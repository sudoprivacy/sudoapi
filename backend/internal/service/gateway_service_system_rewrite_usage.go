// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

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
			"gpt-5.5":         4300,
			"gpt-5.2":         4400,
			"gpt-5.1":         4700,
			"default_openai":  1300,
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
	case strings.HasPrefix(m, "gpt-5.5"):
		m = "gpt-5.5"
	case strings.HasPrefix(m, "gpt-5.2"):
		m = "gpt-5.2"
	case strings.HasPrefix(m, "gpt-5.1"):
		m = "gpt-5.1"
	}
	if tokens, ok := config.Models[m]; ok {
		return tokens
	}
	if tokens, ok := config.Models["default_openai"]; ok {
		return tokens
	}
	return config.Default
}

func applyOpenAIResponsesSystemRewriteUsage(usage *apicompat.ResponsesUsage, systemTokens int) bool {
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

	logger.LegacyPrintf(
		"service.openai_gateway",
		"openai instructions rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		before, usage.InputTokens, systemTokens,
	)
	return true
}
