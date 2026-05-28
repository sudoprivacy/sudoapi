package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelSquareHandler 处理「模型广场」的两个查询入口：
//
//   - GET /api/v1/public/models           — 未登录可访问，仅返回 standard + 非专属分组定价。
//   - GET /api/v1/models                  — 已登录访问，叠加当前用户可见的专属/订阅分组。
//
// 与 AvailableChannelHandler 的差异：把视角从「渠道→模型」倒置为「模型→平台/端点/分组定价」，
// 给用户一个以模型为主体的浏览页面。
type ModelSquareHandler struct {
	modelSquareSvc *service.ModelSquareService
	apiKeyService  *service.APIKeyService
}

// NewModelSquareHandler 构造 ModelSquareHandler。apiKeyService 用于已登录入口
// 拉取用户可访问的分组 ID 集合，与 AvailableChannelHandler 保持一致。
func NewModelSquareHandler(
	modelSquareSvc *service.ModelSquareService,
	apiKeyService *service.APIKeyService,
) *ModelSquareHandler {
	return &ModelSquareHandler{
		modelSquareSvc: modelSquareSvc,
		apiKeyService:  apiKeyService,
	}
}

// ──────────────────────────────────────────────────────────
// DTO（字段白名单 + JSON snake_case 命名）
//
// 关键安全考量：绝不暴露 ChannelID / 调度元数据 / 内部分组结构；
// 也不暴露用户专属倍率（user_rate_multiplier 仅作为前端 join 的辅助，登录态生效）。
// ──────────────────────────────────────────────────────────

type modelEndpointDTO struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type modelGroupPriceDTO struct {
	GroupID              int64                   `json:"group_id"`
	GroupName            string                  `json:"group_name"`
	SubscriptionType     string                  `json:"subscription_type"`
	IsExclusive          bool                    `json:"is_exclusive"`
	BaseRateMultiplier   float64                 `json:"base_rate_multiplier"`
	UserRateMultiplier   *float64                `json:"user_rate_multiplier"`
	BillingMode          string                  `json:"billing_mode"`
	InputPricePerMTok    *float64                `json:"input_price_per_mtok_usd"`
	OutputPricePerMTok   *float64                `json:"output_price_per_mtok_usd"`
	CacheReadPricePerMT  *float64                `json:"cache_read_price_per_mtok_usd"`
	CacheWritePricePerMT *float64                `json:"cache_write_price_per_mtok_usd"`
	CacheCreation5mPMT   *float64                `json:"cache_creation_5m_price_per_mtok_usd"`
	CacheCreation1hPMT   *float64                `json:"cache_creation_1h_price_per_mtok_usd"`
	ImageOutputPricePM   *float64                `json:"image_output_price_per_mtok_usd"`
	PerRequestPriceUSD   *float64                `json:"per_request_price_usd"`
	Intervals            []modelPriceIntervalDTO `json:"intervals"`
	ChannelChain         []string                `json:"channel_chain"`
}

type modelPriceIntervalDTO struct {
	MinTokens            int      `json:"min_tokens"`
	MaxTokens            *int     `json:"max_tokens"`
	TierLabel            string   `json:"tier_label"`
	InputPricePerMTok    *float64 `json:"input_price_per_mtok_usd"`
	OutputPricePerMTok   *float64 `json:"output_price_per_mtok_usd"`
	CacheReadPricePerMT  *float64 `json:"cache_read_price_per_mtok_usd"`
	CacheWritePricePerMT *float64 `json:"cache_write_price_per_mtok_usd"`
	CacheCreation5mPMT   *float64 `json:"cache_creation_5m_price_per_mtok_usd"`
	CacheCreation1hPMT   *float64 `json:"cache_creation_1h_price_per_mtok_usd"`
	PerRequestPriceUSD   *float64 `json:"per_request_price_usd"`
	SortOrder            int      `json:"sort_order"`
}

type modelOfficialPriceDTO struct {
	InputPricePerMTok       *float64 `json:"input_price_per_mtok_usd"`
	OutputPricePerMTok      *float64 `json:"output_price_per_mtok_usd"`
	CacheReadPricePerMTok   *float64 `json:"cache_read_price_per_mtok_usd"`
	CacheWritePricePerMTok  *float64 `json:"cache_write_price_per_mtok_usd"`
	ImageOutputPricePerMTok *float64 `json:"image_output_price_per_mtok_usd"`
	ImagePriceUSD           *float64 `json:"image_price_usd"`
}

type modelPlatformSectionDTO struct {
	Platform    string               `json:"platform"`
	Endpoints   []modelEndpointDTO   `json:"endpoints"`
	GroupPrices []modelGroupPriceDTO `json:"group_prices"`
}

type modelSquareCardDTO struct {
	Name             string                    `json:"name"`
	DisplayName      string                    `json:"display_name"`
	Category         string                    `json:"category"`
	Description      string                    `json:"description"`
	ModelType        string                    `json:"model_type"`
	ContextWindow    int                       `json:"context_window"`
	MaxOutput        int                       `json:"max_output"`
	Capabilities     []string                  `json:"capabilities"`
	InputModalities  []string                  `json:"input_modalities"`
	OutputModalities []string                  `json:"output_modalities"`
	SupportFlags     []string                  `json:"support_flags"`
	Featured         bool                      `json:"featured"`
	IconURL          string                    `json:"icon_url"`
	OfficialPrice    *modelOfficialPriceDTO    `json:"official_price"`
	Platforms        []modelPlatformSectionDTO `json:"platforms"`
}

type liteLLMModelDTO struct {
	Name                            string         `json:"name"`
	SerialNumber                    *int           `json:"serial_number"`
	Provider                        string         `json:"provider"`
	Mode                            string         `json:"mode"`
	Category                        string         `json:"category"`
	MaxTokens                       int            `json:"max_tokens"`
	MaxInputTokens                  int            `json:"max_input_tokens"`
	MaxOutputTokens                 int            `json:"max_output_tokens"`
	InputPricePerMTok               *float64       `json:"input_price_per_mtok_usd"`
	InputPricePriorityPerMTok       *float64       `json:"input_price_priority_per_mtok_usd"`
	OutputPricePerMTok              *float64       `json:"output_price_per_mtok_usd"`
	OutputPricePriorityPerMTok      *float64       `json:"output_price_priority_per_mtok_usd"`
	CacheCreationPerMTok            *float64       `json:"cache_creation_price_per_mtok_usd"`
	CacheCreationAbove1hPerMTok     *float64       `json:"cache_creation_above_1h_price_per_mtok_usd"`
	CacheReadPerMTok                *float64       `json:"cache_read_price_per_mtok_usd"`
	CacheReadPriorityPerMTok        *float64       `json:"cache_read_priority_price_per_mtok_usd"`
	OutputPricePerImage             *float64       `json:"output_price_per_image_usd"`
	OutputPricePerImageMTok         *float64       `json:"output_price_per_image_mtok_usd"`
	LongContextInputTokenThreshold  int            `json:"long_context_input_token_threshold"`
	LongContextInputCostMultiplier  float64        `json:"long_context_input_cost_multiplier"`
	LongContextOutputCostMultiplier float64        `json:"long_context_output_cost_multiplier"`
	SupportsPromptCaching           bool           `json:"supports_prompt_caching"`
	SupportsServiceTier             bool           `json:"supports_service_tier"`
	SupportedModalities             []string       `json:"supported_modalities"`
	OutputModalities                []string       `json:"output_modalities"`
	SupportFlags                    []string       `json:"support_flags"`
	Capabilities                    []string       `json:"capabilities"`
	RawFields                       map[string]any `json:"raw_fields"`
}

type liteLLMModelListDiagnosticsDTO struct {
	CSVOnlyModels []modelSettingMissingLiteLLMDTO `json:"csv_only_models"`
}

type modelSettingMissingLiteLLMDTO struct {
	SerialNumber int    `json:"serial_number"`
	ID           string `json:"id"`
}

type liteLLMModelListResponseDTO struct {
	Items       []liteLLMModelDTO              `json:"items"`
	Diagnostics liteLLMModelListDiagnosticsDTO `json:"diagnostics"`
}

// ListPublic 处理 GET /api/v1/public/models。
// 不需要 JWT；仅返回 standard 非专属分组的定价。
func (h *ModelSquareHandler) ListPublic(c *gin.Context) {
	cards, err := h.modelSquareSvc.ListPublic(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toCardDTOs(cards, nil))
}

// ListLiteLLM handles GET /api/v1/public/litellm-models.
// It returns the loaded LiteLLM model list directly, without channel filtering.
func (h *ModelSquareHandler) ListLiteLLM(c *gin.Context) {
	result := h.modelSquareSvc.ListLiteLLMModelsWithDiagnostics()
	out := make([]liteLLMModelDTO, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, liteLLMModelDTO{
			Name:                            item.Name,
			SerialNumber:                    item.SerialNumber,
			Provider:                        item.Provider,
			Mode:                            item.Mode,
			Category:                        item.Category,
			MaxTokens:                       item.MaxTokens,
			MaxInputTokens:                  item.MaxInputTokens,
			MaxOutputTokens:                 item.MaxOutputTokens,
			InputPricePerMTok:               positivePerMTokPtr(item.InputCostPerToken),
			InputPricePriorityPerMTok:       positivePerMTokPtr(item.InputCostPerTokenPriority),
			OutputPricePerMTok:              positivePerMTokPtr(item.OutputCostPerToken),
			OutputPricePriorityPerMTok:      positivePerMTokPtr(item.OutputCostPerTokenPriority),
			CacheCreationPerMTok:            positivePerMTokPtr(item.CacheCreationCost),
			CacheCreationAbove1hPerMTok:     positivePerMTokPtr(item.CacheCreationCostAbove1h),
			CacheReadPerMTok:                positivePerMTokPtr(item.CacheReadCost),
			CacheReadPriorityPerMTok:        positivePerMTokPtr(item.CacheReadCostPriority),
			OutputPricePerImage:             positivePtr(item.OutputCostPerImage),
			OutputPricePerImageMTok:         positivePerMTokPtr(item.OutputCostPerImageToken),
			LongContextInputTokenThreshold:  item.LongContextInputTokenThreshold,
			LongContextInputCostMultiplier:  item.LongContextInputCostMultiplier,
			LongContextOutputCostMultiplier: item.LongContextOutputCostMultiplier,
			SupportsPromptCaching:           item.SupportsPromptCaching,
			SupportsServiceTier:             item.SupportsServiceTier,
			SupportedModalities:             emptyStrings(item.SupportedModalities),
			OutputModalities:                emptyStrings(item.OutputModalities),
			SupportFlags:                    emptyStrings(item.SupportFlags),
			Capabilities:                    emptyStrings(item.Capabilities),
			RawFields:                       item.RawFields,
		})
	}
	if c.Query("diagnostics") == "1" || strings.EqualFold(c.Query("diagnostics"), "true") {
		response.Success(c, liteLLMModelListResponseDTO{
			Items:       out,
			Diagnostics: toLiteLLMModelListDiagnosticsDTO(result.Diagnostics),
		})
		return
	}
	response.Success(c, out)
}

// ListAuthenticated 处理 GET /api/v1/models。
// 在 public 基础上叠加用户可访问的专属/订阅分组定价；附带用户专属倍率（如配置）。
func (h *ModelSquareHandler) ListAuthenticated(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userGroups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	allowedGroupIDs := make(map[int64]struct{}, len(userGroups))
	for i := range userGroups {
		allowedGroupIDs[userGroups[i].ID] = struct{}{}
	}

	cards, err := h.modelSquareSvc.ListForUser(c.Request.Context(), subject.UserID, allowedGroupIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 用户专属倍率（per-group）通过 /groups/rates 单独获取，这里不预 join，与 API
	// 密钥页保持一致；前端展示时再叠加倍率，便于实时展示「base × user」拆解。
	response.Success(c, toCardDTOs(cards, nil))
}

// toCardDTOs 把 service 层卡片转换为白名单 DTO。
// userRateMultipliers 为可选 join：group_id → multiplier；nil 时所有行 user_rate_multiplier=nil。
func toCardDTOs(cards []service.ModelSquareCard, userRateMultipliers map[int64]float64) []modelSquareCardDTO {
	out := make([]modelSquareCardDTO, 0, len(cards))
	for _, c := range cards {
		out = append(out, modelSquareCardDTO{
			Name:             c.Name,
			DisplayName:      c.DisplayName,
			Category:         c.Category,
			Description:      c.Description,
			ModelType:        c.ModelType,
			ContextWindow:    c.ContextWindow,
			MaxOutput:        c.MaxOutput,
			Capabilities:     c.Capabilities,
			InputModalities:  emptyStrings(c.InputModalities),
			OutputModalities: emptyStrings(c.OutputModalities),
			SupportFlags:     emptyStrings(c.SupportFlags),
			Featured:         c.Featured,
			IconURL:          c.IconURL,
			OfficialPrice:    toOfficialPriceDTO(c.OfficialPrice),
			Platforms:        toPlatformDTOs(c.Platforms, userRateMultipliers),
		})
	}
	return out
}

func toLiteLLMModelListDiagnosticsDTO(in service.LiteLLMModelListDiagnostics) liteLLMModelListDiagnosticsDTO {
	out := make([]modelSettingMissingLiteLLMDTO, 0, len(in.CSVOnlyModels))
	for _, item := range in.CSVOnlyModels {
		out = append(out, modelSettingMissingLiteLLMDTO{
			SerialNumber: item.SerialNumber,
			ID:           item.ID,
		})
	}
	return liteLLMModelListDiagnosticsDTO{CSVOnlyModels: out}
}

func toOfficialPriceDTO(in *service.ModelOfficialPrice) *modelOfficialPriceDTO {
	if in == nil {
		return nil
	}
	return &modelOfficialPriceDTO{
		InputPricePerMTok:       in.InputPricePerMTok,
		OutputPricePerMTok:      in.OutputPricePerMTok,
		CacheReadPricePerMTok:   in.CacheReadPricePerMTok,
		CacheWritePricePerMTok:  in.CacheWritePricePerMTok,
		ImageOutputPricePerMTok: in.ImageOutputPricePerMTok,
		ImagePriceUSD:           in.ImagePriceUSD,
	}
}

func emptyStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func positivePtr(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	vv := v
	return &vv
}

func positivePerMTokPtr(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	vv := v * 1e6
	return &vv
}

func toPlatformDTOs(in []service.ModelPlatformSection, rates map[int64]float64) []modelPlatformSectionDTO {
	out := make([]modelPlatformSectionDTO, 0, len(in))
	for _, p := range in {
		endpoints := make([]modelEndpointDTO, 0, len(p.Endpoints))
		for _, e := range p.Endpoints {
			endpoints = append(endpoints, modelEndpointDTO{Path: e.Path, Method: e.Method})
		}
		prices := make([]modelGroupPriceDTO, 0, len(p.GroupPrices))
		for _, gp := range p.GroupPrices {
			var userRate *float64
			if rates != nil {
				if v, ok := rates[gp.GroupID]; ok {
					vv := v
					userRate = &vv
				}
			}
			prices = append(prices, modelGroupPriceDTO{
				GroupID:              gp.GroupID,
				GroupName:            gp.GroupName,
				SubscriptionType:     gp.SubscriptionType,
				IsExclusive:          gp.IsExclusive,
				BaseRateMultiplier:   gp.BaseRateMult,
				UserRateMultiplier:   userRate,
				BillingMode:          string(gp.BillingMode),
				InputPricePerMTok:    gp.InputPricePerMTok,
				OutputPricePerMTok:   gp.OutputPricePerMTok,
				CacheReadPricePerMT:  gp.CacheReadPricePerMTok,
				CacheWritePricePerMT: gp.CacheWritePricePerMTok,
				CacheCreation5mPMT:   gp.CacheCreation5mPerMTok,
				CacheCreation1hPMT:   gp.CacheCreation1hPerMTok,
				ImageOutputPricePM:   gp.ImageOutputPricePerMTok,
				PerRequestPriceUSD:   gp.PerRequestPrice,
				Intervals:            toPriceIntervalDTOs(gp.Intervals),
				ChannelChain:         gp.ChannelChain,
			})
		}
		out = append(out, modelPlatformSectionDTO{
			Platform:    p.Platform,
			Endpoints:   endpoints,
			GroupPrices: prices,
		})
	}
	return out
}

func toPriceIntervalDTOs(in []service.ModelGroupPriceInterval) []modelPriceIntervalDTO {
	if in == nil {
		return []modelPriceIntervalDTO{}
	}
	out := make([]modelPriceIntervalDTO, 0, len(in))
	for _, iv := range in {
		out = append(out, modelPriceIntervalDTO{
			MinTokens:            iv.MinTokens,
			MaxTokens:            iv.MaxTokens,
			TierLabel:            iv.TierLabel,
			InputPricePerMTok:    iv.InputPricePerMTok,
			OutputPricePerMTok:   iv.OutputPricePerMTok,
			CacheReadPricePerMT:  iv.CacheReadPricePerMTok,
			CacheWritePricePerMT: iv.CacheWritePricePerMTok,
			CacheCreation5mPMT:   iv.CacheCreation5mPerMTok,
			CacheCreation1hPMT:   iv.CacheCreation1hPerMTok,
			PerRequestPriceUSD:   iv.PerRequestPrice,
			SortOrder:            iv.SortOrder,
		})
	}
	return out
}
