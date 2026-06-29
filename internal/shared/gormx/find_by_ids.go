package gormx

import (
	"context"

	"gorm.io/gorm"
)

// FindActiveByIDs loads rows from table where id IN (ids) AND deleted_at IS NULL.
// Returns nil slice when ids is empty.
func FindActiveByIDs[T any](
	ctx context.Context,
	db *gorm.DB,
	table string,
	selectCols string,
	ids []string,
) ([]T, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []T
	q := db.WithContext(ctx).Table(table).Where("id IN ? AND deleted_at IS NULL", ids)
	if selectCols != "" {
		q = q.Select(selectCols)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// IndexByID builds a map keyed by idFn(row) from rows.
func IndexByID[T any](rows []T, idFn func(T) string) map[string]T {
	out := make(map[string]T, len(rows))
	for _, row := range rows {
		out[idFn(row)] = row
	}
	return out
}
