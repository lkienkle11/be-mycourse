package entities

import (
	"time"
)

type ProviderUploadResult struct {
	URL       string
	OriginURL string
	ObjectKey string
	Metadata  RawMetadata
}

type MediaUploadEntityInput struct {
	Kind          string
	Provider      string
	Filename      string
	ContentType   string
	SizeBytes     int64
	Payload       []byte
	Uploaded      ProviderUploadResult
	UploadedMeta  RawMetadata
	B2Bucket      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	GenerateNewID bool
	PreserveID    string
}
