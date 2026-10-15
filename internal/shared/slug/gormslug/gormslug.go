// Package gormslug adapts the pure slug resolver in internal/shared/slug to a
// GORM-backed table: a table-scoped availability check and the shared
// duplicate-key / retry policy used by every slugged entity (course,
// taxonomy, user).
package gormslug

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/slug"
)

// RetryAttempts is how many times a slug write is retried after losing a
// unique-index race (random-suffix collisions can need more than one pass
// under contention). Shared by course, taxonomy and user slug writes.
const RetryAttempts = 5

// AvailableIn returns an AvailableFunc that reports whether a candidate slug
// is free among active rows (deleted_at IS NULL) of newModel's table.
// excludeID, when non-nil and non-blank, excludes that row so resubmitting
// the row's own current slug is never a self-collision.
func AvailableIn[R any](ctx context.Context, db *gorm.DB, newModel func() *R, excludeID *string) slug.AvailableFunc {
	return func(candidate string) (bool, error) {
		q := db.WithContext(ctx).Model(newModel()).Where("deleted_at IS NULL").Where("slug = ?", candidate)
		if excludeID != nil {
			if id := strings.TrimSpace(*excludeID); id != "" {
				q = q.Where("id != ?", id)
			}
		}
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return false, err
		}
		return count == 0, nil
	}
}

// IsDuplicate reports whether err is a unique violation on one of the given
// slug indexes, so callers retry only genuine slug races and never swallow an
// unrelated unique violation.
func IsDuplicate(err error, indexNames ...string) bool {
	return gormx.IsUniqueViolation(err, indexNames...)
}
