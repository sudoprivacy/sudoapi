package llmcompat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

type InboundProtocol string

const (
	ProtocolOpenAIChat      InboundProtocol = "openai_chat_completions"
	ProtocolOpenAIResponses InboundProtocol = "openai_responses"
)

// CanonicalRequest is the protocol-neutral request shape used by provider
// adapters. Phase one intentionally stores the existing Anthropic-compatible IR
// so new providers can be added without rewriting the mature OpenAI converters.
type CanonicalRequest struct {
	Protocol        InboundProtocol
	Model           string
	Stream          bool
	Anthropic       *apicompat.AnthropicRequest
	IncludeUsage    bool
	ReasoningEffort *string
}

func FromOpenAIResponses(body []byte) (*CanonicalRequest, error) {
	var req apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse responses request: %w", err)
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if err := ValidateOpenAIResponsesForGemini(&req); err != nil {
		return nil, err
	}
	anthropicReq, err := apicompat.ResponsesToAnthropicRequest(&req)
	if err != nil {
		return nil, fmt.Errorf("convert responses to canonical request: %w", err)
	}
	return &CanonicalRequest{
		Protocol:        ProtocolOpenAIResponses,
		Model:           req.Model,
		Stream:          req.Stream,
		Anthropic:       anthropicReq,
		ReasoningEffort: normalizeReasoningEffort(req.Reasoning),
	}, nil
}

func FromOpenAIChat(body []byte) (*CanonicalRequest, error) {
	var req apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse chat completions request: %w", err)
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if err := ValidateOpenAIChatForGeminiRaw(body); err != nil {
		return nil, err
	}
	responsesReq, err := apicompat.ChatCompletionsToResponses(&req)
	if err != nil {
		return nil, fmt.Errorf("convert chat completions to responses: %w", err)
	}
	if err := ValidateOpenAIResponsesForGemini(responsesReq); err != nil {
		return nil, err
	}
	anthropicReq, err := apicompat.ResponsesToAnthropicRequest(responsesReq)
	if err != nil {
		return nil, fmt.Errorf("convert chat completions to canonical request: %w", err)
	}
	return &CanonicalRequest{
		Protocol:        ProtocolOpenAIChat,
		Model:           req.Model,
		Stream:          req.Stream,
		Anthropic:       anthropicReq,
		IncludeUsage:    req.StreamOptions != nil && req.StreamOptions.IncludeUsage,
		ReasoningEffort: normalizeChatReasoningEffort(req.ReasoningEffort, body),
	}, nil
}

func normalizeReasoningEffort(reasoning *apicompat.ResponsesReasoning) *string {
	if reasoning == nil {
		return nil
	}
	effort := normalizeEffort(reasoning.Effort)
	if effort == "" {
		return nil
	}
	return &effort
}

func normalizeChatReasoningEffort(flat string, body []byte) *string {
	raw := strings.TrimSpace(flat)
	if raw == "" {
		var envelope struct {
			Reasoning *apicompat.ResponsesReasoning `json:"reasoning"`
		}
		_ = json.Unmarshal(body, &envelope)
		if envelope.Reasoning != nil {
			raw = envelope.Reasoning.Effort
		}
	}
	effort := normalizeEffort(raw)
	if effort == "" {
		return nil
	}
	return &effort
}

func normalizeEffort(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low", "medium", "high", "xhigh":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}
