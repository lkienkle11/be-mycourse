package models

import (
	"time"

	"mycourse-io-be/dbschema"
)

type MediaPendingCloudCleanup struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider     string    `gorm:"column:provider;type:varchar(16);not null"`
	ObjectKey    string    `gorm:"column:object_key;type:varchar(512);not null;default:''"`
	BunnyVideoID string    `gorm:"column:bunny_video_id;type:varchar(255);not null;default:''"`
	Status       string    `gorm:"column:status;type:varchar(32);not null"`
	AttemptCount int       `gorm:"column:attempt_count;not null;default:0"`
	LastError    string    `gorm:"column:last_error;type:text;not null;default:''"`
	NextRunAt    time.Time `gorm:"column:next_run_at;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (MediaPendingCloudCleanup) TableName() string {
	return dbschema.Media.PendingCloudCleanup()
}
