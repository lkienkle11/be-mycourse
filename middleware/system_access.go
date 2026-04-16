package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services"
)

// RequireSystemAccessToken validates the system JWT (Authorization: Bearer) using app_token_env from DB.
func RequireSystemAccessToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "missing bearer token", nil)
			return
		}
		tok := strings.TrimSpace(raw[7:])
		if tok == "" {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "missing bearer token", nil)
			return
		}
		if err := services.VerifySystemAccessToken(db, tok); err != nil {
			response.AbortFail(c, http.StatusUnauthorized, errcode.Unauthorized, "invalid or expired system token", nil)
			return
		}
		c.Next()
	}
}
