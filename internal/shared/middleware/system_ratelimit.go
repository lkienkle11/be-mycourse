package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/ratelimit"
	"mycourse-io-be/internal/shared/response"
)

// SystemIPQuota overrides default /api/system rate limits for a single client IP.
type SystemIPQuota struct {
	Attempts int
	Minutes  int
}

var (
	systemRLMu    sync.RWMutex
	systemRLExtra = map[string]SystemIPQuota{} // client IP → relaxed quota (optional)
	systemRLStore = ratelimit.NewInMemoryStore(true)
)

// SetSystemRateLimitOverride registers a custom quota for an IP (attempts per rolling window of minutes).
// Pass attempts < 1 or minutes < 1 to remove an override. Safe for hot-reload from config later.
func SetSystemRateLimitOverride(ip string, attempts, minutes int) {
	systemRLMu.Lock()
	defer systemRLMu.Unlock()
	if attempts < 1 || minutes < 1 {
		delete(systemRLExtra, ip)
		return
	}
	systemRLExtra[ip] = SystemIPQuota{Attempts: attempts, Minutes: minutes}
}

func systemQuotaForIP(ip string, defaultAttempts, defaultMinutes int) (attempts, minutes int) {
	systemRLMu.RLock()
	q, ok := systemRLExtra[ip]
	systemRLMu.RUnlock()
	if ok && q.Attempts >= 1 && q.Minutes >= 1 {
		return q.Attempts, q.Minutes
	}
	return defaultAttempts, defaultMinutes
}

// RateLimitSystemIP limits each client IP on /api/system (default attempts per rolling window of minutes).
// Per-IP overrides are applied via SetSystemRateLimitOverride.
func RateLimitSystemIP(defaultAttempts, defaultMinutes int) gin.HandlerFunc {
	if defaultAttempts < 1 || defaultMinutes < 1 {
		return func(c *gin.Context) { c.Next() }
	}

	id := atomic.AddUint64(&systemRateLimiterSeq, 1)
	keyPrefix := strconv.FormatUint(id, 10) + "|"

	return func(c *gin.Context) {
		ip := c.ClientIP()
		attempts, minutes := systemQuotaForIP(ip, defaultAttempts, defaultMinutes)
		if attempts < 1 || minutes < 1 {
			c.Next()
			return
		}
		windowSec := int64(minutes) * 60
		key := keyPrefix + ip + "|" + strconv.Itoa(attempts) + "|" + strconv.Itoa(minutes)
		now := time.Now().Unix()
		windowStart := ratelimit.WindowStart(now, windowSec)
		effectiveAttempts := effectiveRateAttempts(attempts)

		if !systemRLStore.Allow(key, windowSec, windowStart, effectiveAttempts) {
			response.AbortFail(c, http.StatusTooManyRequests, errors.TooManyRequests, errors.DefaultMessage(errors.TooManyRequests), nil)
			return
		}
		c.Next()
	}
}

var systemRateLimiterSeq uint64
