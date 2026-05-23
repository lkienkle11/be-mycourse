package gormx

import (
	"context"
	"errors"

	"gorm.io/gorm"

	sharedErrors "mycourse-io-be/internal/shared/errors"
)

// FirstWhere loads the first row matching query into dest or returns ErrNotFound.
func FirstWhere(ctx context.Context, db *gorm.DB, dest any, query string, args ...any) error {
	if err := db.WithContext(ctx).Where(query, args...).First(dest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sharedErrors.ErrNotFound
		}
		return err
	}
	return nil
}
