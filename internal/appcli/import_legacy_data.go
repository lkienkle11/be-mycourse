package appcli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/parsebool"
)

// MaybeRunImportLegacyData returns true when CLI import mode is handled and process should exit.
func MaybeRunImportLegacyData(db *gorm.DB) bool {
	if !parsebool.EnvEnabled("CLI_IMPORT_LEGACY_DATA") {
		return false
	}
	runLegacyImport(db)
	return true
}

func runLegacyImport(db *gorm.DB) {
	if err := guardCLIOperation(context.Background(), constants.CLIOpLegacyImport); err != nil {
		printCLIGuardFailure(err)
		return
	}

	dumpPath := strings.TrimSpace(os.Getenv("CLI_IMPORT_LEGACY_DATA_DUMP"))
	if dumpPath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Failure: CLI_IMPORT_LEGACY_DATA_DUMP is required (absolute path to backup-*.sql).")
		return
	}

	start := time.Now()
	report, err := importLegacyDump(context.Background(), db, dumpPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failure: legacy import failed (%v).\n", err)
		return
	}

	_, _ = fmt.Fprintf(os.Stdout,
		"Legacy import completed in %s. statements=%d imported_rows=%d skipped_rows=%d idmap=%s\n",
		time.Since(start).Round(time.Millisecond),
		report.Statements,
		report.ImportedRows,
		report.SkippedRows,
		report.IDMapPath,
	)
}
