package cache

import (
	"context"
	"testing"
	"time"
)

// These tests only exercise the fail-open path (Redis == nil), since this
// package has no live-Redis test harness (no miniredis/redismock dependency
// in go.mod) and adding one is out of scope for this change. A live
// set-then-get round trip must be verified manually against a running Redis
// instance (see tasks.md task 2.1).

func TestGetJSONNoRedisReturnsZeroValueAndFalse(t *testing.T) {
	if Redis != nil {
		t.Skip("Redis client already configured in this test binary; nil-safety path not exercised")
	}
	got, ok := GetJSON[[]string](context.Background(), "mycourse:catalog:test:missing")
	if ok {
		t.Fatalf("expected ok=false when Redis is unavailable, got true with value %v", got)
	}
	if got != nil {
		t.Fatalf("expected zero value (nil slice), got %v", got)
	}
}

func TestSetJSONNoRedisIsNoop(t *testing.T) {
	if Redis != nil {
		t.Skip("Redis client already configured in this test binary; nil-safety path not exercised")
	}
	// Must not panic when Redis is unavailable.
	SetJSON(context.Background(), "mycourse:catalog:test:missing", time.Minute, []string{"a", "b"})
}
