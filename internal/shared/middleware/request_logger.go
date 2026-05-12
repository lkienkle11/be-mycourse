package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/logger"
)

// RequestLogger attaches a request_id (client header or generated UUID) to the
// Go context and Gin context, echoes it on X-Request-ID, and emits one structured
// access log line per request (no body — avoids PII).
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Writer.Header().Set(HeaderRequestID, rid)
		c.Set(GinContextKeyRequestID, rid)

		ctx := logger.WithRequestID(c.Request.Context(), rid)
		c.Request = c.Request.WithContext(ctx)

		path := c.Request.URL.Path
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		written := c.Writer.Size()

		zap.L().Info("http_request",
			zap.String("request_id", rid),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("response_bytes", written),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
