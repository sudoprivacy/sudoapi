// sudoapi: Deduct proxy-injected system prompt usage.

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func TestGatewayService_AnthropicOAuth_SystemRewriteDeductsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","system":"Original system prompt","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), PlatformAnthropic)
	require.NoError(t, err)

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid-oauth-system-usage-deduct"},
			},
			Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-3-5-sonnet-20241022","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1000,"output_tokens":7}}`)),
		},
	}
	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
		settingService:       NewSettingService(&gatewayTTLSettingRepo{data: map[string]string{}}, cfg),
	}
	account := &Account{
		ID:          303,
		Name:        "anthropic-oauth-system-usage-deduct",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 640, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, int64(640), gjson.Get(rec.Body.String(), "usage.input_tokens").Int())
	require.Equal(t, int64(7), gjson.Get(rec.Body.String(), "usage.output_tokens").Int())

	system := gjson.GetBytes(upstream.lastBody, "system")
	require.True(t, system.IsArray())
	require.Equal(t, claudeCodeSystemPrompt, system.Array()[1].Get("text").String())
}

func TestOpenAIGatewayService_Forward_OAuthCodexBaseInstructionsDeductsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstreamSSE := strings.Join([]string{
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"resp_usage_deduct","model":"gpt-5.2","output":[],"usage":{"input_tokens":5000,"output_tokens":200,"total_tokens":5200,"input_tokens_details":{"cached_tokens":0}}}}`,
		``,
		``,
	}, "\n")
	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid_usage_deduct"}},
			Body:       io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
	account := &Account{
		ID:          11,
		Name:        "openai-oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)

	body := []byte(`{"model":"gpt-5.2","stream":false,"input":"hi"}`)
	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, defaultCodexSynthInstructions("gpt-5.2"), gjson.GetBytes(upstream.lastBody, "instructions").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "stream").Bool())
	require.Equal(t, 600, result.Usage.InputTokens)
	require.Equal(t, 200, result.Usage.OutputTokens)
	require.Equal(t, int64(800), gjson.Get(rec.Body.String(), "usage.total_tokens").Int())
	require.Equal(t, int64(600), gjson.Get(rec.Body.String(), "usage.input_tokens").Int())
	require.Equal(t, int64(0), gjson.Get(rec.Body.String(), "usage.input_tokens_details.cached_tokens").Int())
}

func resetSystemRewriteTokenUsageLoaderForTest() {
	systemRewriteTokenUsageLoader = NewSettingLoader[SystemRewriteTokenUsageConfig](
		systemRewriteTokenUsageSettingKey,
		parseSystemRewriteTokenUsageConfig,
	)
}

func TestSystemRewriteTokenUsageConfigDefaults(t *testing.T) {
	cfg := defaultSystemRewriteTokenUsageConfig()

	require.Equal(t, 360, cfg.Default)
	require.Equal(t, 1300, cfg.DefaultOpenAI)
	require.Equal(t, 500, cfg.Models["claude-opus-4-7"])
	require.Equal(t, 4300, cfg.Models["gpt-5.5"])
}

func TestSystemRewriteTokenUsageConfigParsesOverrides(t *testing.T) {
	raw := `{
		"default": 111,
		"Claude-Opus-4-7": 222,
		"gpt-5.2": 333,
		"default_openai": 444,
		"negative": -1
	}`

	cfg, err := parseSystemRewriteTokenUsageConfig(raw)
	require.NoError(t, err)
	require.Equal(t, 111, cfg.Default)
	require.Equal(t, 222, cfg.Models["claude-opus-4-7"])
	require.Equal(t, 333, cfg.Models["gpt-5.2"])
	require.Equal(t, 444, cfg.DefaultOpenAI)
	require.Equal(t, -1, cfg.Models["negative"])
}

func TestSettingServiceGetSystemRewriteTokenUsageConfigUsesCache(t *testing.T) {
	resetSystemRewriteTokenUsageLoaderForTest()
	repo := newSystemRewriteUsageSettingRepo(map[string]string{
		systemRewriteTokenUsageSettingKey: `{"default":123}`,
	})
	svc := NewSettingService(repo, nil)

	first := svc.getSystemRewriteTokenUsageConfig()
	require.Equal(t, 123, first.Default)

	require.NoError(t, repo.Set(context.Background(), systemRewriteTokenUsageSettingKey, `{"default":456}`))
	second := svc.getSystemRewriteTokenUsageConfig()
	require.Equal(t, 123, second.Default)
	require.Equal(t, 1, repo.getCallCount())
}

func TestSystemRewriteTokenUsageConfigFallsBack(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo SettingRepository
	}{
		{name: "missing", repo: newSystemRewriteUsageSettingRepo(nil)},
		{name: "invalid json", repo: newSystemRewriteUsageSettingRepo(map[string]string{systemRewriteTokenUsageSettingKey: `[`})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetSystemRewriteTokenUsageLoaderForTest()
			svc := NewSettingService(tc.repo, nil)

			cfg := svc.getSystemRewriteTokenUsageConfig()

			require.Equal(t, defaultSystemRewriteTokenUsageConfig(), cfg)
		})
	}
}

func TestGatewayServiceSystemTokens(t *testing.T) {
	resetSystemRewriteTokenUsageLoaderForTest()
	repo := newSystemRewriteUsageSettingRepo(map[string]string{
		systemRewriteTokenUsageSettingKey: `{"default":123,"claude-fable-5":456}`,
	})
	svc := &GatewayService{settingService: NewSettingService(repo, nil)}

	require.Equal(t, 456, svc.systemRewriteTokens("Claude-Fable-5"))
	require.Equal(t, 123, svc.systemRewriteTokens("unknown-claude"))
}

func TestOpenAIGatewayServiceSystemTokens(t *testing.T) {
	resetSystemRewriteTokenUsageLoaderForTest()
	repo := newSystemRewriteUsageSettingRepo(map[string]string{
		systemRewriteTokenUsageSettingKey: `{"default_openai":123,"gpt-5.2":456}`,
	})
	svc := &OpenAIGatewayService{settingService: NewSettingService(repo, nil)}

	require.Equal(t, 456, svc.systemRewriteTokens("gpt-5.2-codex"))
	require.Equal(t, 123, svc.systemRewriteTokens("gpt-4.1"))
}

func TestApplySystemRewriteUsage(t *testing.T) {
	usage := &ClaudeUsage{InputTokens: 1000}
	s := GatewayService{}

	require.True(t, s.applySystemRewriteUsage(usage, 360))
	require.Equal(t, 640, usage.InputTokens)

	cached := &ClaudeUsage{InputTokens: 1000, CacheReadInputTokens: 1}
	require.False(t, s.applySystemRewriteUsage(cached, 360))
	require.Equal(t, 1000, cached.InputTokens)

	small := &ClaudeUsage{InputTokens: 100}
	require.True(t, s.applySystemRewriteUsage(small, 360))
	require.Equal(t, 100, small.InputTokens)
}

func TestApplySystemRewriteUsageMap(t *testing.T) {
	usage := map[string]any{"input_tokens": float64(500)}
	s := GatewayService{}

	require.True(t, s.applySystemRewriteUsageMap(usage, 600))
	require.Equal(t, 500, usage["input_tokens"])

	cached := map[string]any{"input_tokens": float64(500), "cache_read_input_tokens": float64(1)}
	require.False(t, s.applySystemRewriteUsageMap(cached, 100))
	require.Equal(t, float64(500), cached["input_tokens"])

	small := map[string]any{"input_tokens": float64(100)}
	require.True(t, s.applySystemRewriteUsageMap(small, 360))
	require.Equal(t, 100, small["input_tokens"])
}

func TestApplyOpenAIResponsesSystemRewriteUsage(t *testing.T) {
	usage := &apicompat.ResponsesUsage{
		InputTokens:  1000,
		OutputTokens: 200,
		TotalTokens:  1200,
		InputTokensDetails: &apicompat.ResponsesInputTokensDetails{
			CachedTokens: 500,
		},
	}
	s := OpenAIGatewayService{}

	require.True(t, s.applyResponsesSystemRewriteUsage(usage, 300))
	require.Equal(t, 700, usage.InputTokens)
	require.Equal(t, 200, usage.InputTokensDetails.CachedTokens)
	require.Equal(t, 900, usage.TotalTokens)
}

func TestApplyOpenAISystemRewriteUsageJSON(t *testing.T) {
	body := []byte(`{"usage":{"input_tokens":1000,"output_tokens":200,"total_tokens":1200,"input_tokens_details":{"cached_tokens":500}}}`)
	s := OpenAIGatewayService{}
	updated, ok := s.applySystemRewriteUsageJSON(body, 300)

	require.True(t, ok)
	require.JSONEq(t, `{"usage":{"input_tokens":700,"output_tokens":200,"total_tokens":900,"input_tokens_details":{"cached_tokens":200}}}`, string(updated))
}

func TestApplyOpenAISystemRewriteUsageJSONPreservesPromptTokensSchema(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":1000,"completion_tokens":200,"total_tokens":1200,"prompt_tokens_details":{"cached_tokens":500}}}`)
	s := OpenAIGatewayService{}
	updated, ok := s.applySystemRewriteUsageJSON(body, 300)

	require.True(t, ok)
	require.JSONEq(t, `{"usage":{"prompt_tokens":700,"completion_tokens":200,"total_tokens":900,"prompt_tokens_details":{"cached_tokens":200}}}`, string(updated))
	require.False(t, gjson.GetBytes(updated, "usage.input_tokens").Exists())
	require.False(t, gjson.GetBytes(updated, "usage.input_tokens_details").Exists())
}

func TestApplyOpenAISystemRewriteUsageJSONUpdatesTopLevelAndResponseUsage(t *testing.T) {
	body := []byte(`{"usage":{"input_tokens":1000,"output_tokens":200,"total_tokens":1200},"response":{"usage":{"input_tokens":900,"output_tokens":100,"total_tokens":1000}}}`)
	s := OpenAIGatewayService{}
	updated, ok := s.applySystemRewriteUsageJSON(body, 300)

	require.True(t, ok)
	require.JSONEq(t, `{"usage":{"input_tokens":700,"output_tokens":200,"total_tokens":900},"response":{"usage":{"input_tokens":600,"output_tokens":100,"total_tokens":700}}}`, string(updated))
}

func TestApplyOpenAISystemRewriteUsageJSONReturnsFalseWithoutUsage(t *testing.T) {
	body := []byte(`{"type":"response.output_text.delta","delta":"hello"}`)
	s := OpenAIGatewayService{}
	updated, ok := s.applySystemRewriteUsageJSON(body, 300)

	require.False(t, ok)
	require.Equal(t, body, updated)
}

func TestApplyOpenAISystemRewriteUsageJSONSkipsUsageWithoutInputFields(t *testing.T) {
	s := OpenAIGatewayService{}
	for _, body := range [][]byte{
		[]byte(`{"usage":{}}`),
		[]byte(`{"usage":{"total_tokens":100}}`),
	} {
		updated, ok := s.applySystemRewriteUsageJSON(body, 300)

		require.False(t, ok)
		require.Equal(t, body, updated)
	}
}

func TestRewriteOpenAISSEBodySystemRewriteUsage(t *testing.T) {
	body := "event: response.completed\n" +
		`data: {"response":{"usage":{"input_tokens":1000,"output_tokens":200,"total_tokens":1200,"input_tokens_details":{"cached_tokens":500}}}}` + "\n\n"
	s := OpenAIGatewayService{}
	updated, ok := s.rewriteSSEBodySystemRewriteUsage(body, 300)

	require.True(t, ok)
	require.Contains(t, updated, `"input_tokens":700`)
	require.Contains(t, updated, `"total_tokens":900`)
	require.Contains(t, updated, `"cached_tokens":200`)
}

type systemRewriteUsageSettingRepo struct {
	values   map[string]string
	getCalls int
}

func newSystemRewriteUsageSettingRepo(values map[string]string) *systemRewriteUsageSettingRepo {
	if values == nil {
		values = map[string]string{}
	}
	return &systemRewriteUsageSettingRepo{values: values}
}

func (r *systemRewriteUsageSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	r.getCalls++
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value, UpdatedAt: time.Now()}, nil
}

func (r *systemRewriteUsageSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *systemRewriteUsageSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *systemRewriteUsageSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *systemRewriteUsageSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		if err := r.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *systemRewriteUsageSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	result := map[string]string{}
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}

func (r *systemRewriteUsageSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func (r *systemRewriteUsageSettingRepo) getCallCount() int {
	return r.getCalls
}
