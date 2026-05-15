package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/llmcompat"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

type GeminiOpenAICompatService struct {
	gemini *GeminiMessagesCompatService
}

func NewGeminiOpenAICompatService(gemini *GeminiMessagesCompatService) *GeminiOpenAICompatService {
	return &GeminiOpenAICompatService{gemini: gemini}
}

func (s *GeminiOpenAICompatService) ForwardChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	_ *ParsedRequest,
) (*ForwardResult, error) {
	canonical, err := llmcompat.FromOpenAIChat(body)
	if err != nil {
		writeGatewayCCError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	result, err := s.forwardCanonical(ctx, c, account, canonical, body, func(resp *http.Response, geminiStream bool, mappedModel string, start time.Time) (*ForwardResult, error) {
		if canonical.Stream {
			return s.handleChatStreaming(ctx, c, resp, canonical, mappedModel, start)
		}
		return s.handleChatBuffered(c, resp, canonical, geminiStream, mappedModel, start)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GeminiOpenAICompatService) ForwardResponses(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	_ *ParsedRequest,
) (*ForwardResult, error) {
	canonical, err := llmcompat.FromOpenAIResponses(body)
	if err != nil {
		writeResponsesError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	return s.forwardCanonical(ctx, c, account, canonical, body, func(resp *http.Response, geminiStream bool, mappedModel string, start time.Time) (*ForwardResult, error) {
		if canonical.Stream {
			return s.handleResponsesStreaming(ctx, c, resp, canonical, mappedModel, start)
		}
		return s.handleResponsesBuffered(c, resp, canonical, geminiStream, mappedModel, start)
	})
}

type geminiOpenAIResponseHandler func(resp *http.Response, geminiStream bool, mappedModel string, start time.Time) (*ForwardResult, error)

func (s *GeminiOpenAICompatService) forwardCanonical(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	canonical *llmcompat.CanonicalRequest,
	originalBody []byte,
	handle geminiOpenAIResponseHandler,
) (*ForwardResult, error) {
	startTime := time.Now()
	if s == nil || s.gemini == nil {
		err := errors.New("gemini openai compatibility service is not configured")
		s.writeProtocolError(c, canonical.Protocol, http.StatusBadGateway, "server_error", err.Error())
		return nil, err
	}
	if canonical == nil || canonical.Anthropic == nil {
		err := errors.New("canonical request is empty")
		s.writeProtocolError(c, "", http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}

	originalModel := canonical.Model
	mappedModel := originalModel
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeServiceAccount {
		mappedModel = account.GetMappedModel(originalModel)
	}
	canonical.Anthropic.Model = mappedModel
	canonical.Anthropic.Stream = canonical.Stream

	anthropicBody, err := json.Marshal(canonical.Anthropic)
	if err != nil {
		s.writeProtocolError(c, canonical.Protocol, http.StatusBadRequest, "invalid_request_error", "Failed to marshal canonical request")
		return nil, err
	}
	geminiReq, err := convertClaudeMessagesToGeminiGenerateContent(anthropicBody)
	if err != nil {
		s.writeProtocolError(c, canonical.Protocol, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	geminiReq = ensureGeminiFunctionCallThoughtSignatures(geminiReq)

	useUpstreamStream := canonical.Stream
	if account.Type == AccountTypeOAuth && !canonical.Stream && strings.TrimSpace(account.GetCredential("project_id")) != "" {
		useUpstreamStream = true
	}
	action := "generateContent"
	if useUpstreamStream {
		action = "streamGenerateContent"
	}

	resp, requestIDHeader, err := s.gemini.doGenerateContentWithRetry(ctx, c, account, mappedModel, action, useUpstreamStream, geminiReq, anthropicBody)
	if err != nil {
		if !c.Writer.Written() {
			s.writeProtocolError(c, canonical.Protocol, http.StatusBadGateway, "server_error", "Upstream request failed")
		}
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return nil, s.handleUpstreamError(ctx, c, account, canonical.Protocol, resp, requestIDHeader, respBody)
	}

	requestID := getHTTPHeaderCaseInsensitive(resp.Header, requestIDHeader)
	if requestID == "" {
		requestID = getHTTPHeaderCaseInsensitive(resp.Header, "x-goog-request-id")
	}
	if requestID != "" {
		c.Header("x-request-id", requestID)
		if c.Writer != nil {
			c.Writer.Header().Set("x-request-id", requestID)
		}
	}
	if c != nil {
		c.Set(OpsUpstreamRequestBodyKey, string(originalBody))
	}

	result, err := handle(resp, useUpstreamStream, mappedModel, startTime)
	if result != nil {
		result.RequestID = requestID
	}
	return result, err
}

func (s *GeminiOpenAICompatService) handleResponsesBuffered(c *gin.Context, resp *http.Response, canonical *llmcompat.CanonicalRequest, geminiStream bool, mappedModel string, start time.Time) (*ForwardResult, error) {
	anthropicResp, usage, err := geminiResponseToAnthropic(resp.Body, canonical.Model, geminiStream)
	if err != nil {
		writeResponsesError(c, http.StatusBadGateway, "server_error", err.Error())
		return nil, err
	}
	responsesResp := apicompat.AnthropicToResponsesResponse(anthropicResp)
	responsesResp.Model = canonical.Model
	if s.gemini.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.gemini.responseHeaderFilter)
	}
	c.JSON(http.StatusOK, responsesResp)
	return &ForwardResult{
		Usage:           usage,
		Model:           canonical.Model,
		UpstreamModel:   mappedModel,
		ReasoningEffort: canonical.ReasoningEffort,
		Stream:          false,
		Duration:        time.Since(start),
	}, nil
}

func (s *GeminiOpenAICompatService) handleChatBuffered(c *gin.Context, resp *http.Response, canonical *llmcompat.CanonicalRequest, geminiStream bool, mappedModel string, start time.Time) (*ForwardResult, error) {
	anthropicResp, usage, err := geminiResponseToAnthropic(resp.Body, canonical.Model, geminiStream)
	if err != nil {
		writeGatewayCCError(c, http.StatusBadGateway, "server_error", err.Error())
		return nil, err
	}
	responsesResp := apicompat.AnthropicToResponsesResponse(anthropicResp)
	ccResp := apicompat.ResponsesToChatCompletions(responsesResp, canonical.Model)
	if s.gemini.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.gemini.responseHeaderFilter)
	}
	c.JSON(http.StatusOK, ccResp)
	return &ForwardResult{
		Usage:           usage,
		Model:           canonical.Model,
		UpstreamModel:   mappedModel,
		ReasoningEffort: canonical.ReasoningEffort,
		Stream:          false,
		Duration:        time.Since(start),
	}, nil
}

func (s *GeminiOpenAICompatService) handleResponsesStreaming(ctx context.Context, c *gin.Context, resp *http.Response, canonical *llmcompat.CanonicalRequest, mappedModel string, start time.Time) (*ForwardResult, error) {
	if s.gemini.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.gemini.responseHeaderFilter)
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	anthState := apicompat.NewAnthropicEventToResponsesState()
	anthState.Model = canonical.Model
	var usage ClaudeUsage
	var firstTokenMs *int
	first := true

	err := s.streamGeminiAsAnthropic(ctx, resp.Body, canonical.Model, start, func(event *apicompat.AnthropicStreamEvent) bool {
		if first {
			first = false
			ms := int(time.Since(start).Milliseconds())
			firstTokenMs = &ms
		}
		mergeAnthropicEventUsage(&usage, event)
		for _, evt := range apicompat.AnthropicEventToResponsesEvents(event, anthState) {
			sse, err := apicompat.ResponsesEventToSSE(evt)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprint(c.Writer, sse); err != nil {
				return true
			}
		}
		c.Writer.Flush()
		return false
	})
	if err != nil {
		return nil, err
	}
	for _, evt := range apicompat.FinalizeAnthropicResponsesStream(anthState) {
		sse, err := apicompat.ResponsesEventToSSE(evt)
		if err == nil {
			fmt.Fprint(c.Writer, sse) //nolint:errcheck
		}
	}
	c.Writer.Flush()
	return &ForwardResult{Usage: usage, Model: canonical.Model, UpstreamModel: mappedModel, ReasoningEffort: canonical.ReasoningEffort, Stream: true, Duration: time.Since(start), FirstTokenMs: firstTokenMs}, nil
}

func (s *GeminiOpenAICompatService) handleChatStreaming(ctx context.Context, c *gin.Context, resp *http.Response, canonical *llmcompat.CanonicalRequest, mappedModel string, start time.Time) (*ForwardResult, error) {
	if s.gemini.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.gemini.responseHeaderFilter)
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	anthState := apicompat.NewAnthropicEventToResponsesState()
	anthState.Model = canonical.Model
	ccState := apicompat.NewResponsesEventToChatState()
	ccState.Model = canonical.Model
	ccState.IncludeUsage = canonical.IncludeUsage
	var usage ClaudeUsage
	var firstTokenMs *int
	first := true

	writeChunk := func(chunk apicompat.ChatCompletionsChunk) bool {
		sse, err := apicompat.ChatChunkToSSE(chunk)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprint(c.Writer, sse); err != nil {
			return true
		}
		return false
	}

	err := s.streamGeminiAsAnthropic(ctx, resp.Body, canonical.Model, start, func(event *apicompat.AnthropicStreamEvent) bool {
		if first {
			first = false
			ms := int(time.Since(start).Milliseconds())
			firstTokenMs = &ms
		}
		mergeAnthropicEventUsage(&usage, event)
		for _, resEvt := range apicompat.AnthropicEventToResponsesEvents(event, anthState) {
			for _, chunk := range apicompat.ResponsesEventToChatChunks(&resEvt, ccState) {
				if writeChunk(chunk) {
					return true
				}
			}
		}
		c.Writer.Flush()
		return false
	})
	if err != nil {
		return nil, err
	}
	for _, resEvt := range apicompat.FinalizeAnthropicResponsesStream(anthState) {
		for _, chunk := range apicompat.ResponsesEventToChatChunks(&resEvt, ccState) {
			writeChunk(chunk) //nolint:errcheck
		}
	}
	for _, chunk := range apicompat.FinalizeResponsesChatStream(ccState) {
		writeChunk(chunk) //nolint:errcheck
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n") //nolint:errcheck
	c.Writer.Flush()
	return &ForwardResult{Usage: usage, Model: canonical.Model, UpstreamModel: mappedModel, ReasoningEffort: canonical.ReasoningEffort, Stream: true, Duration: time.Since(start), FirstTokenMs: firstTokenMs}, nil
}

func geminiResponseToAnthropic(body io.Reader, originalModel string, geminiStream bool) (*apicompat.AnthropicResponse, ClaudeUsage, error) {
	var geminiResp map[string]any
	var usage *ClaudeUsage
	var err error
	if geminiStream {
		geminiResp, usage, err = collectGeminiSSE(body, true)
		if err != nil {
			return nil, ClaudeUsage{}, fmt.Errorf("failed to read upstream stream")
		}
	} else {
		raw, err := io.ReadAll(io.LimitReader(body, 8<<20))
		if err != nil {
			return nil, ClaudeUsage{}, fmt.Errorf("failed to read upstream response")
		}
		unwrapped, err := unwrapGeminiResponse(raw)
		if err != nil {
			return nil, ClaudeUsage{}, fmt.Errorf("failed to parse upstream response")
		}
		if err := json.Unmarshal(unwrapped, &geminiResp); err != nil {
			return nil, ClaudeUsage{}, fmt.Errorf("failed to parse upstream response")
		}
		usage = extractGeminiUsage(unwrapped)
	}
	raw, _ := json.Marshal(geminiResp)
	claudeMap, convertedUsage := convertGeminiToClaudeMessage(geminiResp, originalModel, raw)
	if usage != nil && (usage.InputTokens > 0 || usage.OutputTokens > 0 || usage.CacheReadInputTokens > 0) {
		convertedUsage = usage
	}
	claudeBytes, _ := json.Marshal(claudeMap)
	var anthropicResp apicompat.AnthropicResponse
	if err := json.Unmarshal(claudeBytes, &anthropicResp); err != nil {
		return nil, ClaudeUsage{}, fmt.Errorf("failed to convert upstream response")
	}
	if convertedUsage == nil {
		convertedUsage = &ClaudeUsage{}
	}
	anthropicResp.Usage = apicompat.AnthropicUsage{
		InputTokens:              convertedUsage.InputTokens,
		OutputTokens:             convertedUsage.OutputTokens,
		CacheCreationInputTokens: convertedUsage.CacheCreationInputTokens,
		CacheReadInputTokens:     convertedUsage.CacheReadInputTokens,
	}
	return &anthropicResp, *convertedUsage, nil
}

func (s *GeminiOpenAICompatService) streamGeminiAsAnthropic(ctx context.Context, body io.Reader, model string, start time.Time, handle func(*apicompat.AnthropicStreamEvent) bool) error {
	messageID := "msg_" + randomHex(12)
	startEvent := &apicompat.AnthropicStreamEvent{
		Type: "message_start",
		Message: &apicompat.AnthropicResponse{
			ID:      messageID,
			Type:    "message",
			Role:    "assistant",
			Model:   model,
			Content: []apicompat.AnthropicContentBlock{},
			Usage:   apicompat.AnthropicUsage{},
		},
	}
	if handle(startEvent) {
		return nil
	}

	var usage ClaudeUsage
	finishReason := ""
	sawToolUse := false
	nextBlockIndex := 0
	openBlockIndex := -1
	openBlockType := ""
	seenText := ""
	openToolIndex := -1
	openToolName := ""
	seenToolJSON := ""

	reader := bufio.NewReader(body)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("stream read error: %w", err)
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload != "" && payload != "[DONE]" {
				unwrapped, unwrapErr := unwrapGeminiResponse([]byte(payload))
				if unwrapErr == nil {
					var geminiResp map[string]any
					if json.Unmarshal(unwrapped, &geminiResp) == nil {
						if fr := extractGeminiFinishReason(geminiResp); fr != "" {
							finishReason = fr
						}
						if u := extractGeminiUsage(unwrapped); u != nil {
							usage = *u
						}
						for _, part := range extractGeminiParts(geminiResp) {
							if stop := streamGeminiPartAsAnthropic(part, &streamAnthropicState{
								handle:         handle,
								nextBlockIndex: &nextBlockIndex,
								openBlockIndex: &openBlockIndex,
								openBlockType:  &openBlockType,
								seenText:       &seenText,
								openToolIndex:  &openToolIndex,
								openToolName:   &openToolName,
								seenToolJSON:   &seenToolJSON,
								sawToolUse:     &sawToolUse,
							}); stop {
								return nil
							}
						}
					}
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	_ = start
	if openBlockIndex >= 0 {
		idx := openBlockIndex
		if handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
			return nil
		}
	}
	if openToolIndex >= 0 {
		idx := openToolIndex
		if handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
			return nil
		}
	}
	stopReason := mapGeminiFinishReasonToClaudeStopReason(finishReason)
	if sawToolUse {
		stopReason = "tool_use"
	}
	if handle(&apicompat.AnthropicStreamEvent{
		Type:  "message_delta",
		Delta: &apicompat.AnthropicDelta{StopReason: stopReason},
		Usage: &apicompat.AnthropicUsage{
			InputTokens:              usage.InputTokens,
			OutputTokens:             usage.OutputTokens,
			CacheCreationInputTokens: usage.CacheCreationInputTokens,
			CacheReadInputTokens:     usage.CacheReadInputTokens,
		},
	}) {
		return nil
	}
	handle(&apicompat.AnthropicStreamEvent{Type: "message_stop"}) //nolint:errcheck
	return nil
}

type streamAnthropicState struct {
	handle         func(*apicompat.AnthropicStreamEvent) bool
	nextBlockIndex *int
	openBlockIndex *int
	openBlockType  *string
	seenText       *string
	openToolIndex  *int
	openToolName   *string
	seenToolJSON   *string
	sawToolUse     *bool
}

func streamGeminiPartAsAnthropic(part map[string]any, st *streamAnthropicState) bool {
	if text, ok := part["text"].(string); ok && text != "" {
		delta, newSeen := computeGeminiTextDelta(*st.seenText, text)
		*st.seenText = newSeen
		if delta == "" {
			return false
		}
		if *st.openBlockType != "text" {
			if *st.openBlockIndex >= 0 {
				idx := *st.openBlockIndex
				if st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
					return true
				}
				*st.openBlockIndex = -1
			}
			if *st.openToolIndex >= 0 {
				idx := *st.openToolIndex
				if st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
					return true
				}
				*st.openToolIndex = -1
				*st.openToolName = ""
				*st.seenToolJSON = ""
			}
			*st.openBlockType = "text"
			*st.openBlockIndex = *st.nextBlockIndex
			(*st.nextBlockIndex)++
			idx := *st.openBlockIndex
			if st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_start", Index: &idx, ContentBlock: &apicompat.AnthropicContentBlock{Type: "text"}}) {
				return true
			}
		}
		idx := *st.openBlockIndex
		return st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &apicompat.AnthropicDelta{Type: "text_delta", Text: delta}})
	}
	if fc, ok := part["functionCall"].(map[string]any); ok && fc != nil {
		name, _ := fc["name"].(string)
		if strings.TrimSpace(name) == "" {
			name = "tool"
		}
		if *st.openBlockIndex >= 0 {
			idx := *st.openBlockIndex
			if st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
				return true
			}
			*st.openBlockIndex = -1
			*st.openBlockType = ""
		}
		if *st.openToolIndex >= 0 && *st.openToolName != name {
			idx := *st.openToolIndex
			if st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_stop", Index: &idx}) {
				return true
			}
			*st.openToolIndex = -1
			*st.openToolName = ""
			*st.seenToolJSON = ""
		}
		if *st.openToolIndex < 0 {
			*st.openToolIndex = *st.nextBlockIndex
			*st.openToolName = name
			(*st.nextBlockIndex)++
			*st.sawToolUse = true
			idx := *st.openToolIndex
			if st.handle(&apicompat.AnthropicStreamEvent{
				Type:  "content_block_start",
				Index: &idx,
				ContentBlock: &apicompat.AnthropicContentBlock{
					Type:  "tool_use",
					ID:    "toolu_" + randomHex(8),
					Name:  name,
					Input: json.RawMessage(`{}`),
				},
			}) {
				return true
			}
		}
		argsJSON := "{}"
		if args := fc["args"]; args != nil {
			if s, ok := args.(string); ok && strings.TrimSpace(s) != "" {
				argsJSON = s
			} else if b, err := json.Marshal(args); err == nil && len(b) > 0 {
				argsJSON = string(b)
			}
		}
		delta, newSeen := computeGeminiTextDelta(*st.seenToolJSON, argsJSON)
		*st.seenToolJSON = newSeen
		if delta != "" {
			idx := *st.openToolIndex
			return st.handle(&apicompat.AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &apicompat.AnthropicDelta{Type: "input_json_delta", PartialJSON: delta}})
		}
	}
	return false
}

func mergeAnthropicEventUsage(dst *ClaudeUsage, event *apicompat.AnthropicStreamEvent) {
	if event == nil {
		return
	}
	if event.Message != nil {
		mergeAnthropicUsage(dst, event.Message.Usage)
	}
	if event.Usage != nil {
		mergeAnthropicUsage(dst, *event.Usage)
	}
}

func (s *GeminiOpenAICompatService) handleUpstreamError(ctx context.Context, c *gin.Context, account *Account, protocol llmcompat.InboundProtocol, resp *http.Response, requestIDHeader string, body []byte) error {
	s.gemini.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	if s.gemini.shouldFailoverGeminiUpstreamError(resp.StatusCode) {
		return &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: body}
	}
	message := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	if message == "" {
		message = "Upstream request failed"
	}
	requestID := getHTTPHeaderCaseInsensitive(resp.Header, requestIDHeader)
	if requestID == "" {
		requestID = getHTTPHeaderCaseInsensitive(resp.Header, "x-goog-request-id")
	}
	setOpsUpstreamError(c, resp.StatusCode, message, "")
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  requestID,
		Kind:               "error",
		Message:            message,
	})
	s.writeProtocolError(c, protocol, mapUpstreamStatusCode(resp.StatusCode), "server_error", message)
	return fmt.Errorf("upstream error: %d %s", resp.StatusCode, message)
}

func (s *GeminiOpenAICompatService) writeProtocolError(c *gin.Context, protocol llmcompat.InboundProtocol, status int, code, message string) {
	if protocol == llmcompat.ProtocolOpenAIChat {
		writeGatewayCCError(c, status, code, message)
		return
	}
	writeResponsesError(c, status, code, message)
}

func getHTTPHeaderCaseInsensitive(header http.Header, key string) string {
	if header == nil || key == "" {
		return ""
	}
	if value := header.Get(key); value != "" {
		return value
	}
	for k, values := range header {
		if strings.EqualFold(k, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func (s *GeminiMessagesCompatService) doGenerateContentWithRetry(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	mappedModel string,
	action string,
	stream bool,
	body []byte,
	opsBody []byte,
) (*http.Response, string, error) {
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var requestIDHeader string
	var resp *http.Response
	for attempt := 1; attempt <= geminiMaxRetries; attempt++ {
		req, idHeader, err := s.buildGenerateContentRequest(ctx, account, mappedModel, action, stream, body)
		if err != nil {
			return nil, "", err
		}
		requestIDHeader = idHeader
		if c != nil {
			c.Set(OpsUpstreamRequestBodyKey, string(opsBody))
		}
		resp, err = s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				Kind:               "request_error",
				Message:            safeErr,
			})
			if attempt < geminiMaxRetries {
				sleepGeminiBackoff(attempt)
				continue
			}
			setOpsUpstreamError(c, 0, safeErr, "")
			return nil, requestIDHeader, fmt.Errorf("upstream request failed after retries: %s", safeErr)
		}
		if matched, rebuilt := s.checkErrorPolicyInLoop(ctx, account, resp); matched {
			return rebuilt, requestIDHeader, nil
		} else {
			resp = rebuilt
		}
		if resp.StatusCode >= 400 && s.shouldRetryGeminiUpstreamError(account, resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			if resp.StatusCode == 403 && isGeminiInsufficientScope(resp.Header, respBody) {
				resp.Body = io.NopCloser(bytes.NewReader(respBody))
				return resp, requestIDHeader, nil
			}
			if resp.StatusCode == 429 {
				s.handleGeminiUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			if attempt < geminiMaxRetries {
				sleepGeminiBackoff(attempt)
				continue
			}
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			return resp, requestIDHeader, nil
		}
		return resp, requestIDHeader, nil
	}
	return resp, requestIDHeader, nil
}

func (s *GeminiMessagesCompatService) buildGenerateContentRequest(ctx context.Context, account *Account, mappedModel string, action string, stream bool, body []byte) (*http.Request, string, error) {
	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := strings.TrimSpace(account.GetCredential("api_key"))
		if apiKey == "" {
			return nil, "", errors.New("gemini api_key not configured")
		}
		baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, "", err
		}
		fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, action)
		if stream {
			fullURL += "?alt=sse"
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(normalizeGeminiRequestForAIStudio(body)))
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", apiKey)
		return req, "x-request-id", nil
	case AccountTypeOAuth:
		if s.tokenProvider == nil {
			return nil, "", errors.New("gemini token provider not configured")
		}
		accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, "", err
		}
		projectID := strings.TrimSpace(account.GetCredential("project_id"))
		if projectID != "" {
			baseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
			if err != nil {
				return nil, "", err
			}
			fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), action)
			if stream {
				fullURL += "?alt=sse"
			}
			var inner any
			if err := json.Unmarshal(body, &inner); err != nil {
				return nil, "", fmt.Errorf("failed to parse gemini request: %w", err)
			}
			wrapped, _ := json.Marshal(map[string]any{"model": mappedModel, "project": projectID, "request": inner})
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(wrapped))
			if err != nil {
				return nil, "", err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
			return req, "x-request-id", nil
		}
		baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, "", err
		}
		fullURL := fmt.Sprintf("%s/v1beta/models/%s:%s", strings.TrimRight(normalizedBaseURL, "/"), mappedModel, action)
		if stream {
			fullURL += "?alt=sse"
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(normalizeGeminiRequestForAIStudio(body)))
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return req, "x-request-id", nil
	case AccountTypeServiceAccount:
		if s.tokenProvider == nil {
			return nil, "", errors.New("gemini token provider not configured")
		}
		accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, "", err
		}
		fullURL, err := buildVertexGeminiURL(account.VertexProjectID(), account.VertexLocation(mappedModel), mappedModel, action, stream)
		if err != nil {
			return nil, "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(normalizeGeminiRequestForAIStudio(body)))
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return req, "x-request-id", nil
	default:
		return nil, "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}
