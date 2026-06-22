package gormx

import "gorm.io/gorm"

// DefaultConfig returns shared GORM settings for application PostgreSQL pools.
func DefaultConfig() *gorm.Config {
	return &gorm.Config{
		Logger: NewSQLLogger(),
	}
}
