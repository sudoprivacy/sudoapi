package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type APIKeyModelRouteResolver interface {
	ResolveAPIKeyGroupForModel(ctx context.Context, apiKey *service.APIKey, model string) *service.Group
}

// ContextKey 定义上下文键类型
type ContextKey string

const (
	// ContextKeyUser 用户上下文键
	ContextKeyUser ContextKey = "user"
	// ContextKeyUserRole 当前用户角色（string）
	ContextKeyUserRole ContextKey = "user_role"
	// ContextKeyAPIKey API密钥上下文键
	ContextKeyAPIKey ContextKey = "api_key"
	// ContextKeySubscription 订阅上下文键
	ContextKeySubscription ContextKey = "subscription"
	// ContextKeyForcePlatform 强制平台（用于 /antigravity 路由）
	ContextKeyForcePlatform ContextKey = "force_platform"
)

// ForcePlatform 返回设置强制平台的中间件
// 同时设置 request.Context（供 Service 使用）和 gin.Context（供 Handler 快速检查）
func ForcePlatform(platform string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置到 request.Context，使用 ctxkey.ForcePlatform 供 Service 层读取
		ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, platform)
		c.Request = c.Request.WithContext(ctx)
		// 同时设置到 gin.Context，供 Handler 快速检查
		c.Set(string(ContextKeyForcePlatform), platform)
		c.Next()
	}
}

// HasForcePlatform 检查是否有强制平台（用于 Handler 跳过分组检查）
func HasForcePlatform(c *gin.Context) bool {
	_, exists := c.Get(string(ContextKeyForcePlatform))
	return exists
}

// GetForcePlatformFromContext 从 gin.Context 获取强制平台
func GetForcePlatformFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyForcePlatform))
	if !exists {
		return "", false
	}
	platform, ok := value.(string)
	return platform, ok
}

func effectiveAPIKeyForRequest(c *gin.Context, apiKey *service.APIKey, defaultPlatform string, routeResolver APIKeyModelRouteResolver) *service.APIKey {
	if apiKey == nil {
		return nil
	}
	platform := defaultPlatform
	if forced, ok := GetForcePlatformFromContext(c); ok && forced != "" {
		// ForcePlatform 由专用路由设置，必须优先于模型推断，避免 /antigravity 等路由被请求体模型改写。
		platform = forced
	}
	if platform == "" {
		platform = inferAPIKeyPlatformFromRequest(c, apiKey, routeResolver)
	}
	if platform == "" {
		return apiKey
	}
	return apiKey.WithEffectiveGroupForPlatform(platform)
}

func inferAPIKeyPlatformFromRequest(c *gin.Context, apiKey *service.APIKey, routeResolver APIKeyModelRouteResolver) string {
	path := c.Request.URL.Path
	if isOpenAITextEndpoint(path) {
		// OpenAI 格式端点可能承载 Gemini/Claude 模型，先按 model 解析 effective group。
		model := readRequestModelAndRestoreBody(c)
		if strings.TrimSpace(model) != "" && routeResolver != nil {
			if group := routeResolver.ResolveAPIKeyGroupForModel(c.Request.Context(), apiKey, model); group != nil && group.Platform != "" {
				return group.Platform
			}
		}
	}
	return inferAPIKeyPlatformFromPath(path, apiKey)
}

func inferAPIKeyPlatformFromPath(path string, apiKey *service.APIKey) string {
	switch {
	case isAnthropicMessagesEndpoint(path):
		if apiKey.HasGroupForPlatform(service.PlatformAnthropic) {
			return service.PlatformAnthropic
		}
		if apiKey.HasGroupForPlatform(service.PlatformOpenAI) {
			return service.PlatformOpenAI
		}
	case isOpenAIChatCompletionsEndpoint(path):
		if apiKey.HasGroupForPlatform(service.PlatformOpenAI) {
			return service.PlatformOpenAI
		}
	case isOpenAIResponsesEndpoint(path):
		if apiKey.HasGroupForPlatform(service.PlatformOpenAI) {
			return service.PlatformOpenAI
		}
	case path == "/images/generations" || path == "/images/edits" || path == "/v1/images/generations" || path == "/v1/images/edits":
		return service.PlatformOpenAI
	}
	return ""
}

func isOpenAITextEndpoint(path string) bool {
	return isOpenAIChatCompletionsEndpoint(path) || isOpenAIResponsesEndpoint(path)
}

func isAnthropicMessagesEndpoint(path string) bool {
	return path == "/messages" || path == "/v1/messages" ||
		path == "/messages/count_tokens" || path == "/v1/messages/count_tokens"
}

func isOpenAIChatCompletionsEndpoint(path string) bool {
	return path == "/chat/completions" || path == "/v1/chat/completions"
}

func isOpenAIResponsesEndpoint(path string) bool {
	return path == "/responses" || strings.HasPrefix(path, "/responses/") ||
		path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/") ||
		path == "/backend-api/codex/responses" || strings.HasPrefix(path, "/backend-api/codex/responses/")
}

func readRequestModelAndRestoreBody(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	// 鉴权阶段只窥探 model，后续 handler 仍需要完整读取原始请求体。
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil || len(body) == 0 {
		return ""
	}
	return strings.TrimSpace(gjson.GetBytes(body, "model").String())
}

// ErrorResponse 标准错误响应结构
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// AbortWithError 中断请求并返回JSON错误
func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

// ──────────────────────────────────────────────────────────
// RequireGroupAssignment — 未分组 Key 拦截中间件
// ──────────────────────────────────────────────────────────

// GatewayErrorWriter 定义网关错误响应格式（不同协议使用不同格式）
type GatewayErrorWriter func(c *gin.Context, status int, message string)

// AnthropicErrorWriter 按 Anthropic API 规范输出错误
func AnthropicErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"type":  "error",
		"error": gin.H{"type": "permission_error", "message": message},
	})
}

// GoogleErrorWriter 按 Google API 规范输出错误
func GoogleErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
}

// RequireGroupAssignment 检查 API Key 是否已分配到分组，
// 如果未分组且系统设置不允许未分组 Key 调度则返回 403。
func RequireGroupAssignment(settingService *service.SettingService, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey.GroupID != nil {
			c.Next()
			return
		}
		// 未分组 Key — 检查系统设置
		if settingService.IsUngroupedKeySchedulingAllowed(c.Request.Context()) {
			c.Next()
			return
		}
		writeError(c, http.StatusForbidden, "API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group.")
		c.Abort()
	}
}
