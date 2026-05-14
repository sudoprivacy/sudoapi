package handler

import (
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
	GroupID              int64    `json:"group_id"`
	GroupName            string   `json:"group_name"`
	SubscriptionType     string   `json:"subscription_type"`
	IsExclusive          bool     `json:"is_exclusive"`
	BaseRateMultiplier   float64  `json:"base_rate_multiplier"`
	UserRateMultiplier   *float64 `json:"user_rate_multiplier"`
	BillingMode          string   `json:"billing_mode"`
	InputPricePerMTok    *float64 `json:"input_price_per_mtok_usd"`
	OutputPricePerMTok   *float64 `json:"output_price_per_mtok_usd"`
	CacheReadPricePerMT  *float64 `json:"cache_read_price_per_mtok_usd"`
	CacheWritePricePerMT *float64 `json:"cache_write_price_per_mtok_usd"`
	ImageOutputPricePM   *float64 `json:"image_output_price_per_mtok_usd"`
	PerRequestPriceUSD   *float64 `json:"per_request_price_usd"`
	ChannelChain         []string `json:"channel_chain"`
}

type modelPlatformSectionDTO struct {
	Platform    string               `json:"platform"`
	Endpoints   []modelEndpointDTO   `json:"endpoints"`
	GroupPrices []modelGroupPriceDTO `json:"group_prices"`
}

type modelSquareCardDTO struct {
	Name          string                    `json:"name"`
	DisplayName   string                    `json:"display_name"`
	Category      string                    `json:"category"`
	Description   string                    `json:"description"`
	ContextWindow int                       `json:"context_window"`
	MaxOutput     int                       `json:"max_output"`
	Capabilities  []string                  `json:"capabilities"`
	Featured      bool                      `json:"featured"`
	IconURL       string                    `json:"icon_url"`
	Platforms     []modelPlatformSectionDTO `json:"platforms"`
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
			Name:          c.Name,
			DisplayName:   c.DisplayName,
			Category:      c.Category,
			Description:   c.Description,
			ContextWindow: c.ContextWindow,
			MaxOutput:     c.MaxOutput,
			Capabilities:  c.Capabilities,
			Featured:      c.Featured,
			IconURL:       c.IconURL,
			Platforms:     toPlatformDTOs(c.Platforms, userRateMultipliers),
		})
	}
	return out
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
				ImageOutputPricePM:   gp.ImageOutputPricePerMTok,
				PerRequestPriceUSD:   gp.PerRequestPrice,
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
