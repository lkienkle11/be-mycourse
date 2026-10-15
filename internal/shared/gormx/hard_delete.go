package gormx

import (
	"context"

	"gorm.io/gorm"
)

// HardDeleteByID permanently deletes the row of model whose primary key id
// equals id. The id is always bound as an "id = ?" parameter: GORM's
// Delete(&model, idString) form inlines a string argument as a raw SQL
// condition, which Postgres rejects for a UUID, so every UUID-keyed permanent
// delete must go through this helper instead.
func HardDeleteByID(ctx context.Context, db *gorm.DB, model any, id string) error {
	return db.WithContext(ctx).Where("id = ?", id).Delete(model).Error
}
