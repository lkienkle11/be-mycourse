package utils

import "testing"

func TestPrepareBulkUserIDs(t *testing.T) {
	t.Parallel()
	got := PrepareBulkUserIDs([]string{" u1 ", "u2", "u1", "", "  "})
	if len(got) != 2 {
		t.Fatalf("got = %v", got)
	}
	if got[0] != "u1" || got[1] != "u2" {
		t.Fatalf("got = %v", got)
	}
}

func TestPrepareBulkUserIDsEmpty(t *testing.T) {
	t.Parallel()
	got := PrepareBulkUserIDs(nil)
	if len(got) != 0 {
		t.Fatalf("got = %v", got)
	}
}
