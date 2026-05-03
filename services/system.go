package services

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"mycourse-io-be/internal/systemauth"
	"mycourse-io-be/models"
)

var (
	ErrSystemAppConfigMissing = errors.New("system_app_config row missing")
	ErrSystemSecretsNotReady  = errors.New("system secrets are not configured in database")
	ErrSystemLoginFailed      = errors.New("invalid system credentials")
)

// GetSystemAppConfig returns the singleton system_app_config row (id=1).
func GetSystemAppConfig(db *gorm.DB) (*models.SystemAppConfig, error) {
	if db == nil {
		return nil, errors.New("nil database")
	}
	var row models.SystemAppConfig
	if err := db.Where("id = ?", 1).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSystemAppConfigMissing
		}
		return nil, err
	}
	return &row, nil
}

// RegisterSystemPrivilegedUser stores HMAC-derived secrets using app_system_env from DB.
func RegisterSystemPrivilegedUser(db *gorm.DB, username, password string) error {
	if db == nil {
		return errors.New("nil database")
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return errors.New("username and password required")
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" {
		return ErrSystemSecretsNotReady
	}
	uh := systemauth.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := systemauth.CredentialHMACHex(cfg.AppSystemEnv, password)
	row := models.SystemPrivilegedUser{
		UsernameSecret: uh,
		PasswordSecret: ph,
	}
	return db.Create(&row).Error
}

// SystemLogin validates privileged user credentials and returns a short-lived system access token.
func SystemLogin(db *gorm.DB, username, password string) (accessToken string, err error) {
	if db == nil {
		return "", errors.New("nil database")
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", ErrSystemLoginFailed
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" || strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return "", ErrSystemSecretsNotReady
	}
	uh := systemauth.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := systemauth.CredentialHMACHex(cfg.AppSystemEnv, password)

	var n int64
	if err := db.Model(&models.SystemPrivilegedUser{}).
		Where("username_secret = ? AND password_secret = ?", uh, ph).
		Count(&n).Error; err != nil {
		return "", err
	}
	if n == 0 {
		return "", ErrSystemLoginFailed
	}
	tok, err := systemauth.MintSystemAccessToken(cfg.AppTokenEnv, uh)
	if err != nil {
		return "", err
	}
	return tok, nil
}

// VerifySystemAccessToken checks the bearer token against app_token_env in DB.
func VerifySystemAccessToken(db *gorm.DB, tokenStr string) error {
	if db == nil {
		return errors.New("nil database")
	}
	cfg, err := GetSystemAppConfig(db)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return ErrSystemSecretsNotReady
	}
	_, err = systemauth.ParseSystemAccessToken(cfg.AppTokenEnv, strings.TrimSpace(tokenStr))
	return err
}
