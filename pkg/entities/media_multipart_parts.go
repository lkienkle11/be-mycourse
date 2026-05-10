package entities

import (
	"mime/multipart"
)

// OpenedUploadPart is an opened multipart file part plus its header metadata.
type OpenedUploadPart struct {
	File   multipart.File
	Header *multipart.FileHeader
}

// PreparedCreatePart holds buffered payload and routing fields for one multipart part in a batch create/update tail.
type PreparedCreatePart struct {
	Header    *multipart.FileHeader
	Payload   []byte
	Filename  string
	Mime      string
	Kind      string
	Provider  string
	ObjectKey string
}

// PreparedUpdateHead holds buffered payload for the primary row in a bundle PUT.
type PreparedUpdateHead struct {
	Header            *multipart.FileHeader
	Payload           []byte
	Filename          string
	Mime              string
	Fingerprint       string
	PayloadNorm       []byte
	FilenameNorm      string
	MimeNorm          string
	Kind              string
	Provider          string
	ResolvedObjectKey string
}
