// Package db provides the shared PostgreSQL GORM handle, migration runner, and
// raw sql.DB accessor for the application.  It combines the old internal/appdb
// (handle storage) and models/setup.go (GORM bootstrap + migration) into a
// single cohesive package so all other internal packages have one import path.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	gomigrate "github.com/golang-migrate/migrate/v4"

	appmigrations "mycourse-io-be/migrations"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
)

// primary is the application-wide GORM connection; set once on startup.
var primary *gorm.DB

// Set stores the shared GORM DB after Setup(). Call once at process startup.
func Set(db *gorm.DB) { primary = db }

// Conn returns the primary DB handle, or nil if Set has not been called.
func Conn() *gorm.DB { return primary }

// Setup opens the GORM connection using setting.DatabaseSetting and stores it
// in the package-level primary handle.  Call once during bootstrap.
func Setup() error {
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN: set URL or Host+Name (and User as needed)")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	primary = db
	return nil
}

// StdDB returns the shared PostgreSQL pool used by GORM, for database/sql raw queries.
func StdDB() (*sql.DB, error) {
	if primary == nil {
		return nil, errors.New("database not initialized")
	}
	return primary.DB()
}

// MigrateDatabase applies pending migrations using the embedded migration files.
func MigrateDatabase() error {
	if primary == nil {
		return errors.New("database not initialized")
	}
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN")
	}
	return migrateUp(context.Background(), dsn, appmigrations.Files, ".")
}

func migrateUp(ctx context.Context, dsn string, fsys fs.FS, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rawDB, err := openPingPostgres(ctx, dsn)
	if err != nil {
		return err
	}
	defer func() { _ = rawDB.Close() }()
	return migrateUpFromIOFS(rawDB, fsys, dir)
}

func openPingPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	rawDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf(constants.MsgMigrationDBOpen, err)
	}
	if err := rawDB.PingContext(ctx); err != nil {
		_ = rawDB.Close()
		return nil, fmt.Errorf(constants.MsgMigrationDBPing, err)
	}
	return rawDB, nil
}

func migrateUpFromIOFS(rawDB *sql.DB, fsys fs.FS, dir string) error {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return fmt.Errorf(constants.MsgMigrationSource, err)
	}
	driver, err := postgresmigrate.WithInstance(rawDB, &postgresmigrate.Config{
		MultiStatementEnabled: true,
	})
	if err != nil {
		return fmt.Errorf(constants.MsgMigrationPostgresDriver, err)
	}
	m, err := gomigrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf(constants.MsgMigrationRun, err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return err
	}
	return nil
}
