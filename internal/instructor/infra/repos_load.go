package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/instructor/domain"
)

func loadActiveRow(ctx context.Context, db *gorm.DB, dest any, query string, args ...any) error {
	return activeScope(db.WithContext(ctx)).Where(query, args...).First(dest).Error
}

func loadApplicationRow(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.Application, error) {
	return loadMappedRow(ctx, db, query, args, func(row *applicationRow) domain.Application { return appRowToDomain(row) })
}

func loadProfileRow(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.Profile, error) {
	return loadMappedRow(ctx, db, query, args, func(row *profileRow) domain.Profile { return profileRowToDomain(row) })
}

func loadMappedRow[T any, D any](ctx context.Context, db *gorm.DB, query string, args []any, mapFn func(*T) D) (*D, error) {
	var row T
	if err := loadActiveRow(ctx, db, &row, query, args...); err != nil {
		return nil, mapNotFound(err)
	}
	out := mapFn(&row)
	return &out, nil
}
