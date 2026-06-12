// sudoapi: Deduct proxy-injected Claude Code system prompt usage.

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type systemRewriteUsageHTTPUpstream struct {
	responses []string
	bodies    [][]byte
}

func (u *systemRewriteUsageHTTPUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		u.bodies = append(u.bodies, body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	body := `{"input_tokens":0}`
	if len(u.responses) > 0 {
		body = u.responses[0]
		u.responses = u.responses[1:]
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

func (u *systemRewriteUsageHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestGatewayService_DeductSystemRewriteUsage_DeductsCountTokensDelta(t *testing.T) {
	systemRewriteUsageCache.Clear()

	originalSystem := "respond tersely"
	originalBody := []byte(`{"model":"claude-sonnet-4-5","system":"respond tersely","messages":[{"role":"user","content":"hello"}],"max_tokens":16}`)
	rewrittenBody := rewriteSystemForNonClaudeCode(originalBody, originalSystem)
	upstream := &systemRewriteUsageHTTPUpstream{
		responses: []string{`{"input_tokens":20}`},
	}
	svc := &GatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          1,
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{},
	}
	usage := &ClaudeUsage{InputTokens: 150}

	systemTokens, err := svc.countSystemRewriteInputTokens(context.Background(), account, rewrittenBody, "test-key", "apikey", "claude-sonnet-4-5")
	require.NoError(t, err)
	require.True(t, applySystemRewriteUsage(usage, systemTokens))

	require.Equal(t, 130, usage.InputTokens)
	require.Len(t, upstream.bodies, 1)
	require.Contains(t, gjson.GetBytes(upstream.bodies[0], "system.0.text").String(), "cc_version=")
}

func TestGatewayService_DeductSystemRewriteUsage_SkipsWhenCacheReadHit(t *testing.T) {
	systemRewriteUsageCache.Clear()

	usage := &ClaudeUsage{InputTokens: 150, CacheReadInputTokens: 10}

	require.False(t, applySystemRewriteUsage(usage, 20))

	require.Equal(t, 150, usage.InputTokens)
}

func TestGatewayService_DeductSystemRewriteUsage_CachesByProxyAddedBody(t *testing.T) {
	systemRewriteUsageCache.Clear()

	originalSystem := "respond tersely"
	originalBody := []byte(`{"model":"claude-sonnet-4-5","system":"respond tersely","messages":[{"role":"user","content":"hello"}]}`)
	rewrittenBody := rewriteSystemForNonClaudeCode(originalBody, originalSystem)
	upstream := &systemRewriteUsageHTTPUpstream{responses: []string{`{"input_tokens":20}`}}
	svc := &GatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{}}
	first := &ClaudeUsage{InputTokens: 150}
	second := &ClaudeUsage{InputTokens: 90}

	firstTokens, err := svc.countSystemRewriteInputTokens(context.Background(), account, rewrittenBody, "test-key", "apikey", "claude-sonnet-4-5")
	require.NoError(t, err)
	secondTokens, err := svc.countSystemRewriteInputTokens(context.Background(), account, rewrittenBody, "test-key", "apikey", "claude-sonnet-4-5")
	require.NoError(t, err)
	require.True(t, applySystemRewriteUsage(first, firstTokens))
	require.True(t, applySystemRewriteUsage(second, secondTokens))

	require.Equal(t, 130, first.InputTokens)
	require.Equal(t, 70, second.InputTokens)
	require.Len(t, upstream.bodies, 1)
}
