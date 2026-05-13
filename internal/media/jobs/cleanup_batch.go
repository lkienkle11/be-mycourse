package jobs

import (
	"context"
	"time"

	"mycourse-io-be/internal/media/domain"           //nolint:depguard // jobs use domain repository interfaces as parameters; no business logic
	mediainfra "mycourse-io-be/internal/media/infra" //nolint:depguard // jobs call infra.RequireInitialized and cloud client APIs; TODO: inject via application service
)

// ProcessPendingCleanupBatch executes one batch of deferred cloud deletes from media_pending_cloud_cleanup.
func ProcessPendingCleanupBatch(ctx context.Context, cleanupRepo domain.PendingCleanupRepository) {
	if err := mediainfra.RequireInitialized(mediainfra.Cloud); err != nil {
		return
	}
	rows, err := cleanupRepo.FindPending(ctx, MediaCleanupBatchSize)
	if err != nil || len(rows) == 0 {
		return
	}
	clients := mediainfra.Cloud
	for _, row := range rows {
		delErr := mediainfra.DeleteStoredObject(ctx, clients, row.ObjectKey, row.Provider, row.BunnyVideoID)
		next := row.AttemptCount + 1
		if delErr == nil {
			_ = cleanupRepo.MarkDone(ctx, row.ID)
			CleanupCloudDeleted.Add(1)
			continue
		}
		if next >= MediaCleanupMaxAttempts {
			_ = cleanupRepo.MarkFailed(ctx, row.ID, delErr.Error(), nil)
			CleanupCloudFailed.Add(1)
			continue
		}
		backoff := time.Duration(next*next) * time.Minute
		if backoff > 30*time.Minute {
			backoff = 30 * time.Minute
		}
		nextRunAt := time.Now().Add(backoff)
		_ = cleanupRepo.MarkFailed(ctx, row.ID, delErr.Error(), nextRunAt)
		CleanupCloudRetried.Add(1)
	}
}
