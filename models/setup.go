package models

import (
	"database/sql"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mycourse-io-be/pkg/setting"
)

var DB *gorm.DB

func Setup() error {
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN: set URL or Host+Name (and User as needed); Supabase Postgres uses [supabase].DBURL separately")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db
	return nil
}

// StdDB returns the shared PostgreSQL pool used by GORM, for database/sql raw queries.
func StdDB() (*sql.DB, error) {
	if DB == nil {
		return nil, errors.New("database not initialized")
	}
	return DB.DB()
}

func MigrateDatabase() error {
	return nil
}
