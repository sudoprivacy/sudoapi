package llmcompat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func TestFromOpenAIChatToCanonical(t *testing.T) {
	body := []byte(`{"model":"gemini-2.5-pro","messages":[{"role":"system","content":"Be terse."},{"role":"user","content":"hello"}],"stream":true,"stream_options":{"include_usage":true}}`)

	req, err := FromOpenAIChat(body)
	if err != nil {
		t.Fatalf("FromOpenAIChat error: %v", err)
	}
	if req.Protocol != ProtocolOpenAIChat {
		t.Fatalf("Protocol = %q, want %q", req.Protocol, ProtocolOpenAIChat)
	}
	if req.Model != "gemini-2.5-pro" || !req.Stream || !req.IncludeUsage {
		t.Fatalf("unexpected canonical request: %+v", req)
	}
	if req.Anthropic == nil {
		t.Fatal("Anthropic is nil")
	}
	if req.Anthropic.Model != "gemini-2.5-pro" {
		t.Fatalf("Anthropic.Model = %q", req.Anthropic.Model)
	}
	if len(req.Anthropic.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(req.Anthropic.Messages))
	}
	var system string
	if err := json.Unmarshal(req.Anthropic.System, &system); err != nil {
		t.Fatalf("system JSON invalid: %v", err)
	}
	if system != "Be terse." {
		t.Fatalf("system = %q", system)
	}
}

func TestFromOpenAIResponsesToCanonical(t *testing.T) {
	input, err := json.Marshal([]map[string]any{{
		"role":    "user",
		"content": "hello",
	}})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"model": "gemini-2.5-flash",
		"input": json.RawMessage(input),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req, err := FromOpenAIResponses(body)
	if err != nil {
		t.Fatalf("FromOpenAIResponses error: %v", err)
	}
	if req.Protocol != ProtocolOpenAIResponses {
		t.Fatalf("Protocol = %q, want %q", req.Protocol, ProtocolOpenAIResponses)
	}
	if req.Model != "gemini-2.5-flash" || req.Stream {
		t.Fatalf("unexpected canonical request: %+v", req)
	}
	if req.Anthropic == nil || len(req.Anthropic.Messages) != 1 {
		t.Fatalf("unexpected anthropic request: %+v", req.Anthropic)
	}
}

func TestValidateOpenAIChatForGeminiRejectsUnsupportedCapability(t *testing.T) {
	err := ValidateOpenAIChatForGeminiRaw([]byte(`{"model":"gemini","messages":[],"logprobs":true}`))
	if err == nil || !contains(err.Error(), "logprobs") {
		t.Fatalf("err = %v, want logprobs error", err)
	}

	err = ValidateOpenAIChatForGeminiRaw([]byte(`{"model":"gemini","messages":[],"n":2}`))
	if err == nil || !contains(err.Error(), "n > 1") {
		t.Fatalf("err = %v, want n > 1 error", err)
	}
}

func TestFromOpenAIChatAllowsWebSearchTool(t *testing.T) {
	body := []byte(`{"model":"gemini","messages":[{"role":"user","content":"search"}],"tools":[{"type":"web_search_preview"}]}`)

	req, err := FromOpenAIChat(body)
	if err != nil {
		t.Fatalf("FromOpenAIChat error: %v", err)
	}
	if req.Anthropic == nil || len(req.Anthropic.Tools) != 1 {
		t.Fatalf("unexpected tools: %+v", req.Anthropic)
	}
	if req.Anthropic.Tools[0].Type != "web_search_20250305" || req.Anthropic.Tools[0].Name != "web_search" {
		t.Fatalf("tool = %+v, want web search", req.Anthropic.Tools[0])
	}
}

func TestValidateOpenAIResponsesForGeminiRejectsUnsupportedTool(t *testing.T) {
	err := ValidateOpenAIResponsesForGemini(&apicompat.ResponsesRequest{
		Model: "gemini",
		Tools: []apicompat.ResponsesTool{{Type: "file_search"}},
	})
	if err == nil || !contains(err.Error(), "file_search") {
		t.Fatalf("err = %v, want file_search error", err)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
