package infra

import (
	"context"

	"gorm.io/gorm"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/gormx"
)

func firstActiveMediaFile(ctx context.Context, db *gorm.DB, query string, args ...any) (*domain.File, error) {
	var row mediaFileListRow
	if err := gormx.FirstWhere(
		ctx,
		gormx.ScopeActiveOnly(db).Joins(mediaOwnerJoinSQL).Select(mediaOwnerSelectSQL),
		&row,
		query,
		args...,
	); err != nil {
		return nil, err
	}
	f := rowToFile(&row.mediaFileRow)
	applyMediaOwnerIdentity(f, row.mediaFileOwnerProjection)
	return f, nil
}
