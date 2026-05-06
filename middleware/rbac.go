package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services/rbac"
)

// RequirePermission allows the request only if the authenticated user has every listed
// permission_name value (e.g. constants.AllPermissions.UserRead).
// Must run after AuthJWT.
//
// Permissions are checked from the JWT context first (no DB round-trip).
// Falls back to a DB query when the JWT context does not carry the permissions set.
func RequirePermission(actions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ContextUserID)
		if !ok {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "not authenticated", nil)
			return
		}
		userID, _ := v.(uint)

		if jwtPermissionsSatisfied(c, actions) {
			return
		}

		okAll, missing, err := rbac.UserHasAllPermissions(userID, actions)
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

// jwtPermissionsSatisfied runs c.Next when JWT carries a permission superset; returns true if handled.
func jwtPermissionsSatisfied(c *gin.Context, actions []string) bool {
	permVal, exists := c.Get(ContextPermissions)
	if !exists {
		return false
	}
	permSet, ok := permVal.(map[string]struct{})
	if !ok {
		return false
	}
	for _, action := range actions {
		if _, has := permSet[action]; !has {
			response.AbortFail(c, http.StatusForbidden, errcode.Forbidden, "forbidden: missing permission "+action, nil)
			return true
		}
	}
	c.Next()
	return true
}
