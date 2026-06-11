package utils

import "testing"

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
