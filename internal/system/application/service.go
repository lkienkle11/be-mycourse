// Package application contains the SYSTEM bounded-context use-case layer.
package application

import (
	"context"
	"strings"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/system/domain"
)

// SystemService provides all SYSTEM use-cases.
type SystemService struct {
	appCfgRepo   domain.AppConfigRepository
	privUserRepo domain.PrivilegedUserRepository
	permSyncer   domain.PermissionSyncer
	roleSyncer   domain.RolePermissionSyncer
	crypto       domain.SystemCrypto
}

// NewSystemService constructs a SystemService.
func NewSystemService(
	appCfgRepo domain.AppConfigRepository,
	privUserRepo domain.PrivilegedUserRepository,
	permSyncer domain.PermissionSyncer,
	roleSyncer domain.RolePermissionSyncer,
	crypto domain.SystemCrypto,
) *SystemService {
	return &SystemService{
		appCfgRepo:   appCfgRepo,
		privUserRepo: privUserRepo,
		permSyncer:   permSyncer,
		roleSyncer:   roleSyncer,
		crypto:       crypto,
	}
}

// SystemLogin validates privileged user credentials and returns a short-lived system access token.
// app_system_env and app_token_env in system_app_config must be bcrypt hashes (cost 14), not plaintext;
// the hash strings are used as HMAC/JWT key material via SystemCrypto.
func (s *SystemService) SystemLogin(ctx context.Context, username, password, machineSecret string) (string, error) {
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
	uh := s.crypto.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := s.crypto.CredentialHMACHex(cfg.AppSystemEnv, password)
	row, err := s.privUserRepo.FindByCredentials(ctx, uh, ph)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", apperrors.ErrSystemLoginFailed
	}
	if row.MachineSecret != machineSecret {
		return "", apperrors.ErrSystemMachineBindingFailed
	}
	return s.crypto.MintSystemAccessToken(cfg.AppTokenEnv, uh)
}

// VerifySystemAccessToken checks the bearer token against app_token_env in DB.
// app_token_env must be a bcrypt hash (cost 14) used as JWT signing key material.
// Implements middleware.SystemTokenVerifier.
func (s *SystemService) VerifySystemAccessToken(tokenStr string) error {
	cfg, err := s.appCfgRepo.Get(context.Background())
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AppTokenEnv) == "" {
		return apperrors.ErrSystemSecretsNotReady
	}
	_, err = s.crypto.ParseSystemAccessToken(cfg.AppTokenEnv, strings.TrimSpace(tokenStr))
	return err
}

// RegisterPrivilegedUser stores HMAC-derived secrets.
// app_system_env must be a bcrypt hash (cost 14) used as HMAC key material for username/password derivation.
func (s *SystemService) RegisterPrivilegedUser(ctx context.Context, username, password, machineSecret string) error {
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
	uh := s.crypto.CredentialHMACHex(cfg.AppSystemEnv, username)
	ph := s.crypto.CredentialHMACHex(cfg.AppSystemEnv, password)
	u := &domain.PrivilegedUser{
		UsernameSecret: uh,
		PasswordSecret: ph,
		MachineSecret:  machineSecret,
	}
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

// systemAccessTokenTTLSeconds matches infra.SystemAccessTokenTTL without importing infra.
const systemAccessTokenTTLSeconds = 90

// SystemAccessTokenTTL exposes the token lifetime for response body use.
func SystemAccessTokenTTL() int {
	return systemAccessTokenTTLSeconds
}
