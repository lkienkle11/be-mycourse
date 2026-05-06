// Package appdb holds the primary application PostgreSQL GORM handle so packages
// that must not import mycourse-io-be/models (e.g. api/ per depguard) can still
// pass the DB into services, middleware, and jobs.
package appdb

import "gorm.io/gorm"

var primary *gorm.DB

// Set stores the shared GORM DB after models.Setup. Call once at process startup.
func Set(db *gorm.DB) {
	primary = db
}

// Conn returns the primary DB handle, or nil if Set has not been called.
func Conn() *gorm.DB {
	return primary
}
