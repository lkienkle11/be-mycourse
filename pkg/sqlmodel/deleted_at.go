package sqlmodel

import "gorm.io/gorm"

// DeletedAt is an alias for GORM soft-delete so model structs under models/ can use
// the same column behavior without importing gorm.io/gorm in those files.
type DeletedAt = gorm.DeletedAt
