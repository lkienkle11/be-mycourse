package dto

import "mycourse-io-be/pkg/entities"

type FileFilter struct {
	BaseFilter
	Provider *string `form:"provider" binding:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	Kind     *string `form:"kind" binding:"omitempty,oneof=FILE VIDEO"`
}

type CreateFileRequest struct {
	Kind      string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	Provider  string         `json:"provider" validate:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	ObjectKey string         `json:"object_key" validate:"omitempty,max=512"`
	Metadata  map[string]any `json:"metadata" validate:"omitempty"`
}

type UpdateFileRequest struct {
	Kind     string         `json:"kind" validate:"omitempty,oneof=FILE VIDEO"`
	Provider string         `json:"provider" validate:"omitempty,oneof=S3 GCS B2 R2 Bunny Local"`
	Metadata map[string]any `json:"metadata" validate:"omitempty"`
}

type UploadFileResponse struct {
	URL       string                `json:"url"`
	OriginURL string                `json:"origin_url"`
	ObjectKey string                `json:"object_key"`
	Provider  string                `json:"provider"`
	Metadata  entities.FileMetadata `json:"metadata"`
}
