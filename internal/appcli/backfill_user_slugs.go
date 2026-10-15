package appcli

import (
	"context"
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"

	authinfra "mycourse-io-be/internal/auth/infra"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/parsebool"
)

// userSlugBackfiller assigns slugs to users that have none; satisfied by
// authinfra.GormUserRepository.
type userSlugBackfiller interface {
	BackfillMissingSlugs(ctx context.Context, batchSize int) (int, error)
}

// MaybeRunBackfillUserSlugs returns true when CLI_BACKFILL_USER_SLUGS mode is
// handled and the process should exit. It assigns a slug to every user whose
// users.slug is NULL (soft-deleted included) with the same generation rules
// used at registration, and is safe to run repeatedly. Run it once per
// environment before applying the NOT NULL migration 000044.
func MaybeRunBackfillUserSlugs(db *gorm.DB) bool {
	if !parsebool.EnvEnabled("CLI_BACKFILL_USER_SLUGS") {
		return false
	}
	if err := guardCLIOperation(context.Background(), constants.CLIOpBackfillUserSlugs); err != nil {
		printCLIGuardFailure(err)
		return true
	}
	runBackfillUserSlugs(context.Background(), authinfra.NewGormUserRepository(db), os.Stdout, os.Stderr)
	return true
}

func runBackfillUserSlugs(ctx context.Context, repo userSlugBackfiller, stdout, stderr *os.File) {
	start := time.Now()
	updated, err := repo.BackfillMissingSlugs(ctx, authinfra.DefaultSlugBackfillBatchSize)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Failure: user slug backfill failed after updating %d users (%v).\n", updated, err)
		return
	}
	_, _ = fmt.Fprintf(stdout, "User slug backfill completed in %s. updated_users=%d\n",
		time.Since(start).Round(time.Millisecond), updated)
}
