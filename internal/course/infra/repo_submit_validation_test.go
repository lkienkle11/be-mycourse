package infra

import (
	"errors"
	"testing"

	"mycourse-io-be/internal/course/domain"
)

type outlineCase struct {
	name      string
	outline   []domain.Section
	validate  func(subLesson domain.SubLesson) error
	wantError error
}

func TestValidateOutlineForSubmit(t *testing.T) {
	tests := []outlineCase{
		outlineEmptyCase(),
		outlineSectionWithoutLessonsCase(),
		outlineLessonWithoutItemsCase(),
		outlineInvalidSubLessonCase(),
		outlineValidCase(),
	}
	runOutlineCases(t, tests)
}

func runOutlineCases(t *testing.T, tests []outlineCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutlineForSubmit(tt.outline, tt.validate)
			if tt.wantError == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

func outlineEmptyCase() outlineCase {
	return outlineCase{
		name:      "empty outline",
		outline:   []domain.Section{},
		validate:  func(subLesson domain.SubLesson) error { return nil },
		wantError: domain.ErrCourseSubmitOutlineIncomplete,
	}
}

func outlineSectionWithoutLessonsCase() outlineCase {
	return outlineCase{
		name: "section without lessons",
		outline: []domain.Section{
			{Lessons: []domain.Lesson{}},
		},
		validate:  func(subLesson domain.SubLesson) error { return nil },
		wantError: domain.ErrCourseSubmitOutlineIncomplete,
	}
}

func outlineLessonWithoutItemsCase() outlineCase {
	return outlineCase{
		name: "lesson without sub lessons",
		outline: []domain.Section{
			{
				Lessons: []domain.Lesson{
					{SubLessons: []domain.SubLesson{}},
				},
			},
		},
		validate:  func(subLesson domain.SubLesson) error { return nil },
		wantError: domain.ErrCourseSubmitOutlineIncomplete,
	}
}

func outlineInvalidSubLessonCase() outlineCase {
	return outlineCase{
		name: "invalid sub lesson",
		outline: []domain.Section{
			{
				Lessons: []domain.Lesson{
					{
						SubLessons: []domain.SubLesson{validSubmitSubLesson()},
					},
				},
			},
		},
		validate: func(subLesson domain.SubLesson) error {
			return errors.New("invalid")
		},
		wantError: domain.ErrCourseSubmitInvalidSubLesson,
	}
}

func outlineValidCase() outlineCase {
	return outlineCase{
		name: "valid outline",
		outline: []domain.Section{
			{
				Lessons: []domain.Lesson{
					{
						SubLessons: []domain.SubLesson{validSubmitSubLesson()},
					},
				},
			},
		},
		validate:  func(subLesson domain.SubLesson) error { return nil },
		wantError: nil,
	}
}

func validSubmitSubLesson() domain.SubLesson {
	return domain.SubLesson{
		Kind:      domain.SubLessonKindText,
		IsPreview: false,
		Text: &domain.TextContent{
			ContentDelta: `{"ops":[{"insert":"hello"}]}`,
		},
	}
}
