//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// makeChannel 构造一个 active 渠道，挂载若干 SupportedModel 与若干 group。
func makeChannel(id int64, name string, groups []AvailableGroupRef, models []SupportedModel) Channel {
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	return Channel{
		ID:       id,
		Name:     name,
		Status:   StatusActive,
		GroupIDs: groupIDs,
	}
}

// stubChannelServiceProvider 让 ModelSquareService 拿到固定的 AvailableChannel 列表。
// 通过把 ChannelService 的依赖伪造起来：repo.ListAll + groupRepo.ListActive。
func stubChannelServiceProvider(t *testing.T, channels []Channel, groups []Group) *ChannelService {
	t.Helper()
	repo := &mockChannelRepository{
		listAllFn: func(ctx context.Context) ([]Channel, error) { return channels, nil },
	}
	groupRepo := &stubGroupRepoForAvailable{activeGroups: groups}
	return NewChannelService(repo, groupRepo, nil, nil)
}

func msPtrFloat(v float64) *float64 { return &v }
func msPtrInt(v int) *int           { return &v }

type stubModelMetadataReader map[string]*ModelMetadataOverride

func (s stubModelMetadataReader) GetOverridesByModelNames(_ context.Context, modelNames []string) (map[string]*ModelMetadataOverride, error) {
	out := make(map[string]*ModelMetadataOverride, len(modelNames))
	for _, name := range modelNames {
		key := normalizeMetadataModelKey(name)
		if override, ok := s[key]; ok {
			out[key] = override
		}
	}
	return out, nil
}

type stubModelEndpointConfigReader struct {
	cfg *ModelEndpointConfig
}

func (s stubModelEndpointConfigReader) GetModelEndpointConfig(context.Context) (*ModelEndpointConfig, error) {
	return s.cfg, nil
}

func TestModelSquareService_PublicScopeExcludesExclusiveAndSubscriptionGroups(t *testing.T) {
	// 三个分组：一个标准公开、一个专属、一个订阅。public scope 只应保留标准公开。
	gStdPublic := Group{
		ID:               1,
		Name:             "auto",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeStandard,
		IsExclusive:      false,
		RateMultiplier:   1.0,
		Status:           StatusActive,
	}
	gExclusive := Group{
		ID:               2,
		Name:             "vip",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeStandard,
		IsExclusive:      true,
		RateMultiplier:   0.5,
		Status:           StatusActive,
	}
	gSub := Group{
		ID:               3,
		Name:             "pro-plan",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		IsExclusive:      false,
		RateMultiplier:   0.8,
		Status:           StatusActive,
	}
	channels := []Channel{makeChannel(10, "ch-A", nil, nil)}
	channels[0].GroupIDs = []int64{gStdPublic.ID, gExclusive.ID, gSub.ID}
	channels[0].ModelPricing = []ChannelModelPricing{{
		ID: 1, ChannelID: 10, Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"},
		BillingMode: BillingModeToken,
		InputPrice:  msPtrFloat(1.75e-5),
		OutputPrice: msPtrFloat(8.75e-5),
	}}

	svc := NewModelSquareService(
		stubChannelServiceProvider(t, channels, []Group{gStdPublic, gExclusive, gSub}),
		nil,
	)
	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "claude-opus-4-7", cards[0].Name)
	require.Len(t, cards[0].Platforms, 1)
	priceRows := cards[0].Platforms[0].GroupPrices
	require.Len(t, priceRows, 1, "public scope only sees the standard non-exclusive group")
	require.Equal(t, "auto", priceRows[0].GroupName)
}

func TestModelSquareService_AuthenticatedScopeIncludesAllowedExclusive(t *testing.T) {
	gStdPublic := Group{
		ID: 1, Name: "auto", Platform: PlatformAnthropic,
		SubscriptionType: SubscriptionTypeStandard, IsExclusive: false,
		RateMultiplier: 1.0, Status: StatusActive,
	}
	gExclusive := Group{
		ID: 2, Name: "vip", Platform: PlatformAnthropic,
		SubscriptionType: SubscriptionTypeStandard, IsExclusive: true,
		RateMultiplier: 0.5, Status: StatusActive,
	}
	channels := []Channel{{
		ID: 10, Name: "ch-A", Status: StatusActive,
		GroupIDs: []int64{gStdPublic.ID, gExclusive.ID},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1.75e-5),
		}},
	}}
	svc := NewModelSquareService(
		stubChannelServiceProvider(t, channels, []Group{gStdPublic, gExclusive}),
		nil,
	)
	cards, err := svc.ListForUser(context.Background(), 42, map[int64]struct{}{
		2: {}, // 用户被授权访问 vip 分组
	})
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Len(t, cards[0].Platforms[0].GroupPrices, 2, "authenticated scope keeps standard + allowed exclusive")
}

func TestModelSquareService_PricePerMillionConversion(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode:          BillingModeToken,
			InputPrice:           msPtrFloat(1.75e-5),   // → 17.5 / MTok
			OutputPrice:          msPtrFloat(8.75e-5),   // → 87.5 / MTok
			CacheReadPrice:       msPtrFloat(1.75e-6),   // → 1.75 / MTok
			CacheWritePrice:      msPtrFloat(2.1875e-5), // → 21.875 / MTok
			CacheCreation5mPrice: msPtrFloat(2.5e-5),    // → 25 / MTok
			CacheCreation1hPrice: msPtrFloat(3e-5),      // → 30 / MTok
		}},
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), nil)
	cards, err := svc.ListPublic(context.Background())
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

func TestModelSquareService_OfficialPriceFromLiteLLM(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 0.5, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"gpt-quote-test"},
			BillingMode: BillingModeToken,
			InputPrice:  msPtrFloat(1e-6),
			OutputPrice: msPtrFloat(5e-6),
		}},
	}}
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
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
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), pricingSvc)

	cards, err := svc.ListPublic(context.Background())
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

func TestModelSquareService_ListLiteLLMModelsUsesWhitelist(t *testing.T) {
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-z": {LiteLLMProvider: "openai", Mode: "chat"},
		"gpt-a": {LiteLLMProvider: "openai", Mode: "chat"},
		"gpt-x": {LiteLLMProvider: "openai", Mode: "chat"},
	}}
	modelSettingSvc := &ModelSettingService{}
	modelSettingSvc.replaceState(map[string]ModelSettingRecord{
		"gpt-z": {SerialNumber: 2, ID: "gpt-z"},
		"gpt-a": {SerialNumber: 1, ID: "gpt-a"},
	}, ModelSettingSummary{LoadedRows: 2}, "test.csv", "test")

	svc := NewModelSquareService(nil, pricingSvc)
	svc.SetModelSettingService(modelSettingSvc)

	items := svc.ListLiteLLMModels()
	require.Len(t, items, 2)
	require.Equal(t, "gpt-z", items[0].Name)
	require.NotNil(t, items[0].SerialNumber)
	require.Equal(t, 2, *items[0].SerialNumber)
	require.Equal(t, "gpt-a", items[1].Name)
}

func TestModelSquareService_ListLiteLLMModelsDiagnosticsCSVOnly(t *testing.T) {
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-a": {LiteLLMProvider: "openai", Mode: "chat"},
	}}
	modelSettingSvc := &ModelSettingService{}
	modelSettingSvc.replaceState(map[string]ModelSettingRecord{
		"gpt-a": {SerialNumber: 2, ID: "gpt-a"},
		"gpt-b": {SerialNumber: 1, ID: "gpt-b"},
	}, ModelSettingSummary{LoadedRows: 2}, "test.csv", "test")

	svc := NewModelSquareService(nil, pricingSvc)
	svc.SetModelSettingService(modelSettingSvc)

	result := svc.ListLiteLLMModelsWithDiagnostics()
	require.Len(t, result.Items, 1)
	require.Equal(t, []ModelSettingRecord{{SerialNumber: 1, ID: "gpt-b"}}, result.Diagnostics.CSVOnlyModels)
}

func TestModelSquareService_OfficialPriceMissingWhenLiteLLMUnmatched(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformAnthropic, Models: []string{"private-model"},
			BillingMode: BillingModeToken,
			InputPrice:  msPtrFloat(1e-6),
		}},
	}}
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), pricingSvc)

	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Nil(t, cards[0].OfficialPrice)
}

func TestModelSquareService_ContextIntervalsConversion(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"},
			BillingMode: BillingModeToken,
			InputPrice:  msPtrFloat(1e-6),
			OutputPrice: msPtrFloat(2e-6),
			Intervals: []PricingInterval{
				{
					MinTokens:            0,
					MaxTokens:            msPtrInt(200000),
					InputPrice:           msPtrFloat(3e-6),
					OutputPrice:          msPtrFloat(1.5e-5),
					CacheReadPrice:       msPtrFloat(3e-7),
					CacheWritePrice:      msPtrFloat(3.75e-6),
					CacheCreation5mPrice: msPtrFloat(4e-6),
					CacheCreation1hPrice: msPtrFloat(6e-6),
					SortOrder:            0,
				},
				{
					MinTokens:  200000,
					MaxTokens:  nil,
					InputPrice: msPtrFloat(6e-6),
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
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), nil)
	cards, err := svc.ListPublic(context.Background())
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

func TestModelSquareService_ZeroPriceTreatedAsUnconfigured(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"gpt-4o"},
			BillingMode: BillingModeToken,
			InputPrice:  msPtrFloat(0), // 显式 0 视为未配置
			OutputPrice: msPtrFloat(2.5e-5),
		}},
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), nil)
	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	p := cards[0].Platforms[0].GroupPrices[0]
	require.Nil(t, p.InputPricePerMTok, "zero must collapse to nil")
	require.NotNil(t, p.OutputPricePerMTok)
}

func TestModelSquareService_PivotAggregatesAcrossPlatformsAndChannels(t *testing.T) {
	// 同一模型 "claude-opus-4-7" 既在 anthropic 平台又通过 antigravity 平台暴露，
	// 同时分属两个不同渠道。期望：一张卡片，2 个 platform section，每 section 一行价格，
	// antigravity 平台的 ChannelChain 包含两个渠道名。
	gA := Group{ID: 1, Name: "auto-anthropic", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	gG := Group{ID: 2, Name: "auto-antigravity", Platform: PlatformAntigravity, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.2, Status: StatusActive}
	channels := []Channel{
		{
			ID: 10, Name: "primary-pool", Status: StatusActive,
			GroupIDs: []int64{1, 2},
			ModelPricing: []ChannelModelPricing{
				{Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: msPtrFloat(1.75e-5)},
				{Platform: PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: msPtrFloat(1.5e-5)},
			},
		},
		{
			ID: 11, Name: "fallback-pool", Status: StatusActive,
			GroupIDs: []int64{2},
			ModelPricing: []ChannelModelPricing{
				{Platform: PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: msPtrFloat(1.7e-5)},
			},
		},
	}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{gA, gG}), nil)
	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "claude-opus-4-7", card.Name)
	require.Len(t, card.Platforms, 2)
	// platforms 按字典序：anthropic 在前，antigravity 在后
	require.Equal(t, PlatformAnthropic, card.Platforms[0].Platform)
	require.Equal(t, PlatformAntigravity, card.Platforms[1].Platform)
	antig := card.Platforms[1]
	require.Len(t, antig.GroupPrices, 1)
	require.ElementsMatch(t, []string{"primary-pool", "fallback-pool"}, antig.GroupPrices[0].ChannelChain)
}

func TestModelSquareService_AppliesModelMetadataOverrides(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"custom-model"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-6),
		}},
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), nil)
	svc.SetModelMetadataReader(stubModelMetadataReader{
		"custom-model": {
			ModelName:     "custom-model",
			DisplayName:   "Custom Model",
			Description:   "Admin maintained description",
			Category:      "gpt",
			ContextWindow: 128000,
			MaxOutput:     8192,
			Capabilities:  []string{"reasoning", "function_calling"},
			Featured:      true,
			IconURL:       "https://example.com/icon.png",
		},
	})

	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "Custom Model", card.DisplayName)
	require.Equal(t, "Admin maintained description", card.Description)
	require.Equal(t, "gpt", card.Category)
	require.Equal(t, 128000, card.ContextWindow)
	require.Equal(t, 8192, card.MaxOutput)
	require.Equal(t, []string{"reasoning", "function_calling"}, card.Capabilities)
	require.True(t, card.Featured)
	require.Equal(t, "https://example.com/icon.png", card.IconURL)
	require.Len(t, card.Platforms[0].GroupPrices, 1, "pricing remains sourced from channel config")
}

func TestModelSquareService_FillsLiteLLMTypeModalitiesAndSupportFlags(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformGemini, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformGemini, Models: []string{"gemini-test"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-6),
		}},
	}}
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
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
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), pricingSvc)

	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	card := cards[0]
	require.Equal(t, "chat", card.ModelType)
	require.Equal(t, []string{"text", "image", "audio", "video"}, card.InputModalities)
	require.Equal(t, []string{"text", "image"}, card.OutputModalities)
	require.Equal(t, []string{"vision", "web_search", "response_schema"}, card.SupportFlags)
	require.Equal(t, []string{"vision", "function_calling"}, card.Capabilities)
}

func TestModelSquareService_EndpointsUseOverriddenModelType(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"custom-image-model"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-6),
		}},
	}}
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"custom-image-model": {Mode: "chat"},
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), pricingSvc)
	svc.SetModelMetadataReader(stubModelMetadataReader{
		"custom-image-model": {ModelName: "custom-image-model", ModelType: "image_generation"},
	})
	svc.SetModelEndpointConfigReader(stubModelEndpointConfigReader{cfg: &ModelEndpointConfig{
		Platforms: map[string]map[string][]ModelEndpoint{
			PlatformOpenAI: {
				"chat":             {{Method: "POST", Path: "/v1/chat/completions"}},
				"image_generation": {{Method: "POST", Path: "/v1/images/generations"}},
			},
		},
	}})

	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "image_generation", cards[0].ModelType)
	require.Equal(t, []ModelEndpoint{{Method: "POST", Path: "/v1/images/generations"}}, cards[0].Platforms[0].Endpoints)
}

func TestModelSquareService_UnconfiguredModelTypeEndpointsEmptyWithExplicitConfig(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"text-embedding-custom"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-6),
		}},
	}}
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"text-embedding-custom": {Mode: "embedding"},
	}}
	svc := NewModelSquareService(stubChannelServiceProvider(t, channels, []Group{g}), pricingSvc)
	svc.SetModelEndpointConfigReader(stubModelEndpointConfigReader{cfg: &ModelEndpointConfig{
		Platforms: map[string]map[string][]ModelEndpoint{
			PlatformOpenAI: {
				"chat": {{Method: "POST", Path: "/v1/chat/completions"}},
			},
		},
	}})

	cards, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Len(t, cards, 1)
	require.Equal(t, "embedding", cards[0].ModelType)
	require.Empty(t, cards[0].Platforms[0].Endpoints)
}

func TestInboundEndpointsForPlatform_RoundTripWithNormalize(t *testing.T) {
	// 该断言保证：模型广场展示的 inbound 端点路径，能被网关的 NormalizeInboundEndpoint
	// 重新映射回同一个金标常量，避免两侧路径常量漂移。
	// （NormalizeInboundEndpoint 在 handler 包，无法在 service 内 import，因此这里只验证
	// service 包内部 ms* 常量与 endpoint.go 中导出常量的字面一致——通过下面的 TestEndpoint
	// Constants 间接保证。）
	cases := []struct {
		platform string
		mode     string
		want     []ModelEndpoint
	}{
		{PlatformAnthropic, "", []ModelEndpoint{{msEndpointMessages, "POST"}}},
		{PlatformOpenAI, "", []ModelEndpoint{
			{msEndpointChatCompletions, "POST"},
			{msEndpointResponses, "POST"},
		}},
		{PlatformOpenAI, "image_generation", []ModelEndpoint{
			{msEndpointImagesGenerations, "POST"},
			{msEndpointImagesEdits, "POST"},
		}},
		{PlatformGemini, "", []ModelEndpoint{{msEndpointGeminiModels, "POST"}}},
		{PlatformAntigravity, "", []ModelEndpoint{
			{msEndpointMessages, "POST"},
			{msEndpointGeminiModels, "POST"},
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
	require.Equal(t, "/v1/messages", msEndpointMessages)
	require.Equal(t, "/v1/chat/completions", msEndpointChatCompletions)
	require.Equal(t, "/v1/responses", msEndpointResponses)
	require.Equal(t, "/v1/images/generations", msEndpointImagesGenerations)
	require.Equal(t, "/v1/images/edits", msEndpointImagesEdits)
	require.Equal(t, "/v1beta/models", msEndpointGeminiModels)
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

func TestModelSquareService_CacheHit(t *testing.T) {
	// 第二次调用应命中缓存：通过断言 ChannelService.ListAvailable 没有被再次调用
	// 来证明（mockChannelRepository.listAllFn 计数）。
	calls := 0
	channels := []Channel{{
		ID: 1, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformAnthropic, Models: []string{"m"},
			BillingMode: BillingModeToken, InputPrice: msPtrFloat(1e-5),
		}},
	}}
	groups := []Group{{ID: 1, Name: "auto", Platform: PlatformAnthropic, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}}
	repo := &mockChannelRepository{
		listAllFn: func(ctx context.Context) ([]Channel, error) {
			calls++
			return channels, nil
		},
	}
	svc := NewModelSquareService(
		NewChannelService(repo, &stubGroupRepoForAvailable{activeGroups: groups}, nil, nil),
		nil,
	)
	_, err := svc.ListPublic(context.Background())
	require.NoError(t, err)
	_, err = svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, calls, "second call must hit cache")

	svc.InvalidateAll()
	_, err = svc.ListPublic(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, calls, "InvalidateAll forces a rebuild")
}

func TestDeriveCapabilities(t *testing.T) {
	lp := &LiteLLMModelPricing{
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
