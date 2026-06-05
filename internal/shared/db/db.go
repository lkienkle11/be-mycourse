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
	"path/filepath"
	"strconv"
	"strings"

	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	gomigrate "github.com/golang-migrate/migrate/v4"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
	appmigrations "mycourse-io-be/migrations"
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

// MigrateDatabaseDownByFile rolls back one specific migration file by moving DB
// version to the immediate previous version of that file.
func MigrateDatabaseDownByFile(migrationVersionFile string) error {
	if primary == nil {
		return errors.New("database not initialized")
	}
	dsn := setting.DatabaseSetting.PostgresDSN()
	if dsn == "" {
		return errors.New("missing [database] DSN")
	}
	targetVersion, err := targetDownVersionFromFileName(migrationVersionFile)
	if err != nil {
		return err
	}
	return migrateDownTo(context.Background(), dsn, appmigrations.Files, ".", targetVersion)
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
	m, err := newMigratorFromIOFS(rawDB, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return err
	}
	return nil
}

func migrateDownTo(ctx context.Context, dsn string, fsys fs.FS, dir string, targetVersion uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rawDB, err := openPingPostgres(ctx, dsn)
	if err != nil {
		return err
	}
	defer func() { _ = rawDB.Close() }()
	return migrateDownToFromIOFS(rawDB, fsys, dir, targetVersion)
}

func migrateDownToFromIOFS(rawDB *sql.DB, fsys fs.FS, dir string, targetVersion uint) error {
	m, err := newMigratorFromIOFS(rawDB, fsys, dir)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	currentVersion, dirty, err := migrationVersion(m)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("cannot run down migration on dirty schema_migrations (current version=%d)", currentVersion)
	}
	if currentVersion <= targetVersion {
		return fmt.Errorf("down-only migration refused: current version=%d is not above target=%d", currentVersion, targetVersion)
	}
	if err := m.Migrate(targetVersion); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return err
	}
	return nil
}

func newMigratorFromIOFS(rawDB *sql.DB, fsys fs.FS, dir string) (*gomigrate.Migrate, error) {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf(constants.MsgMigrationSource, err)
	}
	driver, err := postgresmigrate.WithInstance(rawDB, &postgresmigrate.Config{
		MultiStatementEnabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf(constants.MsgMigrationPostgresDriver, err)
	}
	m, err := gomigrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf(constants.MsgMigrationRun, err)
	}
	return m, nil
}

func migrationVersion(m *gomigrate.Migrate) (uint, bool, error) {
	version, dirty, err := m.Version()
	if errors.Is(err, gomigrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read schema_migrations version: %w", err)
	}
	return version, dirty, nil
}

func targetDownVersionFromFileName(migrationVersionFile string) (uint, error) {
	base := filepath.Base(strings.TrimSpace(migrationVersionFile))
	if base == "" || base == "." {
		return 0, errors.New("MIGRATE_VERSION_FILE is required when MIGRATE=2")
	}
	if !strings.HasSuffix(base, ".down.sql") {
		return 0, fmt.Errorf("MIGRATE_VERSION_FILE must point to a .down.sql file, got %q", base)
	}
	versionToken, _, ok := strings.Cut(base, "_")
	if !ok || versionToken == "" {
		return 0, fmt.Errorf("invalid migration file name %q (expected NNNNNN_name.down.sql)", base)
	}
	versionNumber, err := strconv.ParseUint(versionToken, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %q: %w", base, err)
	}
	if versionNumber == 0 {
		return 0, fmt.Errorf("invalid migration version in %q: must be greater than zero", base)
	}
	return uint(versionNumber - 1), nil
}
