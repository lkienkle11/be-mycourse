package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	rateLimitMaxPerWindow = 60
	rateWindow            = time.Minute
)

type ipBucket struct {
	windowStart int64 // unix seconds, aligned to rateWindow
	count       int
}

var (
	rateMu      sync.Mutex
	rateBuckets = make(map[string]*ipBucket)
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
	cutoff := time.Now().Add(-3 * rateWindow).Unix()
	rateMu.Lock()
	defer rateMu.Unlock()
	for k, b := range rateBuckets {
		if b.windowStart < cutoff {
			delete(rateBuckets, k)
		}
	}
}

// RateLimitLocal allows at most 60 requests per client IP per calendar-aligned minute bucket.
// State is kept in process memory only (not Redis/DB).
func RateLimitLocal() gin.HandlerFunc {
	windowSec := int64(rateWindow.Seconds())
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now().Unix()
		windowStart := now - (now % windowSec)

		rateMu.Lock()
		b := rateBuckets[ip]
		if b == nil || b.windowStart != windowStart {
			rateBuckets[ip] = &ipBucket{windowStart: windowStart, count: 1}
			rateMu.Unlock()
			c.Next()
			return
		}
		b.count++
		n := b.count
		rateMu.Unlock()

		if n > rateLimitMaxPerWindow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message":    "rate limit exceeded",
				"limit":      rateLimitMaxPerWindow,
				"window_sec": int(windowSec),
			})
			return
		}
		c.Next()
	}
}
