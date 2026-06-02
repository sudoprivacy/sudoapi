// sudoapi: Model Square model catalog.

//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestToCardDTOs_FieldWhitelistShape(t *testing.T) {
	// 严格断言序列化结果只包含白名单 JSON key，不能漏暴露 channel_id / 调度元数据。
	card := service.ModelSquareCard{
		Name:             "claude-opus-4-7",
		DisplayName:      "Claude Opus 4.7",
		Category:         "claude",
		Description:      "desc",
		ModelType:        "chat",
		ContextWindow:    200000,
		MaxOutput:        64000,
		Capabilities:     []string{"vision", "reasoning"},
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
		SupportFlags:     []string{"vision", "reasoning", "web_search"},
		Featured:         true,
		IconURL:          "https://cdn/icon.png",
		OfficialPrice: &service.ModelOfficialPrice{
			InputPricePerMTok:      func() *float64 { v := 15.0; return &v }(),
			OutputPricePerMTok:     func() *float64 { v := 75.0; return &v }(),
			CacheReadPricePerMTok:  func() *float64 { v := 1.5; return &v }(),
			CacheWritePricePerMTok: func() *float64 { v := 18.75; return &v }(),
		},
		Platforms: []service.ModelPlatformSection{
			{
				Platform:  "anthropic",
				Endpoints: []service.ModelEndpoint{{Path: "/v1/messages", Method: "POST"}},
				GroupPrices: []service.ModelGroupPrice{
					{
						GroupID:           1,
						GroupName:         "auto",
						SubscriptionType:  "standard",
						IsExclusive:       false,
						BaseRateMult:      1.0,
						BillingMode:       service.BillingModeToken,
						InputPricePerMTok: func() *float64 { v := 17.5; return &v }(),
						Intervals: []service.ModelGroupPriceInterval{{
							MinTokens:          0,
							MaxTokens:          func() *int { v := 200000; return &v }(),
							InputPricePerMTok:  func() *float64 { v := 3.0; return &v }(),
							OutputPricePerMTok: func() *float64 { v := 15.0; return &v }(),

							CacheCreation5mPerMTok: func() *float64 { v := 4.0; return &v }(),
							CacheCreation1hPerMTok: func() *float64 { v := 6.0; return &v }(),
						}},

						CacheCreation5mPerMTok: func() *float64 { v := 20.0; return &v }(),
						CacheCreation1hPerMTok: func() *float64 { v := 25.0; return &v }(),
					},
				},
			},
		},
	}

	dtos := toCardDTOs([]service.ModelSquareCard{card}, nil)
	require.Len(t, dtos, 1)
	raw, err := json.Marshal(dtos[0])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))

	// 白名单：必须存在的字段
	for _, k := range []string{
		"name", "display_name", "category", "description",
		"model_type", "context_window", "max_output", "capabilities",
		"input_modalities", "output_modalities", "support_flags",
		"featured", "icon_url", "official_price", "platforms",
	} {
		_, ok := m[k]
		require.True(t, ok, "missing whitelisted field: %s", k)
	}
	// 禁止字段：channel_id / api_key_id 之类绝不应该出现
	for _, k := range []string{"channel_id", "api_key_id", "raw_pricing", "internal_id"} {
		_, ok := m[k]
		require.False(t, ok, "DTO leaked forbidden field: %s", k)
	}

	platforms, _ := m["platforms"].([]any)
	require.Len(t, platforms, 1)
	officialPrice, _ := m["official_price"].(map[string]any)
	require.Contains(t, officialPrice, "input_price_per_mtok_usd")
	require.Contains(t, officialPrice, "output_price_per_mtok_usd")
	require.Contains(t, officialPrice, "cache_read_price_per_mtok_usd")
	require.Contains(t, officialPrice, "cache_write_price_per_mtok_usd")
	require.Contains(t, officialPrice, "image_output_price_per_mtok_usd")
	require.Contains(t, officialPrice, "image_price_usd")

	platform := platforms[0].(map[string]any)
	prices, _ := platform["group_prices"].([]any)
	require.Len(t, prices, 1)
	row := prices[0].(map[string]any)
	// 价格字段命名遵循 USD/MTok 约定
	require.Contains(t, row, "input_price_per_mtok_usd")
	require.Contains(t, row, "output_price_per_mtok_usd")
	require.Contains(t, row, "cache_read_price_per_mtok_usd")
	require.Contains(t, row, "cache_write_price_per_mtok_usd")
	require.Contains(t, row, "cache_creation_5m_price_per_mtok_usd")
	require.Contains(t, row, "cache_creation_1h_price_per_mtok_usd")
	require.Contains(t, row, "intervals")
	require.Contains(t, row, "channel_chain")
	require.Contains(t, row, "base_rate_multiplier")
	// 未传 userRateMultipliers 时 user_rate_multiplier 必须是 null（json: null → nil 反序列化为 nil）
	require.Nil(t, row["user_rate_multiplier"])

	intervals, _ := row["intervals"].([]any)
	require.Len(t, intervals, 1)
	interval := intervals[0].(map[string]any)
	require.Contains(t, interval, "min_tokens")
	require.Contains(t, interval, "max_tokens")
	require.Contains(t, interval, "input_price_per_mtok_usd")
	require.Contains(t, interval, "output_price_per_mtok_usd")
	require.Contains(t, interval, "cache_creation_5m_price_per_mtok_usd")
	require.Contains(t, interval, "cache_creation_1h_price_per_mtok_usd")
}

func TestToCardDTOs_UserRateMultiplierJoinsByGroupID(t *testing.T) {
	cards := []service.ModelSquareCard{{
		Name: "m1",
		Platforms: []service.ModelPlatformSection{{
			Platform: "anthropic",
			GroupPrices: []service.ModelGroupPrice{
				{GroupID: 1, GroupName: "auto", BaseRateMult: 1.0},
				{GroupID: 2, GroupName: "vip", BaseRateMult: 0.5},
			},
		}},
	}}
	rates := map[int64]float64{1: 0.8} // 仅 group 1 有专属倍率

	dtos := toCardDTOs(cards, rates)
	require.Len(t, dtos, 1)
	prices := dtos[0].Platforms[0].GroupPrices
	require.Len(t, prices, 2)
	require.NotNil(t, prices[0].UserRateMultiplier)
	require.InDelta(t, 0.8, *prices[0].UserRateMultiplier, 1e-9)
	require.Nil(t, prices[1].UserRateMultiplier)
}

func TestModelSquareHandler_ListAuthenticatedJoinsUserGroupRates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inputPrice := 2e-6
	group := service.Group{
		ID:               7,
		Name:             "vip",
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeStandard,
		IsExclusive:      true,
		RateMultiplier:   0.5,
		Status:           service.StatusActive,
	}
	channelRepo := &modelSquareHandlerChannelRepoStub{channels: []service.Channel{{
		ID:       10,
		Name:     "vip-channel",
		Status:   service.StatusActive,
		GroupIDs: []int64{group.ID},
		ModelPricing: []service.ChannelModelPricing{{
			Platform:    service.PlatformAnthropic,
			Models:      []string{"claude-test"},
			BillingMode: service.BillingModeToken,
			InputPrice:  &inputPrice,
		}},
	}}}
	groupRepo := &modelSquareHandlerGroupRepoStub{groups: []service.Group{group}}
	channelSvc := service.NewChannelService(channelRepo, groupRepo, nil, nil)
	modelSquareSvc := service.NewModelSquareService(channelSvc, nil)
	userID := int64(77)
	apiKeySvc := service.NewAPIKeyService(
		nil,
		&modelSquareHandlerUserRepoStub{user: &service.User{ID: userID, AllowedGroups: []int64{group.ID}}},
		groupRepo,
		&modelSquareHandlerSubscriptionRepoStub{},
		&modelSquareHandlerRateRepoStub{rates: map[int64]float64{group.ID: 0.8}},
		nil,
		nil,
	)
	h := NewModelSquareHandler(modelSquareSvc, apiKeySvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})

	h.ListAuthenticated(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int                  `json:"code"`
		Data []modelSquareCardDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Len(t, body.Data, 1)
	prices := body.Data[0].Platforms[0].GroupPrices
	require.Len(t, prices, 1)
	require.Equal(t, group.ID, prices[0].GroupID)
	require.NotNil(t, prices[0].UserRateMultiplier)
	require.InDelta(t, 0.8, *prices[0].UserRateMultiplier, 1e-9)
}

type modelSquareHandlerChannelRepoStub struct {
	service.ChannelRepository
	channels []service.Channel
}

func (s *modelSquareHandlerChannelRepoStub) ListAll(context.Context) ([]service.Channel, error) {
	return s.channels, nil
}

type modelSquareHandlerGroupRepoStub struct {
	service.GroupRepository
	groups []service.Group
}

func (s *modelSquareHandlerGroupRepoStub) ListActive(context.Context) ([]service.Group, error) {
	return s.groups, nil
}

type modelSquareHandlerUserRepoStub struct {
	service.UserRepository
	user *service.User
}

func (s *modelSquareHandlerUserRepoStub) GetByID(context.Context, int64) (*service.User, error) {
	return s.user, nil
}

type modelSquareHandlerSubscriptionRepoStub struct {
	service.UserSubscriptionRepository
}

func (s *modelSquareHandlerSubscriptionRepoStub) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

type modelSquareHandlerRateRepoStub struct {
	service.UserGroupRateRepository
	rates map[int64]float64
}

func (s *modelSquareHandlerRateRepoStub) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	return s.rates, nil
}
