package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/services"
)

// RequirePermission allows the request only if the authenticated user has every listed permission code.
// Must run after AuthJWT (or another middleware that sets ContextUserID).
func RequirePermission(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skip, ok := c.Get(ContextSkipAuth); ok {
			if b, _ := skip.(bool); b {
				c.Next()
				return
			}
		}
		v, ok := c.Get(ContextUserID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "not authenticated"})
			return
		}
		userID, _ := v.(string)
		okAll, missing, err := services.UserHasAllPermissions(userID, codes)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "permission check failed"})
			return
		}
		if !okAll {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden", "missing_permission": missing})
			return
		}
		c.Next()
	}
}
