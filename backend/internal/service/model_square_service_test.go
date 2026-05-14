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

func ptrFloat(v float64) *float64 { return &v }

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
		InputPrice:  ptrFloat(1.75e-5),
		OutputPrice: ptrFloat(8.75e-5),
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
			BillingMode: BillingModeToken, InputPrice: ptrFloat(1.75e-5),
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
			BillingMode:     BillingModeToken,
			InputPrice:      ptrFloat(1.75e-5),  // → 17.5 / MTok
			OutputPrice:     ptrFloat(8.75e-5),  // → 87.5 / MTok
			CacheReadPrice:  ptrFloat(1.75e-6),  // → 1.75 / MTok
			CacheWritePrice: ptrFloat(2.1875e-5), // → 21.875 / MTok
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
}

func TestModelSquareService_ZeroPriceTreatedAsUnconfigured(t *testing.T) {
	g := Group{ID: 1, Name: "auto", Platform: PlatformOpenAI, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1.0, Status: StatusActive}
	channels := []Channel{{
		ID: 10, Name: "ch", Status: StatusActive,
		GroupIDs: []int64{1},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"gpt-4o"},
			BillingMode: BillingModeToken,
			InputPrice:  ptrFloat(0), // 显式 0 视为未配置
			OutputPrice: ptrFloat(2.5e-5),
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
				{Platform: PlatformAnthropic, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: ptrFloat(1.75e-5)},
				{Platform: PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: ptrFloat(1.5e-5)},
			},
		},
		{
			ID: 11, Name: "fallback-pool", Status: StatusActive,
			GroupIDs: []int64{2},
			ModelPricing: []ChannelModelPricing{
				{Platform: PlatformAntigravity, Models: []string{"claude-opus-4-7"}, BillingMode: BillingModeToken, InputPrice: ptrFloat(1.7e-5)},
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
		"claude-opus-4-7":      "claude",
		"Claude-Sonnet-4-6":    "claude",
		"gpt-4o":               "gpt",
		"o3-mini":              "gpt",
		"codex-mini":           "gpt",
		"chatgpt-4o-latest":    "gpt",
		"gemini-2.5-pro":       "gemini",
		"dall-e-3":             "image",
		"imagen-3":             "image",
		"text-embedding-3-small": "embedding",
		"some-random-model":    "other",
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
			BillingMode: BillingModeToken, InputPrice: ptrFloat(1e-5),
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
