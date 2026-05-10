package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/pkg/entities"
)

func toUploadMetadataDTO(meta entities.UploadFileMetadata) dto.UploadFileMetadata {
	return dto.UploadFileMetadata{
		SizeBytes:       meta.SizeBytes,
		WidthBytes:      meta.WidthBytes,
		HeightBytes:     meta.HeightBytes,
		MimeType:        meta.MimeType,
		Extension:       meta.Extension,
		DurationSeconds: meta.DurationSeconds,
		Bitrate:         meta.Bitrate,
		FPS:             meta.FPS,
		VideoCodec:      meta.VideoCodec,
		AudioCodec:      meta.AudioCodec,
		HasAudio:        meta.HasAudio,
		IsHDR:           meta.IsHDR,
		PageCount:       meta.PageCount,
		HasPassword:     meta.HasPassword,
		ArchiveEntries:  meta.ArchiveEntries,
	}
}

func ToUploadFileResponse(file entities.File) dto.UploadFileResponse {
	// Sub 12: canonical/origin URL is not part of dto.UploadFileResponse (no origin_url in public JSON).
	// entities.File / DB still carry OriginURL for persistence, orphan lookup, and cloud delete.
	return dto.UploadFileResponse{
		ID:                 file.ID,
		Kind:               string(file.Kind),
		Filename:           file.Filename,
		MimeType:           file.MimeType,
		SizeBytes:          file.SizeBytes,
		Status:             string(file.Status),
		B2BucketName:       file.B2BucketName,
		URL:                file.URL,
		ObjectKey:          file.ObjectKey,
		BunnyVideoID:       file.BunnyVideoID,
		BunnyLibraryID:     file.BunnyLibraryID,
		VideoID:            file.VideoID,
		ThumbnailURL:       file.ThumbnailURL,
		EmbededHTML:        file.EmbededHTML,
		Duration:           file.Duration,
		VideoProvider:      file.VideoProvider,
		Metadata:           toUploadMetadataDTO(file.Metadata),
		RowVersion:         file.RowVersion,
		ContentFingerprint: file.ContentFingerprint,
	}
}

func ToUploadFileResponses(files []entities.File) []dto.UploadFileResponse {
	out := make([]dto.UploadFileResponse, 0, len(files))
	for _, item := range files {
		out = append(out, ToUploadFileResponse(item))
	}
	return out
}

// ToUploadFileResponsesFromPointers maps non-nil entity pointers to upload DTOs (API response envelope).
func ToUploadFileResponsesFromPointers(files []*entities.File) []dto.UploadFileResponse {
	out := make([]dto.UploadFileResponse, 0, len(files))
	for _, p := range files {
		if p == nil {
			continue
		}
		out = append(out, ToUploadFileResponse(*p))
	}
	return out
}

// ToBatchDeleteMediaFilesResponse builds the batch-delete success payload.
func ToBatchDeleteMediaFilesResponse(deletedCount int) dto.BatchDeleteMediaFilesResponse {
	return dto.BatchDeleteMediaFilesResponse{DeletedCount: deletedCount}
}

// ToLocalURLDecodeResponse maps a decoded object key to the public DTO.
func ToLocalURLDecodeResponse(objectKey string) dto.LocalURLDecodeResponse {
	return dto.LocalURLDecodeResponse{ObjectKey: objectKey}
}

// ToVideoStatusResponse maps a Bunny/video pipeline status string to the public DTO.
func ToVideoStatusResponse(status string) dto.VideoStatusResponse {
	return dto.VideoStatusResponse{Status: status}
}

// ToMediaCleanupMetricsResponse maps worker counters to the public metrics DTO.
func ToMediaCleanupMetricsResponse(deleted, failed, retried uint64) dto.MediaCleanupMetricsResponse {
	return dto.MediaCleanupMetricsResponse{
		CleanupCloudDeleted: deleted,
		CleanupCloudFailed:  failed,
		CleanupCloudRetried: retried,
	}
}
