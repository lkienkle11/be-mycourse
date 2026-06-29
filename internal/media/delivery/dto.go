// Package delivery contains the MEDIA bounded-context HTTP delivery layer.
package delivery

import "mycourse-io-be/internal/shared/utils"

// FileFilterRequest is the query-param DTO for listing media files.
type FileFilterRequest struct {
	utils.BaseFilter
	Search   string  `form:"search"`
	Provider *string `form:"provider" binding:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	Kind     *string `form:"kind" binding:"omitempty,oneof=FILE VIDEO"`
	SortBy   string  `form:"sort_by" binding:"omitempty,oneof=created_at updated_at filename size_bytes"`
	Category *string `form:"category" binding:"omitempty,oneof=image document video"`
}

// BatchDeleteMediaFilesRequest carries the batch delete body.
type BatchDeleteMediaFilesRequest struct {
	ObjectKeys []string `json:"object_keys" binding:"required"`
}

// BatchDeleteMediaFilesResponse is returned after a successful batch delete.
type BatchDeleteMediaFilesResponse struct {
	DeletedCount int `json:"deleted_count"`
}

// LocalURLDecodeResponse is returned by GET .../media/files/local/:token.
type LocalURLDecodeResponse struct {
	ObjectKey string `json:"object_key"`
}

// MediaCleanupMetricsResponse exposes orphan cleanup worker counters.
type MediaCleanupMetricsResponse struct {
	CleanupCloudDeleted uint64 `json:"cleanup_cloud_deleted"`
	CleanupCloudFailed  uint64 `json:"cleanup_cloud_failed"`
	CleanupCloudRetried uint64 `json:"cleanup_cloud_retried"`
}

// UploadFileResponse is the public API response for a file entity.
//
// `metadata` is the only metadata sub-object exposed to clients. It is a
// typed view rebuilt on every read from the underlying `metadata_json`
// JSONB column (see `rowToFile` in `internal/media/infra/repos.go`), so
// callers see a stable shape regardless of which provider wrote the row.
// The raw provider blob is kept server-side only and is not returned.
type UploadFileResponse struct {
	ID                  string             `json:"id,omitempty"`
	Kind                string             `json:"kind,omitempty"`
	Filename            string             `json:"filename,omitempty"`
	MimeType            string             `json:"mime_type,omitempty"`
	SizeBytes           int64              `json:"size_bytes,omitempty"`
	Status              string             `json:"status,omitempty"`
	R2BucketName        string             `json:"r2_bucket_name,omitempty"`
	URL                 string             `json:"url"`
	ObjectKey           string             `json:"object_key"`
	BunnyVideoID        string             `json:"bunny_video_id,omitempty"`
	BunnyLibraryID      string             `json:"bunny_library_id,omitempty"`
	VideoID             string             `json:"video_id,omitempty"`
	ThumbnailURL        string             `json:"thumbnail_url,omitempty"`
	EmbededHTML         string             `json:"embeded_html,omitempty"`
	DirectPlayURL       string             `json:"direct_play_url,omitempty"`
	HLSPlaylistURL      string             `json:"hls_playlist_url,omitempty"`
	PreviewAnimationURL string             `json:"preview_animation_url,omitempty"`
	Duration            int64              `json:"duration,omitempty"`
	VideoProvider       string             `json:"video_provider,omitempty"`
	Metadata            UploadFileMetadata `json:"metadata"`
	RowVersion          int64              `json:"row_version,omitempty"`
	ContentFingerprint  string             `json:"content_fingerprint,omitempty"`
	CreatedAt           int64              `json:"created_at,omitempty"`
	UpdatedAt           int64              `json:"updated_at,omitempty"`
}

// UploadFileMetadata is the metadata sub-object in UploadFileResponse.
type UploadFileMetadata struct {
	SizeBytes       int64   `json:"size_bytes"`
	WidthBytes      int     `json:"width_bytes"`
	HeightBytes     int     `json:"height_bytes"`
	MimeType        string  `json:"mime_type"`
	Extension       string  `json:"extension"`
	DurationSeconds float64 `json:"duration_seconds"`
	Bitrate         int     `json:"bitrate"`
	FPS             float64 `json:"fps"`
	VideoCodec      string  `json:"video_codec"`
	AudioCodec      string  `json:"audio_codec"`
	HasAudio        bool    `json:"has_audio"`
	IsHDR           bool    `json:"is_hdr"`
	PageCount       int     `json:"page_count"`
	HasPassword     bool    `json:"has_password"`
	ArchiveEntries  int     `json:"archive_entries"`
}

// VideoStatusResponse carries the Bunny Stream processing status.
type VideoStatusResponse struct {
	Status string `json:"status"`
}

// BunnyVideoWebhookRequest is the decoded Bunny Stream webhook JSON body.
type BunnyVideoWebhookRequest struct {
	VideoLibraryID int    `json:"VideoLibraryId" binding:"required"`
	VideoGUID      string `json:"VideoGuid" binding:"required"`
	Status         int    `json:"Status" binding:"required,min=0,max=10"`
}
