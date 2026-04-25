package entities

import (
	"time"

	"mycourse-io-be/constants"
)

type FileMetadata map[string]any

type File struct {
	ID        string                 `json:"id"`
	Kind      constants.FileKind     `json:"kind"`
	Provider  constants.FileProvider `json:"provider"`
	Filename  string                 `json:"filename"`
	MimeType  string                 `json:"mime_type"`
	SizeBytes int64                  `json:"size_bytes"`
	URL       string                 `json:"url"`
	OriginURL string                 `json:"origin_url"`
	ObjectKey string                 `json:"object_key"`
	Status    constants.FileStatus   `json:"status"`
	Metadata  FileMetadata           `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
