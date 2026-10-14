package gormx

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

// TestRetryInSavepointReturnsImmediatelyWhenSavepointUnsupported verifies the
// first real GORM code path this function exercises without a live DB
// connection: a *gorm.DB with no configured Dialector cannot satisfy
// gorm.SavePointerDialectorInterface, so tx.SavePoint(name) itself fails
// (gorm.ErrUnsupportedDriver) before attempt is ever called. This repo has
// no DB-seeded test harness anywhere (confirmed: no sqlite/testcontainers
// usage in any *_test.go), so the actual SAVEPOINT/ROLLBACK TO retry
// behavior against a real transaction is verified live against the dev DB
// instead (see tasks.md 1.4/8/10), matching this repo's established
// DB-behavior verification convention rather than introducing a new one.
func TestRetryInSavepointReturnsImmediatelyWhenSavepointUnsupported(t *testing.T) {
	tx := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{}}
	attemptCalls := 0
	err := RetryInSavepoint(tx, "test_savepoint", 5, func(error) bool { return true }, func() error {
		attemptCalls++
		return nil
	})
	if err == nil {
		t.Fatal("RetryInSavepoint() error = nil, want a savepoint-unsupported error")
	}
	if !errors.Is(err, gorm.ErrUnsupportedDriver) {
		t.Fatalf("RetryInSavepoint() error = %v, want gorm.ErrUnsupportedDriver", err)
	}
	if attemptCalls != 0 {
		t.Fatalf("attempt called %d times, want 0 (should fail before ever attempting)", attemptCalls)
	}
}
