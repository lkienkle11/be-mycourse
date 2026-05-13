package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/setting"
)

// EnsureCSRFCookie provisions a CSRF token cookie for browser clients.
func EnsureCSRFCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie(CookieCSRFToken)
		if err != nil {
			token, genErr := generateCSRFToken()
			if genErr != nil {
				response.AbortFail(c, http.StatusInternalServerError, apperrors.InternalError, "failed to issue csrf token", nil)
				return
			}
			setCSRFCookie(c, token)
		}
		c.Next()
	}
}

// RequireCSRF rejects unsafe methods when CSRF header/cookie mismatch.
func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isCSRFProtectedMethod(c.Request.Method) {
			c.Next()
			return
		}
		cookieToken, err := c.Cookie(CookieCSRFToken)
		if err != nil || strings.TrimSpace(cookieToken) == "" {
			response.AbortFail(c, http.StatusForbidden, apperrors.Forbidden, "missing csrf cookie", nil)
			return
		}
		headerToken := strings.TrimSpace(c.GetHeader(HeaderCSRFToken))
		if headerToken == "" {
			response.AbortFail(c, http.StatusForbidden, apperrors.Forbidden, "missing csrf header", nil)
			return
		}
		if headerToken != cookieToken {
			response.AbortFail(c, http.StatusForbidden, apperrors.Forbidden, "invalid csrf token", nil)
			return
		}
		c.Next()
	}
}

func setCSRFCookie(c *gin.Context, token string) {
	secure := strings.EqualFold(strings.TrimSpace(setting.ServerSetting.RunMode), "release")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieCSRFToken, token, 12*60*60, "/", "", secure, false)
}

func generateCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isCSRFProtectedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}
