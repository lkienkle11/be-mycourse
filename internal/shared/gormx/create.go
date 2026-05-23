package gormx

import (
	"context"

	"gorm.io/gorm"
)

// CreateAndThen inserts row and runs then on success (e.g. copy ID/timestamps into domain).
func CreateAndThen(ctx context.Context, db *gorm.DB, row any, then func()) error {
	if err := db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	then()
	return nil
}
