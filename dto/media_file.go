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
	Kind     string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	Metadata map[string]any `json:"metadata" validate:"omitempty"`
}

type UploadFileResponse struct {
	ID             string `json:"id,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Filename       string `json:"filename,omitempty"`
	MimeType       string `json:"mime_type,omitempty"`
	SizeBytes      int64  `json:"size_bytes,omitempty"`
	Status         string `json:"status,omitempty"`
	B2BucketName   string `json:"b2_bucket_name,omitempty"`
	URL            string `json:"url"`
	OriginURL      string `json:"origin_url"`
	ObjectKey      string `json:"object_key"`
	BunnyVideoID   string `json:"bunny_video_id,omitempty"`
	BunnyLibraryID string `json:"bunny_library_id,omitempty"`
	Duration       int64  `json:"duration,omitempty"`
	VideoProvider  string `json:"video_provider,omitempty"`
	Metadata       any    `json:"metadata"`
}

type VideoStatusResponse struct {
	Status string `json:"status"`
}
