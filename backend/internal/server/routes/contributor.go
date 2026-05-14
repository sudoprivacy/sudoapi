package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterContributorRoutes registers account-contributor routes.
func RegisterContributorRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
) {
	contributor := v1.Group("/contributor")
	contributor.Use(gin.HandlerFunc(jwtAuth), middleware.AccountContributorOnly())
	accounts := contributor.Group("/accounts")
	{
		accounts.GET("", h.ContributorAccount.List)
		accounts.POST("", h.ContributorAccount.Create)
		accounts.GET("/:id", h.ContributorAccount.GetByID)
		accounts.PUT("/:id", h.ContributorAccount.Update)
		accounts.POST("/:id/test", h.ContributorAccount.Test)
	}
}
