package dbmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	postgresmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func openPingPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("migration db open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migration db ping: %w", err)
	}
	return db, nil
}

func migrateUpFromIOFS(db *sql.DB, fsys fs.FS, dir string) error {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := postgresmigrate.WithInstance(db, &postgresmigrate.Config{
		MultiStatementEnabled: true,
	})
	if err != nil {
		return fmt.Errorf("migration postgres driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Up applies pending migrations from fsys at dir (e.g. ".") using golang-migrate.
// It uses a short-lived *sql.DB so Close() does not shut down the app’s GORM pool.
func Up(ctx context.Context, dsn string, fsys fs.FS, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db, err := openPingPostgres(ctx, dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return migrateUpFromIOFS(db, fsys, dir)
}
