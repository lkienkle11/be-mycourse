package delivery

import (
	"encoding/json"
	"testing"

	"mycourse-io-be/internal/taxonomy/domain"
)

func TestToCourseTopicResponse_UsesImageFileURLField(t *testing.T) {
	imageFileID := "550e8400-e29b-41d4-a716-446655440000"
	row := domain.CourseTopic{
		ID:           imageFileID,
		Name:         "Math",
		Slug:         "math",
		ImageFileID:  &imageFileID,
		ImageFileURL: "https://cdn.example.com/math.webp",
		Status:       "ACTIVE",
	}

	resp := toCourseTopicResponse(row)
	if resp.ImageFileID != imageFileID {
		t.Fatalf("expected image_file_id %q, got %q", imageFileID, resp.ImageFileID)
	}
	if resp.ImageFileURL != row.ImageFileURL {
		t.Fatalf("expected image_file_url %q, got %q", row.ImageFileURL, resp.ImageFileURL)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := payload["image_file_url"]; !ok {
		t.Fatal("expected image_file_url in response payload")
	}
	if _, ok := payload["image_url"]; ok {
		t.Fatal("did not expect legacy image_url key in response payload")
	}
}

func TestToCourseOutcomeResponse_UsesImageFileURLField(t *testing.T) {
	imageFileID := "550e8400-e29b-41d4-a716-446655440000"
	row := domain.CourseOutcome{
		ID:               imageFileID,
		ShortDescription: "Think critically",
		ImageFileID:      &imageFileID,
		ImageFileURL:     "https://cdn.example.com/outcome.webp",
		Status:           "ACTIVE",
	}

	resp := toCourseOutcomeResponse(row)
	if resp.ImageFileID != imageFileID {
		t.Fatalf("expected image_file_id %q, got %q", imageFileID, resp.ImageFileID)
	}
	if resp.ImageFileURL != row.ImageFileURL {
		t.Fatalf("expected image_file_url %q, got %q", row.ImageFileURL, resp.ImageFileURL)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := payload["image_file_url"]; !ok {
		t.Fatal("expected image_file_url in response payload")
	}
	if _, ok := payload["image_url"]; ok {
		t.Fatal("did not expect legacy image_url key in response payload")
	}
}
