package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/setting"
)

// RequireInternalAPIKey protects routes with X-API-Key matching app.api_key (constant-time compare).
func RequireInternalAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := setting.AppSetting.ApiKey
		if cfg == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "internal api key not configured"})
			return
		}
		key := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if len(key) != len(cfg) || subtle.ConstantTimeCompare([]byte(key), []byte(cfg)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid api key"})
			return
		}
		c.Next()
	}
}
