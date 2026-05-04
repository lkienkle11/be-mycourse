package tests

import (
	"testing"

	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
)

// ---------------------------------------------------------------------------
// ParseImageURLForOrphanCleanup tests
// ---------------------------------------------------------------------------

func TestParseImageURLForOrphanCleanup_Empty(t *testing.T) {
	_, _, _, ok := pkgmedia.ParseImageURLForOrphanCleanup("")
	if ok {
		t.Fatal("empty URL should return ok=false")
	}
}

func TestParseImageURLForOrphanCleanup_Local(t *testing.T) {
	prov, key, bID, ok := pkgmedia.ParseImageURLForOrphanCleanup("/api/v1/media/files/local/abc123token")
	if !ok {
		t.Fatal("local URL should return ok=true")
	}
	if string(prov) != "Local" {
		t.Fatalf("expected provider Local, got %s", prov)
	}
	if key != "" || bID != "" {
		t.Fatal("local URL should have empty objectKey and bunnyVideoID")
	}
}

func TestParseImageURLForOrphanCleanup_External(t *testing.T) {
	_, _, _, ok := pkgmedia.ParseImageURLForOrphanCleanup("https://external.cdn.example.com/images/photo.jpg")
	if ok {
		t.Fatal("external URL should return ok=false")
	}
}

func TestParseImageURLForOrphanCleanup_BunnyStream(t *testing.T) {
	setting.MediaSetting.BunnyStreamBaseURL = "https://iframe.mediadelivery.net/play"
	setting.MediaSetting.BunnyStreamLibraryID = "123456"

	rawURL := "https://iframe.mediadelivery.net/play/123456/abc-def-guid-000"
	prov, key, bID, ok := pkgmedia.ParseImageURLForOrphanCleanup(rawURL)
	if !ok {
		t.Fatalf("bunny URL should be recognised, got ok=false")
	}
	if string(prov) != "Bunny" {
		t.Fatalf("expected provider Bunny, got %s", prov)
	}
	if key != "abc-def-guid-000" {
		t.Fatalf("expected objectKey=abc-def-guid-000, got %s", key)
	}
	if bID != "abc-def-guid-000" {
		t.Fatalf("expected bunnyVideoID=abc-def-guid-000, got %s", bID)
	}
}

func TestParseImageURLForOrphanCleanup_BunnyStream_WithQuery(t *testing.T) {
	setting.MediaSetting.BunnyStreamBaseURL = "https://iframe.mediadelivery.net/play"
	setting.MediaSetting.BunnyStreamLibraryID = "123456"

	rawURL := "https://iframe.mediadelivery.net/play/123456/abc-def-guid-000?token=xyz&expires=99999"
	_, key, _, ok := pkgmedia.ParseImageURLForOrphanCleanup(rawURL)
	if !ok {
		t.Fatal("bunny URL with query params should be recognised")
	}
	if key != "abc-def-guid-000" {
		t.Fatalf("expected objectKey without query, got %s", key)
	}
}

func TestParseImageURLForOrphanCleanup_B2CDN(t *testing.T) {
	setting.MediaSetting.GcoreCDNURL = "https://cdn.mycourse.io"
	setting.MediaSetting.B2Bucket = "mybucket"

	rawURL := "https://cdn.mycourse.io/mybucket/12345678-photo.jpg"
	prov, key, bID, ok := pkgmedia.ParseImageURLForOrphanCleanup(rawURL)
	if !ok {
		t.Fatalf("B2 CDN URL should be recognised, got ok=false")
	}
	if string(prov) != "B2" {
		t.Fatalf("expected provider B2, got %s", prov)
	}
	if key != "12345678-photo.jpg" {
		t.Fatalf("expected objectKey=12345678-photo.jpg, got %s", key)
	}
	if bID != "" {
		t.Fatalf("expected empty bunnyVideoID, got %s", bID)
	}
}

func TestParseImageURLForOrphanCleanup_B2CDN_NoBucketConfigured(t *testing.T) {
	setting.MediaSetting.GcoreCDNURL = "https://cdn.mycourse.io"
	setting.MediaSetting.B2Bucket = "" // not configured → cannot parse

	_, _, _, ok := pkgmedia.ParseImageURLForOrphanCleanup("https://cdn.mycourse.io/mybucket/12345678-photo.jpg")
	if ok {
		t.Fatal("B2 URL without configured bucket should return ok=false")
	}
}

// ---------------------------------------------------------------------------
// ScanJSONBForImageURLs tests
// ---------------------------------------------------------------------------

func TestScanJSONBForImageURLs_Empty(t *testing.T) {
	urls := pkgmedia.ScanJSONBForImageURLs(nil)
	if len(urls) != 0 {
		t.Fatalf("expected 0 URLs from nil, got %d", len(urls))
	}
}

func TestScanJSONBForImageURLs_FlatObject(t *testing.T) {
	raw := []byte(`{"title":"test","cover_url":"https://cdn.mycourse.io/mybucket/cover.jpg","description":"desc"}`)
	urls := pkgmedia.ScanJSONBForImageURLs(raw)
	if len(urls) != 1 || urls[0] != "https://cdn.mycourse.io/mybucket/cover.jpg" {
		t.Fatalf("expected 1 URL, got %v", urls)
	}
}

func TestScanJSONBForImageURLs_Nested(t *testing.T) {
	raw := []byte(`{
		"content": {
			"blocks": [
				{"type":"image","image_url":"https://cdn.example.com/img1.png"},
				{"type":"text","text":"hello"},
				{"type":"image","thumbnail":"https://cdn.example.com/img2.png"}
			]
		}
	}`)
	urls := pkgmedia.ScanJSONBForImageURLs(raw)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d: %v", len(urls), urls)
	}
}

func TestScanJSONBForImageURLs_NoURLKeys(t *testing.T) {
	raw := []byte(`{"title":"hello","count":42,"active":true}`)
	urls := pkgmedia.ScanJSONBForImageURLs(raw)
	if len(urls) != 0 {
		t.Fatalf("expected 0 URLs, got %v", urls)
	}
}
