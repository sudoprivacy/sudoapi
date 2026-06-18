// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/sync/singleflight"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	systemRewriteTokensKey   = "system_rewrite_tokens"
	systemRewriteTokensUsage = "system_rewrite_tokens_usage"

	systemRewriteTokensUsageCacheTTL  = 60 * time.Second
	systemRewriteTokensUsageErrorTTL  = 5 * time.Second
	systemRewriteTokensUsageDBTimeout = 5 * time.Second
)

func defaultSystemRewriteTokenConfig() *systemRewriteTokenConfig {
	return &systemRewriteTokenConfig{
		Models: map[string]int{
			"claude-fable-5":  500,
			"claude-opus-4-7": 500,
			"claude-opus-4-8": 500,
			"codex":           1300,
			"gpt-5.2":         4400,
			"gpt-5":           5000,
		},
		Default: 360,
	}
}

type systemRewriteTokenConfig struct {
	Models    map[string]int
	Default   int
	expiresAt int64
}

var (
	systemRewriteTokenConfigCache atomic.Value
	systemRewriteTokenConfigSF    singleflight.Group
)

func (s *SettingService) getSystemRewriteTokenConfig(ctx context.Context) *systemRewriteTokenConfig {
	if s == nil || s.settingRepo == nil {
		return defaultSystemRewriteTokenConfig()
	}
	if cached, ok := systemRewriteTokenConfigCache.Load().(*systemRewriteTokenConfig); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached
		}
	}
	val, _, _ := systemRewriteTokenConfigSF.Do(systemRewriteTokensUsage, func() (any, error) {
		if cached, ok := systemRewriteTokenConfigCache.Load().(*systemRewriteTokenConfig); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached, nil
			}
		}
		config := defaultSystemRewriteTokenConfig()
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), systemRewriteTokensUsageDBTimeout)
		defer cancel()
		value, err := s.settingRepo.GetValue(dbCtx, systemRewriteTokensUsage)
		if err != nil {
			slog.Warn("failed to get system rewrite tokens settings", "error", err)
			config.expiresAt = time.Now().Add(systemRewriteTokensUsageErrorTTL).UnixNano()
			systemRewriteTokenConfigCache.Store(config)
			return config, nil
		}
		overrides := map[string]int{}
		if err = json.Unmarshal([]byte(value), &overrides); err != nil {
			slog.Warn("failed to parse system rewrite tokens settings", "error", err)
			config.expiresAt = time.Now().Add(systemRewriteTokensUsageErrorTTL).UnixNano()
			systemRewriteTokenConfigCache.Store(config)
			return config, nil
		}
		if defaultValue, ok := overrides["default"]; ok {
			config.Default = defaultValue
			delete(overrides, "default")
		}
		for model, tokens := range overrides {
			config.Models[model] = tokens
		}
		config.expiresAt = time.Now().Add(systemRewriteTokensUsageCacheTTL).UnixNano()
		systemRewriteTokenConfigCache.Store(config)
		return config, nil
	})
	if cfg, ok := val.(*systemRewriteTokenConfig); ok {
		return cfg
	}
	return defaultSystemRewriteTokenConfig()
}

func (s *GatewayService) systemRewriteInputTokens(ctx context.Context, model string) int {
	var config *systemRewriteTokenConfig
	if s == nil || s.settingService == nil {
		config = defaultSystemRewriteTokenConfig()
	} else {
		config = s.settingService.getSystemRewriteTokenConfig(ctx)
	}
	if tokens, ok := config.Models[model]; ok {
		return tokens
	}
	return config.Default
}

func applySystemRewriteUsage(usage *ClaudeUsage, systemTokens int) bool {
	if usage == nil {
		return false
	}
	// cache read 命中时，代理注入的静态 system 已经按缓存读取计费；这里不再修正。
	if usage.CacheReadInputTokens > 0 {
		return false
	}
	if systemTokens <= 0 {
		return false
	}
	systemTokens = min(systemTokens, usage.InputTokens)
	before := usage.InputTokens
	usage.InputTokens -= systemTokens
	logger.LegacyPrintf(
		"service.gateway",
		"system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		before, usage.InputTokens, systemTokens,
	)
	return true
}

func applySystemRewriteUsageMap(usage map[string]any, systemTokens int) bool {
	if usage == nil {
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
	if systemTokens <= 0 {
		return false
	}
	systemTokens = min(systemTokens, inputTokens)
	after := inputTokens - systemTokens
	usage["input_tokens"] = after
	logger.LegacyPrintf(
		"service.gateway",
		"system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		inputTokens, after, systemTokens,
	)
	return true
}

func (s *OpenAIGatewayService) instructionsRewriteInputTokens(ctx context.Context, model string) int {
	var config *systemRewriteTokenConfig
	if s == nil || s.settingService == nil {
		config = defaultSystemRewriteTokenConfig()
	} else {
		config = s.settingService.getSystemRewriteTokenConfig(ctx)
	}

	m := strings.ToLower(strings.TrimSpace(model))
	if tokens, ok := config.Models[m]; ok {
		return tokens
	}

	// see openai.CodexBaseInstructionsForModel
	switch {
	case strings.Contains(m, "codex"):
		m = "codex"
	case strings.HasPrefix(m, "gpt-5.2"):
		m = "gpt-5.2"
	case strings.HasPrefix(m, "gpt-5.1"):
		m = "gpt-5.1"
	case strings.HasPrefix(m, "gpt-5"):
		m = "gpt-5"
	}
	if tokens, ok := config.Models[m]; ok {
		return tokens
	}
	if tokens, ok := config.Models["default_openai"]; ok {
		return tokens
	}
	return config.Default
}

func applyOpenAISystemRewriteUsage(usage *OpenAIUsage, systemTokens int) bool {
	if usage == nil {
		return false
	}
	// cache read 命中时，代理注入的静态 system 已经按缓存读取计费；这里不再修正。
	if usage.CacheReadInputTokens > 0 {
		return false
	}
	if systemTokens <= 0 {
		return false
	}
	systemTokens = min(systemTokens, usage.InputTokens)
	before := usage.InputTokens
	usage.InputTokens -= systemTokens
	logger.LegacyPrintf(
		"service.openai_gateway",
		"openai instructions rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		before, usage.InputTokens, systemTokens,
	)
	return true
}

func applyOpenAIResponsesSystemRewriteUsage(usage *apicompat.ResponsesUsage, systemTokens int) bool {
	if usage == nil {
		return false
	}
	cacheReadInputTokens := 0
	if usage.InputTokensDetails != nil {
		cacheReadInputTokens = usage.InputTokensDetails.CachedTokens
	}
	if cacheReadInputTokens > 0 {
		return false
	}
	if systemTokens <= 0 {
		return false
	}
	systemTokens = min(systemTokens, usage.InputTokens)
	before := usage.InputTokens
	usage.InputTokens -= systemTokens
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	logger.LegacyPrintf(
		"service.openai_gateway",
		"openai instructions rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		before, usage.InputTokens, systemTokens,
	)
	return true
}

func applyOpenAISystemRewriteUsageJSON(body []byte, systemTokens int) ([]byte, bool) {
	usage, ok := extractOpenAIUsageFromJSONBytes(body)
	if !ok {
		return body, false
	}
	if !applyOpenAISystemRewriteUsage(&usage, systemTokens) {
		return body, false
	}

	path := "usage.input_tokens"
	totalPath := "usage.total_tokens"
	if !gjson.GetBytes(body, "usage").Exists() && gjson.GetBytes(body, "response.usage").Exists() {
		path = "response.usage.input_tokens"
		totalPath = "response.usage.total_tokens"
	}
	updated, err := sjson.SetBytes(body, path, usage.InputTokens)
	if err != nil {
		return body, false
	}
	if gjson.GetBytes(updated, totalPath).Exists() {
		if updatedTotal, err := sjson.SetBytes(updated, totalPath, usage.InputTokens+usage.OutputTokens); err == nil {
			updated = updatedTotal
		}
	}
	return updated, true
}

func rewriteOpenAISSEBodySystemRewriteUsage(body string, systemTokens int) (string, bool) {
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
		updated, lineChanged := applyOpenAISystemRewriteUsageJSON([]byte(data), systemTokens)
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
