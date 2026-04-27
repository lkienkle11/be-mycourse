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
