// Package domain contains the MEDIA bounded-context's pure domain types:
// entities, repository interfaces, and domain errors.
package domain

import (
	"mime/multipart"
	"time"
)

// MediaFile is the core media entity persisted in the media_files table.
type MediaFile struct {
	ID                 string
	ObjectKey          string
	Kind               string
	Provider           string
	Filename           string
	MimeType           string
	SizeBytes          int64
	URL                string
	OriginURL          string
	Status             string
	B2BucketName       string
	BunnyVideoID       string
	BunnyLibraryID     string
	VideoID            string
	ThumbnailURL       string
	EmbededHTML        string
	Duration           int64
	VideoProvider      string
	RowVersion         int64
	ContentFingerprint string
	MetadataJSON       []byte
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

// MediaPendingCloudCleanup tracks media objects scheduled for deferred cloud deletion.
type MediaPendingCloudCleanup struct {
	ID           int64
	Provider     string
	ObjectKey    string
	BunnyVideoID string
	Status       string
	AttemptCount int
	LastError    string
	NextRunAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MediaFilePublic is the client-facing subset of a stored media row (no server-only fields).
type MediaFilePublic struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Provider           string `json:"provider"`
	Filename           string `json:"filename"`
	MimeType           string `json:"mime_type"`
	SizeBytes          int64  `json:"size_bytes"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	URL                string `json:"url"`
	Duration           int64  `json:"duration"`
	ContentFingerprint string `json:"content_fingerprint"`
	Status             string `json:"status"`
}

// FileMetadata holds basic file type and dimension metadata.
type FileMetadata struct {
	Size      int64  `json:"size,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Extension string `json:"extension,omitempty"`
}

// UploadFileMetadata carries file intrinsic metadata probed at upload time.
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

// VideoProviderStatus is a normalized provider encoding/processing status.
type VideoProviderStatus struct {
	Status string
}

// File is the full service-layer representation of a media file.
type File struct {
	ID                 string
	Kind               string
	Provider           string
	Filename           string
	MimeType           string
	SizeBytes          int64
	URL                string
	OriginURL          string
	ObjectKey          string
	Status             string
	B2BucketName       string
	BunnyVideoID       string
	BunnyLibraryID     string
	VideoID            string
	ThumbnailURL       string
	EmbededHTML        string
	Duration           int64
	VideoProvider      string
	RowVersion         int64
	ContentFingerprint string
	MetadataJSON       string
	Metadata           UploadFileMetadata
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// RawMetadataMap decodes MetadataJSON into a RawMetadata map.
// Returns nil if MetadataJSON is empty or malformed.
func (f *File) RawMetadataMap() RawMetadata {
	if f.MetadataJSON == "" {
		return nil
	}
	out := make(RawMetadata)
	if err := jsonUnmarshal([]byte(f.MetadataJSON), &out); err != nil {
		return nil
	}
	return out
}

// MetadataJSONBytes returns MetadataJSON as a byte slice.
func (f *File) MetadataJSONBytes() []byte {
	return []byte(f.MetadataJSON)
}

// Provider constants — mirror constants/ so internal packages don't depend on top-level constants.
const (
	ProviderB2    = "B2"
	ProviderBunny = "Bunny"
	ProviderLocal = "Local"
	ProviderGCS   = "GCS"
	ProviderR2    = "R2"
	ProviderS3    = "S3"

	KindFile  = "FILE"
	KindVideo = "VIDEO"

	StatusReady   = "READY"
	StatusDeleted = "DELETED"
	StatusFailed  = "FAILED"
	StatusPending = "PENDING"
)

// --- Provider / upload types ---

// RawMetadata is an untyped metadata map for upstream provider responses.
type RawMetadata map[string]any

// ProviderUploadResult carries the result of a cloud storage upload.
type ProviderUploadResult struct {
	URL       string
	OriginURL string
	ObjectKey string
	Metadata  RawMetadata
}

// MediaUploadEntityInput carries all fields needed to create/update a MediaFile after upload.
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
	GenerateNewID bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PreserveID    string
}

// BunnyVideoDetail mirrors the Bunny Stream get-video payload.
//
// JSON tags below match Bunny's actual response fields (verified from live
// API logs). Note: Bunny does NOT return `bitrate`, `audioCodec`, or
// `thumbnailUrl` — `outputCodecs` and `thumbnailFileName` are the canonical
// keys for codec and thumbnail respectively. Legacy field aliases are kept
// for forward compatibility in case Bunny adds them later.
type BunnyVideoDetail struct {
	VideoLibraryID        int     `json:"videoLibraryId"`
	BunnyNumericID        int64   `json:"id"`
	GUID                  string  `json:"guid"`
	Title                 string  `json:"title"`
	Description           string  `json:"description"`
	DateUploaded          string  `json:"dateUploaded"`
	Views                 int64   `json:"views"`
	IsPublic              bool    `json:"isPublic"`
	Length                float64 `json:"length"`
	Status                int     `json:"status"`
	Framerate             float64 `json:"framerate"`
	Rotation              int     `json:"rotation"`
	Width                 int     `json:"width"`
	Height                int     `json:"height"`
	AvailableResolutions  string  `json:"availableResolutions"`
	OutputCodecs          string  `json:"outputCodecs"`
	ThumbnailCount        int     `json:"thumbnailCount"`
	EncodeProgress        int     `json:"encodeProgress"`
	StorageSize           int64   `json:"storageSize"`
	HasMP4Fallback        bool    `json:"hasMP4Fallback"`
	CollectionID          string  `json:"collectionId"`
	ThumbnailFileName     string  `json:"thumbnailFileName"`
	ThumbnailBlurhash     string  `json:"thumbnailBlurhash"`
	AverageWatchTime      int64   `json:"averageWatchTime"`
	TotalWatchTime        int64   `json:"totalWatchTime"`
	Category              string  `json:"category"`
	JitEncodingEnabled    bool    `json:"jitEncodingEnabled"`
	HasOriginal           bool    `json:"hasOriginal"`
	OriginalHash          string  `json:"originalHash"`
	HasHighQualityPreview bool    `json:"hasHighQualityPreview"`

	// Legacy / forward-compatible fields. Bunny may not populate these.
	Bitrate             int    `json:"bitrate"`
	VideoCodec          string `json:"videoCodec"`
	AudioCodec          string `json:"audioCodec"`
	ThumbnailURL        string `json:"thumbnailUrl"`
	DefaultThumbnailURL string `json:"defaultThumbnailUrl"`
}

// OpenedUploadPart is an opened multipart file part plus its header metadata.
type OpenedUploadPart struct {
	File   multipart.File
	Header *multipart.FileHeader
}

// PreparedCreatePart holds buffered payload and routing fields for one multipart part in a batch create.
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
