// Package delivery contains the MEDIA bounded-context HTTP delivery layer.
package delivery

// FileFilterRequest is the query-param DTO for listing media files.
type FileFilterRequest struct {
	Page     int     `form:"page"`
	PerPage  int     `form:"per_page"`
	Provider *string `form:"provider" binding:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	Kind     *string `form:"kind" binding:"omitempty,oneof=FILE VIDEO"`
}

func (f FileFilterRequest) getPage() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}

func (f FileFilterRequest) getPerPage() int {
	if f.PerPage < 1 {
		return 20
	}
	if f.PerPage > 100 {
		return 100
	}
	return f.PerPage
}

// CreateFileRequest is the multipart/form-data request DTO for file creation.
type CreateFileRequest struct {
	Kind      string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	ObjectKey string         `json:"object_key" validate:"omitempty,max=512"`
	Metadata  map[string]any `json:"metadata" validate:"omitempty"`
}

// UpdateFileRequest is the multipart/form-data request DTO for file update.
type UpdateFileRequest struct {
	Kind                  string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	Metadata              map[string]any `json:"metadata" validate:"omitempty"`
	ReuseMediaID          string         `json:"reuse_media_id"`
	ExpectedRowVersion    *int64         `json:"expected_row_version"`
	SkipUploadIfUnchanged bool           `json:"skip_upload_if_unchanged"`
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
type UploadFileResponse struct {
	ID                 string             `json:"id,omitempty"`
	Kind               string             `json:"kind,omitempty"`
	Filename           string             `json:"filename,omitempty"`
	MimeType           string             `json:"mime_type,omitempty"`
	SizeBytes          int64              `json:"size_bytes,omitempty"`
	Status             string             `json:"status,omitempty"`
	B2BucketName       string             `json:"b2_bucket_name,omitempty"`
	URL                string             `json:"url"`
	ObjectKey          string             `json:"object_key"`
	BunnyVideoID       string             `json:"bunny_video_id,omitempty"`
	BunnyLibraryID     string             `json:"bunny_library_id,omitempty"`
	VideoID            string             `json:"video_id,omitempty"`
	ThumbnailURL       string             `json:"thumbnail_url,omitempty"`
	EmbededHTML        string             `json:"embeded_html,omitempty"`
	Duration           int64              `json:"duration,omitempty"`
	VideoProvider      string             `json:"video_provider,omitempty"`
	Metadata           UploadFileMetadata `json:"metadata"`
	RowVersion         int64              `json:"row_version,omitempty"`
	ContentFingerprint string             `json:"content_fingerprint,omitempty"`
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
