package infra

import (
	"testing"

	"mycourse-io-be/internal/course/domain"
)

func TestValidateQuizSubLesson_correctAnswerRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		quiz    *domain.QuizContent
		wantErr error
	}{
		{
			name: "single choice with multiple correct answers",
			quiz: &domain.QuizContent{
				Prompt:        "Pick one",
				AllowMultiple: false,
				Options: []domain.QuizOption{
					{Body: "A", IsCorrect: true},
					{Body: "B", IsCorrect: true},
				},
			},
			wantErr: domain.ErrCourseQuizSingleChoiceMultipleCorrect,
		},
		{
			name: "single choice with one correct answer",
			quiz: &domain.QuizContent{
				Prompt:        "Pick one",
				AllowMultiple: false,
				Options: []domain.QuizOption{
					{Body: "A", IsCorrect: true},
					{Body: "B", IsCorrect: false},
				},
			},
		},
		{
			name: "multiple choice with multiple correct answers",
			quiz: &domain.QuizContent{
				Prompt:        "Pick many",
				AllowMultiple: true,
				Options: []domain.QuizOption{
					{Body: "A", IsCorrect: true},
					{Body: "B", IsCorrect: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateQuizSubLesson(tt.quiz)
			if err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
