package infra_test

import (
	"encoding/json"
	"strings"
	"testing"

	mediadelivery "mycourse-io-be/internal/media/delivery"
	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/setting"
)

func TestIsBunnyWebhookSignatureValid_Valid(t *testing.T) {
	rawBody := []byte(`{"VideoLibraryId":133,"VideoGuid":"657bb740-a71b-4529-a012-528021c31a92","Status":3}`)
	secret := "readonly-key"
	signature := mediainfra.BunnyWebhookSignatureExpectedHex(rawBody, secret)

	ok := mediainfra.IsBunnyWebhookSignatureValid(
		rawBody,
		signature,
		mediadomain.BunnyWebhookSignatureVersionV1,
		mediadomain.BunnyWebhookSignatureAlgorithmHMAC,
		secret,
	)
	if !ok {
		t.Fatal("expected valid signature")
	}
}

func TestIsBunnyWebhookSignatureValid_InvalidCases(t *testing.T) {
	testBunnyInvalidSignature(t)
	testBunnyInvalidVersion(t)
	testBunnyInvalidAlgorithm(t)
	testBunnyModifiedRawBody(t)
}

func testBunnyInvalidSignature(t *testing.T) {
	rawBody := []byte(`{"VideoLibraryId":133,"VideoGuid":"abc","Status":3}`)
	secret := "readonly-key"
	if mediainfra.IsBunnyWebhookSignatureValid(rawBody, strings.Repeat("a", 64), mediadomain.BunnyWebhookSignatureVersionV1, mediadomain.BunnyWebhookSignatureAlgorithmHMAC, secret) {
		t.Fatal("expected invalid signature")
	}
}

func testBunnyInvalidVersion(t *testing.T) {
	rawBody := []byte(`{"VideoLibraryId":133,"VideoGuid":"abc","Status":3}`)
	secret := "readonly-key"
	signature := mediainfra.BunnyWebhookSignatureExpectedHex(rawBody, secret)
	if mediainfra.IsBunnyWebhookSignatureValid(rawBody, signature, "v2", mediadomain.BunnyWebhookSignatureAlgorithmHMAC, secret) {
		t.Fatal("expected invalid version")
	}
}

func testBunnyInvalidAlgorithm(t *testing.T) {
	rawBody := []byte(`{"VideoLibraryId":133,"VideoGuid":"abc","Status":3}`)
	secret := "readonly-key"
	signature := mediainfra.BunnyWebhookSignatureExpectedHex(rawBody, secret)
	if mediainfra.IsBunnyWebhookSignatureValid(rawBody, signature, mediadomain.BunnyWebhookSignatureVersionV1, "sha256", secret) {
		t.Fatal("expected invalid algorithm")
	}
}

func testBunnyModifiedRawBody(t *testing.T) {
	rawBody := []byte(`{"VideoLibraryId":133,"VideoGuid":"abc","Status":3}`)
	secret := "readonly-key"
	signature := mediainfra.BunnyWebhookSignatureExpectedHex(rawBody, secret)
	modifiedBody := []byte(`{"Status":3,"VideoGuid":"abc","VideoLibraryId":133}`)
	if mediainfra.IsBunnyWebhookSignatureValid(modifiedBody, signature, mediadomain.BunnyWebhookSignatureVersionV1, mediadomain.BunnyWebhookSignatureAlgorithmHMAC, secret) {
		t.Fatal("expected modified body to fail")
	}
}

func TestBunnyWebhookSigningSecret_Priority(t *testing.T) {
	prev := *setting.MediaSetting
	t.Cleanup(func() { *setting.MediaSetting = prev })

	setting.MediaSetting.BunnyStreamReadOnlyAPIKey = "read-only-key"
	setting.MediaSetting.BunnyStreamAPIKey = "api-key"
	if got := mediainfra.BunnyWebhookSigningSecret(); got != "read-only-key" {
		t.Fatalf("expected read-only key, got %q", got)
	}

	setting.MediaSetting.BunnyStreamReadOnlyAPIKey = ""
	if got := mediainfra.BunnyWebhookSigningSecret(); got != "api-key" {
		t.Fatalf("expected fallback api key, got %q", got)
	}
}

func TestBunnyWebhookPayload_DTOFields(t *testing.T) {
	raw := []byte(`{"VideoLibraryId":133,"VideoGuid":"657bb740-a71b-4529-a012-528021c31a92","Status":10}`)
	var req mediadelivery.BunnyVideoWebhookRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if req.VideoLibraryID != 133 || req.VideoGUID == "" || req.Status != 10 {
		t.Fatalf("unexpected payload decode: %+v", req)
	}
}
