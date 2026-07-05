package server

import (
	"context"

	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authinfra "mycourse-io-be/internal/auth/infra"
	instapp "mycourse-io-be/internal/instructor/application"
	instdelivery "mycourse-io-be/internal/instructor/delivery"
	instinfra "mycourse-io-be/internal/instructor/infra"
	mediadomain "mycourse-io-be/internal/media/domain"
	rbacapp "mycourse-io-be/internal/rbac/application"
	"mycourse-io-be/internal/shared/gormx"
	"mycourse-io-be/internal/shared/useraccess"
)

func wireInstructor(
	db *gorm.DB,
	rbacSvc *rbacapp.RBACService,
	authSvc *authapp.AuthService,
	userRepo *authinfra.GormUserRepository,
	fileRepo mediadomain.FileRepository,
) (*instapp.InstructorService, *instdelivery.Handler) {
	instRepo := instinfra.NewGormRepository(db)
	instSvc := instapp.NewInstructorService(instRepo, instapp.InstructorServiceDeps{
		Users:               &instructorUserLookup{repo: userRepo},
		Roles:               newInstructorRoleManager(rbacSvc),
		Perms:               &instructorPermissionChecker{rbac: rbacSvc},
		MeCache:             &instructorMeCache{auth: authSvc},
		MediaVal:            &instructorProfileMediaValidator{files: fileRepo},
		Hydrator:            &instructorAvatarHydrator{resolver: &mediaFileURLResolver{repo: fileRepo}},
		MediaHydr:           &instructorMediaHydrator{repo: fileRepo},
		AssignmentSnapshots: &instructorAssignmentSnapshotLoader{db: db},
	})
	return instSvc, instdelivery.NewHandler(instSvc)
}

type instructorAssignmentSnapshotLoader struct{ db *gorm.DB }

func (l *instructorAssignmentSnapshotLoader) LoadAssignmentSnapshot(
	ctx context.Context,
	userID string,
) (useraccess.AssignmentSnapshot, error) {
	return gormx.LoadAssignmentSnapshotByID(ctx, l.db, userID)
}
