package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/services"
)

// RequirePermission allows the request only if the authenticated user has every listed permissions.code_check
// value (pass the string from a catalog field, e.g. constants.CodeProfileRead.CourseRead). Must run after AuthJWT.
func RequirePermission(codeChecks ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ContextUserID)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "not authenticated"})
			return
		}
		userID, _ := v.(string)
		okAll, missing, err := services.UserHasAllPermissions(userID, codeChecks)
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
