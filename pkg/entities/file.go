package entities

import (
	"time"

	"mycourse-io-be/constants"
)

type RawMetadata map[string]any

type FileMetadata struct {
	Size      int64  `json:"size,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Extension string `json:"extension,omitempty"`
}

type ImageMetadata struct {
	FileMetadata
}

type VideoMetadata struct {
	FileMetadata
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Duration   float64 `json:"duration"`
	Bitrate    int     `json:"bitrate"`
	FPS        float64 `json:"fps"`
	VideoCodec string  `json:"video_codec"`
	AudioCodec string  `json:"audio_codec"`
	HasAudio   bool    `json:"has_audio"`
	IsHDR      bool    `json:"is_hdr"`
}

type DocumentMetadata struct {
	FileMetadata
	PageCount int `json:"page_count,omitempty"`
}

type File struct {
	ID                 string                 `json:"id"`
	Kind               constants.FileKind     `json:"kind"`
	Provider           constants.FileProvider `json:"provider"`
	Filename           string                 `json:"filename"`
	MimeType           string                 `json:"mime_type"`
	SizeBytes          int64                  `json:"size_bytes"`
	URL                string                 `json:"url"`
	OriginURL          string                 `json:"origin_url"`
	ObjectKey          string                 `json:"object_key"`
	Status             constants.FileStatus   `json:"status"`
	B2BucketName       string                 `json:"b2_bucket_name"`
	BunnyVideoID       string                 `json:"bunny_video_id"`
	BunnyLibraryID     string                 `json:"bunny_library_id"`
	Duration           int64                  `json:"duration"`
	VideoProvider      string                 `json:"video_provider"`
	RowVersion         int64                  `json:"row_version,omitempty"`
	ContentFingerprint string                 `json:"content_fingerprint,omitempty"`
	Metadata           any                    `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}
