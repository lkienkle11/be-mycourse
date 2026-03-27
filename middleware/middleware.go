package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func BeforeInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Reserved for authn/authz/tenant/permission checks.
		if c.GetHeader("X-Blocked") == "1" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "blocked"})
			return
		}
		c.Next()
	}
}
