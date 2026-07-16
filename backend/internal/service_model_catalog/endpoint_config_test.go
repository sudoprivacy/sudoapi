// sudoapi: Model catalog.

//go:build unit

package service_model_catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type endpointConfigSettingRepo struct {
	values map[string]string
}

func (r *endpointConfigSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *endpointConfigSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if r.values == nil {
		return "", service.ErrSettingNotFound
	}
	v, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return v, nil
}

func (r *endpointConfigSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = make(map[string]string)
	}
	r.values[key] = value
	return nil
}

func (r *endpointConfigSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *endpointConfigSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (r *endpointConfigSettingRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *endpointConfigSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestModelCatalogEndpointConfigService_DefaultWhenSettingMissing(t *testing.T) {
	svc := NewEndpointConfigService(&endpointConfigSettingRepo{})

	cfg, err := svc.GetEndpointConfig(context.Background())
	require.NoError(t, err)

	require.Equal(t, []Endpoint{
		{Path: endpointEmbeddings, Method: "POST"},
	}, ResolveEndpoints(cfg, service.PlatformOpenAI, "embedding"))
	require.Equal(t, []Endpoint{
		{Path: endpointImagesGenerations, Method: "POST"}, {Path: endpointImagesEdits, Method: "POST"},
	}, ResolveEndpoints(cfg, service.PlatformOpenAI, "image_generation"))
	require.Equal(t, []Endpoint{
		{Path: endpointChatCompletions, Method: "POST"}, {Path: endpointResponses, Method: "POST"},
	}, ResolveEndpoints(cfg, service.PlatformGrok, "chat"))
	require.Equal(t, []Endpoint{
		{Path: endpointImagesGenerations, Method: "POST"}, {Path: endpointImagesEdits, Method: "POST"},
	}, ResolveEndpoints(cfg, service.PlatformGrok, "image_generation"))
	require.Nil(t, ResolveEndpoints(cfg, service.PlatformGrok, "embedding"))
}

func TestNormalizeModelCatalogEndpointConfig_NormalizesAndDeduplicates(t *testing.T) {
	cfg, err := NormalizeEndpointConfig(&EndpointConfig{
		Platforms: map[string]map[string][]Endpoint{
			" OpenAI ": {
				" Chat ": {
					{Method: " post ", Path: " /v1/chat/completions "},
					{Method: "POST", Path: "/v1/chat/completions"},
					{Method: " get ", Path: " /v1/models "},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []Endpoint{
		{Path: "/v1/chat/completions", Method: "POST"},
		{Path: "/v1/models", Method: "GET"},
	}, cfg.Platforms["openai"]["chat"])
}

func TestNormalizeModelCatalogEndpointConfig_RejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		cfg  *EndpointConfig
	}{
		{
			name: "bad platform key",
			cfg: &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
				"bad platform": {"chat": {{Method: "POST", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad model type key",
			cfg: &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
				"openai": {"bad type": {{Method: "POST", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad method",
			cfg: &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
				"openai": {"chat": {{Method: "PATCH", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad path",
			cfg: &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
				"openai": {"chat": {{Method: "POST", Path: "v1/chat/completions"}}},
			}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NormalizeEndpointConfig(c.cfg)
			require.ErrorIs(t, err, ErrEndpointConfigInvalid)
		})
	}
}

func TestResolveModelCatalogEndpoints_ExactPlatformAndModelTypeOnly(t *testing.T) {
	cfg := &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
		"openai": {
			"chat": {{Method: "POST", Path: "/v1/chat/completions"}},
		},
	}}

	require.Equal(t, []Endpoint{{Method: "POST", Path: "/v1/chat/completions"}}, ResolveEndpoints(cfg, " OpenAI ", " Chat "))
	require.Nil(t, ResolveEndpoints(cfg, "openai", "embedding"))
	require.Nil(t, ResolveEndpoints(cfg, "anthropic", "chat"))
}

func TestModelCatalogEndpointConfigService_SetPersistsNormalizedJSON(t *testing.T) {
	repo := &endpointConfigSettingRepo{}
	svc := NewEndpointConfigService(repo)

	cfg, err := svc.SetEndpointConfig(context.Background(), &EndpointConfig{
		Platforms: map[string]map[string][]Endpoint{
			"OpenAI": {
				"CHAT": {{Method: "post", Path: " /v1/chat/completions "}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []Endpoint{{Method: "POST", Path: "/v1/chat/completions"}}, cfg.Platforms["openai"]["chat"])

	var stored EndpointConfig
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyModelCatalogEndpointConfig]), &stored))
	require.Equal(t, cfg, &stored)
}
