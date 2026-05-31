package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/resilience"
	"mycourse-io-be/internal/shared/response"
)

// CircuitBreakerMiddleware fast-fails when the global circuit is open and tracks request load/errors.
func CircuitBreakerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !resilience.Global.Allow() {
			response.AbortFail(
				c,
				http.StatusServiceUnavailable,
				errors.ServiceUnavailable,
				errors.DefaultMessage(errors.ServiceUnavailable),
				nil,
			)
			return
		}
		end := resilience.Global.RecordRequestStart()
		c.Next()
		success := c.Writer.Status() < http.StatusInternalServerError
		end(success)
	}
}
