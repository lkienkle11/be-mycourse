package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/gormx"
)

func firstActiveMediaFile(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.File, error) {
	var row mediaFileListRow
	base := gormx.ScopeActiveOnly(db).Model(&mediaFileRow{})
	if err := gormx.FirstWhere(
		ctx,
		base.Joins(mediaOwnerJoinSQL).Select(mediaOwnerSelectSQL),
		&row,
		query,
		args...,
	); err != nil {
		return nil, err
	}
	mapped := fileFromMediaListRow(&row)
	return &mapped, nil
}
