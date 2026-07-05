package gormx

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScopeActiveOnly restricts queries to rows where deleted_at IS NULL.
func ScopeActiveOnly(db *gorm.DB) *gorm.DB {
	// Resolve deleted_at against the current table/alias to avoid
	// ambiguous-column errors when queries include joins to tables that
	// also have deleted_at.
	return db.Where(clause.Eq{
		Column: clause.Column{Table: clause.CurrentTable, Name: "deleted_at"},
		Value:  nil,
	})
}

// ScopeIncludeDeleted is a no-op scope marker for list endpoints that include soft-deleted rows.
func ScopeIncludeDeleted(db *gorm.DB) *gorm.DB {
	return db
}
