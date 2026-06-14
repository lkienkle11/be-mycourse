package mailtmpl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirModuleRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(wd) })
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestNormalizeLanguageCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ in, want string }{
		{"en", "en"}, {"EN", "en"}, {"vi", "vi"}, {"", "vi"}, {"fr", "vi"},
	} {
		if got := NormalizeLanguageCode(tc.in); got != tc.want {
			t.Fatalf("NormalizeLanguageCode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestInterpolate(t *testing.T) {
	t.Parallel()
	got := Interpolate("Hello, {{displayName}}!", map[string]string{"displayName": "Alice"})
	if got != "Hello, Alice!" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadConfirmAccountTranslations_en(t *testing.T) {
	chdirModuleRoot(t)
	texts, err := LoadConfirmAccountTranslations("en")
	if err != nil {
		t.Fatal(err)
	}
	if texts[KeySubject] == "" || !strings.Contains(texts[KeyGreeting], "{{displayName}}") {
		t.Fatalf("unexpected en translations: %#v", texts)
	}
}

func TestRenderConfirmAccount_vi(t *testing.T) {
	chdirModuleRoot(t)
	html, subject, err := RenderConfirmAccount("vi", "Bình", "https://example.com/confirm")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Xin chào, Bình!") || !strings.Contains(subject, "Xác nhận") {
		t.Fatalf("subject=%q html=%q", subject, html)
	}
}
