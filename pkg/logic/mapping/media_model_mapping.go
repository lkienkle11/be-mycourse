package mapping

import (
	"encoding/json"
	"fmt"
	"strings"

	"mycourse-io-be/constants"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/entities"
	pkgmedia "mycourse-io-be/pkg/media"
)

func uploadMetadataToRaw(meta entities.UploadFileMetadata) entities.RawMetadata {
	return entities.RawMetadata{
		"size":            meta.SizeBytes,
		"width":           meta.WidthBytes,
		"height":          meta.HeightBytes,
		"mime_type":       meta.MimeType,
		"extension":       meta.Extension,
		"duration":        meta.DurationSeconds,
		"bitrate":         meta.Bitrate,
		"fps":             meta.FPS,
		"video_codec":     meta.VideoCodec,
		"audio_codec":     meta.AudioCodec,
		"has_audio":       meta.HasAudio,
		"is_hdr":          meta.IsHDR,
		"page_count":      meta.PageCount,
		"has_password":    meta.HasPassword,
		"archive_entries": meta.ArchiveEntries,
	}
}

func mediaEntityVideoThumbEmbed(raw entities.RawMetadata, row models.MediaFile) (videoID, thumbnailURL, embededHTML string) {
	videoID = strings.TrimSpace(row.VideoID)
	if videoID == "" {
		videoID = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyVideoID]))
	}
	if videoID == "" {
		videoID = strings.TrimSpace(row.BunnyVideoID)
	}
	thumbnailURL = strings.TrimSpace(row.ThumbnailURL)
	if thumbnailURL == "" {
		thumbnailURL = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyThumbnailURL]))
	}
	embededHTML = strings.TrimSpace(row.EmbededHTML)
	if embededHTML == "" {
		embededHTML = strings.TrimSpace(fmt.Sprintf("%v", raw[constants.MediaMetaKeyEmbededHTML]))
	}
	return videoID, thumbnailURL, embededHTML
}

func ToMediaEntity(row models.MediaFile) entities.File {
	raw := entities.RawMetadata{}
	_ = json.Unmarshal(row.MetadataJSON, &raw)
	videoID, thumbnailURL, embededHTML := mediaEntityVideoThumbEmbed(raw, row)
	return entities.File{
		ID:                 row.ID,
		Kind:               row.Kind,
		Provider:           row.Provider,
		Filename:           row.Filename,
		MimeType:           row.MimeType,
		SizeBytes:          row.SizeBytes,
		URL:                row.URL,
		OriginURL:          row.OriginURL,
		ObjectKey:          row.ObjectKey,
		Status:             row.Status,
		B2BucketName:       row.B2BucketName,
		BunnyVideoID:       row.BunnyVideoID,
		BunnyLibraryID:     row.BunnyLibraryID,
		VideoID:            videoID,
		ThumbnailURL:       thumbnailURL,
		EmbededHTML:        embededHTML,
		Duration:           row.Duration,
		VideoProvider:      row.VideoProvider,
		RowVersion:         row.RowVersion,
		ContentFingerprint: row.ContentFingerprint,
		Metadata:           pkgmedia.BuildTypedMetadata(row.Kind, row.MimeType, row.Filename, row.SizeBytes, nil, raw),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func mediaModelMetadataMap(row entities.File) entities.RawMetadata {
	meta := pkgmedia.NormalizeMetadata(uploadMetadataToRaw(row.Metadata))
	if row.BunnyVideoID != "" {
		meta["bunny_video_id"] = row.BunnyVideoID
	}
	if row.BunnyLibraryID != "" {
		meta["bunny_library_id"] = row.BunnyLibraryID
	}
	if row.VideoProvider != "" {
		meta["video_provider"] = row.VideoProvider
	}
	if row.Duration > 0 {
		meta["duration"] = row.Duration
	}
	if row.VideoID != "" {
		meta[constants.MediaMetaKeyVideoID] = row.VideoID
	}
	if row.ThumbnailURL != "" {
		meta[constants.MediaMetaKeyThumbnailURL] = row.ThumbnailURL
	}
	if row.EmbededHTML != "" {
		meta[constants.MediaMetaKeyEmbededHTML] = row.EmbededHTML
	}
	return meta
}

func ToMediaModel(row entities.File) *models.MediaFile {
	blob, _ := json.Marshal(mediaModelMetadataMap(row))
	return &models.MediaFile{
		ID:                 row.ID,
		ObjectKey:          row.ObjectKey,
		Kind:               row.Kind,
		Provider:           row.Provider,
		Filename:           row.Filename,
		MimeType:           row.MimeType,
		SizeBytes:          row.SizeBytes,
		URL:                row.URL,
		OriginURL:          row.OriginURL,
		Status:             row.Status,
		B2BucketName:       row.B2BucketName,
		BunnyVideoID:       row.BunnyVideoID,
		BunnyLibraryID:     row.BunnyLibraryID,
		VideoID:            row.VideoID,
		ThumbnailURL:       row.ThumbnailURL,
		EmbededHTML:        row.EmbededHTML,
		Duration:           row.Duration,
		VideoProvider:      row.VideoProvider,
		RowVersion:         row.RowVersion,
		ContentFingerprint: row.ContentFingerprint,
		MetadataJSON:       blob,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
