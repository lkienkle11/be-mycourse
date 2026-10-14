package taxonomy

import (
	"strings"
	"testing"
)

func TestResolveTreeSlugsAutoGeneratesForNewNodeWithOmittedSlug(t *testing.T) {
	nodes := []TreeNode{{ID: "n1", Name: "Golang Co Ban"}}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if out[0].Slug == "" {
		t.Fatal("expected an auto-generated slug, got empty string")
	}
}

func TestResolveTreeSlugsFallsBackOnUntransliterableName(t *testing.T) {
	nodes := []TreeNode{{ID: "n1", Name: "🔥🔥🔥"}}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if !strings.HasPrefix(out[0].Slug, "node-") {
		t.Fatalf("Slug = %q, want a node-{suffix} fallback", out[0].Slug)
	}
}

func TestResolveTreeSlugsPreservesExistingNodeSlugByIDWhenOmitted(t *testing.T) {
	old := []TreeNode{{ID: "n1", Name: "Old Name", Slug: "manually-chosen-slug"}}
	oldByID := BuildNodeIndexByID(old)
	// Same node id, renamed, slug omitted — must preserve, not recompute from the new name.
	nodes := []TreeNode{{ID: "n1", Name: "Renamed Node"}}
	out, err := ResolveTreeSlugs(nodes, oldByID, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if out[0].Slug != "manually-chosen-slug" {
		t.Fatalf("Slug = %q, want preserved %q", out[0].Slug, "manually-chosen-slug")
	}
}

func TestResolveTreeSlugsUsesManualValueAsIs(t *testing.T) {
	nodes := []TreeNode{{ID: "n1", Name: "Anything", Slug: "explicit-slug"}}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if out[0].Slug != "explicit-slug" {
		t.Fatalf("Slug = %q, want %q", out[0].Slug, "explicit-slug")
	}
}

func TestResolveTreeSlugsAutoResolvesInTreeCollisionAcrossDepths(t *testing.T) {
	// Two nodes at different depths that would derive the same base slug.
	nodes := []TreeNode{
		{ID: "n1", Name: "Basics"},
		{ID: "n2", Name: "Parent", Children: []TreeNode{
			{ID: "n3", Name: "Basics"},
		}},
	}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	first := out[0].Slug
	second := out[1].Children[0].Slug
	if first == second {
		t.Fatalf("expected distinct slugs for colliding nodes, both got %q", first)
	}
	if first != "basics" {
		t.Fatalf("first node Slug = %q, want %q (first claim wins unsuffixed)", first, "basics")
	}
	if !strings.HasPrefix(second, "basics-") {
		t.Fatalf("second node Slug = %q, want a basics-{suffix} auto-resolution", second)
	}
}

func TestResolveTreeSlugsAutoResolvesManualCollisionToo(t *testing.T) {
	nodes := []TreeNode{
		{ID: "n1", Name: "A", Slug: "same-slug"},
		{ID: "n2", Name: "B", Slug: "same-slug"},
	}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if out[0].Slug != "same-slug" {
		t.Fatalf("first node Slug = %q, want %q", out[0].Slug, "same-slug")
	}
	if out[1].Slug == "same-slug" || !strings.HasPrefix(out[1].Slug, "same-slug-") {
		t.Fatalf("second node Slug = %q, want a same-slug-{suffix} auto-resolution", out[1].Slug)
	}
}

func TestResolveTreeSlugsScopedToOneCallOnly(t *testing.T) {
	// Two independent trees (two separate ResolveTreeSlugs calls, mirroring
	// two different topics/skills) may legitimately share a slug — node
	// uniqueness never applies across trees.
	treeA, err := ResolveTreeSlugs([]TreeNode{{ID: "a1", Name: "Basics", Slug: "basics"}}, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	treeB, err := ResolveTreeSlugs([]TreeNode{{ID: "b1", Name: "Basics", Slug: "basics"}}, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if treeA[0].Slug != "basics" || treeB[0].Slug != "basics" {
		t.Fatalf("expected both independent trees to keep slug %q unsuffixed, got %q and %q", "basics", treeA[0].Slug, treeB[0].Slug)
	}
}

func TestResolveTreeSlugsSuffixTruncatesNearMaxLen(t *testing.T) {
	longBase := strings.Repeat("a", DefaultMaxSlugLen)
	nodes := []TreeNode{
		{ID: "n1", Name: "x", Slug: longBase},
		{ID: "n2", Name: "x", Slug: longBase},
	}
	out, err := ResolveTreeSlugs(nodes, nil, "node")
	if err != nil {
		t.Fatalf("ResolveTreeSlugs() error = %v", err)
	}
	if len(out[1].Slug) > DefaultMaxSlugLen {
		t.Fatalf("second node slug length = %d, want <= %d", len(out[1].Slug), DefaultMaxSlugLen)
	}
	if out[1].Slug == longBase {
		t.Fatal("expected the second node's slug to be truncated+suffixed, got the untouched base")
	}
}
