// sudoapi: Model catalog.

package service_model_catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/samber/lo"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewEndpointConfigService(settingRepo service.SettingRepository) *EndpointConfigService {
	return &EndpointConfigService{settingRepo: settingRepo}
}

type EndpointConfigService struct {
	settingRepo service.SettingRepository
}

func (s *EndpointConfigService) GetEndpointConfig(ctx context.Context) (*EndpointConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultModelCatalogEndpointConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelCatalogEndpointConfig)
	if errors.Is(err, service.ErrSettingNotFound) || strings.TrimSpace(raw) == "" {
		return defaultModelCatalogEndpointConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get model catalog endpoint config: %w", err)
	}
	var cfg EndpointConfig
	if err = json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrEndpointConfigInvalid.WithMetadata(map[string]string{"field": "json"})
	}
	cleaned, err := NormalizeEndpointConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return cleaned, nil
}

func (s *EndpointConfigService) SetEndpointConfig(ctx context.Context, cfg *EndpointConfig) (*EndpointConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("model catalog endpoint config service is not configured")
	}
	cleaned, err := NormalizeEndpointConfig(cfg)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("marshal model catalog endpoint config: %w", err)
	}
	if err = s.settingRepo.Set(ctx, SettingKeyModelCatalogEndpointConfig, string(body)); err != nil {
		return nil, fmt.Errorf("set model catalog endpoint config: %w", err)
	}
	return cleaned, nil
}

// 模型目录各项端点的金标常量（来自 handler/endpoint.go，避免循环依赖故在此重复定义）。
// 单测断言两侧一致。
const (
	endpointMessages          = "/v1/messages"
	endpointChatCompletions   = "/v1/chat/completions"
	endpointResponses         = "/v1/responses"
	endpointEmbeddings        = "/v1/embeddings"
	endpointImagesGenerations = "/v1/images/generations"
	endpointImagesEdits       = "/v1/images/edits"
	endpointGeminiModels      = "/v1beta/models"
)

const SettingKeyModelCatalogEndpointConfig = "model_catalog_endpoint_config"

var (
	ErrEndpointConfigInvalid = infraerrors.BadRequest("MODEL_CATALOG_ENDPOINT_CONFIG_INVALID", "model catalog endpoint config is invalid")
	endpointConfigKeyPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

type (
	EndpointConfigReader interface {
		GetEndpointConfig(ctx context.Context) (*EndpointConfig, error)
	}

	EndpointConfig struct {
		Platforms map[string]map[string][]Endpoint `json:"platforms"`
	}

	// Endpoint 模型在某平台对外暴露的入站端点（用户实际请求的路径）。
	Endpoint struct {
		Path   string `json:"path"`
		Method string `json:"method"`
	}
)

func defaultModelCatalogEndpointConfig() *EndpointConfig {
	genericTypes := []string{"chat", "responses", "completion", "embedding", "image", "image_generation", "audio_speech", "audio_transcription"}
	openaiTypes := []string{"chat", "responses", "completion", "audio_speech", "audio_transcription"}

	postEndpoint := func(path string) Endpoint {
		return Endpoint{Path: path, Method: "POST"}
	}
	endpointRulesForTypes := func(types []string, endpoints ...Endpoint) map[string][]Endpoint {
		return lo.SliceToMap(types, func(typ string) (string, []Endpoint) {
			return typ, slices.Clone(endpoints)
		})
	}

	cfg := &EndpointConfig{Platforms: map[string]map[string][]Endpoint{
		service.PlatformAnthropic:   endpointRulesForTypes(genericTypes, postEndpoint(endpointMessages)),
		service.PlatformOpenAI:      endpointRulesForTypes(openaiTypes, postEndpoint(endpointChatCompletions), postEndpoint(endpointResponses)),
		service.PlatformGrok:        endpointRulesForTypes(openaiTypes, postEndpoint(endpointChatCompletions), postEndpoint(endpointResponses)),
		service.PlatformGemini:      endpointRulesForTypes(genericTypes, postEndpoint(endpointGeminiModels)),
		service.PlatformAntigravity: endpointRulesForTypes(genericTypes, postEndpoint(endpointMessages), postEndpoint(endpointGeminiModels)),
	}}

	cfg.Platforms[service.PlatformOpenAI]["embedding"] = []Endpoint{postEndpoint(endpointEmbeddings)}
	cfg.Platforms[service.PlatformOpenAI]["image"] = []Endpoint{postEndpoint(endpointImagesGenerations), postEndpoint(endpointImagesEdits)}
	cfg.Platforms[service.PlatformOpenAI]["image_generation"] = []Endpoint{postEndpoint(endpointImagesGenerations), postEndpoint(endpointImagesEdits)}
	cfg.Platforms[service.PlatformGrok]["image"] = []Endpoint{postEndpoint(endpointImagesGenerations), postEndpoint(endpointImagesEdits)}
	cfg.Platforms[service.PlatformGrok]["image_generation"] = []Endpoint{postEndpoint(endpointImagesGenerations), postEndpoint(endpointImagesEdits)}

	return cfg
}

func NormalizeEndpointConfig(cfg *EndpointConfig) (*EndpointConfig, error) {
	rst := &EndpointConfig{Platforms: map[string]map[string][]Endpoint{}}
	if cfg == nil || len(cfg.Platforms) == 0 {
		return rst, nil
	}

	for rawPlatform, rawRules := range cfg.Platforms {
		platform := strings.ToLower(strings.TrimSpace(rawPlatform))
		if !endpointConfigKeyPattern.MatchString(platform) {
			return nil, ErrEndpointConfigInvalid.WithMetadata(map[string]string{"field": "platform", "value": rawPlatform})
		}

		for rawType, rawEndpoints := range rawRules {
			modelType := strings.ToLower(strings.TrimSpace(rawType))
			if !endpointConfigKeyPattern.MatchString(modelType) {
				return nil, ErrEndpointConfigInvalid.WithMetadata(map[string]string{"field": "model_type", "value": rawType})
			}

			endpoints := make([]Endpoint, 0, len(rawEndpoints))
			seen := make(map[string]struct{}, len(rawEndpoints))
			for _, endpoint := range rawEndpoints {
				method := strings.ToUpper(strings.TrimSpace(endpoint.Method))
				if method != "GET" && method != "POST" {
					return nil, ErrEndpointConfigInvalid.WithMetadata(map[string]string{"field": "method", "value": endpoint.Method})
				}
				path := strings.TrimSpace(endpoint.Path)
				if path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, " \t\r\n") {
					return nil, ErrEndpointConfigInvalid.WithMetadata(map[string]string{"field": "path", "value": endpoint.Path})
				}
				seenKey := method + " " + path
				if _, ok := seen[seenKey]; ok {
					continue
				}
				seen[seenKey] = struct{}{}
				endpoints = append(endpoints, Endpoint{Path: path, Method: method})
			}
			if len(endpoints) > 0 {
				if rst.Platforms[platform] == nil {
					rst.Platforms[platform] = map[string][]Endpoint{}
				}
				rst.Platforms[platform][modelType] = endpoints
			}
		}
	}

	return rst, nil
}

func ResolveEndpoints(cfg *EndpointConfig, platform, modelType string) []Endpoint {
	if cfg == nil || len(cfg.Platforms) == 0 {
		return nil
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	modelType = strings.ToLower(strings.TrimSpace(modelType))
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
	return slices.Clone(endpoints)
}
