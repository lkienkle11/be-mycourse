package models

import "time"

// SystemPrivilegedUser stores credentials for system-level operators (not normal app users).
type SystemPrivilegedUser struct {
	ID             uint      `gorm:"column:id;primaryKey" json:"id"`
	UsernameSecret string    `gorm:"column:username_secret;not null;uniqueIndex" json:"-"`
	PasswordSecret string    `gorm:"column:password_secret;not null" json:"-"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SystemPrivilegedUser) TableName() string { return "system_privileged_users" }
