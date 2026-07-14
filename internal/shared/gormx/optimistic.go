package gormx

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/timex"
)

// OptimisticUpdate applies updates when id + row_version match an active row.
// On zero rows affected it returns lockErr (caller-specific optimistic lock error).
func OptimisticUpdate(
	ctx context.Context,
	tx *gorm.DB,
	model any,
	rowID string,
	expectedRowVersion int64,
	updates map[string]any,
	lockErr error,
) error {
	updates["updated_at"] = timex.NowUnix()
	updates["row_version"] = gorm.Expr("row_version + 1")
	result := tx.WithContext(ctx).
		Model(model).
		Where("id = ? AND row_version = ? AND deleted_at IS NULL", rowID, expectedRowVersion).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return lockErr
	}
	return nil
}
