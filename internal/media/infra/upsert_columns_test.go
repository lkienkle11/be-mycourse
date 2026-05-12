package infra

import (
	"strings"
	"testing"

	"mycourse-io-be/internal/media/domain"
	"mycourse-io-be/internal/shared/constants"
)

// TestBuildUpsertUpdateColumns_persistsWebhookFields verifies that the column
// map used by UpsertByObjectKey contains every field the Bunny webhook
// pipeline needs to write — most importantly `duration` and `metadata_json`.
//
// Regression test for the bug where `duration` in the DB stayed at 0 even
// after Bunny called the webhook with a valid `length`. Root cause: the
// previous `Assign(struct)+FirstOrCreate` GORM pattern silently dropped
// zero/zero-relative fields when generating the UPDATE statement.
func TestBuildUpsertUpdateColumns_persistsWebhookFields(t *testing.T) {
	f := &domain.File{
		ID:           "id-1",
		ObjectKey:    "abc",
		Kind:         constants.FileKindVideo,
		Provider:     constants.FileProviderBunny,
		Filename:     "v.mp4",
		MimeType:     "video/mp4",
		SizeBytes:    1024,
		Status:       constants.FileStatusReady,
		BunnyVideoID: "abc",
		ThumbnailURL: "https://cdn.example/thumb.jpg",
		Duration:     190,
		MetadataJSON: `{"length":190,"duration_seconds":190,"width":1920,"height":1080}`,
	}
	row := fileToRow(f)
	cols := buildUpsertUpdateColumns(row)

	mustHave := []string{
		"kind", "provider", "filename", "mime_type", "size_bytes",
		"url", "origin_url", "status", "b2_bucket_name",
		"bunny_video_id", "bunny_library_id", "video_id",
		"thumbnail_url", "embeded_html", "duration", "video_provider",
		"content_fingerprint", "metadata_json",
	}
	for _, key := range mustHave {
		if _, ok := cols[key]; !ok {
			t.Fatalf("expected column %q in upsert map, missing", key)
		}
	}

	if got, _ := cols["duration"].(int64); got != 190 {
		t.Fatalf("expected duration=190, got %v", cols["duration"])
	}
	if got, _ := cols["status"].(string); got != constants.FileStatusReady {
		t.Fatalf("expected status=%q, got %v", constants.FileStatusReady, cols["status"])
	}
	if got, _ := cols["thumbnail_url"].(string); got != "https://cdn.example/thumb.jpg" {
		t.Fatalf("expected thumbnail_url to be persisted, got %v", cols["thumbnail_url"])
	}
	meta, _ := cols["metadata_json"].([]byte)
	if !strings.Contains(string(meta), `"duration_seconds":190`) {
		t.Fatalf("expected metadata_json to contain duration_seconds:190, got %q", string(meta))
	}
}

// TestBuildUpsertUpdateColumns_zeroValuesStillPresent guards against the
// original regression: even when the incoming row carries a zero/empty value
// (status reset, cleared thumbnail), the key MUST still appear in the map so
// that GORM's map-based UPDATE writes the new value rather than silently
// keeping the old one.
func TestBuildUpsertUpdateColumns_zeroValuesStillPresent(t *testing.T) {
	row := fileToRow(&domain.File{
		ID:        "id-2",
		ObjectKey: "abc",
		Kind:      constants.FileKindFile,
		Provider:  constants.FileProviderLocal,
		// All other fields intentionally left as zero values.
	})
	cols := buildUpsertUpdateColumns(row)

	for _, key := range []string{"duration", "thumbnail_url", "embeded_html", "metadata_json"} {
		if _, ok := cols[key]; !ok {
			t.Fatalf("expected column %q in upsert map even when zero, missing", key)
		}
	}
	if got, _ := cols["duration"].(int64); got != 0 {
		t.Fatalf("expected duration=0, got %v", cols["duration"])
	}
}
