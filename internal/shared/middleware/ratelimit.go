package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/ratelimit"
	"mycourse-io-be/internal/shared/response"
)

var localRateStore = ratelimit.NewInMemoryStore(false)

// RateLimitLocal limits each client IP to at most attempts requests per rolling calendar-aligned
// window of minutes minutes. State is in-process only (global map). Each call returns an
// independent limiter instance (separate counters even when attempts/minutes match), so e.g.
// routerAuthen and routerNotAuthen can use different quotas for the same IP.
//
// attempts < 1 or minutes < 1 produces a no-op middleware.
func RateLimitLocal(attempts, minutes int) gin.HandlerFunc {
	if attempts < 1 || minutes < 1 {
		return func(c *gin.Context) { c.Next() }
	}

	windowSec := int64(minutes) * 60
	id := ratelimit.NextStoreID()
	keyPrefix := strconv.FormatUint(id, 10) + "|"

	return func(c *gin.Context) {
		key := keyPrefix + c.ClientIP()
		now := time.Now().Unix()
		windowStart := ratelimit.WindowStart(now, windowSec)
		effectiveAttempts := effectiveRateAttempts(attempts)
		if !localRateStore.Allow(key, windowSec, windowStart, effectiveAttempts) {
			response.AbortFail(c, http.StatusTooManyRequests, errors.TooManyRequests, errors.DefaultMessage(errors.TooManyRequests), nil)
			return
		}
		c.Next()
	}
}
