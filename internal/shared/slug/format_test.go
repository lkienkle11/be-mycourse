package slug

import (
	"strings"
	"testing"
)

const testMaxLen = 255

func TestValidateManualFormat(t *testing.T) {
	valid := []string{"abc", "abc-123", "abc---123", "a", "123", "golang-course"}
	for _, in := range valid {
		if _, ok := ValidateManualFormat(in, testMaxLen); !ok {
			t.Errorf("ValidateManualFormat(%q) = false, want true", in)
		}
	}

	invalid := []string{
		"-abc", "abc-", "-abc-", "ABC", "abc_123", "abc@123",
		"日本語", "🔥course", "", "   ",
	}
	for _, in := range invalid {
		if trimmed, ok := ValidateManualFormat(in, testMaxLen); ok {
			t.Errorf("ValidateManualFormat(%q) = (%q, true), want ok=false", in, trimmed)
		}
	}
}

func TestValidateManualFormatTrims(t *testing.T) {
	trimmed, ok := ValidateManualFormat("  golang-course  ", testMaxLen)
	if !ok || trimmed != "golang-course" {
		t.Fatalf("got (%q, %v), want (\"golang-course\", true)", trimmed, ok)
	}
}

// TestValidateManualFormatRejectsOverMaxLen guards code-review finding #2:
// update's manual slug had no length check at all before this fix, letting a
// format-valid-but->255-char slug reach the DB and fail with a raw Postgres
// "value too long" error instead of a clean 400.
func TestValidateManualFormatRejectsOverMaxLen(t *testing.T) {
	tooLong := strings.Repeat("a", testMaxLen+1)
	if _, ok := ValidateManualFormat(tooLong, testMaxLen); ok {
		t.Fatalf("expected a %d-char slug to be rejected against maxLen=%d", len(tooLong), testMaxLen)
	}
	exactlyMax := strings.Repeat("a", testMaxLen)
	if _, ok := ValidateManualFormat(exactlyMax, testMaxLen); !ok {
		t.Fatalf("expected a slug of exactly maxLen=%d to be accepted", testMaxLen)
	}
}
