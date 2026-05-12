package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

// SystemTokenVerifier verifies a system bearer token.  Implemented by the system
// application layer so the middleware layer never imports a concrete domain package.
type SystemTokenVerifier interface {
	VerifySystemAccessToken(token string) error
}

// RequireSystemAccessToken validates the system JWT (Authorization: Bearer) using the
// injected verifier (which reads app_token_env from DB).
func RequireSystemAccessToken(verifier SystemTokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, errors.Unauthorized, "missing bearer token", nil)
			return
		}
		tok := strings.TrimSpace(raw[7:])
		if tok == "" {
			response.AbortFail(c, http.StatusUnauthorized, errors.Unauthorized, "missing bearer token", nil)
			return
		}
		if err := verifier.VerifySystemAccessToken(tok); err != nil {
			response.AbortFail(c, http.StatusUnauthorized, errors.Unauthorized, "invalid or expired system token", nil)
			return
		}
		c.Next()
	}
}
