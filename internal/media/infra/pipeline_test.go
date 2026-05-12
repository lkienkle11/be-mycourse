package infra_test

import (
	"regexp"
	"testing"

	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
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

func TestBuildPublicURL_B2_includesBucketInPath(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.GcoreCDNURL = "https://cdn.example.com"
	setting.MediaSetting.B2Bucket = "app-media"
	got := mediainfra.BuildPublicURL(constants.FileProviderB2, "/videos/x.mp4")
	want := "https://cdn.example.com/app-media/videos/x.mp4"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildB2ObjectKey_eightDigitsAndSanitizedName(t *testing.T) {
	re := regexp.MustCompile(`^\d{8}-[\w.-]+\.mp4$`)
	for range 20 {
		k := mediainfra.BuildB2ObjectKey("My File!.mp4")
		if !re.MatchString(k) {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func TestResolveMediaUploadObjectKey_byProvider(t *testing.T) {
	if g := mediainfra.ResolveMediaUploadObjectKey("", "a.mp4", constants.FileProviderBunny); g != "" {
		t.Fatalf("Bunny default key should be empty before GUID, got %q", g)
	}
	if g := mediainfra.ResolveMediaUploadObjectKey("", "a.mp4", constants.FileProviderB2); !regexp.MustCompile(`^\d{8}-`).MatchString(g) {
		t.Fatalf("B2 default key should start with 8 digits, got %q", g)
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

func TestBuildPublicURL_B2_trailingSlashVariants(t *testing.T) {
	const objectKey = "videos/sample.mp4"
	const want = "https://cdn.example.com/my-bucket/videos/sample.mp4"

	cases := []struct {
		name   string
		cdnURL string
		bucket string
	}{
		{"cdn_no_slash_bucket_no_slash", "https://cdn.example.com", "my-bucket"},
		{"cdn_trailing_slash_bucket_no_slash", "https://cdn.example.com/", "my-bucket"},
		{"cdn_no_slash_bucket_trailing_slash", "https://cdn.example.com", "my-bucket/"},
		{"cdn_trailing_slash_bucket_trailing_slash", "https://cdn.example.com/", "my-bucket/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prev := *setting.MediaSetting
			t.Cleanup(func() { *setting.MediaSetting = prev })

			setting.MediaSetting.GcoreCDNURL = tc.cdnURL
			setting.MediaSetting.B2Bucket = tc.bucket
			got := mediainfra.BuildPublicURL(constants.FileProviderB2, objectKey)
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
	}
}

func TestBuildPublicURL_B2_emptyBucket(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.GcoreCDNURL = "https://cdn.example.com"
	setting.MediaSetting.B2Bucket = ""
	got := mediainfra.BuildPublicURL(constants.FileProviderB2, "foo/bar.jpg")
	want := "https://cdn.example.com/foo/bar.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildPublicURL_B2_leadingSlashInKey(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.GcoreCDNURL = "https://cdn.example.com"
	setting.MediaSetting.B2Bucket = "bucket"
	got := mediainfra.BuildPublicURL(constants.FileProviderB2, "/leading/slash.jpg")
	want := "https://cdn.example.com/bucket/leading/slash.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
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
