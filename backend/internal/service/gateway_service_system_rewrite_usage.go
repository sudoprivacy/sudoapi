// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const systemRewriteUsageFixTimeout = 10 * time.Second

const systemRewriteTokensKey = "sudoapi_system_rewrite_tokens"

const fallbackSystemRewriteTokens = 0

var systemRewriteUsageCache sync.Map

func applySystemRewriteUsage(usage *ClaudeUsage, systemTokens int) bool {
	if usage == nil {
		return false
	}
	// cache read 命中时，代理注入的静态 system 已经按缓存读取计费；这里不再修正。
	if usage.CacheReadInputTokens > 0 {
		return false
	}
	if systemTokens <= 0 || usage.InputTokens <= systemTokens {
		return false
	}
	before := usage.InputTokens
	usage.InputTokens -= systemTokens
	logger.LegacyPrintf("service.gateway", "system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		before, usage.InputTokens, systemTokens)
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
	if systemTokens <= 0 || inputTokens <= systemTokens {
		return false
	}
	after := inputTokens - systemTokens
	usage["input_tokens"] = after
	logger.LegacyPrintf("service.gateway", "system rewrite usage deducted: input_tokens %d -> %d deducted_tokens=%d",
		inputTokens, after, systemTokens)
	return true
}

func (s *GatewayService) countSystemRewriteInputTokens(
	ctx context.Context,
	account *Account,
	body []byte,
	token string,
	tokenType string,
	model string,
) (int, error) {
	rewriteSystemMsg := []byte("{}")
	rewriteSystemMsg, _ = sjson.SetRawBytes(rewriteSystemMsg, "model", []byte(gjson.GetBytes(body, "model").Raw))
	rewriteSystemMsg, _ = sjson.SetRawBytes(rewriteSystemMsg, "system", []byte(gjson.GetBytes(body, "system").Raw))
	rewriteSystemMsg, _ = sjson.SetRawBytes(rewriteSystemMsg, "messages", []byte(`[{"role":"user","content":"."}]`))

	cacheKey := string(rewriteSystemMsg)
	if cached, ok := systemRewriteUsageCache.Load(cacheKey); ok {
		if tokens, ok := cached.(int); ok {
			return tokens, nil
		}
	}

	countCtx, cancel := context.WithTimeout(ctx, systemRewriteUsageFixTimeout)
	defer cancel()
	tokens, err := s.countInputTokensForBody(countCtx, account, rewriteSystemMsg, token, tokenType, model, true)
	if err != nil {
		return 0, fmt.Errorf("count proxy-added body: %w", err)
	}

	systemRewriteUsageCache.Store(cacheKey, tokens)
	return tokens, nil
}

func (s *GatewayService) countInputTokensForBody(
	ctx context.Context,
	account *Account,
	body []byte,
	token string,
	tokenType string,
	model string,
	mimicClaudeCode bool,
) (int, error) {
	req, _, err := s.buildCountTokensRequest(ctx, nil, account, body, token, tokenType, model, mimicClaudeCode)
	if err != nil {
		return 0, fmt.Errorf("build count_tokens request: %w", err)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		if !account.IsCustomBaseURLEnabled() || account.GetCustomBaseURL() == "" {
			proxyURL = account.Proxy.URL()
		}
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, nil)
	if err != nil {
		return 0, fmt.Errorf("count_tokens request failed: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return 0, fmt.Errorf("count_tokens response is empty")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read count_tokens response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("count_tokens upstream status=%d message=%s", resp.StatusCode, sanitizeUpstreamErrorMessage(extractUpstreamErrorMessage(respBody)))
	}
	inputTokens := int(gjson.GetBytes(respBody, "input_tokens").Int())
	if inputTokens <= 0 {
		return 0, fmt.Errorf("count_tokens response missing input_tokens")
	}
	return inputTokens, nil
}
