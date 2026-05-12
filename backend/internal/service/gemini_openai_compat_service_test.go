package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGeminiOpenAICompatTestService(upstream HTTPUpstream) *GeminiOpenAICompatService {
	geminiSvc := &GeminiMessagesCompatService{
		httpUpstream: upstream,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{
					AllowInsecureHTTP: true,
				},
			},
		},
	}
	return NewGeminiOpenAICompatService(geminiSvc)
}

func TestGeminiOpenAICompatServiceForwardResponsesNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"x-request-id": []string{"gemini-resp-1"}},
		Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"hello from gemini"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":4}}`)),
	}}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	result, err := svc.ForwardResponses(context.Background(), c, &Account{
		ID:       1,
		Type:     AccountTypeAPIKey,
		Platform: PlatformGemini,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}, []byte(`{"model":"gemini-2.5-flash","input":[{"role":"user","content":"hello"}]}`), nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "gemini-resp-1", result.RequestID)
	require.Equal(t, "gemini-2.5-flash", result.Model)
	require.Equal(t, "gemini-2.5-flash", result.UpstreamModel)
	require.Equal(t, 10, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, "response", payload["object"])
	require.Equal(t, "gemini-2.5-flash", payload["model"])
	output := payload["output"].([]any)
	message := output[0].(map[string]any)
	content := message["content"].([]any)
	require.Equal(t, "hello from gemini", content[0].(map[string]any)["text"])

	require.Equal(t, 1, httpStub.calls)
	require.Contains(t, httpStub.lastReq.URL.String(), "/v1beta/models/gemini-2.5-flash:generateContent")
	require.Equal(t, "sk-test", httpStub.lastReq.Header.Get("x-goog-api-key"))
}

func TestGeminiOpenAICompatServiceForwardChatNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"x-request-id": []string{"gemini-chat-1"}},
		Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"chat answer"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":8,"candidatesTokenCount":3}}`)),
	}}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	result, err := svc.ForwardChatCompletions(context.Background(), c, &Account{
		ID:       2,
		Type:     AccountTypeAPIKey,
		Platform: PlatformGemini,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}, []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hello"}]}`), nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, w.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, "chat.completion", payload["object"])
	choices := payload["choices"].([]any)
	message := choices[0].(map[string]any)["message"].(map[string]any)
	require.Equal(t, "assistant", message["role"])
	require.Equal(t, "chat answer", message["content"])
	require.Equal(t, 8, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)
}

func TestGeminiOpenAICompatServiceForwardChatStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"x-request-id": []string{"gemini-chat-stream-1"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hel\"}]}}]}\n\n" +
				"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":5,\"candidatesTokenCount\":2}}\n\n",
		)),
	}}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	result, err := svc.ForwardChatCompletions(context.Background(), c, &Account{
		ID:          3,
		Type:        AccountTypeAPIKey,
		Platform:    PlatformGemini,
		Credentials: map[string]any{"api_key": "sk-test"},
	}, []byte(`{"model":"gemini-2.5-flash","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hello"}]}`), nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Equal(t, 5, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	require.Contains(t, body, `"object":"chat.completion.chunk"`)
	require.Contains(t, body, `"content":"Hel"`)
	require.Contains(t, body, `"content":"lo"`)
	require.Contains(t, body, "data: [DONE]")
	require.Contains(t, httpStub.lastReq.URL.String(), "streamGenerateContent?alt=sse")
}

func TestGeminiOpenAICompatServiceRejectsUnsupportedChatCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	_, err := svc.ForwardChatCompletions(context.Background(), c, &Account{ID: 4, Type: AccountTypeAPIKey, Platform: PlatformGemini}, []byte(`{"model":"gemini","messages":[{"role":"user","content":"hi"}],"logprobs":true}`), nil)

	require.ErrorContains(t, err, "logprobs")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, 0, httpStub.calls)
}

func TestGeminiOpenAICompatServiceForwardsToolChoiceToGemini(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"city":"Paris"}}}]},"finishReason":"STOP"}]}`)),
	}}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	body := []byte(`{
		"model":"gemini-2.5-flash",
		"input":[{"role":"user","content":"weather"}],
		"tools":[{"type":"function","name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}],
		"tool_choice":{"type":"function","name":"get_weather"}
	}`)

	_, err := svc.ForwardResponses(context.Background(), c, &Account{
		ID:          5,
		Type:        AccountTypeAPIKey,
		Platform:    PlatformGemini,
		Credentials: map[string]any{"api_key": "sk-test"},
	}, body, nil)

	require.NoError(t, err)
	require.Equal(t, 1, httpStub.calls)
	upstreamBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(upstreamBody, &payload))
	toolConfig := payload["toolConfig"].(map[string]any)
	fcc := toolConfig["functionCallingConfig"].(map[string]any)
	require.Equal(t, "ANY", fcc["mode"])
	require.Equal(t, []any{"get_weather"}, fcc["allowedFunctionNames"])
}

func TestGeminiOpenAICompatServiceForwardsWebSearchTool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	httpStub := &geminiCompatHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"searched"}]},"finishReason":"STOP"}]}`)),
	}}
	svc := newGeminiOpenAICompatTestService(httpStub)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	_, err := svc.ForwardChatCompletions(context.Background(), c, &Account{
		ID:          6,
		Type:        AccountTypeAPIKey,
		Platform:    PlatformGemini,
		Credentials: map[string]any{"api_key": "sk-test"},
	}, []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"search"}],"tools":[{"type":"web_search"}]}`), nil)

	require.NoError(t, err)
	require.Equal(t, 1, httpStub.calls)
	upstreamBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(upstreamBody, &payload))
	tools := payload["tools"].([]any)
	require.Len(t, tools, 1)
	require.Contains(t, tools[0].(map[string]any), "google_search")
}
