package infra

import "strings"

// ProfileImageFileAcceptable returns true when a media_files row may be used as
// taxonomy cover art or a user avatar (non-video raster-friendly file kind).
func ProfileImageFileAcceptable(kind string, mimeType, filename string) bool {
	if strings.TrimSpace(kind) != "FILE" {
		return false
	}
	mt := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(mt, "image/") {
		return mt != "image/svg+xml"
	}
	ext := strings.ToLower(filename)
	for _, suf := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".bmp"} {
		if strings.HasSuffix(ext, suf) {
			return true
		}
	}
	return false
}
