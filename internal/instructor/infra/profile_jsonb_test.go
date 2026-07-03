package infra

import (
	"testing"
)

func TestEmptyRejectionHistoryPtrSerializesToJSONArray(t *testing.T) {
	t.Parallel()
	h := emptyRejectionHistoryPtr()
	if h == nil {
		t.Fatal("expected non-nil pointer")
	}
	v, err := h.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	b, ok := v.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", v)
	}
	if string(b) != "[]" {
		t.Fatalf("expected [], got %s", b)
	}
}

func TestRejectionHistoryJSONScanEmptyArray(t *testing.T) {
	t.Parallel()
	var h RejectionHistoryJSON
	if err := (&h).Scan([]byte("[]")); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(h) != 0 {
		t.Fatalf("expected empty slice, got %d", len(h))
	}
}

func TestEnsureApplicationRowDefaultsSetsRejectionHistory(t *testing.T) {
	t.Parallel()
	row := applicationRow{UserID: "user-1"}
	ensureApplicationRowDefaults(&row)
	if row.RejectionHistory == nil {
		t.Fatal("expected rejection history initialized")
	}
	if len(*row.RejectionHistory) != 0 {
		t.Fatalf("expected empty history, got %d", len(*row.RejectionHistory))
	}
}
