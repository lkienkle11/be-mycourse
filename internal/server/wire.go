// Package server wires all bounded contexts together and constructs the HTTP server.
package server

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authdelivery "mycourse-io-be/internal/auth/delivery"
	courseapp "mycourse-io-be/internal/course/application"
	coursedelivery "mycourse-io-be/internal/course/delivery"

	mediaapp "mycourse-io-be/internal/media/application"
	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	mediajobs "mycourse-io-be/internal/media/jobs"

	taxapp "mycourse-io-be/internal/taxonomy/application"
	taxdelivery "mycourse-io-be/internal/taxonomy/delivery"

	rbacapp "mycourse-io-be/internal/rbac/application"
	rbacdelivery "mycourse-io-be/internal/rbac/delivery"

	sysapp "mycourse-io-be/internal/system/application"
	sysdelivery "mycourse-io-be/internal/system/delivery"

	instapp "mycourse-io-be/internal/instructor/application"
	instdelivery "mycourse-io-be/internal/instructor/delivery"
)

// Services holds all application services after wiring.
type Services struct {
	Auth       *authapp.AuthService
	Course     *courseapp.CourseService
	Media      *mediaapp.MediaService
	Taxonomy   *taxapp.TaxonomyService
	RBAC       *rbacapp.RBACService
	System     *sysapp.SystemService
	Instructor *instapp.InstructorService
}

// Handlers holds all delivery handlers after wiring.
type Handlers struct {
	Auth       *authdelivery.Handler
	Course     *coursedelivery.Handler
	Media      *mediadelivery.Handler
	Taxonomy   *taxdelivery.Handler
	RBAC       *rbacdelivery.Handler
	System     *sysdelivery.Handler
	Instructor *instdelivery.Handler
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
	core := wireCore(db, rdb)
	instSvc, instHandler := wireInstructor(db, core.RBAC, core.Auth, core.UserRepo, core.FileRepo)
	courseSvc, courseHandler := wireCourse(db)
	mediaGW := mediainfra.NewStorageGateway()

	svcs := &Services{
		Auth: core.Auth, Course: courseSvc, Media: core.Media, Taxonomy: core.Taxonomy,
		RBAC: core.RBAC, System: core.System, Instructor: instSvc,
	}
	handlers := &Handlers{
		Auth:       authdelivery.NewHandler(core.Auth, &rbacPermissionUseCase{svc: core.RBAC}),
		Course:     courseHandler,
		Media:      mediadelivery.NewHandler(core.Media, mediaGW),
		Taxonomy:   taxdelivery.NewHandler(core.Taxonomy),
		RBAC:       rbacdelivery.NewHandler(core.RBAC),
		System:     sysdelivery.NewHandler(core.System),
		Instructor: instHandler,
	}
	return svcs, handlers, nil
}
