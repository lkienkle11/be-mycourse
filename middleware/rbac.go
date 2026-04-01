package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services"
)

// RequirePermission allows the request only if the authenticated user has every listed
// permission code_check value (e.g. constants.CodeProfileRead.CourseRead).
// Must run after AuthJWT.
//
// Permissions are checked from the JWT context first (no DB round-trip).
// Falls back to a DB query when the JWT context does not carry the permissions set.
func RequirePermission(codeChecks ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ContextUserID)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "not authenticated", nil)
			return
		}
		userID, _ := v.(uint)

		if permVal, exists := c.Get(ContextPermissions); exists {
			if permSet, ok := permVal.(map[string]struct{}); ok {
				for _, cc := range codeChecks {
					if _, has := permSet[cc]; !has {
						response.AbortFail(c, http.StatusForbidden, errcode.Forbidden, "forbidden: missing permission "+cc, nil)
						return
					}
				}
				c.Next()
				return
			}
		}

		// Fallback: query DB (backward compatibility with tokens without embedded permissions).
		okAll, missing, err := services.UserHasAllPermissions(userID, codeChecks)
		if err != nil {
			response.AbortFail(c, http.StatusInternalServerError, errcode.InternalError, "permission check failed", nil)
			return
		}
		if !okAll {
			response.AbortFail(c, http.StatusForbidden, errcode.Forbidden, "forbidden: missing permission "+missing, nil)
			return
		}
		c.Next()
	}
}
