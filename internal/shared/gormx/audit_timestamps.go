package gormx

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/timex"
)

// TouchCreatedUpdated sets both created_at and updated_at to the current Unix second.
func TouchCreatedUpdated(created, updated *int64) {
	now := timex.NowUnix()
	if created != nil {
		*created = now
	}
	if updated != nil {
		*updated = now
	}
}

// TouchUpdated sets updated_at to the current Unix second.
func TouchUpdated(updated *int64) {
	if updated != nil {
		*updated = timex.NowUnix()
	}
}

// SoftDeleteWithAudit sets deleted_at and updated_at to the current Unix second for rows matching query.
func SoftDeleteWithAudit(ctx context.Context, db *gorm.DB, model any, query string, args ...any) error {
	now := timex.NowUnix()
	return db.WithContext(ctx).Model(model).
		Where(query, args...).
		Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error
}
