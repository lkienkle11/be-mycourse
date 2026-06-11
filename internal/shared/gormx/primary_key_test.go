package gormx

import "testing"

func TestEnsureStringIDAssignsV7WhenEmpty(t *testing.T) {
	var id string
	if err := EnsureStringID(&id); err != nil {
		t.Fatalf("EnsureStringID() error = %v", err)
	}
	if id == "" {
		t.Fatal("EnsureStringID() left id empty")
	}
}

func TestEnsureStringIDPreservesExisting(t *testing.T) {
	existing := "019eabc6-a34d-7461-b451-cd008e18c1b5"
	id := existing
	if err := EnsureStringID(&id); err != nil {
		t.Fatalf("EnsureStringID() error = %v", err)
	}
	if id != existing {
		t.Fatalf("EnsureStringID() = %q, want %q", id, existing)
	}
}
