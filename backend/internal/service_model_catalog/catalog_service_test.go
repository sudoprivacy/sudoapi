// sudoapi: Model catalog.

//go:build unit

package service_model_catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// stubChannelServiceProvider 让 ModelCatalogService 拿到固定的 AvailableChannel 列表。
// 通过把 ChannelService 的依赖伪造起来：repo.ListAll + groupRepo.ListActive。
func stubChannelServiceProvider(t *testing.T, channels []service.Channel, groups []service.Group) *service.ChannelService {
	t.Helper()
	repo := &mockChannelRepository{
		listAllFn: func(ctx context.Context) ([]service.Channel, error) { return channels, nil },
	}
	groupRepo := &stubGroupRepoForAvailable{activeGroups: groups}
	return service.NewChannelService(repo, groupRepo, nil, nil)
}

type stubModelCatalogMetadataReader map[string]*MetadataOverride

func (s stubModelCatalogMetadataReader) GetOverridesByModelNames(_ context.Context, modelNames []string) (map[string]*MetadataOverride, error) {
	out := make(map[string]*MetadataOverride, len(modelNames))
	for _, name := range modelNames {
		key := normalizeMetadataModelKey(name)
		if override, ok := s[key]; ok {
			out[key] = override
		}
	}
	return out, nil
}

type stubModelCatalogEndpointConfigReader struct {
	cfg *EndpointConfig
}

func (s stubModelCatalogEndpointConfigReader) GetEndpointConfig(context.Context) (*EndpointConfig, error) {
	return s.cfg, nil
}

func TestModelCatalogService_AuthenticatedScopeKeepsStandardGroupsWithoutExplicitAllowed(t *testing.T) {
	// 三个分组：一个标准公开、一个专属、一个订阅。未显式授权时只保留 standard 非专属分组。
	gStdPublic := service.Group{
		ID:               1,
		Name:             "auto",
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeStandard,
		IsExclusive:      false,
		RateMultiplier:   1.0,
		Status:           service.StatusActive,
	}
	gExclusive := service.Group{
		ID:               2,
		Name:             "vip",
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeStandard,
		IsExclusive:      true,
		RateMultiplier:   0.5,
		Status:           service.StatusActive,
	}
	gSub := service.Group{
		ID:               3,
		Name:             "pro-plan",
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeSubscription,
		IsExclusive:      false,
		RateMultiplier:   0.8,
		Status:           service.StatusActive,
	}
	channels := []service.Channel{{
		ID:     10,
		Name:   "ch-A",
		Status: service.StatusActive,
	}}
	channels[0].GroupIDs = []int64{gStdPublic.ID, gExclusive.ID, gSub.ID}
	channels[0].ModelPricing = []service.ChannelModelPricing{{
		ID: 1, ChannelID: 10, Platform: service.PlatformAnthropic, Models: []string{"claude-opus-4-7"},
		BillingMode: service.BillingModeToken,
		InputPrice:  new(1.75e-5),
		OutputPrice: new(8.75e-5),
	}}

	svc := NewModelCatalogService(
		stubChannelServiceProvider(t, channels, []service.Group{gStdPublic, gExclusive, gSub}),
		nil,
		nil,
		nil,
	)
	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "claude-opus-4-7", cards[0].Name)
	require.Len(t, cards[0].Platforms, 1)
	priceRows := cards[0].Platforms[0].GroupPrices
	require.Len(t, priceRows, 1, "authenticated scope keeps standard non-exclusive groups without explicit allowed groups")
	require.Equal(t, "auto", priceRows[0].GroupName)
}

func TestModelCatalogService_AuthenticatedScopeIncludesAllowedExclusive(t *testing.T) {
	gStdPublic := service.Group{
		ID: 1, Name: "auto", Platform: service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeStandard, IsExclusive: false,
		RateMultiplier: 1.0, Status: service.StatusActive,
	}
	gExclusive := service.Group{
		ID: 2, Name: "vip", Platform: service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeStandard, IsExclusive: true,
		RateMultiplier: 0.5, Status: service.StatusActive,
	}
	channels := []service.Channel{{
		ID: 10, Name: "ch-A", Status: service.StatusActive,
		GroupIDs: []int64{gStdPublic.ID, gExclusive.ID},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode: service.BillingModeToken, InputPrice: new(1.75e-5),
		}},
	}}
	svc := NewModelCatalogService(
		stubChannelServiceProvider(t, channels, []service.Group{gStdPublic, gExclusive}),
		nil,
		nil,
		nil,
	)
	cards, err := svc.ListForUser(context.Background(), 42, map[int64]struct{}{
		2: {}, // 用户被授权访问 vip 分组
	})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Len(t, cards[0].Platforms[0].GroupPrices, 2, "authenticated scope keeps standard + allowed exclusive")
}

func TestModelCatalogService_PricePerMillionConversion(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode:     service.BillingModeToken,
			InputPrice:      new(1.75e-5),   // → 17.5 / MTok
			OutputPrice:     new(8.75e-5),   // → 87.5 / MTok
			CacheReadPrice:  new(1.75e-6),   // → 1.75 / MTok
			CacheWritePrice: new(2.1875e-5), // → 21.875 / MTok

			CacheCreation5mPrice: new(2.5e-5), // → 25 / MTok
			CacheCreation1hPrice: new(3e-5),   // → 30 / MTok
		}},
	}}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), nil, nil, nil)
	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	p := cards[0].Platforms[0].GroupPrices[0]
	require.NotNil(t, p.InputPricePerMTok)
	require.InDelta(t, 17.5, *p.InputPricePerMTok, 1e-6)
	require.InDelta(t, 87.5, *p.OutputPricePerMTok, 1e-6)
	require.InDelta(t, 1.75, *p.CacheReadPricePerMTok, 1e-6)
	require.InDelta(t, 21.875, *p.CacheWritePricePerMTok, 1e-6)
	require.InDelta(t, 25, *p.CacheCreation5mPerMTok, 1e-6)
	require.InDelta(t, 30, *p.CacheCreation1hPerMTok, 1e-6)
}

func TestModelCatalogService_OfficialPriceFromLiteLLM(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 0.5, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"gpt-quote-test"},
			BillingMode: service.BillingModeToken,
			InputPrice:  new(1e-6),
			OutputPrice: new(5e-6),
		}},
	}}
	pricingSvc := stubPricingReader{
		"gpt-quote-test": {
			InputCostPerToken:           2e-6,
			OutputCostPerToken:          10e-6,
			CacheReadInputTokenCost:     0.2e-6,
			CacheCreationInputTokenCost: 2.5e-6,
			OutputCostPerImageToken:     3e-6,
			OutputCostPerImage:          0.04,
			LiteLLMProvider:             "openai",
			Mode:                        "chat",
		},
	}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), pricingSvc, nil, nil)

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.NotNil(t, cards[0].OfficialPrice)
	require.InDelta(t, 2.0, *cards[0].OfficialPrice.InputPricePerMTok, 1e-9)
	require.InDelta(t, 10.0, *cards[0].OfficialPrice.OutputPricePerMTok, 1e-9)
	require.InDelta(t, 0.2, *cards[0].OfficialPrice.CacheReadPricePerMTok, 1e-9)
	require.InDelta(t, 2.5, *cards[0].OfficialPrice.CacheWritePricePerMTok, 1e-9)
	require.InDelta(t, 3.0, *cards[0].OfficialPrice.ImageOutputPricePerMTok, 1e-9)
	require.InDelta(t, 0.04, *cards[0].OfficialPrice.ImagePriceUSD, 1e-12)
}

func TestModelCatalogService_OfficialPriceMissingWhenLiteLLMUnmatched(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformAnthropic, Models: []string{"private-model"},
			BillingMode: service.BillingModeToken,
			InputPrice:  new(1e-6),
		}},
	}}
	pricingSvc := stubPricingReader{}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), pricingSvc, nil, nil)

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Nil(t, cards[0].OfficialPrice)
}

func TestModelCatalogService_ContextIntervalsConversion(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode: service.BillingModeToken,
			InputPrice:  new(1e-6),
			OutputPrice: new(2e-6),
			Intervals: []service.PricingInterval{
				{
					MinTokens:       0,
					MaxTokens:       new(200000),
					InputPrice:      new(3e-6),
					OutputPrice:     new(1.5e-5),
					CacheReadPrice:  new(3e-7),
					CacheWritePrice: new(3.75e-6),
					SortOrder:       0,

					CacheCreation5mPrice: new(4e-6),
					CacheCreation1hPrice: new(6e-6),
				},
				{
					MinTokens:  200000,
					MaxTokens:  nil,
					InputPrice: new(6e-6),
					SortOrder:  1,
				},
				{
					MinTokens: 400000,
					MaxTokens: nil,
					SortOrder: 2,
				},
			},
		}},
	}}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), nil, nil, nil)
	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	intervals := cards[0].Platforms[0].GroupPrices[0].Intervals
	require.Len(t, intervals, 2, "empty intervals are not shown")
	require.Equal(t, 0, intervals[0].MinTokens)
	require.NotNil(t, intervals[0].MaxTokens)
	require.Equal(t, 200000, *intervals[0].MaxTokens)
	require.InDelta(t, 3, *intervals[0].InputPricePerMTok, 1e-9)
	require.InDelta(t, 15, *intervals[0].OutputPricePerMTok, 1e-9)
	require.InDelta(t, 0.3, *intervals[0].CacheReadPricePerMTok, 1e-9)
	require.InDelta(t, 3.75, *intervals[0].CacheWritePricePerMTok, 1e-9)
	require.InDelta(t, 4, *intervals[0].CacheCreation5mPerMTok, 1e-9)
	require.InDelta(t, 6, *intervals[0].CacheCreation1hPerMTok, 1e-9)
	require.Equal(t, 200000, intervals[1].MinTokens)
	require.Nil(t, intervals[1].MaxTokens)
	require.InDelta(t, 6, *intervals[1].InputPricePerMTok, 1e-9)
}

func TestModelCatalogService_ZeroPriceTreatedAsUnconfigured(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"gpt-4o"},
			BillingMode: service.BillingModeToken,
			InputPrice:  new(float64(0)), // 显式 0 视为未配置
			OutputPrice: new(2.5e-5),
		}},
	}}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), nil, nil, nil)
	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	p := cards[0].Platforms[0].GroupPrices[0]
	require.Nil(t, p.InputPricePerMTok, "zero must collapse to nil")
	require.NotNil(t, p.OutputPricePerMTok)
}

func TestModelCatalogService_PivotAggregatesAcrossPlatformsAndChannels(t *testing.T) {
	// 同一模型 "claude-opus-4-7" 既在 anthropic 平台又通过 antigravity 平台暴露，
	// 同时分属两个不同渠道。期望：一张卡片，2 个 platform section，每 section 一行价格，
	// antigravity 平台的 ChannelChain 包含两个渠道名。
	gA := service.Group{ID: 1, Name: "auto-anthropic", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	gG := service.Group{ID: 2, Name: "auto-antigravity", Platform: service.PlatformAntigravity, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.2, Status: service.StatusActive}
	channels := []service.Channel{
		{
			ID: 10, Name: "primary-pool", Status: service.StatusActive,
			GroupIDs: []int64{1, 2},
			ModelPricing: []service.ChannelModelPricing{
				{Platform: service.PlatformAnthropic, Models: []string{"claude-opus-4-7"}, BillingMode: service.BillingModeToken, InputPrice: new(1.75e-5)},
				{Platform: service.PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: service.BillingModeToken, InputPrice: new(1.5e-5)},
			},
		},
		{
			ID: 11, Name: "fallback-pool", Status: service.StatusActive,
			GroupIDs: []int64{2},
			ModelPricing: []service.ChannelModelPricing{
				{Platform: service.PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: service.BillingModeToken, InputPrice: new(1.7e-5)},
			},
		},
	}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{gA, gG}), nil, nil, nil)
	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "claude-opus-4-7", card.Name)
	require.Len(t, card.Platforms, 2)
	// platforms 按字典序：anthropic 在前，antigravity 在后
	require.Equal(t, service.PlatformAnthropic, card.Platforms[0].Platform)
	require.Equal(t, service.PlatformAntigravity, card.Platforms[1].Platform)
	antig := card.Platforms[1]
	require.Len(t, antig.GroupPrices, 1)
	require.ElementsMatch(t, []string{"primary-pool", "fallback-pool"}, antig.GroupPrices[0].ChannelChain)
}

func TestModelCatalogService_AppliesModelCatalogMetadataOverrides(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"custom-model"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-6),
		}},
	}}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), nil, stubModelCatalogMetadataReader{
		"custom-model": {
			ModelName:     "custom-model",
			DisplayName:   "Custom Model",
			Description:   "Admin maintained description",
			Category:      "OpenAI",
			ContextWindow: 128000,
			MaxOutput:     8192,
			Capabilities:  []string{"reasoning", "function_calling"},
			Featured:      true,
			IconURL:       "https://example.com/icon.png",
		},
	}, nil)

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "Custom Model", card.DisplayName)
	require.Equal(t, "Admin maintained description", card.Description)
	require.Equal(t, "OpenAI", card.Category)
	require.Equal(t, 128000, card.ContextWindow)
	require.Equal(t, 8192, card.MaxOutput)
	require.Equal(t, []string{"reasoning", "function_calling"}, card.Capabilities)
	require.True(t, card.Featured)
	require.Equal(t, "https://example.com/icon.png", card.IconURL)
	require.Len(t, card.Platforms[0].GroupPrices, 1, "pricing remains sourced from channel config")
}

func TestModelCatalogService_FillsLiteLLMTypeModalitiesAndSupportFlags(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformGemini, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformGemini, Models: []string{"gemini-test"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-6),
		}},
	}}
	pricingSvc := stubPricingReader{
		"gemini-test": {
			Mode:                      "chat",
			MaxInputTokens:            1000000,
			MaxOutputTokens:           8192,
			SupportedModalities:       []string{"text", "image", "audio", "video"},
			SupportedOutputModalities: []string{"text", "image"},
			SupportFlags:              []string{"vision", "web_search", "response_schema"},
			SupportsVision:            true,
			SupportsFunctionCalling:   true,
		},
	}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), pricingSvc, nil, nil)

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "chat", card.ModelType)
	require.Equal(t, []string{"text", "image", "audio", "video"}, card.InputModalities)
	require.Equal(t, []string{"text", "image"}, card.OutputModalities)
	require.Equal(t, []string{"vision", "web_search", "response_schema"}, card.SupportFlags)
	require.Equal(t, []string{"vision", "function_calling"}, card.Capabilities)
}

func TestModelCatalogService_EndpointsUseOverriddenModelType(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"custom-image-model"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-6),
		}},
	}}
	pricingSvc := stubPricingReader{
		"custom-image-model": {Mode: "chat"},
	}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), pricingSvc, stubModelCatalogMetadataReader{
		"custom-image-model": {ModelName: "custom-image-model", ModelType: "image_generation"},
	}, stubModelCatalogEndpointConfigReader{cfg: &EndpointConfig{
		Platforms: map[string]map[string][]Endpoint{
			service.PlatformOpenAI: {
				"chat":             {{Method: "POST", Path: "/v1/chat/completions"}},
				"image_generation": {{Method: "POST", Path: "/v1/images/generations"}},
			},
		},
	}})

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "image_generation", cards[0].ModelType)
	require.Equal(t, []Endpoint{{Method: "POST", Path: "/v1/images/generations"}}, cards[0].Platforms[0].Endpoints)
}

func TestModelCatalogService_UnconfiguredModelTypeEndpointsEmptyWithExplicitConfig(t *testing.T) {
	g := service.Group{ID: 1, Name: "auto", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}
	channels := []service.Channel{{
		ID: 10, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformOpenAI, Models: []string{"text-embedding-custom"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-6),
		}},
	}}
	pricingSvc := stubPricingReader{
		"text-embedding-custom": {Mode: "embedding"},
	}
	svc := NewModelCatalogService(stubChannelServiceProvider(t, channels, []service.Group{g}), pricingSvc, nil, stubModelCatalogEndpointConfigReader{cfg: &EndpointConfig{
		Platforms: map[string]map[string][]Endpoint{
			service.PlatformOpenAI: {
				"chat": {{Method: "POST", Path: "/v1/chat/completions"}},
			},
		},
	}})

	cards, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "embedding", cards[0].ModelType)
	require.Empty(t, cards[0].Platforms[0].Endpoints)
}

func TestInboundEndpointsForPlatform_RoundTripWithNormalize(t *testing.T) {
	// 该断言保证：模型目录展示的 inbound 端点路径，能被网关的 NormalizeInboundEndpoint
	// 重新映射回同一个金标常量，避免两侧路径常量漂移。
	// （NormalizeInboundEndpoint 在 handler 包，无法在 service 内 import，因此这里只验证
	// service 包内部 ms* 常量与 endpoint.go 中导出常量的字面一致——通过下面的 TestEndpoint
	// Constants 间接保证。）
	cases := []struct {
		platform string
		mode     string
		want     []Endpoint
	}{
		{service.PlatformAnthropic, "", []Endpoint{{endpointMessages, "POST"}}},
		{service.PlatformOpenAI, "", []Endpoint{
			{endpointChatCompletions, "POST"},
			{endpointResponses, "POST"},
		}},
		{service.PlatformOpenAI, "image_generation", []Endpoint{
			{endpointImagesGenerations, "POST"},
			{endpointImagesEdits, "POST"},
		}},
		{service.PlatformGemini, "", []Endpoint{{endpointGeminiModels, "POST"}}},
		{service.PlatformAntigravity, "", []Endpoint{
			{endpointMessages, "POST"},
			{endpointGeminiModels, "POST"},
		}},
		{"unknown", "", nil},
	}
	for _, c := range cases {
		require.Equal(t, c.want, InboundEndpointsForPlatform(c.platform, c.mode), "platform=%s mode=%s", c.platform, c.mode)
	}
}

func TestEndpointConstantsMatchHandlerPackage(t *testing.T) {
	// service 包内的端点字面必须与 handler 包的金标常量逐字一致，否则前端展示路径会与
	// 网关实际接受路径漂移。这里硬编码做一遍交叉校验。
	require.Equal(t, "/v1/messages", endpointMessages)
	require.Equal(t, "/v1/chat/completions", endpointChatCompletions)
	require.Equal(t, "/v1/responses", endpointResponses)
	require.Equal(t, "/v1/images/generations", endpointImagesGenerations)
	require.Equal(t, "/v1/images/edits", endpointImagesEdits)
	require.Equal(t, "/v1beta/models", endpointGeminiModels)
}

func TestInferCategoryFromName(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-7":        "claude",
		"Claude-Sonnet-4-6":      "claude",
		"gpt-4o":                 "gpt",
		"o3-mini":                "gpt",
		"codex-mini":             "gpt",
		"chatgpt-4o-latest":      "gpt",
		"gemini-2.5-pro":         "gemini",
		"dall-e-3":               "image",
		"imagen-3":               "image",
		"text-embedding-3-small": "embedding",
		"some-random-model":      "other",
	}
	for in, want := range cases {
		require.Equal(t, want, inferCategoryFromName(in), "in=%s", in)
	}
}

func TestModelCatalogService_CacheHit(t *testing.T) {
	// 第二次调用应命中缓存：通过断言 ChannelService.ListAvailable 没有被再次调用
	// 来证明（mockChannelRepository.listAllFn 计数）。
	calls := 0
	channels := []service.Channel{{
		ID: 1, Name: "ch", Status: service.StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []service.ChannelModelPricing{{
			Platform: service.PlatformAnthropic, Models: []string{"m"},
			BillingMode: service.BillingModeToken, InputPrice: new(1e-5),
		}},
	}}
	groups := []service.Group{{ID: 1, Name: "auto", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1.0, Status: service.StatusActive}}
	repo := &mockChannelRepository{
		listAllFn: func(ctx context.Context) ([]service.Channel, error) {
			calls++
			return channels, nil
		},
	}
	svc := NewModelCatalogService(
		service.NewChannelService(repo, &stubGroupRepoForAvailable{activeGroups: groups}, nil, nil),
		nil,
		nil,
		nil,
	)
	_, err := svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	_, err = svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Equal(t, 1, calls, "second call must hit cache")

	svc.InvalidateAll()
	_, err = svc.ListForUser(context.Background(), 1, nil)
	require.NoError(t, err)
	require.Equal(t, 2, calls, "InvalidateAll forces a rebuild")
}

func TestDeriveCapabilities(t *testing.T) {
	lp := &service.LiteLLMModelPricing{
		SupportsVision:          true,
		SupportsFunctionCalling: true,
		SupportsReasoning:       true,
		SupportsAudioInput:      true,
		SupportsPromptCaching:   true,
	}
	caps := deriveCapabilities(lp)
	require.ElementsMatch(t, []string{"vision", "function_calling", "reasoning", "audio_input", "prompt_caching"}, caps)
	require.Nil(t, deriveCapabilities(nil))
}
