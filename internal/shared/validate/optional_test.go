package validate

import (
	"encoding/json"
	"testing"
)

// TestOptionalDistinguishesOmittedFromExplicitNull is the core guarantee this
// type exists for: a plain *string cannot tell these two cases apart with
// encoding/json (verified separately against a *string field — both leave it
// nil), which was a real bug found by code review in
// updateBasicInfoRequest.Slug before this type replaced *string there.
func TestOptionalDistinguishesOmittedFromExplicitNull(t *testing.T) {
	type req struct {
		Slug Optional[string] `json:"slug"`
	}

	var omitted req
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if omitted.Slug.Set {
		t.Fatal("expected Set=false when the key is omitted")
	}

	var explicitNull req
	if err := json.Unmarshal([]byte(`{"slug":null}`), &explicitNull); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !explicitNull.Slug.Set {
		t.Fatal("expected Set=true for an explicit null (this is the bug fix: omitted and null must differ)")
	}
	if explicitNull.Slug.Value != "" {
		t.Fatalf("expected zero value for explicit null, got %q", explicitNull.Slug.Value)
	}

	var explicitValue req
	if err := json.Unmarshal([]byte(`{"slug":"golang-course"}`), &explicitValue); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !explicitValue.Slug.Set || explicitValue.Slug.Value != "golang-course" {
		t.Fatalf("got Set=%v Value=%q, want Set=true Value=\"golang-course\"", explicitValue.Slug.Set, explicitValue.Slug.Value)
	}
}

func TestOptionalInvalidJSONPropagatesError(t *testing.T) {
	type req struct {
		Count Optional[int] `json:"count"`
	}
	var r req
	if err := json.Unmarshal([]byte(`{"count":"not-a-number"}`), &r); err == nil {
		t.Fatal("expected an error unmarshaling a string into Optional[int]")
	}
}
