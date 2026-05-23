package infra

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
)

func firstActiveMediaFile(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.File, error) {
	var row mediaFileRow
	if err := db.WithContext(ctx).Where(query, args...).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return rowToFile(&row), nil
}
