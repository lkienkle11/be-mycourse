package server

import (
	"gorm.io/gorm"

	authapp "mycourse-io-be/internal/auth/application"
	authinfra "mycourse-io-be/internal/auth/infra"
	instapp "mycourse-io-be/internal/instructor/application"
	instdelivery "mycourse-io-be/internal/instructor/delivery"
	instinfra "mycourse-io-be/internal/instructor/infra"
	mediadomain "mycourse-io-be/internal/media/domain"
	rbacapp "mycourse-io-be/internal/rbac/application"
)

func wireInstructor(
	db *gorm.DB,
	rbacSvc *rbacapp.RBACService,
	authSvc *authapp.AuthService,
	userRepo *authinfra.GormUserRepository,
	fileRepo mediadomain.FileRepository,
) (*instapp.InstructorService, *instdelivery.Handler) {
	instRepo := instinfra.NewGormRepository(db)
	instSvc := instapp.NewInstructorService(
		instRepo,
		&instructorUserLookup{repo: userRepo},
		newInstructorRoleManager(rbacSvc),
		&instructorMeCache{auth: authSvc},
		&instructorProfileMediaValidator{files: fileRepo},
		&instructorAvatarHydrator{resolver: &mediaFileURLResolver{repo: fileRepo}},
	)
	return instSvc, instdelivery.NewHandler(instSvc)
}
