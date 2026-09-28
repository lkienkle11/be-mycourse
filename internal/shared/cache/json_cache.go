package cache

import (
	"context"
	"encoding/json"
	"time"
)

// GetJSON reads a JSON-encoded value from Redis. It returns (zero, false) on
// a cache miss, a decode error, or when Redis is unavailable — callers must
// always be able to fall through to the source of truth.
func GetJSON[T any](ctx context.Context, key string) (T, bool) {
	var out T
	if !RedisAvailable() {
		return out, false
	}
	raw, err := Redis.Get(ctx, key).Bytes()
	if err != nil {
		return out, false
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, false
	}
	return out, true
}

// SetJSON writes a JSON-encoded value to Redis with the given TTL. It is a
// no-op (not an error) when Redis is unavailable.
func SetJSON[T any](ctx context.Context, key string, ttl time.Duration, value T) {
	if !RedisAvailable() {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	Redis.Set(ctx, key, raw, ttl)
}
