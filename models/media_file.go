package models

import (
	"time"

	"mycourse-io-be/constants"
	"mycourse-io-be/dbschema"
)

type MediaFile struct {
	ID                 string                 `gorm:"column:id;type:uuid;primaryKey"`
	ObjectKey          string                 `gorm:"column:object_key;type:varchar(512);uniqueIndex;not null"`
	Kind               constants.FileKind     `gorm:"column:kind;type:varchar(16);not null"`
	Provider           constants.FileProvider `gorm:"column:provider;type:varchar(16);not null"`
	Filename           string                 `gorm:"column:filename;type:varchar(512);not null"`
	MimeType           string                 `gorm:"column:mime_type;type:varchar(255);not null;default:''"`
	SizeBytes          int64                  `gorm:"column:size_bytes;not null;default:0"`
	URL                string                 `gorm:"column:url;type:text;not null"`
	OriginURL          string                 `gorm:"column:origin_url;type:text;not null"`
	Status             constants.FileStatus   `gorm:"column:status;type:varchar(16);not null"`
	B2BucketName       string                 `gorm:"column:b2_bucket_name;type:varchar(255);not null;default:''"`
	BunnyVideoID       string                 `gorm:"column:bunny_video_id;type:varchar(255);index"`
	BunnyLibraryID     string                 `gorm:"column:bunny_library_id;type:varchar(255)"`
	VideoID            string                 `gorm:"column:video_id;type:varchar(255);not null;default:''"`
	ThumbnailURL       string                 `gorm:"column:thumbnail_url;type:text;not null;default:''"`
	EmbededHTML        string                 `gorm:"column:embeded_html;type:text;not null;default:''"`
	Duration           int64                  `gorm:"column:duration;not null;default:0"`
	VideoProvider      string                 `gorm:"column:video_provider;type:varchar(64);not null;default:''"`
	RowVersion         int64                  `gorm:"column:row_version;not null;default:1"`
	ContentFingerprint string                 `gorm:"column:content_fingerprint;type:varchar(128);not null;default:''"`
	MetadataJSON       []byte                 `gorm:"column:metadata_json;type:jsonb;not null;default:'{}'::jsonb"`
	CreatedAt          time.Time              `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time              `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt          *time.Time             `gorm:"column:deleted_at"`
}

func (MediaFile) TableName() string { return dbschema.Media.Files() }
