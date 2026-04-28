package entities

import (
	"time"

	"mycourse-io-be/constants"
	"mycourse-io-be/dto"
)

type MediaUploadEntityInput struct {
	Kind          constants.FileKind
	Provider      constants.FileProvider
	Filename      string
	ContentType   string
	SizeBytes     int64
	Payload       []byte
	Uploaded      dto.UploadFileResponse
	UploadedMeta  RawMetadata
	B2Bucket      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	GenerateNewID bool
	PreserveID    string
}
