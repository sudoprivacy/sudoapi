// sudoapi: Model catalog.

package service_model_catalog

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// NewModelCatalogService 构造 ModelCatalogService。
// pricingSvc、metadata、endpointConfig 均可为 nil（测试/默认配置场景）。
func NewModelCatalogService(
	channelSvc *service.ChannelService,
	pricingSvc PricingService,
	metadata MetadataOverrideReader,
	endpointConfig EndpointConfigReader,
) *ModelCatalogService {
	return &ModelCatalogService{
		channelSvc:     channelSvc,
		pricingSvc:     pricingSvc,
		metadata:       metadata,
		endpointConfig: endpointConfig,
		userEntries:    make(map[int64]*cacheEntry),
	}
}

// InvalidateAll 清空所有 scope 的缓存。
// 在 ChannelService.Update / GroupService.Update 等修改路径后调用。
func (s *ModelCatalogService) InvalidateAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userEntries = make(map[int64]*cacheEntry)
}

// ListForUser 返回已登录 scope 的模型目录条目列表。
// allowedGroupIDs 是当前用户可访问的分组集合（含 standard + 专属/订阅），
// 由调用方按 APIKeyService.GetAvailableGroups 获取。
func (s *ModelCatalogService) ListForUser(ctx context.Context, userID int64, allowedGroupIDs map[int64]struct{}) ([]Card, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.Lock()
	cached, ok := s.userEntries[userID]
	s.mu.Unlock()
	if ok && cached != nil && time.Since(cached.loadedAt) < modelCatalogCacheAuthTTL {
		return cloneCards(cached.cards), nil
	}

	cards, err := s.build(ctx, allowedGroupIDs)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.userEntries[userID] = &cacheEntry{cards: cards, loadedAt: time.Now()}
	s.mu.Unlock()
	return cloneCards(cards), nil
}

// build 是核心 pivot：把 channel-keyed 数据倒置为 model-keyed。
// 登录态默认展示 standard 非专属分组，并叠加 allowedGroupIDs 内的专属/订阅分组。
//
// 同模型在多个渠道下都被定价时：当前实现把渠道名追加到调用链（ChannelChain），
// 选用第一个命中的渠道作为「主」定价行。SupportedModel 当前未携带渠道名，调用链
// 暂用「渠道索引」表达，前端有 channel_chain 字段即可展示「→ → 」效果。
func (s *ModelCatalogService) build(
	ctx context.Context,
	allowedGroupIDs map[int64]struct{},
) ([]Card, error) {
	channels, err := s.channelSvc.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("model catalog list available channels: %w", err)
	}

	// modelKey → card；card.Platforms 按 (platform → section) 进一步索引以便快速合入。
	cardByModel := make(map[string]*modelCatalogCardBuilder)

	for _, ch := range channels {
		if ch.Status != service.StatusActive {
			continue
		}
		visibleGroups := filterGroupsForUser(ch.Groups, allowedGroupIDs)
		if len(visibleGroups) == 0 {
			continue
		}

		// 按平台索引可见分组，避免每个模型再扫一次。
		groupsByPlatform := lo.GroupBy(
			lo.Filter(visibleGroups, func(g service.AvailableGroupRef, _ int) bool { return g.Platform != "" }),
			func(g service.AvailableGroupRef) string { return g.Platform },
		)
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

			section := cb.ensurePlatform(m.Platform, nil)

			for _, g := range groups {
				_, existingIdx, find := lo.FindIndexOf(section.GroupPrices, func(p ModelGroupPrice) bool {
					return p.GroupID == g.ID
				})
				if find {
					// 同模型同分组在另一渠道又出现：追加到调用链路（去重，稳定排序）。
					section.GroupPrices[existingIdx].ChannelChain = appendUniqueSorted(section.GroupPrices[existingIdx].ChannelChain, channelLabel)
					continue
				}
				price := buildGroupPrice(g, m.Pricing)
				price.ChannelChain = []string{channelLabel}
				section.GroupPrices = append(section.GroupPrices, price)
			}
		}
	}

	overrides, err := s.loadMetadataOverrides(ctx, cardByModel)
	if err != nil {
		return nil, err
	}
	endpointConfig, err := s.loadEndpointConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 收尾：把元数据（描述/上下文/能力/category）补齐 + 管理员覆盖 + 排序输出。
	cards := make([]Card, 0, len(cardByModel))
	for _, cb := range cardByModel {
		fillModelCatalogMetadata(s.pricingSvc, cb)

		applyModelCatalogMetadataOverride(cb, overrides[normalizeMetadataModelKey(cb.Name)])
		cb.applyEndpointConfig(endpointConfig)
		cb.sortPlatforms()
		cards = append(cards, cb.toCard())
	}
	slices.SortStableFunc(cards, func(a, b Card) int {
		if a.Featured != b.Featured {
			if a.Featured {
				return -1 // featured first
			}
			return 1
		}
		leftCategory := strings.ToLower(a.Category)
		rightCategory := strings.ToLower(b.Category)
		if leftCategory != rightCategory {
			return strings.Compare(leftCategory, rightCategory)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return cards, nil
}

func (s *ModelCatalogService) loadMetadataOverrides(ctx context.Context, cards map[string]*modelCatalogCardBuilder) (map[string]*MetadataOverride, error) {
	if s == nil || s.metadata == nil || len(cards) == 0 {
		return map[string]*MetadataOverride{}, nil
	}
	modelNames := lo.MapToSlice(cards, func(_ string, cb *modelCatalogCardBuilder) string {
		return cb.Name
	})
	overrides, err := s.metadata.GetOverridesByModelNames(ctx, modelNames)
	if err != nil {
		return nil, fmt.Errorf("model catalog load metadata overrides: %w", err)
	}
	return overrides, nil
}

func (s *ModelCatalogService) loadEndpointConfig(ctx context.Context) (*EndpointConfig, error) {
	if s == nil || s.endpointConfig == nil {
		return defaultModelCatalogEndpointConfig(), nil
	}
	cfg, err := s.endpointConfig.GetEndpointConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("model catalog load endpoint config: %w", err)
	}
	if cfg == nil {
		return &EndpointConfig{Platforms: map[string]map[string][]Endpoint{}}, nil
	}
	return cfg, nil
}

const (
	// modelCatalogCacheTTL 控制 process-local TTL 缓存。
	// authenticated 按 userID key，30s。
	modelCatalogCacheAuthTTL = 30 * time.Second
)

type (

	// ModelCatalogService 把 ChannelService.ListAvailable 的「渠道→模型」视图倒置为
	// 「模型→平台→分组定价」视图，并叠加 LiteLLM 元数据（描述/上下文/能力）。
	//
	// 计算成本主要在 ChannelService.ListAvailable（已有缓存），上层做一次 map merge，
	// 量级 channels × models，毫秒级；为应对首页大流量再加 process-local TTL 缓存。
	ModelCatalogService struct {
		channelSvc *service.ChannelService
		pricingSvc PricingService

		metadata       MetadataOverrideReader
		endpointConfig EndpointConfigReader

		mu sync.Mutex
		// userEntries 按 userID 索引。
		userEntries map[int64]*cacheEntry
	}

	// modelCatalogCardBuilder 是组装 Card 的中间态，方便平台二级索引。
	modelCatalogCardBuilder struct {
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

		ModelType        string
		InputModalities  []string
		OutputModalities []string
		SupportFlags     []string
		OfficialPrice    *ModelOfficialPrice
	}

	PricingService interface {
		GetModelPricing(modelName string) *service.LiteLLMModelPricing
	}

	cacheEntry struct {
		cards    []Card
		loadedAt time.Time
	}

	// Card 单个模型在广场上的展示卡片。
	//
	// Category 默认由模型名或 LiteLLM mode 推断，也可被后台元数据覆盖为自定义展示分类。
	// ContextWindow / MaxOutput 来自 LiteLLM 的 max_input_tokens / max_output_tokens。
	// Capabilities 是派生标签（参见 deriveCapabilities），key 由前端 i18n 渲染。
	Card struct {
		Name          string
		DisplayName   string
		Category      string
		Description   string
		ContextWindow int
		MaxOutput     int
		Capabilities  []string
		Featured      bool
		IconURL       string
		Platforms     []ModelPlatformSection

		ModelType        string
		InputModalities  []string
		OutputModalities []string
		SupportFlags     []string
		OfficialPrice    *ModelOfficialPrice
	}
	// ModelPlatformSection 单个模型在某平台下的完整切片。
	ModelPlatformSection struct {
		Platform    string
		Endpoints   []Endpoint
		GroupPrices []ModelGroupPrice
	}
	// ModelGroupPrice 单个分组下的定价行。
	//
	// 价格统一转换为「USD per 1M tokens」，前端无需再做单位换算；
	// per-request / image 模式时 PerRequestPrice 是「USD per 1 call」。
	// 倍率分两层：BaseRateMultiplier 是分组默认，UserRateMultiplier 仅在登录态填值
	// （即 ScopeAuthenticated 且当前用户在 /groups/rates 配了专属倍率时）。
	ModelGroupPrice struct {
		GroupID          int64
		GroupName        string
		SubscriptionType string
		IsExclusive      bool
		BaseRateMult     float64
		BillingMode      service.BillingMode

		// USD per 1M tokens
		InputPricePerMTok       *float64
		OutputPricePerMTok      *float64
		CacheReadPricePerMTok   *float64
		CacheWritePricePerMTok  *float64
		ImageOutputPricePerMTok *float64

		// per_request / image：USD per call
		PerRequestPrice *float64

		// 上下文区间定价。token 模式价格统一为 USD per 1M tokens；
		// per_request / image 模式价格保持 USD per call。
		Intervals []ModelGroupPriceInterval

		// 同模型在多渠道提供时记录调用链路（按渠道名稳定排序），仅展示用。
		ChannelChain []string

		CacheCreation5mPerMTok *float64
		CacheCreation1hPerMTok *float64
	}
	// ModelGroupPriceInterval 单个分组价格下的上下文区间。
	ModelGroupPriceInterval struct {
		MinTokens int
		MaxTokens *int
		TierLabel string

		// USD per 1M tokens
		InputPricePerMTok      *float64
		OutputPricePerMTok     *float64
		CacheReadPricePerMTok  *float64
		CacheWritePricePerMTok *float64

		// per_request / image：USD per call
		PerRequestPrice *float64

		SortOrder int

		CacheCreation5mPerMTok *float64
		CacheCreation1hPerMTok *float64
	}
	// ModelOfficialPrice is the LiteLLM reference price for a model.
	//
	// Token prices are normalized to USD per 1M tokens for display. ImagePriceUSD
	// keeps LiteLLM's per-image unit where available.
	ModelOfficialPrice struct {
		InputPricePerMTok       *float64
		OutputPricePerMTok      *float64
		CacheReadPricePerMTok   *float64
		CacheWritePricePerMTok  *float64
		ImageOutputPricePerMTok *float64
		ImagePriceUSD           *float64
	}
)

// appendUniqueSorted 把 name 加入 list 并按字典序去重稳定排序（不区分大小写比较，保留原大小写）。
func appendUniqueSorted(list []string, name string) []string {
	if lo.ContainsBy(list, func(s string) bool {
		return strings.EqualFold(s, name)
	}) {
		return list
	}
	out := append(list, name)
	slices.SortStableFunc(out, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return out
}

func newCardBuilder(modelName string) *modelCatalogCardBuilder {
	return &modelCatalogCardBuilder{
		Name:           modelName,
		DisplayName:    modelName,
		platformByName: make(map[string]*ModelPlatformSection),
	}
}

func (b *modelCatalogCardBuilder) ensurePlatform(platform string, endpoints []Endpoint) *ModelPlatformSection {
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

func (b *modelCatalogCardBuilder) sortPlatforms() {
	slices.Sort(b.platformOrder)
	for _, sec := range b.platformByName {
		slices.SortStableFunc(sec.GroupPrices, func(a, b ModelGroupPrice) int {
			return strings.Compare(strings.ToLower(a.GroupName), strings.ToLower(b.GroupName))
		})
	}
}

func (b *modelCatalogCardBuilder) applyEndpointConfig(cfg *EndpointConfig) {
	if b == nil {
		return
	}
	for _, platform := range b.platformOrder {
		if sec := b.platformByName[platform]; sec != nil {
			sec.Endpoints = ResolveEndpoints(cfg, platform, b.ModelType)
		}
	}
}

func (b *modelCatalogCardBuilder) toCard() Card {
	platforms := lo.Map(b.platformOrder, func(platform string, _ int) ModelPlatformSection {
		return *b.platformByName[platform]
	})
	return Card{
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

		ModelType:        b.ModelType,
		InputModalities:  b.InputModalities,
		OutputModalities: b.OutputModalities,
		SupportFlags:     b.SupportFlags,
		OfficialPrice:    cloneOfficialPrice(b.OfficialPrice),
	}
}

func fillModelCatalogMetadata(pricingSvc PricingService, cb *modelCatalogCardBuilder) {
	if cb == nil {
		return
	}
	cb.Category = inferCategoryFromName(cb.Name)
	if pricingSvc == nil {
		return
	}
	lp := pricingSvc.GetModelPricing(cb.Name)
	if lp == nil {
		return
	}
	cb.OfficialPrice = officialPriceFromLiteLLM(lp)
	if cb.ModelType == "" {
		cb.ModelType = strings.TrimSpace(lp.Mode)
	}
	if len(cb.InputModalities) == 0 {
		cb.InputModalities = slices.Clone(lp.SupportedModalities)
	}
	if len(cb.OutputModalities) == 0 {
		cb.OutputModalities = slices.Clone(lp.SupportedOutputModalities)
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
	cb.Capabilities = lo.Uniq(slices.Concat(cb.Capabilities, deriveCapabilities(lp)))
	cb.SupportFlags = lo.Uniq(slices.Concat(cb.SupportFlags, lp.SupportFlags))

	// Mode 进一步细化 category（image_generation / embedding 等）。
	if cb.Category == "other" {
		switch strings.ToLower(strings.TrimSpace(lp.Mode)) {
		case "image_generation", "image":
			cb.Category = "image"
		case "embedding":
			cb.Category = "embedding"
		case "audio_transcription", "audio_speech":
			cb.Category = "audio"
		}
	}
}

func applyModelCatalogMetadataOverride(cb *modelCatalogCardBuilder, override *MetadataOverride) {
	if cb == nil || override == nil {
		return
	}
	if v := strings.TrimSpace(override.DisplayName); v != "" {
		cb.DisplayName = v
	}
	if v := strings.TrimSpace(override.Description); v != "" {
		cb.Description = v
	}
	if v := strings.TrimSpace(override.ModelType); v != "" {
		cb.ModelType = v
	}
	if v := strings.TrimSpace(override.Category); v != "" {
		cb.Category = v
	}
	if override.ContextWindow > 0 {
		cb.ContextWindow = override.ContextWindow
	}
	if override.MaxOutput > 0 {
		cb.MaxOutput = override.MaxOutput
	}
	if len(override.Capabilities) > 0 {
		cb.Capabilities = slices.Clone(override.Capabilities)
	}
	if len(override.InputModalities) > 0 {
		cb.InputModalities = slices.Clone(override.InputModalities)
	}
	if len(override.OutputModalities) > 0 {
		cb.OutputModalities = slices.Clone(override.OutputModalities)
	}
	if len(override.SupportFlags) > 0 {
		cb.SupportFlags = slices.Clone(override.SupportFlags)
	}
	cb.Featured = override.Featured
	if v := strings.TrimSpace(override.IconURL); v != "" {
		cb.IconURL = v
	}
}

// InboundEndpointsForPlatform 给定平台返回该平台对外暴露的入站端点列表。
// 这是 handler/endpoint.go::DeriveUpstreamEndpoint 的反函数，结果用于在
// 模型目录条目里展示「这个模型可以从哪些端点访问」。
//
// mode 取自 LiteLLM 的 Mode 字段（chat / image_generation / embedding 等），
// 用于在 openai 平台下区分图片模型只暴露 /v1/images/* 端点。
func InboundEndpointsForPlatform(platform, mode string) []Endpoint {
	switch platform {
	case service.PlatformAnthropic:
		return []Endpoint{{Path: endpointMessages, Method: "POST"}}
	case service.PlatformOpenAI:
		if isImageMode(mode) {
			return []Endpoint{
				{Path: endpointImagesGenerations, Method: "POST"},
				{Path: endpointImagesEdits, Method: "POST"},
			}
		}
		return []Endpoint{
			{Path: endpointChatCompletions, Method: "POST"},
			{Path: endpointResponses, Method: "POST"},
		}
	case service.PlatformGemini:
		return []Endpoint{{Path: endpointGeminiModels, Method: "POST"}}
	case service.PlatformAntigravity:
		// Antigravity 同时接 Claude 与 Gemini 协议。
		return []Endpoint{
			{Path: endpointMessages, Method: "POST"},
			{Path: endpointGeminiModels, Method: "POST"},
		}
	}
	return nil
}

func isImageMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "image_generation" || m == "image"
}

// filterGroupsForUser 保留 standard 非专属分组，并叠加 allowedGroupIDs 内的专属/订阅分组。
func filterGroupsForUser(
	groups []service.AvailableGroupRef,
	allowedGroupIDs map[int64]struct{},
) []service.AvailableGroupRef {
	return lo.Filter(groups, func(g service.AvailableGroupRef, _ int) bool {
		if isPublicVisibleGroup(g) {
			return true
		}
		if allowedGroupIDs != nil {
			if _, ok := allowedGroupIDs[g.ID]; ok {
				return true
			}
		}
		return false
	})
}

func isPublicVisibleGroup(g service.AvailableGroupRef) bool {
	return !g.IsExclusive && g.SubscriptionType == service.SubscriptionTypeStandard
}

// buildGroupPrice 把渠道定价转换为模型目录的分组价格行；
// 单位统一为 USD per 1M tokens（per_request / image 模式保持 USD per call）。
func buildGroupPrice(g service.AvailableGroupRef, p *service.ChannelModelPricing) ModelGroupPrice {
	row := ModelGroupPrice{
		GroupID:          g.ID,
		GroupName:        g.Name,
		SubscriptionType: g.SubscriptionType,
		IsExclusive:      g.IsExclusive,
		BaseRateMult:     g.RateMultiplier,
		BillingMode:      service.BillingModeToken,
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
		row.PerRequestPrice = new(*p.PerRequestPrice)
	}
	row.Intervals = lo.FilterMap(p.Intervals, func(iv service.PricingInterval, _ int) (ModelGroupPriceInterval, bool) {
		if iv.InputPrice == nil && iv.OutputPrice == nil &&
			iv.CacheWritePrice == nil && iv.CacheReadPrice == nil &&
			iv.CacheCreation5mPrice == nil && iv.CacheCreation1hPrice == nil &&
			iv.PerRequestPrice == nil {
			return ModelGroupPriceInterval{}, false
		}
		return ModelGroupPriceInterval{
			MinTokens:              iv.MinTokens,
			MaxTokens:              iv.MaxTokens,
			TierLabel:              iv.TierLabel,
			InputPricePerMTok:      scaleIntervalPtrPerMillion(iv.InputPrice),
			OutputPricePerMTok:     scaleIntervalPtrPerMillion(iv.OutputPrice),
			CacheReadPricePerMTok:  scaleIntervalPtrPerMillion(iv.CacheReadPrice),
			CacheWritePricePerMTok: scaleIntervalPtrPerMillion(iv.CacheWritePrice),
			PerRequestPrice:        cloneFloatPtr(iv.PerRequestPrice),
			SortOrder:              iv.SortOrder,
			CacheCreation5mPerMTok: scaleIntervalPtrPerMillion(iv.CacheCreation5mPrice),
			CacheCreation1hPerMTok: scaleIntervalPtrPerMillion(iv.CacheCreation1hPrice),
		}, true
	})
	row.CacheCreation5mPerMTok = scalePtrPerMillion(p.CacheCreation5mPrice)
	row.CacheCreation1hPerMTok = scalePtrPerMillion(p.CacheCreation1hPrice)
	return row
}

func officialPriceFromLiteLLM(lp *service.LiteLLMModelPricing) *ModelOfficialPrice {
	const million = 1_000_000
	if lp == nil {
		return nil
	}
	return &ModelOfficialPrice{
		InputPricePerMTok:       new(lp.InputCostPerToken * million),
		OutputPricePerMTok:      new(lp.OutputCostPerToken * million),
		CacheReadPricePerMTok:   new(lp.CacheReadInputTokenCost * million),
		CacheWritePricePerMTok:  new(lp.CacheCreationInputTokenCost * million),
		ImageOutputPricePerMTok: new(lp.OutputCostPerImageToken * million),
		ImagePriceUSD:           lo.Ternary(lp.OutputCostPerImage == 0, nil, new(lp.OutputCostPerImage)),
	}
}

// scaleIntervalPtrPerMillion 用于区间展示，保留显式 0 价格以表达免费区间。
func scaleIntervalPtrPerMillion(v *float64) *float64 {
	if v == nil {
		return nil
	}
	return new(*v * 1_000_000)
}

// scalePtrPerMillion 把每 token 价格乘 1e6 转为「USD per 1M tokens」。
// nil 维持 nil；0 也视为「未配置」转 nil（与现有 SupportedModelChip 一致）。
func scalePtrPerMillion(v *float64) *float64 {
	if v == nil || *v == 0 {
		return nil
	}
	return new(*v * 1_000_000)
}

// deriveCapabilities 把 LiteLLM 各 Supports* bool 字段映射为前端展示标签。
// 标签 key 不本地化（前端按 key 走 i18n）。
func deriveCapabilities(lp *service.LiteLLMModelPricing) []string {
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

// cloneCards 返回深拷贝，避免缓存值被调用方意外修改。
func cloneCards(in []Card) []Card {
	out := make([]Card, len(in))
	for i, c := range in {
		out[i] = c
		out[i].OfficialPrice = cloneOfficialPrice(c.OfficialPrice)
		if c.Capabilities != nil {
			out[i].Capabilities = slices.Clone(c.Capabilities)
		}
		if c.InputModalities != nil {
			out[i].InputModalities = slices.Clone(c.InputModalities)
		}
		if c.OutputModalities != nil {
			out[i].OutputModalities = slices.Clone(c.OutputModalities)
		}
		if c.SupportFlags != nil {
			out[i].SupportFlags = slices.Clone(c.SupportFlags)
		}
		if c.Platforms != nil {
			out[i].Platforms = make([]ModelPlatformSection, len(c.Platforms))
			for j, p := range c.Platforms {
				out[i].Platforms[j] = p
				if p.Endpoints != nil {
					out[i].Platforms[j].Endpoints = slices.Clone(p.Endpoints)
				}
				if p.GroupPrices != nil {
					out[i].Platforms[j].GroupPrices = make([]ModelGroupPrice, len(p.GroupPrices))
					for k, gp := range p.GroupPrices {
						out[i].Platforms[j].GroupPrices[k] = gp
						if gp.Intervals != nil {
							out[i].Platforms[j].GroupPrices[k].Intervals = slices.Clone(gp.Intervals)
						}
						if gp.ChannelChain != nil {
							out[i].Platforms[j].GroupPrices[k].ChannelChain = slices.Clone(gp.ChannelChain)
						}
					}
				}
			}
		}
	}
	return out
}

func cloneOfficialPrice(in *ModelOfficialPrice) *ModelOfficialPrice {
	if in == nil {
		return nil
	}
	return &ModelOfficialPrice{
		InputPricePerMTok:       cloneFloatPtr(in.InputPricePerMTok),
		OutputPricePerMTok:      cloneFloatPtr(in.OutputPricePerMTok),
		CacheReadPricePerMTok:   cloneFloatPtr(in.CacheReadPricePerMTok),
		CacheWritePricePerMTok:  cloneFloatPtr(in.CacheWritePricePerMTok),
		ImageOutputPricePerMTok: cloneFloatPtr(in.ImageOutputPricePerMTok),
		ImagePriceUSD:           cloneFloatPtr(in.ImagePriceUSD),
	}
}

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	return new(*v)
}
