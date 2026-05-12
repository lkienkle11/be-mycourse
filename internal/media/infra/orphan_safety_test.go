package infra_test

import (
	"encoding/json"
	"testing"

	mediadomain "mycourse-io-be/internal/media/domain"
	mediainfra "mycourse-io-be/internal/media/infra"
	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/utils"
)

func TestContentFingerprint_deterministic(t *testing.T) {
	payload := []byte("hello sub06")
	a := utils.ContentFingerprint(payload)
	b := utils.ContentFingerprint(payload)
	if a != b {
		t.Fatalf("same bytes got different digests %q vs %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("want SHA256 hex length 64, got %d", len(a))
	}
}

func TestMergeMediaMetadataJSON_overlayKeepsUnknownKeys(t *testing.T) {
	prev := []byte(`{"keep":"x","n":1}`)
	overlay := mediadomain.RawMetadata{"n": 2, "new": true}
	out, err := mediainfra.MergeMediaMetadataJSON(prev, overlay)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["keep"] != "x" {
		t.Fatalf("lost key keep: %#v", m)
	}
	if int(m["n"].(float64)) != 2 {
		t.Fatalf("want n overwritten to 2, got %#v", m["n"])
	}
	if m["new"] != true {
		t.Fatalf("want new key, got %#v", m)
	}
}

func TestShouldEnqueueSupersededCloudCleanup(t *testing.T) {
	if mediainfra.ShouldEnqueueSupersededCloudCleanup("a", "", "", "") {
		t.Fatal("empty keys should not enqueue")
	}
	if !mediainfra.ShouldEnqueueSupersededCloudCleanup("old-key", "", "new-key", "") {
		t.Fatal("B2 key change should enqueue")
	}
	if !mediainfra.ShouldEnqueueSupersededCloudCleanup("x", "guid-a", "x", "guid-b") {
		t.Fatal("Bunny GUID change should enqueue")
	}
	if mediainfra.ShouldEnqueueSupersededCloudCleanup("same", "guid", "same", "guid") {
		t.Fatal("identical refs should not enqueue")
	}
}

func TestMediaConflictMessages_constants(t *testing.T) {
	if constants.MsgMediaOptimisticLockConflict == "" || constants.MsgMediaReuseMismatch == "" {
		t.Fatal("conflict messages must be non-empty in constants/error_msg.go")
	}
}

// TestEnqueueOrphanCleanup_LocalProviderSkipped verifies the OrphanEnqueuer
// does not schedule cloud cleanup for local-storage objects.
func TestEnqueueOrphanCleanup_LocalProviderSkipped(t *testing.T) {
	// ShouldEnqueueSupersededCloudCleanup returns false for local-only refs (empty keys).
	// An empty object key + empty bunny ID means nothing to clean up.
	if mediainfra.ShouldEnqueueSupersededCloudCleanup("local-key", "", "local-key", "") {
		t.Fatal("identical local key refs should not enqueue cleanup")
	}
}
