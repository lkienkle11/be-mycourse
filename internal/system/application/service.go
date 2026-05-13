// Package application contains the SYSTEM bounded-context use-case layer.
package application

import (
	"context"
	"strings"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/system/domain"
	"mycourse-io-be/internal/system/infra" //nolint:depguard // application calls infra.CredentialHMACHex and infra.MintSystemAccessToken crypto helpers; TODO: inject via domain port
)

// SystemService provides all SYSTEM use-cases.
type SystemService struct {
	appCfgRepo   domain.AppConfigRepository
	privUserRepo domain.PrivilegedUserRepository
	permSyncer   domain.PermissionSyncer
	roleSyncer   domain.RolePermissionSyncer
}

// NewSystemService constructs a SystemService.
func NewSystemService(
	appCfgRepo domain.AppConfigRepository,
	privUserRepo domain.PrivilegedUserRepository,
	permSyncer domain.PermissionSyncer,
	roleSyncer domain.RolePermissionSyncer,
) *SystemService {
	return &SystemService{
		appCfgRepo:   appCfgRepo,
		privUserRepo: privUserRepo,
		permSyncer:   permSyncer,
		roleSyncer:   roleSyncer,
	}
}

// SystemLogin validates privileged user credentials and returns a short-lived system access token.
func (s *SystemService) SystemLogin(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return "", apperrors.ErrSystemLoginFailed
	}
	cfg, err := s.appCfgRepo.Get(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" || strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return "", apperrors.ErrSystemSecretsNotReady
	}
	uh := infra.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := infra.CredentialHMACHex(cfg.AppSystemEnv, password)
	n, err := s.privUserRepo.MatchCount(ctx, uh, ph)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", apperrors.ErrSystemLoginFailed
	}
	return infra.MintSystemAccessToken(cfg.AppTokenEnv, uh)
}

// VerifySystemAccessToken checks the bearer token against app_token_env in DB.
// Implements middleware.SystemTokenVerifier.
func (s *SystemService) VerifySystemAccessToken(tokenStr string) error {
	cfg, err := s.appCfgRepo.Get(context.Background())
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return apperrors.ErrSystemSecretsNotReady
	}
	_, err = infra.ParseSystemAccessToken(cfg.AppTokenEnv, strings.TrimSpace(tokenStr))
	return err
}

// RegisterPrivilegedUser stores HMAC-derived secrets.
func (s *SystemService) RegisterPrivilegedUser(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return apperrors.ErrSystemUsernamePasswordRequired
	}
	cfg, err := s.appCfgRepo.Get(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppSystemEnv) == "" {
		return apperrors.ErrSystemSecretsNotReady
	}
	uh := infra.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := infra.CredentialHMACHex(cfg.AppSystemEnv, password)
	u := &domain.PrivilegedUser{UsernameSecret: uh, PasswordSecret: ph}
	return s.privUserRepo.Create(ctx, u)
}

// SyncPermissions upserts all catalog permissions from constants into the database.
func (s *SystemService) SyncPermissions(ctx context.Context) (int, error) {
	entries := AllPermissionEntries()
	if len(entries) == 0 {
		return 0, nil
	}
	return s.permSyncer.SyncPermissionsFromCatalog(ctx, entries)
}

// SyncRolePermissions replaces all role_permissions rows with the matrix from constants.
func (s *SystemService) SyncRolePermissions(ctx context.Context) (int, error) {
	pairs := AllRolePermissionPairs()
	if len(pairs) == 0 {
		return 0, nil
	}
	return s.roleSyncer.SyncRolePermissionsFromCatalog(ctx, pairs)
}

// SystemAccessTokenTTL exposes the token lifetime for response body use.
func SystemAccessTokenTTL() int {
	return int(infra.SystemAccessTokenTTL.Seconds())
}
