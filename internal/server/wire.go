// Package server wires all bounded contexts together and constructs the HTTP server.
package server

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authdelivery "mycourse-io-be/internal/auth/delivery"
	authinfra "mycourse-io-be/internal/auth/infra"

	mediaapp "mycourse-io-be/internal/media/application"
	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediainfra "mycourse-io-be/internal/media/infra"
	mediajobs "mycourse-io-be/internal/media/jobs"

	taxapp "mycourse-io-be/internal/taxonomy/application"
	taxdelivery "mycourse-io-be/internal/taxonomy/delivery"
	taxinfra "mycourse-io-be/internal/taxonomy/infra"

	rbacapp "mycourse-io-be/internal/rbac/application"
	rbacdelivery "mycourse-io-be/internal/rbac/delivery"
	rbacinfra "mycourse-io-be/internal/rbac/infra"

	sysapp "mycourse-io-be/internal/system/application"
	sysdelivery "mycourse-io-be/internal/system/delivery"
	sysinfra "mycourse-io-be/internal/system/infra"

	mediadomain "mycourse-io-be/internal/media/domain"
)

// Services holds all application services after wiring.
type Services struct {
	Auth     *authapp.AuthService
	Media    *mediaapp.MediaService
	Taxonomy *taxapp.TaxonomyService
	RBAC     *rbacapp.RBACService
	System   *sysapp.SystemService
}

// Handlers holds all delivery handlers after wiring.
type Handlers struct {
	Auth     *authdelivery.Handler
	Media    *mediadelivery.Handler
	Taxonomy *taxdelivery.Handler
	RBAC     *rbacdelivery.Handler
	System   *sysdelivery.Handler
}

// --- adapters ----------------------------------------------------------------

// rbacPermissionReader adapts RBACService to authapp.PermissionReader
// (no context parameter — uses Background context).
type rbacPermissionReader struct{ svc *rbacapp.RBACService }

func (r *rbacPermissionReader) PermissionCodesForUser(userID uint) (map[string]struct{}, error) {
	return r.svc.PermissionCodesForUser(context.Background(), userID)
}

// rbacPermissionUseCase adapts RBACService to authdelivery.PermissionUseCase.
type rbacPermissionUseCase struct{ svc *rbacapp.RBACService }

func (r *rbacPermissionUseCase) PermissionCodesForUser(userID uint) (map[string]struct{}, error) {
	return r.svc.PermissionCodesForUser(context.Background(), userID)
}

// mediaProfileImageValidator adapts MediaService to authapp.MediaFileValidator
// (ValidateProfileImageFile(fileID string) error).
type mediaProfileImageValidator struct{ svc *mediaapp.MediaService }

func (m *mediaProfileImageValidator) ValidateProfileImageFile(fileID string) error {
	_, err := m.svc.LoadValidatedProfileImageFile(context.Background(), fileID)
	return err
}

// taxMediaFileValidator adapts MediaService to taxapp.MediaFileValidator
// (LoadValidatedProfileImageFile(ctx, fileID) (imageURL string, err error)).
type taxMediaFileValidator struct{ svc *mediaapp.MediaService }

func (t *taxMediaFileValidator) LoadValidatedProfileImageFile(ctx context.Context, fileID string) (string, error) {
	f, err := t.svc.LoadValidatedProfileImageFile(ctx, fileID)
	if err != nil {
		return "", err
	}
	return f.URL, nil
}

// authOrphanEnqueuer adapts OrphanEnqueuer to authapp.OrphanCleanupEnqueuer.
type authOrphanEnqueuer struct{ e *mediajobs.OrphanEnqueuer }

func (a *authOrphanEnqueuer) EnqueueOrphanCleanup(fileID string) {
	a.e.EnqueueSupersededPendingCleanup(fileID, "", "")
}

// taxOrphanEnqueuer adapts domain repositories to taxapp.OrphanImageEnqueuer.
type taxOrphanEnqueuer struct {
	fileRepo    mediadomain.FileRepository
	cleanupRepo mediadomain.PendingCleanupRepository
}

func (t *taxOrphanEnqueuer) EnqueueOrphanCleanupForFileID(ctx context.Context, fileID string) {
	// Try to resolve fileID → objectKey for cleanup via EnqueueOrphanImageCleanupByURL
	mediajobs.EnqueueOrphanImageCleanupByURL(ctx, t.fileRepo, t.cleanupRepo, fileID)
}

// Wire constructs all services and handlers using the provided DB and Redis connections.
func Wire(db *gorm.DB, rdb *redis.Client) (*Services, *Handlers, error) {
	// --- RBAC (no further deps on other domains) ---------------------------------
	permRepo := rbacinfra.NewGormPermissionRepository(db)
	roleRepo := rbacinfra.NewGormRoleRepository(db)
	userRoleRepo := rbacinfra.NewGormUserRoleRepository(db)
	userPermRepo := rbacinfra.NewGormUserPermissionRepository(db)
	rbacSvc := rbacapp.NewRBACService(permRepo, roleRepo, userRoleRepo, userPermRepo)

	// --- System ------------------------------------------------------------------
	appCfgRepo := sysinfra.NewGormAppConfigRepository(db)
	privUserRepo := sysinfra.NewGormPrivilegedUserRepository(db)
	permSyncer := sysinfra.NewGormPermissionSyncer(db)
	roleSyncer := sysinfra.NewGormRolePermissionSyncer(db)
	sysSvc := sysapp.NewSystemService(appCfgRepo, privUserRepo, permSyncer, roleSyncer)

	// --- Media -------------------------------------------------------------------
	fileRepo := mediainfra.NewGormFileRepository(db)
	cleanupRepo := mediainfra.NewGormPendingCleanupRepository(db)
	orphanEnqueuer := mediajobs.NewOrphanEnqueuer(cleanupRepo)
	mediaSvc := mediaapp.NewMediaService(fileRepo, cleanupRepo, orphanEnqueuer, mediajobs.GlobalCounters)

	// Non-fatal: if cloud clients can't be loaded, media uploads will fail at request time.
	_, _ = mediainfra.NewCloudClientsFromSetting()

	// --- Taxonomy ----------------------------------------------------------------
	catRepo := taxinfra.NewGormCategoryRepository(db)
	tagRepo := taxinfra.NewGormTagRepository(db)
	levelRepo := taxinfra.NewGormCourseLevelRepository(db)
	taxSvc := taxapp.NewTaxonomyService(
		catRepo, tagRepo, levelRepo,
		&taxMediaFileValidator{svc: mediaSvc},
		&taxOrphanEnqueuer{fileRepo: fileRepo, cleanupRepo: cleanupRepo},
	)

	// --- Auth (depends on RBAC + Media) ------------------------------------------
	userRepo := authinfra.NewGormUserRepository(db)
	sessRepo := authinfra.NewGormRefreshSessionRepository(db)
	authSvc := authapp.NewAuthService(
		userRepo,
		sessRepo,
		&rbacPermissionReader{svc: rbacSvc},
		&mediaProfileImageValidator{svc: mediaSvc},
		&authOrphanEnqueuer{e: orphanEnqueuer},
		rdb,
	)

	svcs := &Services{
		Auth:     authSvc,
		Media:    mediaSvc,
		Taxonomy: taxSvc,
		RBAC:     rbacSvc,
		System:   sysSvc,
	}

	handlers := &Handlers{
		Auth:     authdelivery.NewHandler(authSvc, &rbacPermissionUseCase{svc: rbacSvc}),
		Media:    mediadelivery.NewHandler(mediaSvc),
		Taxonomy: taxdelivery.NewHandler(taxSvc),
		RBAC:     rbacdelivery.NewHandler(rbacSvc),
		System:   sysdelivery.NewHandler(sysSvc),
	}

	return svcs, handlers, nil
}
