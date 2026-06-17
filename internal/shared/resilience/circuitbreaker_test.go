package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreakerOpensAfterDBFailures(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		DBFailuresToOpen:   2,
		OpenCooldownSec:    30,
		HalfOpenProbeQuota: 1,
	})
	if !cb.Allow() {
		t.Fatal("initial request should be allowed")
	}
	cb.recordDBFailure()
	if !cb.Allow() {
		t.Fatal("one failure should not open circuit")
	}
	cb.recordDBFailure()
	if cb.Allow() {
		t.Fatal("circuit should be open after threshold failures")
	}
}

func TestCircuitBreakerHalfOpenThenCloses(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		DBFailuresToOpen:   1,
		OpenCooldownSec:    0,
		HalfOpenProbeQuota: 1,
	})
	cb.recordDBFailure()
	cb.mu.Lock()
	cb.openUntil = time.Now().Add(-time.Second)
	cb.mu.Unlock()
	if !cb.Allow() {
		t.Fatal("half-open should allow probe")
	}
	cb.recordDBSuccess()
	if !cb.Allow() {
		t.Fatal("circuit should be closed after successful probe")
	}
}

func TestAttemptsFactorDegradedInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		DegradedAttemptsFactor: 0.5,
		HalfOpenProbeQuota:     2,
	})
	cb.mu.Lock()
	cb.state = stateHalfOpen
	cb.mu.Unlock()
	if cb.AttemptsFactor() != 0.5 {
		t.Fatalf("expected 0.5 factor, got %v", cb.AttemptsFactor())
	}
	cb.mu.Lock()
	cb.state = stateClosed
	cb.mu.Unlock()
	if cb.AttemptsFactor() != 1.0 {
		t.Fatalf("expected 1.0 factor when closed, got %v", cb.AttemptsFactor())
	}
}
