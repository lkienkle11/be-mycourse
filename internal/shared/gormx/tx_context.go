package gormx

import (
	"context"

	"gorm.io/gorm"
)

type txContextKey struct{}

// WithTx returns a context carrying tx. A repository that only receives a context.Context (not
// the caller's *gorm.DB directly, e.g. one belonging to a different bounded context) can use
// DBFromContext to join tx instead of running its own, separately committed transaction.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// DBFromContext returns the transaction stored by WithTx, if any, otherwise fallback.WithContext(ctx).
func DBFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback.WithContext(ctx)
}
