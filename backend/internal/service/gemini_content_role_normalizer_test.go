// sudoapi: Gemini native request role normalization.

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEnsureGeminiContentRoles(t *testing.T) {
	body := []byte(`{
		"systemInstruction":{"parts":[{"text":"system"}]},
		"contents":[
			{"parts":[{"text":"hi"}]},
			{"role":"model","parts":[{"text":"hello"}]},
			{"role":"","parts":[{"functionCall":{"name":"tool","args":{}}}]},
			{"parts":[{"functionResponse":{"name":"tool","response":{"ok":true}}}]}
		]
	}`)

	out := ensureGeminiContentRoles(body)
	var got struct {
		SystemInstruction map[string]any `json:"systemInstruction"`
		Contents          []struct {
			Role string `json:"role"`
		} `json:"contents"`
	}
	require.NoError(t, json.Unmarshal(out, &got))
	require.NotContains(t, got.SystemInstruction, "role")
	require.Equal(t, []string{"user", "model", "model", "user"}, []string{
		got.Contents[0].Role,
		got.Contents[1].Role,
		got.Contents[2].Role,
		got.Contents[3].Role,
	})
}

func TestGeminiForwardNative_AddsMissingContentRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`))),
		},
	}
	svc := &GeminiMessagesCompatService{
		httpUpstream: httpStub,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       1,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "gemini-api-key",
		},
		Concurrency: 1,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))

	result, err := svc.ForwardNative(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, httpStub.lastReq)

	sentBody, err := io.ReadAll(httpStub.lastReq.Body)
	require.NoError(t, err)
	var sent struct {
		Contents []struct {
			Role string `json:"role"`
		} `json:"contents"`
	}
	require.NoError(t, json.Unmarshal(sentBody, &sent))
	require.Len(t, sent.Contents, 1)
	require.Equal(t, "user", sent.Contents[0].Role)
}
