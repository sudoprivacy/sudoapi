// sudoapi: Model Square model catalog.

package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes 注册不需要认证的公开 API。
//
// 承载「模型广场」等不登录可见的只读 API，便于统一审计哪些数据对游客可见。
//
// 注意：所有公开端点必须在 service 层做字段白名单 + scope 过滤，
// 不能把敏感的渠道结构 / 内部 ID / 调度元数据返回给未鉴权调用方。
func RegisterPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	public := v1.Group("/public")
	{
		// 模型广场（未登录可见）：仅展示 standard 非专属分组的模型与定价。
		public.GET("/models", h.ModelSquare.ListPublic)
	}
}
