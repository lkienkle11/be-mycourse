package gormx

import "gorm.io/gorm"

// RetryInSavepoint runs attempt up to attempts times inside a SAVEPOINT taken
// on tx, rolling back to that savepoint (not the whole surrounding
// transaction) whenever attempt's error satisfies isRetryable — most
// commonly a Postgres unique-constraint violation the caller is about to
// retry with a freshly generated candidate.
//
// This exists because Postgres aborts an ENTIRE transaction on any statement
// error (unique violations included): every subsequent statement on that
// same transaction, including a naive "just try the write again" retry,
// then fails immediately with "current transaction is aborted" instead of
// running at all. A SAVEPOINT is the only way to recover from one failed
// statement without discarding everything already done earlier in the same
// outer transaction (e.g. other fields already updated in the same
// request). Found the hard way: internal/course/infra/repo_instructor.go's
// original applyUpdateSlugWithRetry retried a write on the same *gorm.DB
// without a savepoint and could never actually succeed past the first real
// collision.
//
// attempt is responsible for doing its own work (e.g. re-resolving a
// candidate value before writing) on every call, including the first;
// RetryInSavepoint does not cache or reuse anything between attempts.
// Requires a driver that implements gorm.SavePointerDialectorInterface —
// gorm.io/driver/postgres does.
func RetryInSavepoint(tx *gorm.DB, name string, attempts int, isRetryable func(error) bool, attempt func() error) error {
	if err := tx.SavePoint(name).Error; err != nil {
		return err
	}
	var lastErr error
	for range attempts {
		err := attempt()
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		if rbErr := tx.RollbackTo(name).Error; rbErr != nil {
			return rbErr
		}
		lastErr = err
	}
	return lastErr
}
