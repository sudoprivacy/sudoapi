// sudoapi: Deduct proxy-injected system prompt usage.

package service

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	systemRewriteTokenKey             = "system_rewrite_token"
	systemRewriteTokenUsageSettingKey = "system_rewrite_token_usage"
)

type SystemRewriteTokenUsageConfig struct {
	Models        map[string]int
	Default       int
	DefaultOpenAI int
}

var systemRewriteTokenUsageLoader = NewSettingLoader[SystemRewriteTokenUsageConfig](systemRewriteTokenUsageSettingKey, parseSystemRewriteTokenUsageConfig)

func defaultSystemRewriteTokenUsageConfig() SystemRewriteTokenUsageConfig {
	return SystemRewriteTokenUsageConfig{
		Models: map[string]int{
			"claude-fable-5":  500,
			"claude-opus-4-7": 500,
			"claude-opus-4-8": 500,
			"gpt-5.5":         4300,
			"gpt-5.2":         4400,
			"gpt-5.1":         4700,
		},
		Default:       360,
		DefaultOpenAI: 1300,
	}
}

func parseSystemRewriteTokenUsageConfig(raw string) (SystemRewriteTokenUsageConfig, error) {
	config := defaultSystemRewriteTokenUsageConfig()
	var overrides map[string]int
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		return config, err
	}
	for key, tokens := range overrides {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "default" {
			config.Default = tokens
			continue
		}
		if key == "default_openai" {
			config.DefaultOpenAI = tokens
			continue
		}
		config.Models[key] = tokens
	}
	return config, nil
}

func (s *SettingService) getSystemRewriteTokenUsageConfig() SystemRewriteTokenUsageConfig {
	fallback := defaultSystemRewriteTokenUsageConfig()
	if s == nil || s.settingRepo == nil {
		return fallback
	}
	return systemRewriteTokenUsageLoader.Get(s.settingRepo, fallback)
}

func (s *GatewayService) systemRewriteTokens(model string) int {
	config := s.settingService.getSystemRewriteTokenUsageConfig()
	model = strings.ToLower(strings.TrimSpace(model))
	if tokens, ok := config.Models[model]; ok {
		return tokens
	}
	return config.Default
}

func (s *GatewayService) applySystemRewriteUsage(usage *ClaudeUsage, systemTokens int) bool {
	if usage == nil || systemTokens <= 0 {
		return false
	}
	// cache read 命中时，代理注入的静态 system 已经按缓存读取计费；这里不再修正。
	if usage.CacheReadInputTokens > 0 {
		return false
	}
	before := usage.InputTokens
	if usage.InputTokens > systemTokens {
		usage.InputTokens -= systemTokens
	}
	logger.LegacyPrintf("service.gateway", "system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d", before, usage.InputTokens, systemTokens)
	return true
}

func (s *GatewayService) applySystemRewriteUsageMap(usage map[string]any, systemTokens int) bool {
	if usage == nil || systemTokens <= 0 {
		return false
	}
	cacheRead, _ := parseSSEUsageInt(usage["cache_read_input_tokens"])
	if cacheRead > 0 {
		return false
	}
	inputTokens, ok := parseSSEUsageInt(usage["input_tokens"])
	if !ok {
		return false
	}
	after := inputTokens
	if inputTokens > systemTokens {
		after = inputTokens - systemTokens
	}
	usage["input_tokens"] = after
	logger.LegacyPrintf("service.gateway", "system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d", inputTokens, after, systemTokens)
	return true
}

func (s *OpenAIGatewayService) systemRewriteTokens(model string) int {
	config := s.settingService.getSystemRewriteTokenUsageConfig()
	model = strings.ToLower(strings.TrimSpace(model))
	if tokens, ok := config.Models[model]; ok {
		return tokens
	}
	// see openai.CodexBaseInstructionsForModel
	for _, prefix := range []string{"gpt-5.5", "gpt-5.2", "gpt-5.1"} {
		if strings.HasPrefix(model, prefix) {
			if tokens, ok := config.Models[prefix]; ok {
				return tokens
			}
		}
	}
	return config.DefaultOpenAI
}

func (s *OpenAIGatewayService) applyResponsesSystemRewriteUsage(usage *apicompat.ResponsesUsage, systemTokens int) bool {
	if usage == nil || systemTokens <= 0 {
		return false
	}
	// 无论是不是首轮, 一定包含 instructions
	before := usage.InputTokens
	if usage.InputTokens > systemTokens {
		usage.InputTokens -= systemTokens
	}
	// 缓存命中则扣减
	if usage.InputTokensDetails != nil {
		if usage.InputTokensDetails.CachedTokens >= systemTokens {
			usage.InputTokensDetails.CachedTokens -= systemTokens
		} else {
			usage.InputTokensDetails.CachedTokens = 0
		}
	}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	logger.LegacyPrintf("service.openai_gateway", "openai instructions rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d", before, usage.InputTokens, systemTokens)
	return true
}

func (s *OpenAIGatewayService) applySystemRewriteUsageJSON(body []byte, systemTokens int) ([]byte, bool) {
	if systemTokens <= 0 {
		return body, false
	}

	updated := body
	changed := false

	for _, usagePath := range []string{"usage", "response.usage"} {
		usageObj := gjson.GetBytes(body, usagePath)
		hasPromptTokens := usageObj.Get("prompt_tokens").Exists()
		hasInputTokens := usageObj.Get("input_tokens").Exists()
		if !hasPromptTokens && !hasInputTokens {
			continue
		}
		usage, ok := openAIUsageFromGJSON(usageObj)
		if !ok {
			continue
		}

		// 无论是不是首轮, 一定包含 instructions
		before := usage.InputTokens
		if usage.InputTokens > systemTokens {
			usage.InputTokens -= systemTokens
		}
		// 缓存命中则扣减
		if usage.CacheReadInputTokens >= systemTokens {
			usage.CacheReadInputTokens -= systemTokens
		} else {
			usage.CacheReadInputTokens = 0
		}

		next := updated
		var err error
		if hasPromptTokens {
			if next, err = sjson.SetBytes(next, usagePath+".prompt_tokens", usage.InputTokens); err != nil {
				slog.Warn("failed to update input tokens", "error", err)
				continue
			}
		}
		if hasInputTokens {
			if next, err = sjson.SetBytes(next, usagePath+".input_tokens", usage.InputTokens); err != nil {
				slog.Warn("failed to update input tokens", "error", err)
				continue
			}
		}
		if usageObj.Get("total_tokens").Exists() {
			if next, err = sjson.SetBytes(next, usagePath+".total_tokens", usage.InputTokens+usage.OutputTokens); err != nil {
				slog.Warn("failed to update total tokens", "error", err)
				continue
			}
		}
		if usageObj.Get("prompt_tokens_details.cached_tokens").Exists() {
			if next, err = sjson.SetBytes(next, usagePath+".prompt_tokens_details.cached_tokens", usage.CacheReadInputTokens); err != nil {
				slog.Warn("failed to update cached tokens", "error", err)
				continue
			}
		}
		if usageObj.Get("input_tokens_details.cached_tokens").Exists() {
			if next, err = sjson.SetBytes(next, usagePath+".input_tokens_details.cached_tokens", usage.CacheReadInputTokens); err != nil {
				slog.Warn("failed to update cached tokens", "error", err)
				continue
			}
		}
		updated = next
		changed = true
		logger.LegacyPrintf("service.openai_gateway", "openai instructions rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d", before, usage.InputTokens, systemTokens)
	}

	return updated, changed
}

func (s *OpenAIGatewayService) rewriteSSEBodySystemRewriteUsage(body string, systemTokens int) (string, bool) {
	if systemTokens <= 0 || strings.TrimSpace(body) == "" {
		return body, false
	}
	lines := strings.Split(body, "\n")
	changed := false
	for i, line := range lines {
		data, ok := extractOpenAISSEDataLine(line)
		if !ok {
			continue
		}
		updated, lineChanged := s.applySystemRewriteUsageJSON([]byte(data), systemTokens)
		if !lineChanged {
			continue
		}
		prefix := "data:"
		if strings.HasPrefix(line, "data: ") {
			prefix = "data: "
		}
		lines[i] = prefix + string(updated)
		changed = true
	}
	if !changed {
		return body, false
	}
	return strings.Join(lines, "\n"), true
}
