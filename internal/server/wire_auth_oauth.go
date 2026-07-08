package server

import (
	"context"

	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authdomain "mycourse-io-be/internal/auth/domain"
	authinfra "mycourse-io-be/internal/auth/infra"
	rbacinfra "mycourse-io-be/internal/rbac/infra"
	"mycourse-io-be/internal/shared/setting"
)

func wireAuthOAuth(
	db *gorm.DB,
	authSvc *authapp.AuthService,
	userRepo *authinfra.GormUserRepository,
	userRoleRepo *rbacinfra.GormUserRoleRepository,
	roleRepo *rbacinfra.GormRoleRepository,
) {
	oauthIdentityRepo := authinfra.NewGormOAuthIdentityRepository(db)
	oauthWriter := newAuthOAuthAccountWriter(db, userRepo, oauthIdentityRepo, userRoleRepo, roleRepo)
	var googleOAuth authapp.GoogleOAuthClient
	if setting.OAuthGoogleConfigured() {
		googleOAuth = authapp.NewGoogleOAuthVerifier(
			setting.OAuthSetting.GoogleClientID,
			setting.OAuthSetting.GoogleClientSecret,
		)
	}
	var xOAuth authapp.XOAuthClient
	if setting.OAuthXConfigured() {
		xOAuth = authapp.NewXOAuthVerifier(
			setting.OAuthSetting.XClientID,
			setting.OAuthSetting.XClientSecret,
			setting.OAuthSetting.XCallbackURL,
		)
	}
	var discordOAuth authapp.DiscordOAuthClient
	if setting.OAuthDiscordConfigured() {
		discordOAuth = authapp.NewDiscordOAuthVerifier(
			setting.OAuthSetting.DiscordClientID,
			setting.OAuthSetting.DiscordClientSecret,
			setting.OAuthSetting.DiscordCallbackURL,
		)
	}
	authSvc.AttachOAuth(oauthIdentityRepo, oauthWriter, googleOAuth, xOAuth, discordOAuth)
}

type authOAuthAccountWriter struct {
	db           *gorm.DB
	userRepo     *authinfra.GormUserRepository
	oauthRepo    *authinfra.GormOAuthIdentityRepository
	userRoleRepo *rbacinfra.GormUserRoleRepository
	roleRepo     *rbacinfra.GormRoleRepository
}

func newAuthOAuthAccountWriter(
	db *gorm.DB,
	userRepo *authinfra.GormUserRepository,
	oauthRepo *authinfra.GormOAuthIdentityRepository,
	userRoleRepo *rbacinfra.GormUserRoleRepository,
	roleRepo *rbacinfra.GormRoleRepository,
) authapp.OAuthAccountWriter {
	return &authOAuthAccountWriter{
		db: db, userRepo: userRepo, oauthRepo: oauthRepo,
		userRoleRepo: userRoleRepo, roleRepo: roleRepo,
	}
}

func (w *authOAuthAccountWriter) CreateUserWithIdentityAndLearnerRole(
	ctx context.Context,
	user *authdomain.User,
	identity *authdomain.UserOAuthIdentity,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.userRepo.CreateWithDB(ctx, tx, user); err != nil {
			return err
		}
		identity.UserID = user.ID
		if err := w.oauthRepo.CreateWithDB(ctx, tx, identity); err != nil {
			return err
		}
		return w.assignLearnerRole(ctx, tx, user.ID)
	})
}

func (w *authOAuthAccountWriter) LinkIdentityAndUpdateUser(
	ctx context.Context,
	user *authdomain.User,
	identity *authdomain.UserOAuthIdentity,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.userRepo.SaveWithDB(ctx, tx, user); err != nil {
			return err
		}
		identity.UserID = user.ID
		if err := w.oauthRepo.CreateWithDB(ctx, tx, identity); err != nil {
			return err
		}
		return w.ensureLearnerIfConfirmed(ctx, tx, user)
	})
}

func (w *authOAuthAccountWriter) assignLearnerRole(ctx context.Context, tx *gorm.DB, userID string) error {
	return assignLearnerRoleWithDB(ctx, tx, w.userRoleRepo, w.roleRepo, userID)
}

func (w *authOAuthAccountWriter) ensureLearnerIfConfirmed(ctx context.Context, tx *gorm.DB, user *authdomain.User) error {
	if !user.EmailConfirmed {
		return nil
	}
	return w.assignLearnerRole(ctx, tx, user.ID)
}
