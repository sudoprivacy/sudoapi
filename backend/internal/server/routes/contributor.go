// sudoapi: Account contributor review workflow.

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
		// sudoapi: Contributor account self-service authorization.
		accounts.POST("/generate-auth-url", h.ContributorAccount.GenerateAuthURL)
		accounts.POST("/generate-setup-token-url", h.ContributorAccount.GenerateSetupTokenURL)
		accounts.POST("/exchange-code", h.ContributorAccount.ExchangeCode)
		accounts.POST("/exchange-setup-token-code", h.ContributorAccount.ExchangeSetupTokenCode)
		// sudoapi: Contributor account OpenAI OAuth self-service authorization.
		accounts.POST("/openai/generate-auth-url", h.ContributorAccount.GenerateOpenAIAuthURL)
		accounts.POST("/openai/exchange-code", h.ContributorAccount.ExchangeOpenAICode)
		accounts.POST("/openai/refresh-token", h.ContributorAccount.RefreshOpenAIToken)
	}
	proxies := contributor.Group("/proxies")
	{
		// sudoapi: Account contributor review workflow.
		proxies.GET("/all", h.ContributorAccount.ListProxies)
		proxies.POST("/reservation/release", h.ContributorAccount.ReleaseProxyReservation)
	}
}
