//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyModelRouteResolver_ChannelPricingRoutesWithoutSchedulableAccount(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	channel := Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{2},
		ModelPricing: []ChannelModelPricing{
			{ID: 100, Platform: PlatformGemini, Models: []string{"gemini-2.5-flash"}},
		},
	}
	channelService := newTestChannelService(makeStandardRepo(channel, map[int64]string{2: PlatformGemini}))
	resolver := NewAPIKeyModelRouteResolver(channelService)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "gemini-2.5-flash")

	require.NotNil(t, got)
	require.Equal(t, PlatformGemini, got.Platform)
}

func TestAPIKeyModelRouteResolver_ChannelMappingRoutesAlias(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	anthropicGroup := &Group{ID: 2, Platform: PlatformAnthropic}
	channel := Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{2},
		ModelMapping: map[string]map[string]string{
			PlatformAnthropic: {"sonnet-alias": "claude-sonnet-4-5"},
		},
	}
	channelService := newTestChannelService(makeStandardRepo(channel, map[int64]string{2: PlatformAnthropic}))
	resolver := NewAPIKeyModelRouteResolver(channelService)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, anthropicGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "sonnet-alias")

	require.NotNil(t, got)
	require.Equal(t, PlatformAnthropic, got.Platform)
}

func TestAPIKeyModelRouteResolver_ReturnsNilOnCrossPlatformExplicitSignal(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	channel := Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{1, 2},
		ModelPricing: []ChannelModelPricing{
			{ID: 100, Platform: PlatformOpenAI, Models: []string{"shared-model"}},
			{ID: 200, Platform: PlatformGemini, Models: []string{"shared-model"}},
		},
	}
	channelService := newTestChannelService(makeStandardRepo(channel, map[int64]string{
		1: PlatformOpenAI,
		2: PlatformGemini,
	}))
	resolver := NewAPIKeyModelRouteResolver(channelService)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	require.Nil(t, resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "shared-model"))
}

func TestAPIKeyModelRouteResolver_ModelFamilyRoutesToBoundPlatform(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	anthropicGroup := &Group{ID: 3, Platform: PlatformAnthropic}
	resolver := NewAPIKeyModelRouteResolver(nil)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup, anthropicGroup}}

	tests := []struct {
		name     string
		model    string
		platform string
	}{
		{name: "claude", model: "claude-sonnet-4-5", platform: PlatformAnthropic},
		{name: "anthropic prefix", model: "anthropic.claude-sonnet-4-5", platform: PlatformAnthropic},
		{name: "gemini", model: "gemini-2.5-flash", platform: PlatformGemini},
		{name: "gemini models path", model: "models/gemini-2.5-pro", platform: PlatformGemini},
		{name: "imagen", model: "imagen-4.0-generate-001", platform: PlatformGemini},
		{name: "gpt", model: "gpt-5.2", platform: PlatformOpenAI},
		{name: "o series", model: "o5-mini", platform: PlatformOpenAI},
		{name: "embedding", model: "text-embedding-3-small", platform: PlatformOpenAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, tt.model)

			require.NotNil(t, got)
			require.Equal(t, tt.platform, got.Platform)
		})
	}
}

func TestAPIKeyModelRouteResolver_ModelFamilyDoesNotRouteToUnboundPlatform(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	resolver := NewAPIKeyModelRouteResolver(nil)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "gemini-2.5-flash")

	require.NotNil(t, got)
	require.Equal(t, PlatformOpenAI, got.Platform)
}

func TestAPIKeyModelRouteResolver_UnknownModelFallsBackToOpenAIGroup(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	resolver := NewAPIKeyModelRouteResolver(nil)
	apiKey := &APIKey{Group: geminiGroup, Groups: []*Group{geminiGroup, openAIGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "custom-unknown-model")

	require.NotNil(t, got)
	require.Equal(t, PlatformOpenAI, got.Platform)
}

func TestAPIKeyModelRouteResolver_UnknownModelWithoutOpenAIGroupReturnsNil(t *testing.T) {
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	anthropicGroup := &Group{ID: 3, Platform: PlatformAnthropic}
	resolver := NewAPIKeyModelRouteResolver(nil)
	apiKey := &APIKey{Group: geminiGroup, Groups: []*Group{geminiGroup, anthropicGroup}}

	require.Nil(t, resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "custom-unknown-model"))
}
