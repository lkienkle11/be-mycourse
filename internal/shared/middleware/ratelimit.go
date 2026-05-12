package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
)

type rateBucket struct {
	windowStart int64 // unix seconds, aligned to window of windowSec
	windowSec   int64
	count       int
}

var (
	rateMu        sync.Mutex
	rateBuckets   = make(map[string]*rateBucket) // key: limiter instance + "|" + client IP
	nextLimiterID uint64
)

func init() {
	t := time.NewTicker(5 * time.Minute)
	go func() {
		for range t.C {
			cleanupRateBuckets()
		}
	}()
}

func cleanupRateBuckets() {
	now := time.Now().Unix()
	rateMu.Lock()
	defer rateMu.Unlock()
	for k, b := range rateBuckets {
		if now-b.windowStart > 2*b.windowSec {
			delete(rateBuckets, k)
		}
	}
}

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
	id := atomic.AddUint64(&nextLimiterID, 1)
	keyPrefix := strconv.FormatUint(id, 10) + "|"

	return func(c *gin.Context) {
		key := keyPrefix + c.ClientIP()
		now := time.Now().Unix()
		windowStart := now - (now % windowSec)
		if !rateLocalAllow(key, windowSec, windowStart, attempts) {
			response.AbortFail(c, http.StatusTooManyRequests, errors.TooManyRequests, errors.DefaultMessage(errors.TooManyRequests), nil)
			return
		}
		c.Next()
	}
}

func rateLocalAllow(key string, windowSec, windowStart int64, attempts int) bool {
	rateMu.Lock()
	b := rateBuckets[key]
	if b == nil || b.windowStart != windowStart {
		rateBuckets[key] = &rateBucket{
			windowStart: windowStart,
			windowSec:   windowSec,
			count:       1,
		}
		rateMu.Unlock()
		return true
	}
	b.count++
	n := b.count
	rateMu.Unlock()
	return n <= attempts
}
