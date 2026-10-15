package gormx

import (
	"context"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type hardDeleteRow struct {
	ID string `gorm:"primaryKey"`
}

// A dry-run handle plus a callback that records the built statement proves the
// helper binds the UUID as a parameter, without needing a database.
func TestHardDeleteByIDBindsIDAsParameter(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=127.0.0.1 port=1 user=x dbname=x sslmode=disable", PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Skipf("cannot build a dry-run gorm handle: %v", err)
	}
	var gotSQL string
	var gotVars []any
	if err := db.Callback().Delete().After("gorm:delete").Register("test:capture", func(tx *gorm.DB) {
		gotSQL = tx.Statement.SQL.String()
		gotVars = tx.Statement.Vars
	}); err != nil {
		t.Fatal(err)
	}

	const id = "01a13f01-97b3-7cb9-a193-a9a561082ea4"
	if err := HardDeleteByID(context.Background(), db, &hardDeleteRow{}, id); err != nil {
		t.Fatalf("HardDeleteByID() err = %v", err)
	}
	if want := `DELETE FROM "hard_delete_rows" WHERE id = $1`; gotSQL != want {
		t.Fatalf("SQL = %q, want %q", gotSQL, want)
	}
	if len(gotVars) != 1 || gotVars[0] != id {
		t.Fatalf("vars = %v, want [%s]", gotVars, id)
	}
}
