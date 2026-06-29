package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/gormx"
)

func firstActiveMediaFile(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.File, error) {
	var row mediaFileRow
	if err := gormx.FirstWhere(ctx, gormx.ScopeActiveOnly(db), &row, query, args...); err != nil {
		return nil, err
	}
	return rowToFile(&row), nil
}
