package ratelimit

import "sync"

type bucket struct {
	windowStart int64
	windowSec   int64
	count       int
}

// WindowStart aligns now to the start of a fixed calendar window of windowSec seconds.
func WindowStart(now, windowSec int64) int64 {
	return now - (now % windowSec)
}

// AllowFixedWindow records one attempt for key and returns whether it is within attempts.
// When matchWindowSec is true, a bucket is reset if its stored windowSec differs (system IP overrides).
func AllowFixedWindow(
	buckets map[string]*bucket,
	mu *sync.Mutex,
	key string,
	windowSec, windowStart int64,
	attempts int,
	matchWindowSec bool,
) bool {
	mu.Lock()
	defer mu.Unlock()

	b := buckets[key]
	if b == nil || b.windowStart != windowStart || (matchWindowSec && b.windowSec != windowSec) {
		buckets[key] = &bucket{
			windowStart: windowStart,
			windowSec:   windowSec,
			count:       1,
		}
		return true
	}
	b.count++
	return b.count <= attempts
}

// CleanupStaleBuckets removes entries older than 2× their window.
func CleanupStaleBuckets(buckets map[string]*bucket, mu *sync.Mutex, now int64) {
	mu.Lock()
	defer mu.Unlock()
	for k, b := range buckets {
		if now-b.windowStart > 2*b.windowSec {
			delete(buckets, k)
		}
	}
}
