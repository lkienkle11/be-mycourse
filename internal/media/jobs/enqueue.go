package jobs

import (
	"context"
	"strings"

	"mycourse-io-be/internal/media/domain" //nolint:depguard // jobs use domain repository interfaces and entity types as parameter types
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

// EnqueueOrphanImageCleanup schedules deferred cloud-object deletion for a raw URL or object key.
// Resolution: DB lookup first, then infra-level URL parsing.
// Returns true when a pending-cleanup row was inserted.
func EnqueueOrphanImageCleanupByURL(
	ctx context.Context,
	fileRepo domain.FileRepository,
	cleanupRepo domain.PendingCleanupRepository,
	imageURL string,
) bool {
	url := strings.TrimSpace(imageURL)
	if url == "" {
		return false
	}
	var prov, objectKey, bunnyVideoID string
	if row, err := fileRepo.GetByObjectKey(ctx, url); err == nil && row != nil {
		prov = row.Provider
		objectKey = row.ObjectKey
		bunnyVideoID = row.BunnyVideoID
	}
	if prov == "" {
		return false
	}
	if prov == constants.FileProviderLocal {
		return false
	}
	if strings.TrimSpace(objectKey) == "" && strings.TrimSpace(bunnyVideoID) == "" {
		return false
	}
	_ = cleanupRepo.Create(ctx, &domain.MediaPendingCloudCleanup{
		Provider:     prov,
		ObjectKey:    strings.TrimSpace(objectKey),
		BunnyVideoID: strings.TrimSpace(bunnyVideoID),
	})
	return true
}
