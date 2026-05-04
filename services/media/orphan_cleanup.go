package media

import (
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	pkgmedia "mycourse-io-be/pkg/media"
	mediarepo "mycourse-io-be/repository/media"
)

// EnqueueOrphanImageCleanup schedules deferred cloud-object deletion for imageURL.
//
// Resolution order:
//  1. DB lookup via media_files (url / origin_url) — uses stored provider/key when found.
//  2. URL-pattern fallback via pkg/media.ParseImageURLForOrphanCleanup (runtime MediaSetting).
//
// Returns true when a pending-cleanup row was inserted.
// No-op (false) for: empty URL, Local provider, external/unrecognised URL, DB insert error.
//
// Compensation rule: call AFTER the owning DB record has been committed.
// If enqueue fails, the cloud object is stranded until a later sweep — acceptable,
// since no user-visible data is lost.
//
// Future domains (course cover_image, user avatar, lesson JSONB images) MUST call
// this function after their own DB delete/update commits.
func orphanCleanupResolveTargets(repo *mediarepo.FileRepository, url string) (
	prov constants.FileProvider,
	objectKey, bunnyVideoID string,
	ok bool,
) {
	if row, err := repo.GetByURL(url); err == nil && row != nil {
		return row.Provider, row.ObjectKey, row.BunnyVideoID, true
	}
	var parsedOK bool
	prov, objectKey, bunnyVideoID, parsedOK = pkgmedia.ParseImageURLForOrphanCleanup(url)
	if !parsedOK {
		return prov, objectKey, bunnyVideoID, false
	}
	return prov, objectKey, bunnyVideoID, true
}

func EnqueueOrphanImageCleanup(imageURL string) bool {
	url := strings.TrimSpace(imageURL)
	if url == "" {
		return false
	}
	repo := mediaRepository()
	prov, objectKey, bunnyVideoID, resolved := orphanCleanupResolveTargets(repo, url)
	if !resolved {
		return false
	}
	if prov == constants.FileProviderLocal {
		return false
	}
	if strings.TrimSpace(objectKey) == "" && strings.TrimSpace(bunnyVideoID) == "" {
		return false
	}
	row := &models.MediaPendingCloudCleanup{
		Provider:     prov,
		ObjectKey:    strings.TrimSpace(objectKey),
		BunnyVideoID: strings.TrimSpace(bunnyVideoID),
	}
	return repo.InsertPendingCleanup(row) == nil
}

// enqueuePendingCleanupFromMediaRow schedules cloud deletion from a concrete media_files row.
func enqueuePendingCleanupFromMediaRow(row *models.MediaFile) bool {
	if row == nil {
		return false
	}
	if row.Provider == constants.FileProviderLocal {
		return false
	}
	key := strings.TrimSpace(row.ObjectKey)
	bid := strings.TrimSpace(row.BunnyVideoID)
	if key == "" && bid == "" {
		return false
	}
	pending := &models.MediaPendingCloudCleanup{
		Provider:     row.Provider,
		ObjectKey:    key,
		BunnyVideoID: bid,
	}
	return mediaRepository().InsertPendingCleanup(pending) == nil
}

// EnqueueOrphanCleanupForMediaFileRow schedules deferred cleanup from an in-memory media row
// (used after DB unlink when the row is still readable).
func EnqueueOrphanCleanupForMediaFileRow(row *models.MediaFile) bool {
	return enqueuePendingCleanupFromMediaRow(row)
}

// EnqueueOrphanCleanupForMediaFileID loads media_files by id and enqueues cloud cleanup.
// Safe to call after the referencing user/category row is removed; idempotent inserts are acceptable.
func EnqueueOrphanCleanupForMediaFileID(fileID string) bool {
	id := strings.TrimSpace(fileID)
	if id == "" {
		return false
	}
	row, err := mediaRepository().GetByID(id)
	if err != nil || row == nil {
		return false
	}
	return enqueuePendingCleanupFromMediaRow(row)
}
