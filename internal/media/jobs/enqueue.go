package jobs

import (
	"context"
	"strings"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
)

// OrphanEnqueuer implements application.OrphanCleanupEnqueuer using domain repos.
type OrphanEnqueuer struct {
	cleanupRepo domain.PendingCleanupRepository
}

// NewOrphanEnqueuer creates an OrphanEnqueuer.
func NewOrphanEnqueuer(r domain.PendingCleanupRepository) *OrphanEnqueuer {
	return &OrphanEnqueuer{cleanupRepo: r}
}

// EnqueueSupersededPendingCleanup inserts a pending-cleanup row for a replaced/superseded cloud object.
func (e *OrphanEnqueuer) EnqueueSupersededPendingCleanup(objectKey, provider, bunnyVideoID string) {
	key := strings.TrimSpace(objectKey)
	bid := strings.TrimSpace(bunnyVideoID)
	if provider == constants.FileProviderLocal {
		return
	}
	if key == "" && bid == "" {
		return
	}
	_ = e.cleanupRepo.Create(context.Background(), &domain.MediaPendingCloudCleanup{
		Provider:     provider,
		ObjectKey:    key,
		BunnyVideoID: bid,
	})
}

func enqueueOrphanFromFileRow(ctx context.Context, cleanupRepo domain.PendingCleanupRepository, row *domain.File) bool {
	if row == nil {
		return false
	}
	prov := strings.TrimSpace(row.Provider)
	objectKey := strings.TrimSpace(row.ObjectKey)
	bunnyVideoID := strings.TrimSpace(row.BunnyVideoID)
	if prov == "" || prov == constants.FileProviderLocal {
		return false
	}
	if objectKey == "" && bunnyVideoID == "" {
		return false
	}
	_ = cleanupRepo.Create(ctx, &domain.MediaPendingCloudCleanup{
		Provider:     prov,
		ObjectKey:    objectKey,
		BunnyVideoID: bunnyVideoID,
	})
	return true
}

func enqueueOrphanCleanupFromResolvedRow(ctx context.Context, cleanupRepo domain.PendingCleanupRepository, row *domain.File, err error) bool {
	if err != nil || row == nil {
		return false
	}
	return enqueueOrphanFromFileRow(ctx, cleanupRepo, row)
}

// EnqueueOrphanCleanupForFileID schedules deferred cloud-object deletion for a media_files.id (UUID).
// Returns true when a pending-cleanup row was inserted.
func EnqueueOrphanCleanupForFileID(
	ctx context.Context,
	fileRepo domain.FileRepository,
	cleanupRepo domain.PendingCleanupRepository,
	fileID string,
) bool {
	id := strings.TrimSpace(fileID)
	if id == "" {
		return false
	}
	row, err := fileRepo.GetByID(ctx, id)
	return enqueueOrphanCleanupFromResolvedRow(ctx, cleanupRepo, row, err)
}

// EnqueueOrphanCleanupByObjectKey schedules deferred cloud-object deletion when objectKey matches media_files.object_key.
// Accepts object_key only — does not parse raw CDN or Bunny URLs. Returns true when a row was inserted.
func EnqueueOrphanCleanupByObjectKey(
	ctx context.Context,
	fileRepo domain.FileRepository,
	cleanupRepo domain.PendingCleanupRepository,
	objectKey string,
) bool {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return false
	}
	row, err := fileRepo.GetByObjectKey(ctx, key)
	return enqueueOrphanCleanupFromResolvedRow(ctx, cleanupRepo, row, err)
}
