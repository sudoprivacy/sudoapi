package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelSquareScope 决定模型广场展示哪些分组。
//
//   - ScopePublic：仅展示公开 standard 分组（未登录或未传入用户身份的入口）。
//   - ScopeAuthenticated：在 public 基础上叠加用户专属/订阅分组（已登录入口）。
type ModelSquareScope string

const (
	ModelSquareScopePublic        ModelSquareScope = "public"
	ModelSquareScopeAuthenticated ModelSquareScope = "authenticated"
)

// modelSquareCacheTTL 控制 process-local TTL 缓存。
// public scope 量级小、变更慢，60s 足够；authenticated 按 userID key，30s。
const (
	modelSquareCachePublicTTL = 60 * time.Second
	modelSquareCacheAuthTTL   = 30 * time.Second
)

// 模型广场各项端点的金标常量（来自 handler/endpoint.go，避免循环依赖故在此重复定义）。
// 单测断言两侧一致。
const (
	msEndpointMessages          = "/v1/messages"
	msEndpointChatCompletions   = "/v1/chat/completions"
	msEndpointResponses         = "/v1/responses"
	msEndpointImagesGenerations = "/v1/images/generations"
	msEndpointImagesEdits       = "/v1/images/edits"
	msEndpointGeminiModels      = "/v1beta/models"
)

// ModelEndpoint 模型在某平台对外暴露的入站端点（用户实际请求的路径）。
type ModelEndpoint struct {
	Path   string
	Method string
}

// ModelGroupPrice 单个分组下的定价行。
//
// 价格统一转换为「USD per 1M tokens」，前端无需再做单位换算；
// per-request / image 模式时 PerRequestPrice 是「USD per 1 call」。
// 倍率分两层：BaseRateMultiplier 是分组默认，UserRateMultiplier 仅在登录态填值
// （即 ScopeAuthenticated 且当前用户在 /groups/rates 配了专属倍率时）。
type ModelGroupPrice struct {
	GroupID          int64
	GroupName        string
	SubscriptionType string
	IsExclusive      bool
	BaseRateMult     float64
	BillingMode      BillingMode

	// USD per 1M tokens
	InputPricePerMTok       *float64
	OutputPricePerMTok      *float64
	CacheReadPricePerMTok   *float64
	CacheWritePricePerMTok  *float64
	ImageOutputPricePerMTok *float64

	// per_request / image：USD per call
	PerRequestPrice *float64

	// 同模型在多渠道提供时记录调用链路（按渠道名稳定排序），仅展示用。
	ChannelChain []string
}

// ModelPlatformSection 单个模型在某平台下的完整切片。
type ModelPlatformSection struct {
	Platform    string
	Endpoints   []ModelEndpoint
	GroupPrices []ModelGroupPrice
}

// ModelSquareCard 单个模型在广场上的展示卡片。
type ModelSquareCard struct {
	Name          string
	DisplayName   string
	Category      string // claude / gpt / gemini / antigravity / image / embedding / other
	Description   string
	ContextWindow int      // max_input_tokens
	MaxOutput     int      // max_output_tokens
	Capabilities  []string // 派生标签，参见 deriveCapabilities
	Featured      bool
	IconURL       string
	Platforms     []ModelPlatformSection
}

// ModelSquareService 把 ChannelService.ListAvailable 的「渠道→模型」视图倒置为
// 「模型→平台→分组定价」视图，并叠加 LiteLLM 元数据（描述/上下文/能力）。
//
// 计算成本主要在 ChannelService.ListAvailable（已有缓存），上层做一次 map merge，
// 量级 channels × models，毫秒级；为应对首页大流量再加 process-local TTL 缓存。
type ModelSquareService struct {
	channelSvc *ChannelService
	pricingSvc *PricingService

	mu          sync.Mutex
	publicEntry *modelSquareCacheEntry           // 全局共享
	userEntries map[int64]*modelSquareCacheEntry // per-user
}

type modelSquareCacheEntry struct {
	cards    []ModelSquareCard
	loadedAt time.Time
}

// NewModelSquareService 构造 ModelSquareService。
// pricingSvc 可为 nil（测试场景），元数据回落到空值。
func NewModelSquareService(channelSvc *ChannelService, pricingSvc *PricingService) *ModelSquareService {
	return &ModelSquareService{
		channelSvc:  channelSvc,
		pricingSvc:  pricingSvc,
		userEntries: make(map[int64]*modelSquareCacheEntry),
	}
}

// InvalidateAll 清空所有 scope 的缓存。
// 在 ChannelService.Update / GroupService.Update 等修改路径后调用。
func (s *ModelSquareService) InvalidateAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicEntry = nil
	s.userEntries = make(map[int64]*modelSquareCacheEntry)
}

// ListPublic 返回公开 scope（未登录可见）的模型广场卡片列表。
func (s *ModelSquareService) ListPublic(ctx context.Context) ([]ModelSquareCard, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	cached := s.publicEntry
	s.mu.Unlock()
	if cached != nil && time.Since(cached.loadedAt) < modelSquareCachePublicTTL {
		return cloneCards(cached.cards), nil
	}

	cards, err := s.build(ctx, ModelSquareScopePublic, nil)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.publicEntry = &modelSquareCacheEntry{cards: cards, loadedAt: time.Now()}
	s.mu.Unlock()
	return cloneCards(cards), nil
}

// ListForUser 返回已登录 scope 的模型广场卡片列表。
// allowedGroupIDs 是当前用户可访问的分组集合（含 standard + 专属/订阅），
// 由调用方按 APIKeyService.GetAvailableGroups 获取。
func (s *ModelSquareService) ListForUser(
	ctx context.Context,
	userID int64,
	allowedGroupIDs map[int64]struct{},
) ([]ModelSquareCard, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	cached, ok := s.userEntries[userID]
	s.mu.Unlock()
	if ok && cached != nil && time.Since(cached.loadedAt) < modelSquareCacheAuthTTL {
		return cloneCards(cached.cards), nil
	}

	cards, err := s.build(ctx, ModelSquareScopeAuthenticated, allowedGroupIDs)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.userEntries[userID] = &modelSquareCacheEntry{cards: cards, loadedAt: time.Now()}
	s.mu.Unlock()
	return cloneCards(cards), nil
}

// build 是核心 pivot：把 channel-keyed 数据倒置为 model-keyed。
//
// scope == ScopePublic：仅保留 standard + 非专属分组。
// scope == ScopeAuthenticated：在 public 基础上叠加 allowedGroupIDs 内的分组。
//
// 同模型在多个渠道下都被定价时：当前实现把渠道名追加到调用链（ChannelChain），
// 选用第一个命中的渠道作为「主」定价行。SupportedModel 当前未携带渠道名，调用链
// 暂用「渠道索引」表达，前端有 channel_chain 字段即可展示「→ → 」效果。
func (s *ModelSquareService) build(
	ctx context.Context,
	scope ModelSquareScope,
	allowedGroupIDs map[int64]struct{},
) ([]ModelSquareCard, error) {
	channels, err := s.channelSvc.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("model square: list available channels: %w", err)
	}

	// modelKey → card；card.Platforms 按 (platform → section) 进一步索引以便快速合入。
	cardByModel := make(map[string]*modelSquareCardBuilder)

	for _, ch := range channels {
		if ch.Status != StatusActive {
			continue
		}
		visibleGroups := filterGroupsForScope(ch.Groups, scope, allowedGroupIDs)
		if len(visibleGroups) == 0 {
			continue
		}

		// 按平台索引可见分组，避免每个模型再扫一次。
		groupsByPlatform := make(map[string][]AvailableGroupRef, 4)
		for _, g := range visibleGroups {
			if g.Platform == "" {
				continue
			}
			groupsByPlatform[g.Platform] = append(groupsByPlatform[g.Platform], g)
		}
		if len(groupsByPlatform) == 0 {
			continue
		}

		channelLabel := strings.TrimSpace(ch.Name)
		if channelLabel == "" {
			channelLabel = fmt.Sprintf("channel-%d", ch.ID)
		}

		for _, m := range ch.SupportedModels {
			groups, ok := groupsByPlatform[m.Platform]
			if !ok || len(groups) == 0 {
				continue
			}
			modelKey := strings.ToLower(m.Name)
			cb, exists := cardByModel[modelKey]
			if !exists {
				cb = newCardBuilder(m.Name)
				cardByModel[modelKey] = cb
			}

			endpoints := InboundEndpointsForPlatform(m.Platform, modeFromPricing(s.pricingSvc, m.Name))
			section := cb.ensurePlatform(m.Platform, endpoints)

			for _, g := range groups {
				existingIdx := section.findPriceIdx(g.ID)
				if existingIdx >= 0 {
					// 同模型同分组在另一渠道又出现：追加到调用链路（去重，稳定排序）。
					section.GroupPrices[existingIdx].ChannelChain = appendUniqueSorted(
						section.GroupPrices[existingIdx].ChannelChain, channelLabel)
					continue
				}
				price := buildGroupPrice(g, m.Pricing)
				price.ChannelChain = []string{channelLabel}
				section.GroupPrices = append(section.GroupPrices, price)
			}
		}
	}

	// 收尾：把元数据（描述/上下文/能力/category）补齐 + 排序输出。
	cards := make([]ModelSquareCard, 0, len(cardByModel))
	for _, cb := range cardByModel {
		s.fillMetadata(cb)
		cb.sortPlatforms()
		cards = append(cards, cb.toCard())
	}
	sort.SliceStable(cards, func(i, j int) bool {
		if cards[i].Featured != cards[j].Featured {
			return cards[i].Featured // featured first
		}
		if cards[i].Category != cards[j].Category {
			return cards[i].Category < cards[j].Category
		}
		return cards[i].Name < cards[j].Name
	})
	return cards, nil
}

// appendUniqueSorted 把 name 加入 list 并按字典序去重稳定排序（不区分大小写比较，保留原大小写）。
func appendUniqueSorted(list []string, name string) []string {
	for _, s := range list {
		if strings.EqualFold(s, name) {
			return list
		}
	}
	out := append(list, name)
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

// modelSquareCardBuilder 是组装 ModelSquareCard 的中间态，方便平台二级索引。
type modelSquareCardBuilder struct {
	Name           string
	DisplayName    string
	Category       string
	Description    string
	ContextWindow  int
	MaxOutput      int
	Capabilities   []string
	Featured       bool
	IconURL        string
	platformOrder  []string
	platformByName map[string]*ModelPlatformSection
}

func newCardBuilder(modelName string) *modelSquareCardBuilder {
	return &modelSquareCardBuilder{
		Name:           modelName,
		DisplayName:    modelName,
		platformByName: make(map[string]*ModelPlatformSection),
	}
}

func (b *modelSquareCardBuilder) ensurePlatform(platform string, endpoints []ModelEndpoint) *ModelPlatformSection {
	if sec, ok := b.platformByName[platform]; ok {
		return sec
	}
	sec := &ModelPlatformSection{
		Platform:  platform,
		Endpoints: endpoints,
	}
	b.platformByName[platform] = sec
	b.platformOrder = append(b.platformOrder, platform)
	return sec
}

func (b *modelSquareCardBuilder) sortPlatforms() {
	sort.Strings(b.platformOrder)
	for _, sec := range b.platformByName {
		sort.SliceStable(sec.GroupPrices, func(i, j int) bool {
			return strings.ToLower(sec.GroupPrices[i].GroupName) < strings.ToLower(sec.GroupPrices[j].GroupName)
		})
	}
}

func (b *modelSquareCardBuilder) toCard() ModelSquareCard {
	platforms := make([]ModelPlatformSection, 0, len(b.platformOrder))
	for _, p := range b.platformOrder {
		platforms = append(platforms, *b.platformByName[p])
	}
	return ModelSquareCard{
		Name:          b.Name,
		DisplayName:   b.DisplayName,
		Category:      b.Category,
		Description:   b.Description,
		ContextWindow: b.ContextWindow,
		MaxOutput:     b.MaxOutput,
		Capabilities:  b.Capabilities,
		Featured:      b.Featured,
		IconURL:       b.IconURL,
		Platforms:     platforms,
	}
}

// findPriceIdx 返回该平台 section 中 groupID 的索引；不存在返回 -1。
func (sec *ModelPlatformSection) findPriceIdx(groupID int64) int {
	for i, p := range sec.GroupPrices {
		if p.GroupID == groupID {
			return i
		}
	}
	return -1
}

// fillMetadata 根据 LiteLLM 全局数据填充描述、上下文、能力、分类。
// pricingSvc 为 nil 时只做模型名前缀的 category 推断。
func (s *ModelSquareService) fillMetadata(cb *modelSquareCardBuilder) {
	cb.Category = inferCategoryFromName(cb.Name)
	if s == nil || s.pricingSvc == nil {
		return
	}
	lp := s.pricingSvc.GetModelPricing(cb.Name)
	if lp == nil {
		return
	}
	if cb.ContextWindow == 0 {
		cb.ContextWindow = lp.MaxInputTokens
		if cb.ContextWindow == 0 {
			cb.ContextWindow = lp.MaxTokens
		}
	}
	if cb.MaxOutput == 0 {
		cb.MaxOutput = lp.MaxOutputTokens
	}
	caps := deriveCapabilities(lp)
	cb.Capabilities = mergeUniqueStrings(cb.Capabilities, caps)
	// Mode 进一步细化 category（image_generation / embedding 等）。
	if cat := categoryFromMode(lp.Mode); cat != "" && cb.Category == "other" {
		cb.Category = cat
	}
}

// InboundEndpointsForPlatform 给定平台返回该平台对外暴露的入站端点列表。
// 这是 handler/endpoint.go::DeriveUpstreamEndpoint 的反函数，结果用于在
// 模型广场卡片里展示「这个模型可以从哪些端点访问」。
//
// mode 取自 LiteLLM 的 Mode 字段（chat / image_generation / embedding 等），
// 用于在 openai 平台下区分图片模型只暴露 /v1/images/* 端点。
func InboundEndpointsForPlatform(platform, mode string) []ModelEndpoint {
	switch platform {
	case PlatformAnthropic:
		return []ModelEndpoint{{Path: msEndpointMessages, Method: "POST"}}
	case PlatformOpenAI:
		if isImageMode(mode) {
			return []ModelEndpoint{
				{Path: msEndpointImagesGenerations, Method: "POST"},
				{Path: msEndpointImagesEdits, Method: "POST"},
			}
		}
		return []ModelEndpoint{
			{Path: msEndpointChatCompletions, Method: "POST"},
			{Path: msEndpointResponses, Method: "POST"},
		}
	case PlatformGemini:
		return []ModelEndpoint{{Path: msEndpointGeminiModels, Method: "POST"}}
	case PlatformAntigravity:
		// Antigravity 同时接 Claude 与 Gemini 协议。
		return []ModelEndpoint{
			{Path: msEndpointMessages, Method: "POST"},
			{Path: msEndpointGeminiModels, Method: "POST"},
		}
	}
	return nil
}

func isImageMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "image_generation" || m == "image"
}

// modeFromPricing 查模型 mode；pricingSvc 为 nil 或查不到时返回空串。
func modeFromPricing(pricingSvc *PricingService, modelName string) string {
	if pricingSvc == nil {
		return ""
	}
	lp := pricingSvc.GetModelPricing(modelName)
	if lp == nil {
		return ""
	}
	return lp.Mode
}

// filterGroupsForScope 根据 scope 过滤可见分组。
//
//   - ScopePublic：standard + 非专属（IsExclusive=false）。
//   - ScopeAuthenticated：standard + 非专属，叠加 allowedGroupIDs（任何类型，包括订阅/专属）。
//
// 注意：标准非专属分组在 public 已可见，在 authenticated 也仍然要可见 —— 这正是
// allowedGroupIDs 的超集语义（API 密钥页拿到的 allowed 是全集）。这里做并集即可，
// 但为防止 allowedGroupIDs 误配（如未把 standard 全部带回），public 通道独立判定。
func filterGroupsForScope(
	groups []AvailableGroupRef,
	scope ModelSquareScope,
	allowedGroupIDs map[int64]struct{},
) []AvailableGroupRef {
	out := make([]AvailableGroupRef, 0, len(groups))
	for _, g := range groups {
		if isPublicVisibleGroup(g) {
			out = append(out, g)
			continue
		}
		if scope == ModelSquareScopeAuthenticated && allowedGroupIDs != nil {
			if _, ok := allowedGroupIDs[g.ID]; ok {
				out = append(out, g)
			}
		}
	}
	return out
}

func isPublicVisibleGroup(g AvailableGroupRef) bool {
	return !g.IsExclusive && g.SubscriptionType == SubscriptionTypeStandard
}

// buildGroupPrice 把渠道定价转换为模型广场的分组价格行；
// 单位统一为 USD per 1M tokens（per_request / image 模式保持 USD per call）。
func buildGroupPrice(g AvailableGroupRef, p *ChannelModelPricing) ModelGroupPrice {
	row := ModelGroupPrice{
		GroupID:          g.ID,
		GroupName:        g.Name,
		SubscriptionType: g.SubscriptionType,
		IsExclusive:      g.IsExclusive,
		BaseRateMult:     g.RateMultiplier,
		BillingMode:      BillingModeToken,
	}
	if p == nil {
		return row
	}
	if p.BillingMode != "" {
		row.BillingMode = p.BillingMode
	}
	row.InputPricePerMTok = scalePtrPerMillion(p.InputPrice)
	row.OutputPricePerMTok = scalePtrPerMillion(p.OutputPrice)
	row.CacheReadPricePerMTok = scalePtrPerMillion(p.CacheReadPrice)
	row.CacheWritePricePerMTok = scalePtrPerMillion(p.CacheWritePrice)
	row.ImageOutputPricePerMTok = scalePtrPerMillion(p.ImageOutputPrice)
	if p.PerRequestPrice != nil {
		v := *p.PerRequestPrice
		row.PerRequestPrice = &v
	}
	return row
}

// scalePtrPerMillion 把每 token 价格乘 1e6 转为「USD per 1M tokens」。
// nil 维持 nil；0 也视为「未配置」转 nil（与现有 SupportedModelChip 一致）。
func scalePtrPerMillion(v *float64) *float64 {
	if v == nil {
		return nil
	}
	if *v == 0 {
		return nil
	}
	scaled := *v * 1_000_000
	return &scaled
}

// deriveCapabilities 把 LiteLLM 各 Supports* bool 字段映射为前端展示标签。
// 标签 key 不本地化（前端按 key 走 i18n）。
func deriveCapabilities(lp *LiteLLMModelPricing) []string {
	if lp == nil {
		return nil
	}
	out := make([]string, 0, 8)
	if lp.SupportsVision {
		out = append(out, "vision")
	}
	if lp.SupportsFunctionCalling {
		out = append(out, "function_calling")
	}
	if lp.SupportsReasoning {
		out = append(out, "reasoning")
	}
	if lp.SupportsAudioInput {
		out = append(out, "audio_input")
	}
	if lp.SupportsAudioOutput {
		out = append(out, "audio_output")
	}
	if lp.SupportsPDFInput {
		out = append(out, "pdf_input")
	}
	if lp.SupportsPromptCaching {
		out = append(out, "prompt_caching")
	}
	if lp.SupportsParallelTools {
		out = append(out, "parallel_tools")
	}
	return out
}

// inferCategoryFromName 用模型名前缀推断分类。
// 与后台元数据覆盖时的 category 字段语义一致；新模型未来自动归类。
func inferCategoryFromName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasPrefix(n, "claude"):
		return "claude"
	case strings.HasPrefix(n, "gpt") || strings.HasPrefix(n, "o1") || strings.HasPrefix(n, "o3") || strings.HasPrefix(n, "o4") || strings.HasPrefix(n, "chatgpt") || strings.HasPrefix(n, "codex"):
		return "gpt"
	case strings.HasPrefix(n, "gemini"):
		return "gemini"
	case strings.HasPrefix(n, "dall-e") || strings.HasPrefix(n, "dalle") || strings.HasPrefix(n, "imagen") || strings.HasPrefix(n, "stable-"):
		return "image"
	case strings.HasPrefix(n, "text-embedding") || strings.Contains(n, "embedding"):
		return "embedding"
	}
	return "other"
}

func categoryFromMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "image_generation", "image":
		return "image"
	case "embedding":
		return "embedding"
	case "audio_transcription", "audio_speech":
		return "audio"
	}
	return ""
}

// mergeUniqueStrings 合并两份字符串切片去重，保持出现顺序。
func mergeUniqueStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range b {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// cloneCards 返回深拷贝，避免缓存值被调用方意外修改。
func cloneCards(in []ModelSquareCard) []ModelSquareCard {
	out := make([]ModelSquareCard, len(in))
	for i, c := range in {
		out[i] = c
		if c.Capabilities != nil {
			out[i].Capabilities = append([]string(nil), c.Capabilities...)
		}
		if c.Platforms != nil {
			out[i].Platforms = make([]ModelPlatformSection, len(c.Platforms))
			for j, p := range c.Platforms {
				out[i].Platforms[j] = p
				if p.Endpoints != nil {
					out[i].Platforms[j].Endpoints = append([]ModelEndpoint(nil), p.Endpoints...)
				}
				if p.GroupPrices != nil {
					out[i].Platforms[j].GroupPrices = make([]ModelGroupPrice, len(p.GroupPrices))
					for k, gp := range p.GroupPrices {
						out[i].Platforms[j].GroupPrices[k] = gp
						if gp.ChannelChain != nil {
							out[i].Platforms[j].GroupPrices[k].ChannelChain = append([]string(nil), gp.ChannelChain...)
						}
					}
				}
			}
		}
	}
	return out
}
