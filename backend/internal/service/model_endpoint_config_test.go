// sudoapi: Model market.

//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type endpointConfigSettingRepo struct {
	values map[string]string
}

func (r *endpointConfigSettingRepo) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (r *endpointConfigSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if r.values == nil {
		return "", ErrSettingNotFound
	}
	v, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
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

func TestModelEndpointConfigService_DefaultWhenSettingMissing(t *testing.T) {
	svc := NewModelEndpointConfigService(&endpointConfigSettingRepo{})

	cfg, err := svc.GetModelEndpointConfig(context.Background())
	require.NoError(t, err)

	require.Equal(t, []ModelEndpoint{
		{Path: msEndpointChatCompletions, Method: "POST"},
		{Path: msEndpointResponses, Method: "POST"},
	}, ResolveModelEndpoints(cfg, PlatformOpenAI, "embedding"))
	require.Equal(t, []ModelEndpoint{
		{Path: msEndpointImagesGenerations, Method: "POST"},
		{Path: msEndpointImagesEdits, Method: "POST"},
	}, ResolveModelEndpoints(cfg, PlatformOpenAI, "image_generation"))
}

func TestNormalizeModelEndpointConfig_NormalizesAndDeduplicates(t *testing.T) {
	cfg, err := NormalizeModelEndpointConfig(&ModelEndpointConfig{
		Platforms: map[string]map[string][]ModelEndpoint{
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
	require.Equal(t, []ModelEndpoint{
		{Path: "/v1/chat/completions", Method: "POST"},
		{Path: "/v1/models", Method: "GET"},
	}, cfg.Platforms["openai"]["chat"])
}

func TestNormalizeModelEndpointConfig_RejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		cfg  *ModelEndpointConfig
	}{
		{
			name: "bad platform key",
			cfg: &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{
				"bad platform": {"chat": {{Method: "POST", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad model type key",
			cfg: &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{
				"openai": {"bad type": {{Method: "POST", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad method",
			cfg: &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{
				"openai": {"chat": {{Method: "PATCH", Path: "/v1/chat/completions"}}},
			}},
		},
		{
			name: "bad path",
			cfg: &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{
				"openai": {"chat": {{Method: "POST", Path: "v1/chat/completions"}}},
			}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NormalizeModelEndpointConfig(c.cfg)
			require.ErrorIs(t, err, ErrModelEndpointConfigInvalid)
		})
	}
}

func TestResolveModelEndpoints_ExactPlatformAndModelTypeOnly(t *testing.T) {
	cfg := &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{
		"openai": {
			"chat": {{Method: "POST", Path: "/v1/chat/completions"}},
		},
	}}

	require.Equal(t, []ModelEndpoint{{Method: "POST", Path: "/v1/chat/completions"}}, ResolveModelEndpoints(cfg, " OpenAI ", " Chat "))
	require.Nil(t, ResolveModelEndpoints(cfg, "openai", "embedding"))
	require.Nil(t, ResolveModelEndpoints(cfg, "anthropic", "chat"))
}

func TestModelEndpointConfigService_SetPersistsNormalizedJSON(t *testing.T) {
	repo := &endpointConfigSettingRepo{}
	svc := NewModelEndpointConfigService(repo)

	cfg, err := svc.SetModelEndpointConfig(context.Background(), &ModelEndpointConfig{
		Platforms: map[string]map[string][]ModelEndpoint{
			"OpenAI": {
				"CHAT": {{Method: "post", Path: " /v1/chat/completions "}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []ModelEndpoint{{Method: "POST", Path: "/v1/chat/completions"}}, cfg.Platforms["openai"]["chat"])

	var stored ModelEndpointConfig
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyModelEndpointConfig]), &stored))
	require.Equal(t, cfg, &stored)
}
