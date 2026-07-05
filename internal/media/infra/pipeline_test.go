package infra_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/google/uuid"

	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/utils"
)

func TestBunnyVideoStatus_StatusString(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   string
	}{
		{name: "queued", status: mediadomain.BunnyQueued, want: "queued"},
		{name: "processing", status: mediadomain.BunnyProcessing, want: "processing"},
		{name: "encoding", status: mediadomain.BunnyEncoding, want: "encoding"},
		{name: "finished", status: mediadomain.BunnyFinished, want: "finished"},
		{name: "resolution_finished", status: mediadomain.BunnyResolutionFinished, want: "resolution_finished"},
		{name: "failed", status: mediadomain.BunnyFailed, want: "failed"},
		{name: "presigned_upload_started", status: mediadomain.BunnyPresignedUploadStarted, want: "presigned_upload_started"},
		{name: "presigned_upload_finished", status: mediadomain.BunnyPresignedUploadFinished, want: "presigned_upload_finished"},
		{name: "presigned_upload_failed", status: mediadomain.BunnyPresignedUploadFailed, want: "presigned_upload_failed"},
		{name: "captions_generated", status: mediadomain.BunnyCaptionsGenerated, want: "captions_generated"},
		{name: "title_or_description_generated", status: mediadomain.BunnyTitleOrDescriptionGenerated, want: "title_or_description_generated"},
		{name: "unknown", status: 999, want: "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mediainfra.BunnyStatusString(tc.status)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestJoinURLPathSegments_noDoubleSlash(t *testing.T) {
	got := utils.JoinURLPathSegments("https://cdn.example.com/", "bucket", "path/to/obj")
	want := "https://cdn.example.com/bucket/path/to/obj"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildPublicURL_R2_publicURLPlusKey(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.R2.PublicURL = "https://cdn.example.com"
	got, err := mediainfra.BuildPublicURL(constants.FileProviderR2, "12345678-photo.webp")
	if err != nil {
		t.Fatalf("BuildPublicURL: %v", err)
	}
	want := "https://cdn.example.com/12345678-photo.webp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveMediaUploadObjectKey_R2UsesEightDigitPrefix(t *testing.T) {
	if g := mediainfra.ResolveMediaUploadObjectKey("", "", "a.webp", constants.FileProviderR2); !regexp.MustCompile(`^\d{8}-`).MatchString(g) {
		t.Fatalf("R2 default key should start with 8 digits, got %q", g)
	}
}

func TestResolveMediaUploadObjectKey_R2UsesUserCodePrefix(t *testing.T) {
	got := mediainfra.ResolveMediaUploadObjectKey("", "UCODE123", "a.webp", constants.FileProviderR2)
	if !regexp.MustCompile(`^UCODE123/\d{8}-`).MatchString(got) {
		t.Fatalf("R2 scoped key should be user_code/8digits-name, got %q", got)
	}
}

func TestBuildObjectStorageKey_eightDigitsAndSanitizedName(t *testing.T) {
	re := regexp.MustCompile(`^\d{8}-[\w.-]+\.mp4$`)
	for range 20 {
		k := mediainfra.BuildObjectStorageKey("My File!.mp4")
		if !re.MatchString(k) {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func TestResolveMediaUploadObjectKey_byProvider(t *testing.T) {
	if g := mediainfra.ResolveMediaUploadObjectKey("", "", "a.mp4", constants.FileProviderBunny); g != "" {
		t.Fatalf("Bunny default key should be empty before GUID, got %q", g)
	}
	if g := mediainfra.ResolveMediaUploadObjectKey("", "", "a.mp4", constants.FileProviderR2); !regexp.MustCompile(`^\d{8}-`).MatchString(g) {
		t.Fatalf("R2 default key should start with 8 digits, got %q", g)
	}
}

func TestBuildMediaFileEntityFromUpload_persistsTypedMetadataKeys(t *testing.T) {
	entity := mediainfra.BuildMediaFileEntityFromUpload(mediadomain.MediaUploadEntityInput{
		Kind:        constants.FileKindVideo,
		Provider:    constants.FileProviderBunny,
		Filename:    "video.mp4",
		ContentType: "video/mp4",
		SizeBytes:   123,
		Uploaded: mediadomain.ProviderUploadResult{
			URL:       "https://iframe.mediadelivery.net/play/650694/guid",
			OriginURL: "https://iframe.mediadelivery.net/play/650694/guid",
			ObjectKey: "guid",
			Metadata: mediadomain.RawMetadata{
				"bunny_video_id": "guid",
				"length":         190.0,
				"width":          1920,
				"height":         1080,
				"framerate":      23.976,
				"video_codec":    "x264",
			},
		},
		CreatedAt: timex.NowUnix(),
		UpdatedAt: timex.NowUnix(),
	})

	if entity.Metadata.DurationSeconds != 190 {
		t.Fatalf("expected typed duration_seconds 190, got %+v", entity.Metadata)
	}
	if entity.Duration != 190 {
		t.Fatalf("expected flat duration 190, got %d", entity.Duration)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(entity.MetadataJSON), &raw); err != nil {
		t.Fatalf("metadata_json should be valid JSON: %v", err)
	}
	if got := raw["duration_seconds"]; got != float64(190) {
		t.Fatalf("expected metadata_json.duration_seconds=190, got %#v in %s", got, entity.MetadataJSON)
	}
	if got := raw["width_bytes"]; got != float64(1920) {
		t.Fatalf("expected metadata_json.width_bytes=1920, got %#v in %s", got, entity.MetadataJSON)
	}
	if got := raw["height_bytes"]; got != float64(1080) {
		t.Fatalf("expected metadata_json.height_bytes=1080, got %#v in %s", got, entity.MetadataJSON)
	}
	if got := raw["fps"]; got != 23.976 {
		t.Fatalf("expected metadata_json.fps=23.976, got %#v in %s", got, entity.MetadataJSON)
	}
}

func TestBuildMediaFileEntityFromUpload_assignsUUIDv7ForNewCreate(t *testing.T) {
	entity := mediainfra.BuildMediaFileEntityFromUpload(mediadomain.MediaUploadEntityInput{
		Kind:          constants.FileKindFile,
		Provider:      constants.FileProviderR2,
		Filename:      "photo.png",
		ContentType:   "image/png",
		SizeBytes:     42,
		GenerateNewID: true,
		Uploaded: mediadomain.ProviderUploadResult{
			URL:       "https://cdn.example/photo.png",
			OriginURL: "https://cdn.example/photo.png",
			ObjectKey: "01USER/12345678-photo.png",
		},
		CreatedAt: timex.NowUnix(),
		UpdatedAt: timex.NowUnix(),
	})
	if entity.ID == "" {
		t.Fatal("expected non-empty id for new create")
	}
	parsed, err := uuid.Parse(entity.ID)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	if parsed.Version() != 7 {
		t.Fatalf("expected UUID v7, got version %d (%s)", parsed.Version(), entity.ID)
	}
}

func TestBuildMediaFileEntityFromUpload_preservesExistingIDOnUpdate(t *testing.T) {
	const existing = "0195f8ac-214f-7e08-b180-6114ea8f09d6"
	entity := mediainfra.BuildMediaFileEntityFromUpload(mediadomain.MediaUploadEntityInput{
		Kind:        constants.FileKindFile,
		Provider:    constants.FileProviderR2,
		Filename:    "photo.png",
		ContentType: "image/png",
		SizeBytes:   42,
		PreserveID:  existing,
		Uploaded: mediadomain.ProviderUploadResult{
			URL:       "https://cdn.example/photo.png",
			OriginURL: "https://cdn.example/photo.png",
			ObjectKey: "01USER/12345678-photo.png",
		},
		CreatedAt: timex.NowUnix(),
		UpdatedAt: timex.NowUnix(),
	})
	if entity.ID != existing {
		t.Fatalf("expected preserved id %q, got %q", existing, entity.ID)
	}
}

// TestGenerateRandomDigits — task 07: length=8, all decimal digits, uniqueness across 20 samples.
func TestGenerateRandomDigits(t *testing.T) {
	const n = 8
	s := utils.GenerateRandomDigits(n)
	if len(s) != n {
		t.Fatalf("expected length %d, got %d (value: %q)", n, len(s), s)
	}
	for i, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("char at index %d is %q — not a digit (value: %q)", i, c, s)
		}
	}
	seen := make(map[string]struct{}, 20)
	for range 20 {
		seen[utils.GenerateRandomDigits(n)] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("20 calls produced only %d distinct value(s) — generator appears broken", len(seen))
	}
}

func TestToUploadFileResponse_videoFieldsFromBunnyFixture(t *testing.T) {
	in := mediadomain.File{
		URL:            "https://iframe.mediadelivery.net/play/123/abc",
		OriginURL:      "https://iframe.mediadelivery.net/play/123/abc",
		ObjectKey:      "abc",
		BunnyVideoID:   "abc",
		BunnyLibraryID: "123",
		VideoID:        "999",
		ThumbnailURL:   "https://cdn.example/thumb.jpg",
		EmbededHTML:    `<iframe src="https://iframe.mediadelivery.net/embed/123/abc"></iframe>`,
		Duration:       157,
		VideoProvider:  "bunny_stream",
		Metadata: mediadomain.UploadFileMetadata{
			MimeType:        "video/mp4",
			WidthBytes:      1920,
			HeightBytes:     1080,
			DurationSeconds: 157.8,
			FPS:             29.97,
		},
	}

	got := mediadelivery.ToUploadFileResponse(in)
	if got.BunnyVideoID != "abc" || got.BunnyLibraryID != "123" || got.Duration != 157 || got.VideoProvider != "bunny_stream" {
		t.Fatalf("unexpected mapped video fields: %+v", got)
	}
	if got.VideoID != "999" || got.ThumbnailURL != "https://cdn.example/thumb.jpg" || got.EmbededHTML != in.EmbededHTML {
		t.Fatalf("unexpected Bunny parity fields: %+v", got)
	}
}

func TestResolveBunnyEmbedURL_playBaseToEmbed(t *testing.T) {
	got := mediainfra.ResolveBunnyEmbedURL("123", "abc", "https://iframe.mediadelivery.net/play")
	want := "https://iframe.mediadelivery.net/embed/123/abc"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestProfileImageFileAcceptable(t *testing.T) {
	if !mediainfra.ProfileImageFileAcceptable("FILE", "image/png", "x.png") {
		t.Fatal("expected png acceptable")
	}
	if mediainfra.ProfileImageFileAcceptable("VIDEO", "image/png", "x.png") {
		t.Fatal("video kind must reject")
	}
	if mediainfra.ProfileImageFileAcceptable("FILE", "image/svg+xml", "x.svg") {
		t.Fatal("svg must reject")
	}
	if !mediainfra.ProfileImageFileAcceptable("FILE", "", "photo.JPG") {
		t.Fatal("jpg extension fallback")
	}
}

func TestToUploadFileResponsePtr_Nil(t *testing.T) {
	if mediadelivery.ToUploadFileResponsePtr(nil) != nil {
		t.Fatal("nil entity -> nil dto")
	}
}

func TestToUploadFileResponsePtr_WidthHeight(t *testing.T) {
	ent := &mediadomain.File{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Kind:      constants.FileKindFile,
		Filename:  "a.png",
		MimeType:  "image/png",
		SizeBytes: 12,
		URL:       "https://cdn.example/a.png",
		Metadata: mediadomain.UploadFileMetadata{
			WidthBytes:  100,
			HeightBytes: 50,
		},
	}
	pub := mediadelivery.ToUploadFileResponsePtr(ent)
	if pub == nil || pub.Metadata.WidthBytes != 100 || pub.Metadata.HeightBytes != 50 {
		t.Fatalf("unexpected public mapping: %+v", pub)
	}
}

func TestSanitizeMetadataURL_rejectsNilPlaceholder(t *testing.T) {
	if got := mediainfra.SanitizeMetadataURL("<nil>"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestResolveBunnyStreamDeliveryURLs(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })
	setting.MediaSetting.BunnyStreamCDNHostname = "vz-test1234-5678.b-cdn.net"

	guid := "f76b2795-ba9c-4330-bd43-73c368e9b8a9"
	lib := "650694"
	cdn := "vz-test1234-5678.b-cdn.net"
	direct := mediainfra.ResolveBunnyDirectPlayURL(lib, guid)
	wantDirect := "https://player.mediadelivery.net/play/650694/f76b2795-ba9c-4330-bd43-73c368e9b8a9"
	if direct != wantDirect {
		t.Fatalf("direct play: got %q want %q", direct, wantDirect)
	}
	hls := mediainfra.ResolveBunnyHLSPlaylistURL(cdn, guid)
	if hls != "https://"+cdn+"/"+guid+"/playlist.m3u8" {
		t.Fatalf("unexpected hls url: %q", hls)
	}
	detail := &mediadomain.BunnyVideoDetail{GUID: guid, ThumbnailFileName: "thumbnail.jpg"}
	file := &mediadomain.File{BunnyLibraryID: lib}
	mediainfra.ApplyBunnyStreamFileColumns(file, detail, lib, "https://iframe.mediadelivery.net/play")
	if file.ThumbnailURL != "https://"+cdn+"/"+guid+"/thumbnail.jpg" {
		t.Fatalf("unexpected thumbnail: %q", file.ThumbnailURL)
	}
	if file.DirectPlayURL != wantDirect {
		t.Fatalf("unexpected direct play on file: %q", file.DirectPlayURL)
	}
}
