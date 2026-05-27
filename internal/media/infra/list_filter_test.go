package infra

import (
	"strings"
	"testing"
)

func TestMediaListOrderClause(t *testing.T) {
	t.Parallel()
	tests := []struct {
		sortBy    string
		sortOrder string
		want      string
	}{
		{"", "", "created_at DESC"},
		{"filename", "asc", "filename ASC"},
		{"size_bytes", "desc", "size_bytes DESC"},
		{"updated_at", "ASC", "updated_at ASC"},
		{"invalid", "desc", "created_at DESC"},
		{"created_at", "invalid", "created_at DESC"},
	}
	for _, tt := range tests {
		got := mediaListOrderClause(tt.sortBy, tt.sortOrder)
		if got != tt.want {
			t.Errorf("mediaListOrderClause(%q, %q) = %q, want %q", tt.sortBy, tt.sortOrder, got, tt.want)
		}
	}
}

func TestBuildDocumentCategorySQL_includesExtensions(t *testing.T) {
	t.Parallel()
	sql := buildDocumentCategorySQL()
	for _, ext := range []string{".pdf", ".docx", ".zip", ".gz"} {
		if !strings.Contains(sql, ext) {
			t.Errorf("document SQL missing extension %s: %s", ext, sql)
		}
	}
}

func TestImageCategorySQL_matchesResolverExtensions(t *testing.T) {
	t.Parallel()
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".tif"} {
		if !strings.Contains(imageCategorySQL, ext) {
			t.Errorf("image SQL missing extension %s", ext)
		}
	}
	if !strings.Contains(imageCategorySQL, "image/%") {
		t.Error("image SQL should match image/* mime prefix")
	}
}

func TestMediaFilenameSearchValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		search string
		want   string
		ok     bool
	}{
		{name: "blank", search: "", want: "", ok: false},
		{name: "spaces", search: "   ", want: "", ok: false},
		{name: "trimmed", search: "  intro  ", want: "%intro%", ok: true},
		{name: "normal", search: "slide", want: "%slide%", ok: true},
	}
	for _, tt := range tests {
		got, ok := mediaFilenameSearchValue(tt.search)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("%s: mediaFilenameSearchValue(%q) = (%q, %v), want (%q, %v)", tt.name, tt.search, got, ok, tt.want, tt.ok)
		}
	}
}
