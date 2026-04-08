package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

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

// AuthJWT validates a JWT from the Authorization: Bearer header and populates
// context values for downstream handlers.
//
// When the token is expired the middleware sets the X-Token-Expired: true response
// header before returning 401, so clients can distinguish expiry from other auth
// failures and call POST /api/v1/auth/refresh to obtain a new token pair.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireJWT(c) {
			return
		}
		c.Next()
	}
}

func requireJWT(c *gin.Context) bool {
	tok := extractBearerToken(c)
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
		if errors.Is(err, jwt.ErrTokenExpired) {
			c.Header("X-Token-Expired", "true")
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "token expired", nil)
		} else {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "invalid token", nil)
		}
		return false
	}
	if claims.UserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "token missing user_id", nil)
		return false
	}

	populateContext(c, claims)
	return true
}

// populateContext writes JWT claims into the Gin context for downstream handlers and RBAC.
func populateContext(c *gin.Context, claims *token.Claims) {
	c.Set(ContextUserID, claims.UserID)
	c.Set(ContextUserCode, claims.UserCode)
	c.Set(ContextEmail, claims.Email)
	c.Set(ContextDisplayName, claims.DisplayName)

	permSet := make(map[string]struct{}, len(claims.Permissions))
	for _, p := range claims.Permissions {
		permSet[p] = struct{}{}
	}
	c.Set(ContextPermissions, permSet)
}

// extractBearerToken reads the raw JWT string from the Authorization: Bearer header.
// Returns an empty string when the header is absent or malformed.
func extractBearerToken(c *gin.Context) string {
	const prefix = "Bearer "
	if raw := c.GetHeader("Authorization"); strings.HasPrefix(raw, prefix) {
		if tok := strings.TrimSpace(raw[len(prefix):]); tok != "" {
			return tok
		}
	}
	return ""
}
