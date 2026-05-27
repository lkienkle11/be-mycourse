package delivery

import (
	"testing"

	"mycourse-io-be/internal/shared/constants"
)

func TestToFilterDomain_setsSearchAndDefaults(t *testing.T) {
	t.Parallel()
	q := FileFilterRequest{
		Page:      0,
		PerPage:   0,
		Search:    "  intro  ",
		SortBy:    "",
		SortOrder: "",
	}

	got := toFilterDomain(q)
	if got.Page != 1 {
		t.Fatalf("Page = %d, want 1", got.Page)
	}
	if got.PageSize != 20 {
		t.Fatalf("PageSize = %d, want 20", got.PageSize)
	}
	if got.SortBy != "created_at" {
		t.Fatalf("SortBy = %q, want created_at", got.SortBy)
	}
	if got.SortOrder != "desc" {
		t.Fatalf("SortOrder = %q, want desc", got.SortOrder)
	}
	if got.Search != "intro" {
		t.Fatalf("Search = %q, want intro", got.Search)
	}
}

func TestToFilterDomain_categoryForcesKind(t *testing.T) {
	t.Parallel()
	image := "image"
	video := "video"
	doc := "document"

	gotImage := toFilterDomain(FileFilterRequest{Category: &image})
	if gotImage.Kind == nil || *gotImage.Kind != constants.FileKindFile {
		t.Fatalf("image kind = %v, want FILE", gotImage.Kind)
	}

	gotDoc := toFilterDomain(FileFilterRequest{Category: &doc})
	if gotDoc.Kind == nil || *gotDoc.Kind != constants.FileKindFile {
		t.Fatalf("document kind = %v, want FILE", gotDoc.Kind)
	}

	gotVideo := toFilterDomain(FileFilterRequest{Category: &video})
	if gotVideo.Kind == nil || *gotVideo.Kind != constants.FileKindVideo {
		t.Fatalf("video kind = %v, want VIDEO", gotVideo.Kind)
	}
}
