package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

// InMemoryStore holds fixed-window counters in process memory (HTTP middleware).
type InMemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	matchWS bool
}

// NewInMemoryStore returns a store. matchWindowSec enables windowSec matching on bucket reset (system IP quotas).
func NewInMemoryStore(matchWindowSec bool) *InMemoryStore {
	s := &InMemoryStore{
		buckets: make(map[string]*bucket),
		matchWS: matchWindowSec,
	}
	t := time.NewTicker(5 * time.Minute)
	go func() {
		for range t.C {
			CleanupStaleBuckets(s.buckets, &s.mu, time.Now().Unix())
		}
	}()
	return s
}

// Allow records one attempt for key within the fixed window and returns whether it is permitted.
func (s *InMemoryStore) Allow(key string, windowSec, windowStart int64, attempts int) bool {
	return AllowFixedWindow(s.buckets, &s.mu, key, windowSec, windowStart, attempts, s.matchWS)
}

var nextStoreID uint64

// NextStoreID returns a unique prefix for isolating limiter instances (same IP, different route groups).
func NextStoreID() uint64 {
	return atomic.AddUint64(&nextStoreID, 1)
}
