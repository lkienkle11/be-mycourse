package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/gormx"
)

func findActiveUserWhere(ctx context.Context, db *gorm.DB, dest any, query string, args ...any) error {
	return gormx.FirstWhere(ctx, db, dest, query+" AND deleted_at IS NULL", args...)
}
