package helper

import (
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/logic/utils"
	"mycourse-io-be/pkg/setting"
)

// ParseImageURLForOrphanCleanup inspects a stored image URL and returns the
// provider + storage keys if the URL is a system-managed cloud asset.
//
// Resolution order (first match wins):
//  1. Empty → not ok.
//  2. /api/v1/media/files/local/… → provider=Local, ok=true (no remote delete needed).
//  3. Bunny Stream URL matching BunnyStreamBaseURL + library-id prefix →
//     provider=Bunny, objectKey=GUID, bunnyVideoID=GUID.
//  4. B2/CDN URL matching GcoreCDNURL + B2Bucket prefix →
//     provider=B2, objectKey=object key after bucket segment.
//  5. Other (external URL) → not ok.
//
// Pure function: no I/O, reads only from pkg/setting (already loaded at startup).
func ParseImageURLForOrphanCleanup(rawURL string) (
	provider constants.FileProvider,
	objectKey string,
	bunnyVideoID string,
	ok bool,
) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return
	}

	if strings.HasPrefix(u, "/api/v1/media/files/local/") {
		provider = constants.FileProviderLocal
		ok = true
		return
	}

	bunnyBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "")
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	if bunnyBase != "" && libraryID != "" {
		prefix := bunnyBase + "/" + libraryID + "/"
		if strings.HasPrefix(u, prefix) {
			remainder := strings.SplitN(u, "?", 2)[0]
			remainder = strings.SplitN(remainder, "#", 2)[0]
			remainder = strings.TrimPrefix(remainder, prefix)
			guid := strings.TrimSpace(strings.SplitN(remainder, "/", 2)[0])
			if guid != "" {
				provider = constants.FileProviderBunny
				objectKey = guid
				bunnyVideoID = guid
				ok = true
				return
			}
		}
	}

	cdnBase := utils.NormalizeBaseURL(setting.MediaSetting.GcoreCDNURL, "")
	bucket := strings.TrimSpace(setting.MediaSetting.B2Bucket)
	if cdnBase != "" && bucket != "" {
		prefix := utils.JoinURLPathSegments(cdnBase, bucket) + "/"
		if strings.HasPrefix(u, prefix) {
			remainder := strings.TrimPrefix(u, prefix)
			remainder = strings.SplitN(remainder, "?", 2)[0]
			remainder = strings.SplitN(remainder, "#", 2)[0]
			key := strings.TrimSpace(remainder)
			if key != "" {
				provider = constants.FileProviderB2
				objectKey = key
				ok = true
				return
			}
		}
	}

	return
}
