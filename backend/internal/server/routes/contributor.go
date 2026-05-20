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
		accounts.POST("/generate-auth-url", h.ContributorAccount.GenerateAuthURL)
		accounts.POST("/generate-setup-token-url", h.ContributorAccount.GenerateSetupTokenURL)
		accounts.POST("/exchange-code", h.ContributorAccount.ExchangeCode)
		accounts.POST("/exchange-setup-token-code", h.ContributorAccount.ExchangeSetupTokenCode)
		accounts.GET("/:id", h.ContributorAccount.GetByID)
		accounts.PUT("/:id", h.ContributorAccount.Update)
		accounts.POST("/:id/test", h.ContributorAccount.Test)
	}
	proxies := contributor.Group("/proxies")
	{
		proxies.GET("/all", h.ContributorAccount.ListProxies)
	}
}
