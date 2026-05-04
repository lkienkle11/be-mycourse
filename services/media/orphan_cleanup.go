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
