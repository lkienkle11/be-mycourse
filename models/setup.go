package models

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appmigrations "mycourse-io-be/migrations"
	"mycourse-io-be/pkg/dbmigrate"
	"mycourse-io-be/pkg/setting"
)

var DB *gorm.DB

func Setup() error {
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN: set URL or Host+Name (and User as needed); Supabase Postgres uses [supabase].DBURL separately")
	}

	// fmt.Printf("dsn database: %s\n", dsn)

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
	if DB == nil {
		return errors.New("database not initialized")
	}
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN")
	}
	return dbmigrate.Up(context.Background(), dsn, appmigrations.Files, ".")
}
