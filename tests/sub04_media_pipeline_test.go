package tests

import (
	"regexp"
	"testing"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/logic/helper"
	"mycourse-io-be/pkg/logic/utils"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
)

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
	got := pkgmedia.BuildPublicURL(constants.FileProviderB2, "/videos/x.mp4")
	want := "https://cdn.example.com/app-media/videos/x.mp4"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildB2ObjectKey_eightDigitsAndSanitizedName(t *testing.T) {
	re := regexp.MustCompile(`^\d{8}-[\w.-]+\.mp4$`)
	for range 20 {
		k := helper.BuildB2ObjectKey("My File!.mp4")
		if !re.MatchString(k) {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func TestResolveMediaUploadObjectKey_byProvider(t *testing.T) {
	if g := helper.ResolveMediaUploadObjectKey("", "a.mp4", constants.FileProviderBunny); g != "" {
		t.Fatalf("Bunny default key should be empty before GUID, got %q", g)
	}
	if g := helper.ResolveMediaUploadObjectKey("", "a.mp4", constants.FileProviderB2); !regexp.MustCompile(`^\d{8}-`).MatchString(g) {
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
	// Collect 20 samples; collision probability per pair is ~1e-7 so ≥2 distinct values is near-certain.
	seen := make(map[string]struct{}, 20)
	for range 20 {
		seen[utils.GenerateRandomDigits(n)] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("20 calls produced only %d distinct value(s) — generator appears broken", len(seen))
	}
}

// TestBuildPublicURL_B2_trailingSlashVariants — task 06: 4 combinations of
// CDN URL (with/without trailing slash) × B2 bucket (with/without trailing slash).
// All four must produce <cdn>/<bucket>/<key> with no double slashes.
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
			got := pkgmedia.BuildPublicURL(constants.FileProviderB2, objectKey)
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
	}
}

// TestBuildPublicURL_B2_emptyBucket — task 06 edge case: no bucket → cdn/<key> only.
func TestBuildPublicURL_B2_emptyBucket(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.GcoreCDNURL = "https://cdn.example.com"
	setting.MediaSetting.B2Bucket = ""
	got := pkgmedia.BuildPublicURL(constants.FileProviderB2, "foo/bar.jpg")
	want := "https://cdn.example.com/foo/bar.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestBuildPublicURL_B2_leadingSlashInKey — task 06 edge case: leading slash stripped from objectKey.
func TestBuildPublicURL_B2_leadingSlashInKey(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.GcoreCDNURL = "https://cdn.example.com"
	setting.MediaSetting.B2Bucket = "bucket"
	got := pkgmedia.BuildPublicURL(constants.FileProviderB2, "/leading/slash.jpg")
	want := "https://cdn.example.com/bucket/leading/slash.jpg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
