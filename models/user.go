package models

import (
	"time"

	"gorm.io/gorm"
)

// User is the application user stored in the custom `users` table.
// UserCode is a UUIDv7 string generated at the application layer before insert.
// It is used as the user_id in user_roles and user_permissions (RBAC tables).
type User struct {
	ID                 uint           `gorm:"primaryKey;autoIncrement"                json:"id"`
	UserCode           string         `gorm:"type:uuid;uniqueIndex;not null"          json:"user_code"`
	Email              string         `gorm:"size:255;uniqueIndex;not null"           json:"email"`
	HashPassword       string         `gorm:"size:255;not null"                       json:"-"`
	DisplayName        string         `gorm:"size:255;not null;default:''"            json:"display_name"`
	AvatarURL          string         `gorm:"type:text;not null;default:''"           json:"avatar_url"`
	IsDisable          bool           `gorm:"not null;default:false"                  json:"is_disable"`
	EmailConfirmed     bool           `gorm:"not null;default:false"                  json:"email_confirmed"`
	ConfirmationToken  *string        `gorm:"size:128"                                json:"-"`
	ConfirmationSentAt *time.Time     `                                               json:"-"`
	CreatedAt          time.Time      `                                               json:"created_at"`
	UpdatedAt          time.Time      `                                               json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index"                                   json:"deleted_at,omitempty"`
}

func (User) TableName() string { return "users" }
