package utils

import "testing"

func TestClampQueryLimit(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		def  int
		max  int
		want int
	}{
		{"empty falls back to default", "", 8, 24, 8},
		{"invalid falls back to default", "abc", 8, 24, 8},
		{"zero falls back to default", "0", 8, 24, 8},
		{"negative falls back to default", "-5", 8, 24, 8},
		{"within bounds is used as-is", "5", 8, 24, 5},
		{"above max is clamped to max", "100", 8, 24, 24},
		{"exactly max is unchanged", "24", 8, 24, 24},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampQueryLimit(tc.raw, tc.def, tc.max); got != tc.want {
				t.Errorf("ClampQueryLimit(%q, %d, %d) = %d, want %d", tc.raw, tc.def, tc.max, got, tc.want)
			}
		})
	}
}
