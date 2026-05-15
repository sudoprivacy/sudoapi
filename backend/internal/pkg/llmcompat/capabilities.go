package llmcompat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func ValidateOpenAIResponsesForGemini(req *apicompat.ResponsesRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	for _, tool := range req.Tools {
		if !isGeminiResponsesToolSupported(tool.Type) {
			return fmt.Errorf("gemini adapter does not support tool type %q", tool.Type)
		}
	}
	if err := validateGeminiResponsesToolChoice(req.ToolChoice); err != nil {
		return err
	}
	for _, include := range req.Include {
		switch strings.TrimSpace(include) {
		case "", "reasoning.encrypted_content":
			continue
		default:
			return fmt.Errorf("gemini adapter does not support include=%q", include)
		}
	}
	if req.Text != nil && strings.TrimSpace(req.Text.Verbosity) != "" {
		return fmt.Errorf("gemini adapter does not support text.verbosity")
	}
	return nil
}

func isGeminiResponsesToolSupported(toolType string) bool {
	switch strings.ToLower(strings.TrimSpace(toolType)) {
	case "", "function", "web_search", "web_search_preview", "web_search_preview_2025_03_11", "google_search", "web_search_20250305":
		return true
	default:
		return false
	}
}

func validateGeminiResponsesToolChoice(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "", "auto", "required", "none":
			return nil
		default:
			return fmt.Errorf("gemini adapter does not support tool_choice=%q", s)
		}
	}
	var obj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("parse tool_choice: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(obj.Type)) {
	case "", "auto", "required", "none", "function", "tool", "web_search", "web_search_preview", "web_search_preview_2025_03_11", "google_search", "web_search_20250305":
		return nil
	default:
		return fmt.Errorf("gemini adapter does not support tool_choice type %q", obj.Type)
	}
}

func ValidateOpenAIChatForGeminiRaw(body []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	for _, field := range []string{"logprobs", "top_logprobs"} {
		if _, ok := raw[field]; ok {
			return fmt.Errorf("gemini adapter does not support %s", field)
		}
	}
	if rawN, ok := raw["n"]; ok {
		var n int
		if err := json.Unmarshal(rawN, &n); err == nil && n > 1 {
			return fmt.Errorf("gemini adapter does not support n > 1")
		}
	}
	return nil
}
