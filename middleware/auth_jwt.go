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
	"mycourse-io-be/services"
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
//
// Transparent cookie refresh: when the access token arrives via cookie (not the
// Authorization header) and has expired, the middleware automatically attempts to
// rotate tokens using the refresh_token and session_id cookies.  On success it
// reissues all three cookies and continues the request with the fresh claims —
// the client and the handler are completely unaware of the rotation.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireJWT(c) {
			return
		}
		c.Next()
	}
}

func requireJWT(c *gin.Context) bool {
	fromHeader := false
	tok := extractTokenWithSource(c, &fromHeader)
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
		// Transparent refresh only when the token originated from a cookie and the
		// sole reason for failure is expiry.  Bearer-header tokens are never silently
		// refreshed — the caller is responsible for managing its own token lifecycle.
		if !fromHeader && errors.Is(err, jwt.ErrTokenExpired) {
			return tryTokenRefreshFromCookie(c, secret)
		}
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "invalid token", nil)
		return false
	}
	if claims.UserID == 0 {
		response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "token missing user_id", nil)
		return false
	}

	populateContext(c, claims)
	return true
}

// tryTokenRefreshFromCookie reads the session_id and refresh_token cookies, calls the
// session-rotation service, re-issues all three auth cookies with the new tokens, and
// populates the Gin context from the freshly issued access token.
func tryTokenRefreshFromCookie(c *gin.Context, secret string) bool {
	sessionStr, err := c.Cookie("session_id")
	if err != nil || sessionStr == "" {
		response.AbortFail(c, http.StatusUnauthorized, errcode.InvalidSession, errcode.DefaultMessage(errcode.InvalidSession), nil)
		return false
	}

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.AbortFail(c, http.StatusUnauthorized, errcode.InvalidSession, errcode.DefaultMessage(errcode.InvalidSession), nil)
		return false
	}

	result, svcErr := services.RefreshSession(sessionStr, refreshToken)
	if svcErr != nil {
		switch {
		case errors.Is(svcErr, services.ErrRefreshTokenExpired):
			response.AbortFail(c, http.StatusUnauthorized, errcode.RefreshTokenExpired, errcode.DefaultMessage(errcode.RefreshTokenExpired), nil)
		case errors.Is(svcErr, services.ErrInvalidSession), errors.Is(svcErr, services.ErrUserNotFound):
			response.AbortFail(c, http.StatusUnauthorized, errcode.InvalidSession, errcode.DefaultMessage(errcode.InvalidSession), nil)
		case errors.Is(svcErr, services.ErrUserDisabled):
			response.AbortFail(c, http.StatusForbidden, errcode.UserDisabled, errcode.DefaultMessage(errcode.UserDisabled), nil)
		default:
			response.AbortFail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return false
	}

	// Reissue all three cookies with the rotated token set.
	secure := setting.ServerSetting.RunMode == "release"
	refreshMaxAge := int(result.RefreshTTL.Seconds())
	accessMaxAge := int(services.AccessTokenTTL.Seconds())

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", result.AccessToken, accessMaxAge, "/", "", secure, true)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", result.RefreshToken, refreshMaxAge, "/", "", secure, true)
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session_id", result.SessionStr, refreshMaxAge, "/", "", secure, true)

	// Parse the new access token to populate the request context.
	newClaims, parseErr := token.ParseAccess(secret, result.AccessToken)
	if parseErr != nil || newClaims.UserID == 0 {
		response.AbortFail(c, http.StatusInternalServerError, errcode.InternalError, "failed to parse rotated token", nil)
		return false
	}

	populateContext(c, newClaims)
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

// extractTokenWithSource reads the raw JWT string from Authorization header first,
// then falls back to the access_token HttpOnly cookie.
// fromHeader is set to true when the token comes from the Authorization header.
func extractTokenWithSource(c *gin.Context, fromHeader *bool) string {
	const prefix = "Bearer "
	if raw := c.GetHeader("Authorization"); strings.HasPrefix(raw, prefix) {
		if tok := strings.TrimSpace(raw[len(prefix):]); tok != "" {
			*fromHeader = true
			return tok
		}
	}
	if tok, err := c.Cookie("access_token"); err == nil && tok != "" {
		return tok
	}
	return ""
}
