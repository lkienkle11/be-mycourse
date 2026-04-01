package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
)

func BeforeInterceptor() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Reserved for authn/authz/tenant/permission checks.
		if c.GetHeader("X-Blocked") == "1" {
			response.AbortFail(c, http.StatusForbidden, errcode.Forbidden, "blocked", nil)
			return
		}
		c.Next()
	}
}
