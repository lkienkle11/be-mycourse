package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/utils"
)

type CourseService struct {
	repo domain.Repository
}

func NewCourseService(repo domain.Repository) *CourseService {
	return &CourseService{repo: repo}
}

func courseTitleAndSlug(title string) (string, string, error) {
	title = strings.TrimSpace(title)
	if utils.CountNonWhitespace(title) < 5 {
		return "", "", domain.ErrCourseTitleTooShort
	}
	slug := utils.SlugifyName(title)
	if len(slug) < 1 {
		return "", "", domain.ErrCourseInvalidSlug
	}
	return title, slug, nil
}

func (s *CourseService) ListEditableCourses(ctx context.Context, userID string) ([]domain.CourseListItem, error) {
	return s.repo.ListEditableCourses(ctx, userID)
}

func (s *CourseService) CreateCourse(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	title, slug, err := courseTitleAndSlug(in.Title)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateCourse(ctx, domain.CreateCourseInput{
		ActorUserID: in.ActorUserID,
		Title:       title,
		Slug:        slug,
	})
}

func (s *CourseService) GetCourseDetail(ctx context.Context, courseID string, userID string, includeDraft bool) (*domain.CourseDetail, error) {
	return s.repo.GetCourseDetail(ctx, courseID, userID, includeDraft)
}

func (s *CourseService) PrepareDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return s.repo.PrepareDraft(ctx, courseID, actorUserID)
}

func (s *CourseService) UpdateBasicInfo(ctx context.Context, courseID string, actorUserID string, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
	if in.Title != nil {
		title, slug, err := courseTitleAndSlug(*in.Title)
		if err != nil {
			return nil, err
		}
		in.Title = &title
		in.Slug = &slug
	}
	return s.repo.UpdateBasicInfo(ctx, courseID, actorUserID, in)
}

func (s *CourseService) DeleteCourse(ctx context.Context, courseID string, actorUserID string) error {
	return s.repo.DeleteCourse(ctx, courseID, actorUserID)
}

func (s *CourseService) ListCollaborators(ctx context.Context, courseID string, actorUserID string) ([]domain.Collaborator, error) {
	return s.repo.ListCollaborators(ctx, courseID, actorUserID)
}

func (s *CourseService) AddCollaborator(ctx context.Context, courseID string, actorUserID, userID string, role string) ([]domain.Collaborator, error) {
	return s.repo.AddCollaborator(ctx, courseID, actorUserID, userID, role)
}

func (s *CourseService) RemoveCollaborator(ctx context.Context, courseID string, actorUserID, userID string) ([]domain.Collaborator, error) {
	return s.repo.RemoveCollaborator(ctx, courseID, actorUserID, userID)
}

func (s *CourseService) CreateSection(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSectionInput) (*domain.Section, error) {
	return s.repo.CreateSection(ctx, courseID, actorUserID, in)
}

func (s *CourseService) UpdateSection(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSectionInput) (*domain.Section, error) {
	return s.repo.UpdateSection(ctx, courseID, actorUserID, in)
}

func (s *CourseService) DeleteSection(ctx context.Context, courseID string, actorUserID string, sectionID string) ([]domain.Section, error) {
	return s.repo.DeleteSection(ctx, courseID, actorUserID, sectionID)
}

func (s *CourseService) ReorderSections(ctx context.Context, courseID string, actorUserID string, orderedStableIDs []string) ([]domain.Section, error) {
	return s.repo.ReorderSections(ctx, courseID, actorUserID, orderedStableIDs)
}

func (s *CourseService) CreateLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertLessonInput) (*domain.Lesson, error) {
	return s.repo.CreateLesson(ctx, courseID, actorUserID, in)
}

func (s *CourseService) UpdateLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertLessonInput) (*domain.Lesson, error) {
	return s.repo.UpdateLesson(ctx, courseID, actorUserID, in)
}

func (s *CourseService) DeleteLesson(ctx context.Context, courseID string, actorUserID string, lessonID string) ([]domain.Section, error) {
	return s.repo.DeleteLesson(ctx, courseID, actorUserID, lessonID)
}

func (s *CourseService) ReorderLessons(ctx context.Context, courseID string, actorUserID string, sectionID string, orderedStableIDs []string) ([]domain.Lesson, error) {
	return s.repo.ReorderLessons(ctx, courseID, actorUserID, sectionID, orderedStableIDs)
}

func (s *CourseService) CreateSubLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSubLessonInput) (*domain.SubLesson, error) {
	return s.repo.CreateSubLesson(ctx, courseID, actorUserID, in)
}

func (s *CourseService) UpdateSubLesson(ctx context.Context, courseID string, actorUserID string, in domain.UpsertSubLessonInput) (*domain.SubLesson, error) {
	return s.repo.UpdateSubLesson(ctx, courseID, actorUserID, in)
}

func (s *CourseService) DeleteSubLesson(ctx context.Context, courseID string, actorUserID string, subLessonID string) ([]domain.Section, error) {
	return s.repo.DeleteSubLesson(ctx, courseID, actorUserID, subLessonID)
}

func (s *CourseService) ReorderSubLessons(ctx context.Context, courseID string, actorUserID string, lessonID string, orderedStableIDs []string) ([]domain.SubLesson, error) {
	return s.repo.ReorderSubLessons(ctx, courseID, actorUserID, lessonID, orderedStableIDs)
}

func (s *CourseService) AcquireLease(ctx context.Context, courseID string, actorUserID string, in domain.AcquireLeaseInput) (*domain.Lease, error) {
	return s.repo.AcquireLease(ctx, courseID, actorUserID, in)
}

func (s *CourseService) HeartbeatLease(ctx context.Context, courseID string, actorUserID string, in domain.LeaseHeartbeatInput) (*domain.Lease, error) {
	return s.repo.HeartbeatLease(ctx, courseID, actorUserID, in)
}

func (s *CourseService) ReleaseLease(ctx context.Context, courseID string, actorUserID string, in domain.ReleaseLeaseInput) error {
	return s.repo.ReleaseLease(ctx, courseID, actorUserID, in)
}

func (s *CourseService) SubmitForReview(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return s.repo.SubmitForReview(ctx, courseID, actorUserID)
}

func (s *CourseService) ReopenDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return s.repo.ReopenDraft(ctx, courseID, actorUserID)
}

func (s *CourseService) ListPendingReviews(ctx context.Context) ([]domain.CourseListItem, error) {
	return s.repo.ListPendingReviews(ctx)
}

func (s *CourseService) ListAdminCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	return s.repo.ListAdminCourses(ctx)
}

func (s *CourseService) ListTrashedCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	return s.repo.ListTrashedCourses(ctx)
}

func (s *CourseService) TrashCourse(ctx context.Context, courseID string) error {
	return s.repo.TrashCourse(ctx, courseID)
}

func (s *CourseService) RestoreCourse(ctx context.Context, courseID string) error {
	return s.repo.RestoreCourse(ctx, courseID)
}

func (s *CourseService) PermanentDeleteCourse(ctx context.Context, courseID string) error {
	return s.repo.PermanentDeleteCourse(ctx, courseID)
}

func (s *CourseService) ApproveDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return s.repo.ApproveDraft(ctx, courseID, actorUserID)
}

func (s *CourseService) RejectDraft(ctx context.Context, courseID string, actorUserID string, reason string) (*domain.CourseDetail, error) {
	return s.repo.RejectDraft(ctx, courseID, actorUserID, reason)
}

func (s *CourseService) ListPublishedCourses(ctx context.Context) ([]domain.CourseListItem, error) {
	return s.repo.ListPublishedCourses(ctx)
}

func (s *CourseService) GetLearningCourse(ctx context.Context, courseID string, userID string) (*domain.CourseDetail, error) {
	return s.repo.GetLearningCourse(ctx, courseID, userID)
}

func (s *CourseService) Enroll(ctx context.Context, courseID string, userID string) (*domain.Enrollment, error) {
	return s.repo.Enroll(ctx, courseID, userID)
}

func (s *CourseService) GetProgress(ctx context.Context, courseID string, userID string) (*domain.CourseProgress, error) {
	return s.repo.GetProgress(ctx, courseID, userID)
}

func (s *CourseService) SaveProgress(ctx context.Context, courseID string, userID string, in domain.SaveProgressInput) (*domain.CourseProgress, error) {
	return s.repo.SaveProgress(ctx, courseID, userID, in)
}
