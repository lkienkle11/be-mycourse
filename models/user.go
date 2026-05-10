package models

import (
	"time"

	"mycourse-io-be/dbschema"
	gormjsonbauth "mycourse-io-be/pkg/gormjsonb/auth"
)

// User is the application user stored in the custom `users` table.
// UserCode is a UUIDv7 string generated at the application layer before insert.
// It is used as the user_id in user_roles and user_permissions (RBAC tables).
type User struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement"         json:"id"`
	UserCode           string     `gorm:"type:uuid;uniqueIndex;not null"   json:"user_code"`
	Email              string     `gorm:"size:255;uniqueIndex;not null"    json:"email"`
	HashPassword       string     `gorm:"size:255;not null"                json:"-"`
	DisplayName        string     `gorm:"size:255;not null;default:''"     json:"display_name"`
	AvatarFileID       *string    `gorm:"column:avatar_file_id;type:uuid" json:"-"`
	AvatarFile         *MediaFile `gorm:"foreignKey:AvatarFileID;references:ID"`
	IsDisable          bool       `gorm:"not null;default:false"           json:"is_disable"`
	EmailConfirmed     bool       `gorm:"not null;default:false"           json:"email_confirmed"`
	ConfirmationToken  *string    `gorm:"size:128"                         json:"-"`
	ConfirmationSentAt *time.Time `                                        json:"-"`
	// RegistrationEmailSendTotal counts successful Brevo confirmation emails while pending; never exposed in public JSON.
	RegistrationEmailSendTotal int                                  `gorm:"column:registration_email_send_total;not null;default:0" json:"-"`
	RefreshTokenSession        gormjsonbauth.RefreshTokenSessionMap `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	CreatedAt                  time.Time                            `                                        json:"created_at"`
	UpdatedAt                  time.Time                            `                                        json:"updated_at"`
	DeletedAt                  DeletedAt                            `gorm:"index"                            json:"deleted_at,omitempty"`
}

func (User) TableName() string { return dbschema.AppUser.Table() }
