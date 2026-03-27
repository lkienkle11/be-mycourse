package ports

import (
	"context"

	"mycourse-io-be/internal/core/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

type CourseRepository interface {
	List(ctx context.Context) ([]domain.Course, error)
	Create(ctx context.Context, course *domain.Course) error
	FindByID(ctx context.Context, id string) (*domain.Course, error)
}

type LessonRepository interface {
	ListByCourseID(ctx context.Context, courseID string) ([]domain.Lesson, error)
}

type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *domain.Enrollment) error
	ExistsActive(ctx context.Context, studentID, courseID string) (bool, error)
}
