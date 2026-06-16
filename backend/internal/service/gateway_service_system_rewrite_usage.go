// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

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
