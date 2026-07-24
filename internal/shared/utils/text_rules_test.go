package utils

import "testing"

func TestCountRunes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"ýệ", 2},
		{"ệ", 1},
		{"👋", 1},
		{"a👋b", 3},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := CountRunes(tc.in); got != tc.want {
				t.Fatalf("CountRunes(%q)=%d want %d (bytes=%d)", tc.in, got, tc.want, len(tc.in))
			}
		})
	}
}

func TestCountNonWhitespace(t *testing.T) {
	if got := CountNonWhitespace("a b c"); got != 3 {
		t.Fatalf("got %d want 3", got)
	}
	if got := CountNonWhitespace("khóa học 1"); got != 8 {
		t.Fatalf("got %d want 8", got)
	}
}

func TestCountDeltaNonWhitespace(t *testing.T) {
	delta := `{"ops":[{"insert":"Hello course about text here."}]}`
	if got := CountDeltaNonWhitespace(delta); got < 20 {
		t.Fatalf("got %d want >= 20", got)
	}
}
