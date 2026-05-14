//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

func TestToCardDTOs_FieldWhitelistShape(t *testing.T) {
	// 严格断言序列化结果只包含白名单 JSON key，不能漏暴露 channel_id / 调度元数据。
	card := service.ModelSquareCard{
		Name:          "claude-opus-4-7",
		DisplayName:   "Claude Opus 4.7",
		Category:      "claude",
		Description:   "desc",
		ContextWindow: 200000,
		MaxOutput:     64000,
		Capabilities:  []string{"vision", "reasoning"},
		Featured:      true,
		IconURL:       "https://cdn/icon.png",
		Platforms: []service.ModelPlatformSection{
			{
				Platform:  "anthropic",
				Endpoints: []service.ModelEndpoint{{Path: "/v1/messages", Method: "POST"}},
				GroupPrices: []service.ModelGroupPrice{
					{
						GroupID:          1,
						GroupName:        "auto",
						SubscriptionType: "standard",
						IsExclusive:      false,
						BaseRateMult:     1.0,
						BillingMode:      service.BillingModeToken,
						InputPricePerMTok: func() *float64 { v := 17.5; return &v }(),
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
		"context_window", "max_output", "capabilities",
		"featured", "icon_url", "platforms",
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
	platform := platforms[0].(map[string]any)
	prices, _ := platform["group_prices"].([]any)
	require.Len(t, prices, 1)
	row := prices[0].(map[string]any)
	// 价格字段命名遵循 USD/MTok 约定
	require.Contains(t, row, "input_price_per_mtok_usd")
	require.Contains(t, row, "output_price_per_mtok_usd")
	require.Contains(t, row, "cache_read_price_per_mtok_usd")
	require.Contains(t, row, "cache_write_price_per_mtok_usd")
	require.Contains(t, row, "channel_chain")
	require.Contains(t, row, "base_rate_multiplier")
	// 未传 userRateMultipliers 时 user_rate_multiplier 必须是 null（json: null → nil 反序列化为 nil）
	require.Nil(t, row["user_rate_multiplier"])
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
