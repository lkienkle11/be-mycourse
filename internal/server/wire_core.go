package server

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authinfra "mycourse-io-be/internal/auth/infra"

	mediaapp "mycourse-io-be/internal/media/application"
	mediainfra "mycourse-io-be/internal/media/infra"
	mediajobs "mycourse-io-be/internal/media/jobs"

	taxapp "mycourse-io-be/internal/taxonomy/application"
	taxinfra "mycourse-io-be/internal/taxonomy/infra"

	rbacapp "mycourse-io-be/internal/rbac/application"
	rbacinfra "mycourse-io-be/internal/rbac/infra"

	sysapp "mycourse-io-be/internal/system/application"
	sysinfra "mycourse-io-be/internal/system/infra"

	mediadomain "mycourse-io-be/internal/media/domain"
)

// coreWiring holds services constructed before instructor and HTTP handlers.
type coreWiring struct {
	RBAC     *rbacapp.RBACService
	System   *sysapp.SystemService
	Media    *mediaapp.MediaService
	Taxonomy *taxapp.TaxonomyService
	Auth     *authapp.AuthService
	FileRepo mediadomain.FileRepository
	UserRepo *authinfra.GormUserRepository
}

func wireCore(db *gorm.DB, rdb *redis.Client) *coreWiring {
	permRepo := rbacinfra.NewGormPermissionRepository(db)
	roleRepo := rbacinfra.NewGormRoleRepository(db)
	userRoleRepo := rbacinfra.NewGormUserRoleRepository(db)
	userPermRepo := rbacinfra.NewGormUserPermissionRepository(db)
	rbacSvc := rbacapp.NewRBACService(permRepo, roleRepo, userRoleRepo, userPermRepo)

	appCfgRepo := sysinfra.NewGormAppConfigRepository(db)
	privUserRepo := sysinfra.NewGormPrivilegedUserRepository(db)
	permSyncer := sysinfra.NewGormPermissionSyncer(db)
	roleSyncer := sysinfra.NewGormRolePermissionSyncer(db)
	sysCrypto := sysinfra.NewSystemCryptoAdapter()
	sysSvc := sysapp.NewSystemService(appCfgRepo, privUserRepo, permSyncer, roleSyncer, sysCrypto)

	fileRepo := mediainfra.NewGormFileRepository(db)
	cleanupRepo := mediainfra.NewGormPendingCleanupRepository(db)
	orphanEnqueuer := mediajobs.NewOrphanEnqueuer(cleanupRepo)
	mediaGW := mediainfra.NewStorageGateway()
	mediaSvc := mediaapp.NewMediaService(fileRepo, cleanupRepo, orphanEnqueuer, mediajobs.GlobalCounters, mediaGW)
	_, _ = mediainfra.NewCloudClientsFromSetting()

	topicRepo := taxinfra.NewGormCourseTopicRepository(db)
	outcomeRepo := taxinfra.NewGormCourseOutcomeRepository(db)
	skillRepo := taxinfra.NewGormCourseSkillRepository(db)
	tagRepo := taxinfra.NewGormTagRepository(db)
	levelRepo := taxinfra.NewGormCourseLevelRepository(db)
	taxSvc := taxapp.NewTaxonomyService(
		topicRepo, outcomeRepo, skillRepo, tagRepo, levelRepo,
		&taxMediaFileValidator{svc: mediaSvc},
		&taxOrphanEnqueuer{fileRepo: fileRepo, cleanupRepo: cleanupRepo},
	)

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

	return &coreWiring{
		RBAC: rbacSvc, System: sysSvc, Media: mediaSvc, Taxonomy: taxSvc,
		Auth: authSvc, FileRepo: fileRepo, UserRepo: userRepo,
	}
}
