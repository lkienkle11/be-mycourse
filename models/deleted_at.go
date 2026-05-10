package models

import "gorm.io/gorm"

// DeletedAt is a GORM soft-delete column type alias for User and other models.
type DeletedAt = gorm.DeletedAt
