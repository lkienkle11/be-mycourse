package slug

import (
	"errors"
	"strings"
	"testing"
)

func TestParseManual(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{"empty means auto", "", "", nil},
		{"whitespace means auto", "   ", "", nil},
		{"valid trimmed", "  golang-101 ", "golang-101", nil},
		{"uppercase", "Golang", "", ErrInvalidFormat},
		{"underscore", "a_b", "", ErrInvalidFormat},
		{"leading hyphen", "-abc", "", ErrInvalidFormat},
		{"non ascii", "日本語", "", ErrInvalidFormat},
		{"too long", strings.Repeat("a", 256), "", ErrInvalidFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseManual(tc.raw, 255)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseManual(%q) = (%q, %v), want (%q, %v)", tc.raw, got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestParseUpdateInput(t *testing.T) {
	cases := []struct {
		name    string
		set     bool
		value   string
		want    *string
		wantErr error
	}{
		{"omitted leaves unchanged", false, "", nil, nil},
		{"explicit null or empty rejected", true, "", nil, ErrEmpty},
		{"whitespace rejected", true, "   ", nil, ErrEmpty},
		{"valid trimmed", true, " new-slug ", ptr("new-slug"), nil},
		{"invalid format", true, "Bad Slug", nil, ErrInvalidFormat},
		{"too long", true, strings.Repeat("a", 256), nil, ErrInvalidFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseUpdateInput(tc.set, tc.value, 255)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			switch {
			case tc.want == nil && got != nil:
				t.Fatalf("got %q, want nil", *got)
			case tc.want != nil && (got == nil || *got != *tc.want):
				t.Fatalf("got %v, want %q", got, *tc.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }
