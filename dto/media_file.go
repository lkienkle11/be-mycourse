package dto

type FileFilter struct {
	BaseFilter
	Provider *string `form:"provider" binding:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	Kind     *string `form:"kind" binding:"omitempty,oneof=FILE VIDEO"`
}

type CreateFileRequest struct {
	Kind      string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	ObjectKey string         `json:"object_key" validate:"omitempty,max=512"`
	Metadata  map[string]any `json:"metadata" validate:"omitempty"`
}

type UpdateFileRequest struct {
	Kind                  string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	Metadata              map[string]any `json:"metadata" validate:"omitempty"`
	ReuseMediaID          string         `json:"reuse_media_id"`
	ExpectedRowVersion    *int64         `json:"expected_row_version"`
	SkipUploadIfUnchanged bool           `json:"skip_upload_if_unchanged"`
}

type UploadFileResponse struct {
	ID                 string             `json:"id,omitempty"`
	Kind               string             `json:"kind,omitempty"`
	Filename           string             `json:"filename,omitempty"`
	MimeType           string             `json:"mime_type,omitempty"`
	SizeBytes          int64              `json:"size_bytes,omitempty"`
	Status             string             `json:"status,omitempty"`
	B2BucketName       string             `json:"b2_bucket_name,omitempty"`
	URL                string             `json:"url"`
	OriginURL          string             `json:"origin_url"`
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

type VideoStatusResponse struct {
	Status string `json:"status"`
}
