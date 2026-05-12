package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/setting"
)

// RequireInternalAPIKey protects routes with X-API-Key matching app.api_key (constant-time compare).
func RequireInternalAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := setting.AppSetting.ApiKey
		if cfg == "" {
			response.AbortFail(c, http.StatusServiceUnavailable, errors.InternalError, "internal api key not configured", nil)
			return
		}
		key := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if len(key) != len(cfg) || subtle.ConstantTimeCompare([]byte(key), []byte(cfg)) != 1 {
			response.AbortFail(c, http.StatusUnauthorized, errors.Unauthorized, "invalid api key", nil)
			return
		}
		c.Next()
	}
}
