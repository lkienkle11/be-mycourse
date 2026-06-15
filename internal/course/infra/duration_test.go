package infra

import (
	"testing"

	"mycourse-io-be/internal/course/domain"
)

func TestResolveSubLessonEstimatedDurationMs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sub     domain.SubLesson
		mediaMs int64
		want    int64
	}{
		{
			name:    "video uses media duration",
			sub:     domain.SubLesson{Kind: domain.SubLessonKindVideo},
			mediaMs: 125_000,
			want:    125_000,
		},
		{
			name:    "video missing media is zero",
			sub:     domain.SubLesson{Kind: domain.SubLessonKindVideo},
			mediaMs: 0,
			want:    0,
		},
		{
			name:    "text uses stored column",
			sub:     domain.SubLesson{Kind: domain.SubLessonKindText, EstimatedDurationMs: 90_000},
			mediaMs: 999_000,
			want:    90_000,
		},
		{
			name:    "quiz uses stored column",
			sub:     domain.SubLesson{Kind: domain.SubLessonKindQuiz, EstimatedDurationMs: 60_000},
			want:    60_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveSubLessonEstimatedDurationMs(tt.sub, tt.mediaMs)
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestNormalizeSubLessonEstimatedDurationMs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    string
		inputMs int64
		want    int64
		wantErr bool
	}{
		{name: "video ignores manual input", kind: domain.SubLessonKindVideo, inputMs: 60_000, want: 0},
		{name: "text persists valid ms", kind: domain.SubLessonKindText, inputMs: 5_130_000, want: 5_130_000},
		{name: "quiz rejects negative", kind: domain.SubLessonKindQuiz, inputMs: -1, wantErr: true},
		{name: "text rejects over max", kind: domain.SubLessonKindText, inputMs: maxEstimatedDurationMs + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeSubLessonEstimatedDurationMs(tt.kind, tt.inputMs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestApplyOutlineEstimatedDurations(t *testing.T) {
	t.Parallel()

	sections := []domain.Section{{
		Lessons: []domain.Lesson{{
			SubLessons: []domain.SubLesson{
				{
					Kind:                domain.SubLessonKindVideo,
					EstimatedDurationMs: 0,
					Video:               &domain.VideoContent{MediaFileID: "vid-1"},
				},
				{
					Kind:                domain.SubLessonKindText,
					EstimatedDurationMs: 30_000,
				},
				{
					Kind:                domain.SubLessonKindQuiz,
					EstimatedDurationMs: 20_000,
				},
			},
		}},
	}}

	applyOutlineEstimatedDurations(sections, map[string]int64{"vid-1": 100_000})

	if sections[0].Lessons[0].SubLessons[0].EstimatedDurationMs != 100_000 {
		t.Fatalf("video sub-lesson expected 100000, got %d", sections[0].Lessons[0].SubLessons[0].EstimatedDurationMs)
	}
	if sections[0].Lessons[0].SubLessons[1].EstimatedDurationMs != 30_000 {
		t.Fatalf("text sub-lesson expected 30000, got %d", sections[0].Lessons[0].SubLessons[1].EstimatedDurationMs)
	}
	if sections[0].Lessons[0].EstimatedDurationMs != 150_000 {
		t.Fatalf("lesson total expected 150000, got %d", sections[0].Lessons[0].EstimatedDurationMs)
	}
	if sections[0].EstimatedDurationMs != 150_000 {
		t.Fatalf("section total expected 150000, got %d", sections[0].EstimatedDurationMs)
	}
}

func TestMediaDurationSecondsFromStored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration int64
		meta     string
		want     int64
	}{
		{name: "column wins", duration: 240, meta: `{"length":190}`, want: 240},
		{name: "metadata length", duration: 0, meta: `{"length":190}`, want: 190},
		{name: "metadata duration_seconds", duration: 0, meta: `{"duration_seconds":125}`, want: 125},
		{name: "empty metadata", duration: 0, meta: `{}`, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mediaDurationSecondsFromStored(tt.duration, []byte(tt.meta))
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestApplyOutlineEstimatedDurations_missingMediaTreatedAsZero(t *testing.T) {
	t.Parallel()

	sections := []domain.Section{{
		Lessons: []domain.Lesson{{
			SubLessons: []domain.SubLesson{{
				Kind:  domain.SubLessonKindVideo,
				Video: &domain.VideoContent{MediaFileID: "missing"},
			}},
		}},
	}}

	applyOutlineEstimatedDurations(sections, map[string]int64{})

	if sections[0].Lessons[0].SubLessons[0].EstimatedDurationMs != 0 {
		t.Fatalf("expected 0 for missing media, got %d", sections[0].Lessons[0].SubLessons[0].EstimatedDurationMs)
	}
}
