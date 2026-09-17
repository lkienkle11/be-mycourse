package server

import (
	"gorm.io/gorm"

	courseapp "mycourse-io-be/internal/course/application"
	coursedelivery "mycourse-io-be/internal/course/delivery"
	courseinfra "mycourse-io-be/internal/course/infra"
)

func wireCourse(db *gorm.DB, authz *AuthorizationServices) (*courseapp.CourseService, *coursedelivery.Handler) {
	repo := courseinfra.NewGormRepository(db, authz.Authorizer, authz.RoleBindingService)
	svc := courseapp.NewCourseService(repo)
	return svc, coursedelivery.NewHandler(svc)
}
