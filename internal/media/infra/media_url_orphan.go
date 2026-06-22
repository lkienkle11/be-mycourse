package infra

import (
	"strings"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/utils"
)

// ParseImageURLForOrphanCleanup inspects a stored image URL and returns the
// provider + storage keys if the URL is a system-managed cloud asset.
//
// Resolution order (first match wins):
//  1. Empty → not ok.
//  2. /api/v1/media/files/local/… → provider=Local, ok=true (no remote delete needed).
//  3. Bunny Stream URL matching BunnyStreamBaseURL + library-id prefix →
//     provider=Bunny, objectKey=GUID, bunnyVideoID=GUID.
//  4. R2 URL matching R2PublicURL prefix → provider=R2, objectKey=remainder.
//  5. Other (external URL) → not ok.
//
// Pure function: no I/O, reads only from pkg/setting (already loaded at startup).
func ParseImageURLForOrphanCleanup(rawURL string) (
	provider string,
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

	if p, key, bid, hit := orphanCleanupBunnyMatch(u); hit {
		return p, key, bid, true
	}
	if p, key, hit := orphanCleanupR2Match(u); hit {
		return p, key, "", true
	}

	return
}

func orphanCleanupBunnyMatch(u string) (string, string, string, bool) {
	bunnyBase := utils.NormalizeBaseURL(setting.MediaSetting.BunnyStreamBaseURL, "")
	libraryID := strings.TrimSpace(setting.MediaSetting.BunnyStreamLibraryID)
	if bunnyBase == "" || libraryID == "" {
		return string(""), "", "", false
	}
	prefix := bunnyBase + "/" + libraryID + "/"
	if !strings.HasPrefix(u, prefix) {
		return string(""), "", "", false
	}
	remainder := strings.SplitN(u, "?", 2)[0]
	remainder = strings.SplitN(remainder, "#", 2)[0]
	remainder = strings.TrimPrefix(remainder, prefix)
	guid := strings.TrimSpace(strings.SplitN(remainder, "/", 2)[0])
	if guid == "" {
		return string(""), "", "", false
	}
	return constants.FileProviderBunny, guid, guid, true
}

func orphanCleanupR2Match(u string) (string, string, bool) {
	publicBase := utils.NormalizeBaseURL(setting.MediaSetting.R2.PublicURL, "")
	if publicBase == "" {
		return "", "", false
	}
	prefix := publicBase + "/"
	if !strings.HasPrefix(u, prefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(u, prefix)
	remainder = strings.SplitN(remainder, "?", 2)[0]
	remainder = strings.SplitN(remainder, "#", 2)[0]
	key := strings.TrimSpace(remainder)
	if key == "" {
		return "", "", false
	}
	return constants.FileProviderR2, key, true
}
