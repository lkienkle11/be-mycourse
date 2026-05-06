package mapping

import (
	"strings"

	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
)

// ToMediaFilePublicFromModel maps a loaded media_files row to the public DTO shape.
func ToMediaFilePublicFromModel(row *models.MediaFile) *dto.MediaFilePublic {
	if row == nil {
		return nil
	}
	ent := ToMediaEntity(*row)
	return ToMediaFilePublicFromEntity(&ent)
}

// ToMediaFilePublicFromEntity maps entities.File to the API-safe media object (no origin_url).
func ToMediaFilePublicFromEntity(ent *entities.File) *dto.MediaFilePublic {
	if ent == nil {
		return nil
	}
	return &dto.MediaFilePublic{
		ID:                 ent.ID,
		Kind:               string(ent.Kind),
		Provider:           string(ent.Provider),
		Filename:           ent.Filename,
		MimeType:           ent.MimeType,
		SizeBytes:          ent.SizeBytes,
		Width:              ent.Metadata.WidthBytes,
		Height:             ent.Metadata.HeightBytes,
		URL:                ent.URL,
		Duration:           ent.Duration,
		ContentFingerprint: ent.ContentFingerprint,
		Status:             string(ent.Status),
	}
}

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
