package services

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/systemauth"
	"mycourse-io-be/models"
	pkgerrors "mycourse-io-be/pkg/errors"
)

// GetSystemAppConfig returns the singleton system_app_config row (id=1).
func GetSystemAppConfig(db *gorm.DB) (*models.SystemAppConfig, error) {
	if db == nil {
		return nil, pkgerrors.ErrNilDatabase
	}
	var row models.SystemAppConfig
	if err := db.Where("id = ?", 1).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.ErrSystemAppConfigMissing
		}
		return nil, err
	}
	return &row, nil
}

// RegisterSystemPrivilegedUser stores HMAC-derived secrets using app_system_env from DB.
func RegisterSystemPrivilegedUser(db *gorm.DB, username, password string) error {
	if db == nil {
		return pkgerrors.ErrNilDatabase
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return pkgerrors.ErrSystemUsernamePasswordRequired
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" {
		return pkgerrors.ErrSystemSecretsNotReady
	}
	uh := systemauth.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := systemauth.CredentialHMACHex(cfg.AppSystemEnv, password)
	row := models.SystemPrivilegedUser{
		UsernameSecret: uh,
		PasswordSecret: ph,
	}
	return db.Create(&row).Error
}

func systemPrivilegedUserMatchCount(db *gorm.DB, usernameSecret, passwordSecret string) (int64, error) {
	var n int64
	err := db.Model(&models.SystemPrivilegedUser{}).
		Where("username_secret = ? AND password_secret = ?", usernameSecret, passwordSecret).
		Count(&n).Error
	return n, err
}

// SystemLogin validates privileged user credentials and returns a short-lived system access token.
func SystemLogin(db *gorm.DB, username, password string) (accessToken string, err error) {
	if db == nil {
		return "", pkgerrors.ErrNilDatabase
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", pkgerrors.ErrSystemLoginFailed
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" || strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return "", pkgerrors.ErrSystemSecretsNotReady
	}
	uh := systemauth.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := systemauth.CredentialHMACHex(cfg.AppSystemEnv, password)
	n, err := systemPrivilegedUserMatchCount(db, uh, ph)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", pkgerrors.ErrSystemLoginFailed
	}
	return systemauth.MintSystemAccessToken(cfg.AppTokenEnv, uh)
}

// VerifySystemAccessToken checks the bearer token against app_token_env in DB.
func VerifySystemAccessToken(db *gorm.DB, tokenStr string) error {
	if db == nil {
		return pkgerrors.ErrNilDatabase
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return pkgerrors.ErrSystemSecretsNotReady
	}
	_, err = systemauth.ParseSystemAccessToken(cfg.AppTokenEnv, strings.TrimSpace(tokenStr))
	return err
}
