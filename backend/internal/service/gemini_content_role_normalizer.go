// sudoapi: Gemini native request role normalization.

package service

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ensureGeminiContentRoles(body []byte) []byte {
	contents := gjson.GetBytes(body, "contents")
	if !contents.IsArray() || len(contents.Array()) == 0 {
		return body
	}

	out := body
	idx := -1
	contents.ForEach(func(_, content gjson.Result) bool {
		idx++
		role := content.Get("role")
		if role.Exists() && strings.TrimSpace(role.String()) != "" {
			return true
		}
		next, err := sjson.SetBytes(out, "contents."+strconv.Itoa(idx)+".role", inferMissingGeminiContentRole(content))
		if err == nil {
			out = next
		}
		return true
	})
	return out
}

func inferMissingGeminiContentRole(content gjson.Result) string {
	hasFunctionCall := false
	parts := content.Get("parts")
	if !parts.IsArray() {
		return "user"
	}
	parts.ForEach(func(_, part gjson.Result) bool {
		if part.Get("functionResponse").Exists() {
			hasFunctionCall = false
			return false
		}
		if part.Get("functionCall").Exists() {
			hasFunctionCall = true
		}
		return true
	})
	if hasFunctionCall {
		return "model"
	}
	return "user"
}
