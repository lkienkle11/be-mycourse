package infra

import (
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/constants"
)

// userRow is the GORM model for the users table.
type userRow struct {
	ID                         uint    `gorm:"primaryKey;autoIncrement"`
	UserCode                   string  `gorm:"type:uuid;uniqueIndex;not null"`
	Email                      string  `gorm:"size:255;uniqueIndex;not null"`
	HashPassword               string  `gorm:"size:255;not null"`
	DisplayName                string  `gorm:"size:255;not null;default:''"`
	AvatarFileID               *string `gorm:"column:avatar_file_id;type:uuid"`
	IsDisable                  bool    `gorm:"not null;default:false"`
	EmailConfirmed             bool    `gorm:"not null;default:false"`
	ConfirmationToken          *string `gorm:"size:128"`
	ConfirmationSentAt         *time.Time
	RegistrationEmailSendTotal int                    `gorm:"column:registration_email_send_total;not null;default:0"`
	RefreshTokenSession        RefreshTokenSessionMap `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	DeletedAt                  gorm.DeletedAt `gorm:"index"`
}

func (userRow) TableName() string { return constants.TableAppUsers }

func deletedAtToDomain(d gorm.DeletedAt) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func deletedAtToRow(d *time.Time) gorm.DeletedAt {
	if d == nil {
		return gorm.DeletedAt{}
	}
	return gorm.DeletedAt{Time: *d, Valid: true}
}

func toUserDomain(r *userRow) *domain.User {
	return &domain.User{
		ID:                         r.ID,
		UserCode:                   r.UserCode,
		Email:                      r.Email,
		HashPassword:               r.HashPassword,
		DisplayName:                r.DisplayName,
		AvatarFileID:               r.AvatarFileID,
		IsDisable:                  r.IsDisable,
		EmailConfirmed:             r.EmailConfirmed,
		ConfirmationToken:          r.ConfirmationToken,
		ConfirmationSentAt:         r.ConfirmationSentAt,
		RegistrationEmailSendTotal: r.RegistrationEmailSendTotal,
		RefreshTokenSession:        toDomainSessionMap(r.RefreshTokenSession),
		CreatedAt:                  r.CreatedAt,
		UpdatedAt:                  r.UpdatedAt,
		DeletedAt:                  deletedAtToDomain(r.DeletedAt),
	}
}

func toUserRow(u *domain.User) *userRow {
	row := &userRow{
		ID:                         u.ID,
		UserCode:                   u.UserCode,
		Email:                      u.Email,
		HashPassword:               u.HashPassword,
		DisplayName:                u.DisplayName,
		AvatarFileID:               u.AvatarFileID,
		IsDisable:                  u.IsDisable,
		EmailConfirmed:             u.EmailConfirmed,
		ConfirmationToken:          u.ConfirmationToken,
		ConfirmationSentAt:         u.ConfirmationSentAt,
		RegistrationEmailSendTotal: u.RegistrationEmailSendTotal,
		CreatedAt:                  u.CreatedAt,
		UpdatedAt:                  u.UpdatedAt,
		DeletedAt:                  deletedAtToRow(u.DeletedAt),
	}
	if u.RefreshTokenSession != nil {
		row.RefreshTokenSession = RefreshTokenSessionMap(u.RefreshTokenSession)
	}
	return row
}
