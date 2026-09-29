package application

import (
	"context"
	"testing"

	"mycourse-io-be/internal/course/domain"
)

// fakeSlugTestRepo embeds domain.Repository as a nil interface: only methods
// explicitly overridden below are usable; calling any other method panics on
// a nil-pointer dereference, which is exactly the assertion these tests need
// ("the repo is never reached" fails loudly instead of silently).
type fakeSlugTestRepo struct {
	domain.Repository
	createCourseCalled bool
	createCourseInput  domain.CreateCourseInput
	updateInput        domain.UpdateBasicInfoInput
}

func (f *fakeSlugTestRepo) CreateCourse(_ context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	f.createCourseCalled = true
	f.createCourseInput = in
	return &domain.CourseDetail{}, nil
}

func (f *fakeSlugTestRepo) UpdateBasicInfo(_ context.Context, _ string, _ string, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
	f.updateInput = in
	return &domain.CourseDetail{}, nil
}

func TestCreateCourseRejectsInvalidManualSlugBeforeRepoCall(t *testing.T) {
	repo := &fakeSlugTestRepo{}
	svc := NewCourseService(repo)

	_, err := svc.CreateCourse(context.Background(), domain.CreateCourseInput{
		ActorUserID: "user-1",
		Title:       "A valid title",
		Slug:        "Not Valid Slug", // uppercase + space -> invalid format
	})
	if err != domain.ErrCourseInvalidSlug {
		t.Fatalf("err = %v, want domain.ErrCourseInvalidSlug", err)
	}
	if repo.createCourseCalled {
		t.Fatal("repo.CreateCourse must not be called when manual slug is invalid")
	}
}

func TestCreateCourseTreatsWhitespaceSlugAsAutoGenerate(t *testing.T) {
	repo := &fakeSlugTestRepo{}
	svc := NewCourseService(repo)

	if _, err := svc.CreateCourse(context.Background(), domain.CreateCourseInput{
		ActorUserID: "user-1",
		Title:       "A valid title",
		Slug:        "   ", // whitespace-only -> must be treated as "no slug supplied"
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.createCourseCalled {
		t.Fatal("expected repo.CreateCourse to be called")
	}
	if repo.createCourseInput.Slug != "" {
		t.Fatalf("Slug passed to repo = %q, want \"\" (auto-generate signal)", repo.createCourseInput.Slug)
	}
}

func TestUpdateBasicInfoPassesSlugThroughUntouched(t *testing.T) {
	repo := &fakeSlugTestRepo{}
	svc := NewCourseService(repo)

	title := "A new valid title"
	newSlug := "already-validated-slug"
	if _, err := svc.UpdateBasicInfo(context.Background(), "course-1", "user-1", domain.UpdateBasicInfoInput{
		Title: &title,
		Slug:  &newSlug,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateInput.Slug == nil || *repo.updateInput.Slug != newSlug {
		t.Fatalf("Slug passed to repo = %v, want pointer to %q", repo.updateInput.Slug, newSlug)
	}
}

func TestUpdateBasicInfoLeavesNilSlugAlone(t *testing.T) {
	repo := &fakeSlugTestRepo{}
	svc := NewCourseService(repo)

	title := "A new valid title"
	if _, err := svc.UpdateBasicInfo(context.Background(), "course-1", "user-1", domain.UpdateBasicInfoInput{
		Title: &title,
		Slug:  nil,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateInput.Slug != nil {
		t.Fatalf("Slug passed to repo = %v, want nil (omitted -> unchanged)", repo.updateInput.Slug)
	}
}
