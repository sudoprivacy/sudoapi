//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyModelRouteResolver_UsesExplicitAccountMapping(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	repo := &mockAccountRepoForPlatform{accounts: []Account{
		{
			ID:          10,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gpt-5.2": "gpt-5.2"},
			},
		},
		{
			ID:          20,
			Platform:    PlatformGemini,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"gemini-2.5-flash": "gemini-2.5-flash"},
			},
		},
	}}
	resolver := NewAPIKeyModelRouteResolver(repo, nil)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "gemini-2.5-flash")

	require.NotNil(t, got)
	require.Equal(t, PlatformGemini, got.Platform)
}

func TestAPIKeyModelRouteResolver_UsesDefaultPlatformSupportWhenNoExplicitMapping(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	repo := &mockAccountRepoForPlatform{accounts: []Account{
		{ID: 10, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Credentials: map[string]any{}},
		{ID: 20, Platform: PlatformGemini, Status: StatusActive, Schedulable: true, Credentials: map[string]any{}},
	}}
	resolver := NewAPIKeyModelRouteResolver(repo, nil)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "gemini-2.5-flash")

	require.NotNil(t, got)
	require.Equal(t, PlatformGemini, got.Platform)
}

func TestAPIKeyModelRouteResolver_ReturnsNilOnCrossPlatformTie(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	repo := &mockAccountRepoForPlatform{accounts: []Account{
		{
			ID:          10,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"shared-model": "gpt-5.2"},
			},
		},
		{
			ID:          20,
			Platform:    PlatformGemini,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{"shared-model": "gemini-2.5-flash"},
			},
		},
	}}
	resolver := NewAPIKeyModelRouteResolver(repo, nil)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	require.Nil(t, resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "shared-model"))
}

func TestAPIKeyModelRouteResolver_ChannelPricingRequiresSchedulableAccount(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	repo := &mockAccountRepoForPlatform{accounts: []Account{
		{ID: 10, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Credentials: map[string]any{}},
	}}
	channel := Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{2},
		ModelPricing: []ChannelModelPricing{
			{ID: 100, Platform: PlatformGemini, Models: []string{"gemini-2.5-flash"}},
		},
	}
	channelService := newTestChannelService(makeStandardRepo(channel, map[int64]string{2: PlatformGemini}))
	resolver := NewAPIKeyModelRouteResolver(repo, channelService)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	require.Nil(t, resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "gemini-2.5-flash"))
}

func TestAPIKeyModelRouteResolver_ChannelMappingCanRouteAliasWithUnrestrictedAccount(t *testing.T) {
	openAIGroup := &Group{ID: 1, Platform: PlatformOpenAI}
	geminiGroup := &Group{ID: 2, Platform: PlatformGemini}
	repo := &mockAccountRepoForPlatform{accounts: []Account{
		{ID: 10, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Credentials: map[string]any{}},
		{ID: 20, Platform: PlatformGemini, Status: StatusActive, Schedulable: true, Credentials: map[string]any{}},
	}}
	channel := Channel{
		ID:       1,
		Status:   StatusActive,
		GroupIDs: []int64{2},
		ModelMapping: map[string]map[string]string{
			PlatformGemini: {"flash-alias": "gemini-2.5-flash"},
		},
	}
	channelService := newTestChannelService(makeStandardRepo(channel, map[int64]string{2: PlatformGemini}))
	resolver := NewAPIKeyModelRouteResolver(repo, channelService)
	apiKey := &APIKey{Group: openAIGroup, Groups: []*Group{openAIGroup, geminiGroup}}

	got := resolver.ResolveAPIKeyGroupForModel(context.Background(), apiKey, "flash-alias")

	require.NotNil(t, got)
	require.Equal(t, PlatformGemini, got.Platform)
}
