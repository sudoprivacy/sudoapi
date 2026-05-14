package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountContributorOnly allows only account contributors to access contributor APIs.
// Admins intentionally use the admin APIs, keeping the contributor surface narrow.
func AccountContributorOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := GetUserRoleFromContext(c)
		if !ok {
			AbortWithError(c, 401, "UNAUTHORIZED", "User not found in context")
			return
		}
		if role != service.RoleAccountContributor {
			AbortWithError(c, 403, "FORBIDDEN", "Account contributor access required")
			return
		}
		c.Next()
	}
}
