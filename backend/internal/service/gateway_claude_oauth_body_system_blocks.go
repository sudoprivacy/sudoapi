// sudoapi: Preserve system block cache control.

package service

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

func systemCacheControlTTLForBody(body []byte) string {
	if len(body) == 0 {
		return cacheTTLTarget5m
	}
	if topCC := gjson.GetBytes(body, "cache_control"); topCC.Exists() && topCC.Get("type").String() == "ephemeral" && topCC.Get("ttl").String() == cacheTTLTarget1h {
		return cacheTTLTarget1h
	}
	jsonArrayHasMatch := func(array gjson.Result, matches func(gjson.Result) bool) bool {
		if !array.IsArray() {
			return false
		}
		found := false
		array.ForEach(func(_, item gjson.Result) bool {
			found = matches(item)
			return !found
		})
		return found
	}
	match1h := func(value gjson.Result) bool {
		cc := value.Get("cache_control")
		return cc.Exists() && cc.Get("type").String() == "ephemeral" && cc.Get("ttl").String() == cacheTTLTarget1h
	}
	if jsonArrayHasMatch(gjson.GetBytes(body, "system"), match1h) {
		return cacheTTLTarget1h
	}
	if jsonArrayHasMatch(gjson.GetBytes(body, "tools"), match1h) {
		return cacheTTLTarget1h
	}
	if jsonArrayHasMatch(gjson.GetBytes(body, "messages"), func(msg gjson.Result) bool { return jsonArrayHasMatch(msg.Get("content"), match1h) }) {
		return cacheTTLTarget1h
	}
	return cacheTTLTarget5m
}

// see decodeClaudeOAuthSystemPromptCacheControl
func mixingCacheControl(raw json.RawMessage, ttlForBody string) any {
	cc := gjson.ParseBytes(raw)
	if cc.Type == gjson.True {
		if ttlForBody == cacheTTLTarget1h {
			return &anthropicCacheControlPayload{Type: "ephemeral", TTL: cacheTTLTarget1h}
		}
		return &anthropicCacheControlPayload{Type: "ephemeral", TTL: cacheTTLTarget5m}
	}
	if ttl := cc.Get("ttl"); ttl.Exists() {
		if ttlForBody == cacheTTLTarget1h {
			return &anthropicCacheControlPayload{Type: "ephemeral", TTL: cacheTTLTarget1h}
		}
		return &anthropicCacheControlPayload{Type: "ephemeral", TTL: ttl.String()}
	}
	return nil
}

// see extractSystemTextAndCacheControl
func extractSystemBlocks(system any) []map[string]any {
	ccPromptTrimmed := strings.TrimSpace(claudeCodeSystemPrompt)
	switch v := system.(type) {
	case string:
		textTrimmed := strings.TrimSpace(v)
		if textTrimmed != "" && textTrimmed != ccPromptTrimmed && !hasClaudeCodePrefix(textTrimmed) {
			return []map[string]any{{"type": "text", "text": "[System Instructions]\n" + textTrimmed}}
		}
		return nil
	case []any:
		originalSystemBlocks := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			text, ok := m["text"].(string)
			if !ok {
				continue
			}
			textTrimmed := strings.TrimSpace(text)
			if textTrimmed != "" && textTrimmed != ccPromptTrimmed && !hasClaudeCodePrefix(textTrimmed) {
				block := map[string]any{"type": "text", "text": textTrimmed}
				if cacheControl, ok := m["cache_control"]; ok {
					block["cache_control"] = cacheControl
				}
				originalSystemBlocks = append(originalSystemBlocks, block)
			}
		}
		if len(originalSystemBlocks) > 0 {
			originalSystemBlocks[0]["text"] = "[System Instructions]\n" + originalSystemBlocks[0]["text"].(string)
		}
		return originalSystemBlocks
	default:
		return nil
	}
}
