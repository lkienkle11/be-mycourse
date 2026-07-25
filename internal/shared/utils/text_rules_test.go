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
	// JSON without an `ops` array counts as legacy plain text (FE parity).
	if got := CountDeltaNonWhitespace(`{"a":1}`); got != 7 {
		t.Fatalf("json without ops: got %d want 7", got)
	}
}

func TestCountDeltaRunes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: 0},
		{name: "whitespace_only", in: "   ", want: 0},
		{name: "legacy_plain", in: "abc ýệ", want: 6},
		{name: "legacy_trim", in: "  abc  ", want: 3},
		{name: "delta_simple", in: `{"ops":[{"insert":"Hello"}]}`, want: 5},
		{name: "delta_includes_whitespace", in: `{"ops":[{"insert":"a b\n"}]}`, want: 4},
		{name: "delta_multi_op", in: `{"ops":[{"insert":"ab"},{"insert":"cd"}]}`, want: 4},
		{name: "delta_skips_embed_insert", in: `{"ops":[{"insert":{"image":"x"}},{"insert":"hi"}]}`, want: 2},
		{name: "invalid_json_fallback", in: "{not-json}", want: 10},
		{name: "json_without_ops_is_legacy_plain", in: `{"a":1}`, want: 7},
		{name: "json_ops_null_is_legacy_plain", in: `{"ops":null}`, want: 12},
		{name: "delta_empty_ops_array_counts_zero", in: `{"ops":[]}`, want: 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := CountDeltaRunes(tc.in); got != tc.want {
				t.Fatalf("CountDeltaRunes(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}
