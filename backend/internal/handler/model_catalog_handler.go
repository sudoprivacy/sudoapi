// sudoapi: Model catalog.

package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service_model_catalog"
)

// ModelCatalogHandler 处理「模型目录」查询入口：
//
//   - GET /api/v1/models — 已登录访问，叠加当前用户可见的专属/订阅分组。
//
// 与 AvailableChannelHandler 的差异：把视角从「渠道→模型」倒置为「模型→平台/端点/分组定价」，
// 给用户一个以模型为主体的浏览页面。
type ModelCatalogHandler struct {
	catalogService *service_model_catalog.ModelCatalogService
	apiKeyService  *service.APIKeyService
}

// NewModelCatalogHandler 构造 ModelCatalogHandler。apiKeyService 用于已登录入口
// 拉取用户可访问的分组 ID 集合，与 AvailableChannelHandler 保持一致。
func NewModelCatalogHandler(modelCatalogSvc *service_model_catalog.ModelCatalogService, apiKeyService *service.APIKeyService) *ModelCatalogHandler {
	return &ModelCatalogHandler{catalogService: modelCatalogSvc, apiKeyService: apiKeyService}
}

// List 处理 GET /api/v1/models。
// 展示 standard 非专属分组，并叠加用户可访问的专属/订阅分组定价；附带用户专属倍率（如配置）。
func (h *ModelCatalogHandler) List(c *gin.Context) {
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

	cards, err := h.catalogService.ListForUser(c.Request.Context(), subject.UserID, allowedGroupIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	userRateMultipliers, err := h.apiKeyService.GetUserGroupRates(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toCardDTOs(cards, userRateMultipliers))
}

type (
	/*
	   ──────────────────────────────────────────────────────────
	   DTO（字段白名单 + JSON snake_case 命名）

	   关键安全考量：绝不暴露 ChannelID / 调度元数据 / 内部分组结构；
	   也不暴露用户专属倍率（user_rate_multiplier 仅作为前端 join 的辅助，登录态生效）。
	   ──────────────────────────────────────────────────────────
	*/

	ModelPlatformSectionDTO struct {
		Platform    string               `json:"platform"`
		Endpoints   []ModelEndpointDTO   `json:"endpoints"`
		GroupPrices []ModelGroupPriceDTO `json:"group_prices"`
	}
	ModelEndpointDTO struct {
		Path   string `json:"path"`
		Method string `json:"method"`
	}
	ModelGroupPriceDTO struct {
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
		ImageOutputPricePM   *float64                `json:"image_output_price_per_mtok_usd"`
		PerRequestPriceUSD   *float64                `json:"per_request_price_usd"`
		ChannelChain         []string                `json:"channel_chain"`
		Intervals            []ModelPriceIntervalDTO `json:"intervals"`
		CacheCreation5mPMT   *float64                `json:"cache_creation_5m_price_per_mtok_usd"`
		CacheCreation1hPMT   *float64                `json:"cache_creation_1h_price_per_mtok_usd"`
	}
	ModelPriceIntervalDTO struct {
		MinTokens            int      `json:"min_tokens"`
		MaxTokens            *int     `json:"max_tokens"`
		TierLabel            string   `json:"tier_label"`
		InputPricePerMTok    *float64 `json:"input_price_per_mtok_usd"`
		OutputPricePerMTok   *float64 `json:"output_price_per_mtok_usd"`
		CacheReadPricePerMT  *float64 `json:"cache_read_price_per_mtok_usd"`
		CacheWritePricePerMT *float64 `json:"cache_write_price_per_mtok_usd"`
		PerRequestPriceUSD   *float64 `json:"per_request_price_usd"`
		SortOrder            int      `json:"sort_order"`
		CacheCreation5mPMT   *float64 `json:"cache_creation_5m_price_per_mtok_usd"`
		CacheCreation1hPMT   *float64 `json:"cache_creation_1h_price_per_mtok_usd"`
	}
	ModelCatalogCardDTO struct {
		Name             string                    `json:"name"`
		DisplayName      string                    `json:"display_name"`
		Category         string                    `json:"category"`
		Description      string                    `json:"description"`
		ContextWindow    int                       `json:"context_window"`
		MaxOutput        int                       `json:"max_output"`
		Capabilities     []string                  `json:"capabilities"`
		Featured         bool                      `json:"featured"`
		IconURL          string                    `json:"icon_url"`
		Platforms        []ModelPlatformSectionDTO `json:"platforms"`
		ModelType        string                    `json:"model_type"`
		InputModalities  []string                  `json:"input_modalities"`
		OutputModalities []string                  `json:"output_modalities"`
		SupportFlags     []string                  `json:"support_flags"`
		OfficialPrice    *ModelOfficialPriceDTO    `json:"official_price"`
	}
	ModelOfficialPriceDTO struct {
		InputPricePerMTok       *float64 `json:"input_price_per_mtok_usd"`
		OutputPricePerMTok      *float64 `json:"output_price_per_mtok_usd"`
		CacheReadPricePerMTok   *float64 `json:"cache_read_price_per_mtok_usd"`
		CacheWritePricePerMTok  *float64 `json:"cache_write_price_per_mtok_usd"`
		ImageOutputPricePerMTok *float64 `json:"image_output_price_per_mtok_usd"`
		ImagePriceUSD           *float64 `json:"image_price_usd"`
	}
)

// toCardDTOs 把 service 层卡片转换为白名单 DTO。
// userRateMultipliers 为可选 join：group_id → multiplier；nil 时所有行 user_rate_multiplier=nil。
func toCardDTOs(cards []service_model_catalog.Card, userRateMultipliers map[int64]float64) []ModelCatalogCardDTO {
	out := make([]ModelCatalogCardDTO, 0, len(cards))
	for _, c := range cards {
		out = append(out, ModelCatalogCardDTO{
			Name:             c.Name,
			DisplayName:      c.DisplayName,
			Category:         c.Category,
			Description:      c.Description,
			ContextWindow:    c.ContextWindow,
			MaxOutput:        c.MaxOutput,
			Capabilities:     c.Capabilities,
			Featured:         c.Featured,
			IconURL:          c.IconURL,
			Platforms:        toPlatformDTOs(c.Platforms, userRateMultipliers),
			ModelType:        c.ModelType,
			InputModalities:  c.InputModalities,
			OutputModalities: c.OutputModalities,
			SupportFlags:     c.SupportFlags,
			OfficialPrice:    toOfficialPriceDTO(c.OfficialPrice),
		})
	}
	return out
}

func toOfficialPriceDTO(in *service_model_catalog.ModelOfficialPrice) *ModelOfficialPriceDTO {
	if in == nil {
		return nil
	}
	return &ModelOfficialPriceDTO{
		InputPricePerMTok:       in.InputPricePerMTok,
		OutputPricePerMTok:      in.OutputPricePerMTok,
		CacheReadPricePerMTok:   in.CacheReadPricePerMTok,
		CacheWritePricePerMTok:  in.CacheWritePricePerMTok,
		ImageOutputPricePerMTok: in.ImageOutputPricePerMTok,
		ImagePriceUSD:           in.ImagePriceUSD,
	}
}

func toPlatformDTOs(in []service_model_catalog.ModelPlatformSection, rates map[int64]float64) []ModelPlatformSectionDTO {
	return lo.Map(in, func(p service_model_catalog.ModelPlatformSection, _ int) ModelPlatformSectionDTO {
		return ModelPlatformSectionDTO{
			Platform: p.Platform,
			Endpoints: lo.Map(p.Endpoints, func(e service_model_catalog.Endpoint, _ int) ModelEndpointDTO {
				return ModelEndpointDTO{Path: e.Path, Method: e.Method}
			}),
			GroupPrices: lo.Map(p.GroupPrices, func(gp service_model_catalog.ModelGroupPrice, _ int) ModelGroupPriceDTO {
				var userRate *float64
				if r, ok := rates[gp.GroupID]; ok {
					userRate = new(r)
				}
				return ModelGroupPriceDTO{
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
					ImageOutputPricePM:   gp.ImageOutputPricePerMTok,
					PerRequestPriceUSD:   gp.PerRequestPrice,
					ChannelChain:         gp.ChannelChain,
					Intervals: lo.Map(gp.Intervals, func(iv service_model_catalog.ModelGroupPriceInterval, _ int) ModelPriceIntervalDTO {
						return ModelPriceIntervalDTO{
							MinTokens:            iv.MinTokens,
							MaxTokens:            iv.MaxTokens,
							TierLabel:            iv.TierLabel,
							InputPricePerMTok:    iv.InputPricePerMTok,
							OutputPricePerMTok:   iv.OutputPricePerMTok,
							CacheReadPricePerMT:  iv.CacheReadPricePerMTok,
							CacheWritePricePerMT: iv.CacheWritePricePerMTok,
							PerRequestPriceUSD:   iv.PerRequestPrice,
							SortOrder:            iv.SortOrder,
							CacheCreation5mPMT:   iv.CacheCreation5mPerMTok,
							CacheCreation1hPMT:   iv.CacheCreation1hPerMTok,
						}
					}),
					CacheCreation5mPMT: gp.CacheCreation5mPerMTok,
					CacheCreation1hPMT: gp.CacheCreation1hPerMTok,
				}
			}),
		}
	})
}
