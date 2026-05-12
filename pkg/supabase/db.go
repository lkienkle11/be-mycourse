package supabase

import (
	"database/sql"
	"errors"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
)

// GormDB is a separate pool to the Supabase-hosted Postgres (session/pooler URL).
// Use models.DB for your primary [database] PostgreSQL; use this when you talk to Supabase Postgres over DBURL.
var GormDB *gorm.DB

// SetupDatabase opens GormDB when [supabase].DBURL is set; no-op when empty.
func SetupDatabase() error {
	dsn := strings.TrimSpace(setting.SupabaseSetting.DBURL)
	if dsn == "" {
		return nil
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	GormDB = db
	return nil
}

// StdDB returns the shared pool for GormDB (raw SQL). Nil if DBURL was not configured.
func StdDB() (*sql.DB, error) {
	if GormDB == nil {
		return nil, errors.New(constants.MsgSupabaseDBPoolNotInitialized)
	}
	return GormDB.DB()
}
