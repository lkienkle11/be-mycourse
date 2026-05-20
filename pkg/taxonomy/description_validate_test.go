package taxonomy_test

import (
	"strings"
	"testing"

	"mycourse-io-be/pkg/taxonomy"
)

func TestValidateDescriptionParagraphs_ok(t *testing.T) {
	if err := taxonomy.ValidateDescriptionParagraphs([]string{"one", "two"}, 8, 120); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidateDescriptionParagraphs_tooLong(t *testing.T) {
	long := strings.Repeat("x", 121)
	if err := taxonomy.ValidateDescriptionParagraphs([]string{long}, 8, 120); err == nil {
		t.Fatal("expected length error")
	}
}
