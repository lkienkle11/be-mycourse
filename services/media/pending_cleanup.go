package media

import (
	"context"
	"time"

	"mycourse-io-be/constants"
	pkgmedia "mycourse-io-be/pkg/media"
	mediarepo "mycourse-io-be/repository/media"
)

func ProcessPendingCleanupBatch(ctx context.Context, repo *mediarepo.FileRepository) {
	if err := pkgmedia.RequireInitialized(pkgmedia.Cloud); err != nil {
		return
	}
	rows, err := repo.ListPendingCleanupDue(constants.MediaCleanupBatchSize)
	if err != nil || len(rows) == 0 {
		return
	}
	clients := pkgmedia.Cloud
	for _, row := range rows {
		delErr := pkgmedia.DeleteStoredObject(ctx, clients, row.ObjectKey, row.Provider, row.BunnyVideoID)
		next := row.AttemptCount + 1
		if delErr == nil {
			_ = repo.MarkPendingCleanupDone(row.ID)
			CleanupCloudDeleted.Add(1)
			continue
		}
		if next >= constants.MediaCleanupMaxAttempts {
			_ = repo.MarkPendingCleanupFailed(row.ID, delErr.Error())
			CleanupCloudFailed.Add(1)
			continue
		}
		backoff := time.Duration(next*next) * time.Minute
		if backoff > 30*time.Minute {
			backoff = 30 * time.Minute
		}
		_ = repo.SchedulePendingCleanupRetry(row.ID, delErr.Error(), next, time.Now().Add(backoff))
		CleanupCloudRetried.Add(1)
	}
}
