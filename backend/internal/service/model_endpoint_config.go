// sudoapi: Model market.

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrModelEndpointConfigInvalid = infraerrors.BadRequest("MODEL_ENDPOINT_CONFIG_INVALID", "model endpoint config is invalid")
	modelEndpointConfigKeyPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

type ModelEndpointConfig struct {
	Platforms map[string]map[string][]ModelEndpoint `json:"platforms"`
}

type ModelEndpointConfigService struct {
	settingRepo SettingRepository
}

type ModelEndpointConfigReader interface {
	GetModelEndpointConfig(ctx context.Context) (*ModelEndpointConfig, error)
}

func NewModelEndpointConfigService(settingRepo SettingRepository) *ModelEndpointConfigService {
	return &ModelEndpointConfigService{settingRepo: settingRepo}
}

func (s *ModelEndpointConfigService) GetModelEndpointConfig(ctx context.Context) (*ModelEndpointConfig, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultModelEndpointConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelEndpointConfig)
	if errors.Is(err, ErrSettingNotFound) || strings.TrimSpace(raw) == "" {
		return DefaultModelEndpointConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model endpoint config: %w", err)
	}
	var cfg ModelEndpointConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrModelEndpointConfigInvalid.WithMetadata(map[string]string{"field": "json"})
	}
	cleaned, err := NormalizeModelEndpointConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return cleaned, nil
}

func (s *ModelEndpointConfigService) SetModelEndpointConfig(ctx context.Context, cfg *ModelEndpointConfig) (*ModelEndpointConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("model endpoint config service is not configured")
	}
	cleaned, err := NormalizeModelEndpointConfig(cfg)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("marshal model endpoint config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelEndpointConfig, string(body)); err != nil {
		return nil, fmt.Errorf("set model endpoint config: %w", err)
	}
	return cleaned, nil
}

func DefaultModelEndpointConfig() *ModelEndpointConfig {
	openAIChatLike := []string{"chat", "responses", "completion", "embedding", "audio_speech", "audio_transcription"}
	genericTypes := []string{"chat", "responses", "completion", "embedding", "image", "image_generation", "audio_speech", "audio_transcription"}
	cfg := &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{}}

	cfg.Platforms[PlatformAnthropic] = endpointRulesForTypes(genericTypes, []ModelEndpoint{{Path: msEndpointMessages, Method: "POST"}})
	cfg.Platforms[PlatformOpenAI] = endpointRulesForTypes(openAIChatLike, []ModelEndpoint{
		{Path: msEndpointChatCompletions, Method: "POST"},
		{Path: msEndpointResponses, Method: "POST"},
	})
	cfg.Platforms[PlatformOpenAI]["image"] = []ModelEndpoint{
		{Path: msEndpointImagesGenerations, Method: "POST"},
		{Path: msEndpointImagesEdits, Method: "POST"},
	}
	cfg.Platforms[PlatformOpenAI]["image_generation"] = []ModelEndpoint{
		{Path: msEndpointImagesGenerations, Method: "POST"},
		{Path: msEndpointImagesEdits, Method: "POST"},
	}
	cfg.Platforms[PlatformGemini] = endpointRulesForTypes(genericTypes, []ModelEndpoint{{Path: msEndpointGeminiModels, Method: "POST"}})
	cfg.Platforms[PlatformAntigravity] = endpointRulesForTypes(genericTypes, []ModelEndpoint{
		{Path: msEndpointMessages, Method: "POST"},
		{Path: msEndpointGeminiModels, Method: "POST"},
	})
	return cfg
}

func endpointRulesForTypes(types []string, endpoints []ModelEndpoint) map[string][]ModelEndpoint {
	out := make(map[string][]ModelEndpoint, len(types))
	for _, t := range types {
		out[t] = append([]ModelEndpoint(nil), endpoints...)
	}
	return out
}

func NormalizeModelEndpointConfig(cfg *ModelEndpointConfig) (*ModelEndpointConfig, error) {
	out := &ModelEndpointConfig{Platforms: map[string]map[string][]ModelEndpoint{}}
	if cfg == nil || len(cfg.Platforms) == 0 {
		return out, nil
	}
	platforms := make([]string, 0, len(cfg.Platforms))
	for platform := range cfg.Platforms {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)
	for _, rawPlatform := range platforms {
		platform := normalizeEndpointConfigKey(rawPlatform)
		if !validEndpointConfigKey(platform) {
			return nil, ErrModelEndpointConfigInvalid.WithMetadata(map[string]string{"field": "platform", "value": rawPlatform})
		}
		modelRules := cfg.Platforms[rawPlatform]
		if len(modelRules) == 0 {
			continue
		}
		types := make([]string, 0, len(modelRules))
		for modelType := range modelRules {
			types = append(types, modelType)
		}
		sort.Strings(types)
		for _, rawType := range types {
			modelType := normalizeEndpointConfigKey(rawType)
			if !validEndpointConfigKey(modelType) {
				return nil, ErrModelEndpointConfigInvalid.WithMetadata(map[string]string{"field": "model_type", "value": rawType})
			}
			endpoints, err := normalizeModelEndpoints(modelRules[rawType])
			if err != nil {
				return nil, err
			}
			if len(endpoints) == 0 {
				continue
			}
			if out.Platforms[platform] == nil {
				out.Platforms[platform] = map[string][]ModelEndpoint{}
			}
			out.Platforms[platform][modelType] = endpoints
		}
	}
	return out, nil
}

func ResolveModelEndpoints(cfg *ModelEndpointConfig, platform, modelType string) []ModelEndpoint {
	if cfg == nil || len(cfg.Platforms) == 0 {
		return nil
	}
	platform = normalizeEndpointConfigKey(platform)
	modelType = normalizeEndpointConfigKey(modelType)
	if platform == "" || modelType == "" {
		return nil
	}
	rules := cfg.Platforms[platform]
	if len(rules) == 0 {
		return nil
	}
	endpoints := rules[modelType]
	if len(endpoints) == 0 {
		return nil
	}
	return append([]ModelEndpoint(nil), endpoints...)
}

func normalizeModelEndpoints(in []ModelEndpoint) ([]ModelEndpoint, error) {
	if len(in) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]ModelEndpoint, 0, len(in))
	for _, ep := range in {
		method := strings.ToUpper(strings.TrimSpace(ep.Method))
		path := strings.TrimSpace(ep.Path)
		if method != "GET" && method != "POST" {
			return nil, ErrModelEndpointConfigInvalid.WithMetadata(map[string]string{"field": "method", "value": ep.Method})
		}
		if path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, " \t\r\n") {
			return nil, ErrModelEndpointConfigInvalid.WithMetadata(map[string]string{"field": "path", "value": ep.Path})
		}
		key := method + " " + path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ModelEndpoint{Path: path, Method: method})
	}
	return out, nil
}

func normalizeEndpointConfigKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validEndpointConfigKey(value string) bool {
	return modelEndpointConfigKeyPattern.MatchString(value)
}
