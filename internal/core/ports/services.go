package ports

import (
	"context"

	"mycourse-io-be/internal/core/domain"
)

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*domain.User, error)
	Login(ctx context.Context, input LoginInput) (*TokenPair, error)
}

type UserService interface {
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
}

type CourseService interface {
	ListCourses(ctx context.Context) ([]domain.Course, error)
	CreateCourse(ctx context.Context, input CreateCourseInput) (*domain.Course, error)
	GetCourseByID(ctx context.Context, id string) (*domain.Course, error)
}

type EnrollmentService interface {
	Enroll(ctx context.Context, studentID, courseID string) (*domain.Enrollment, error)
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type CreateCourseInput struct {
	InstructorID string
	Title        string
	Description  string
	Price        float64
	ThumbnailURL string
}
