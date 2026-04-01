package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/pkg/token"
)

const (
	// ContextUserID holds the numeric users.id (uint) from the JWT — used for RBAC lookups.
	ContextUserID = "user_id"
	// ContextUserCode holds the UUID user_code string — the external-facing identifier.
	ContextUserCode    = "ctx_user_code"
	ContextEmail       = "ctx_email"
	ContextDisplayName = "ctx_display_name"
	ContextPermissions = "ctx_permissions"
)

// AuthJWT validates a JWT from either the Authorization: Bearer header or the
// access_token cookie, then populates context values for downstream handlers.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireJWT(c) {
			return
		}
		c.Next()
	}
}

func requireJWT(c *gin.Context) bool {
	tok := extractToken(c)
	if tok == "" {
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "missing bearer token", nil)
		return false
	}

	secret := setting.AppSetting.JWTSecret
	if secret == "" {
		response.AbortFail(c, http.StatusInternalServerError, errcode.InternalError, "jwt not configured", nil)
		return false
	}

	claims, err := token.ParseAccess(secret, tok)
	if err != nil {
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "invalid token", nil)
		return false
	}
	if claims.UserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "token missing user_id", nil)
		return false
	}

	c.Set(ContextUserID, claims.UserID)       // uint
	c.Set(ContextUserCode, claims.UserCode)   // string UUID
	c.Set(ContextEmail, claims.Email)
	c.Set(ContextDisplayName, claims.DisplayName)

	permSet := make(map[string]struct{}, len(claims.Permissions))
	for _, p := range claims.Permissions {
		permSet[p] = struct{}{}
	}
	c.Set(ContextPermissions, permSet)
	return true
}

// extractToken reads the raw JWT string from Authorization header first,
// then falls back to the access_token HttpOnly cookie.
func extractToken(c *gin.Context) string {
	const prefix = "Bearer "
	if raw := c.GetHeader("Authorization"); strings.HasPrefix(raw, prefix) {
		if tok := strings.TrimSpace(raw[len(prefix):]); tok != "" {
			return tok
		}
	}
	if tok, err := c.Cookie("access_token"); err == nil && tok != "" {
		return tok
	}
	return ""
}
