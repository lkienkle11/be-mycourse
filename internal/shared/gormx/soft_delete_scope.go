package gormx

import "gorm.io/gorm"

// ScopeActiveOnly restricts queries to rows where deleted_at IS NULL.
func ScopeActiveOnly(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

// ScopeIncludeDeleted is a no-op scope marker for list endpoints that include soft-deleted rows.
func ScopeIncludeDeleted(db *gorm.DB) *gorm.DB {
	return db
}
