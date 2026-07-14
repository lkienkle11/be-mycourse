package taxonomy_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"mycourse-io-be/internal/shared/taxonomy"
)

func TestValidateTree_ok(t *testing.T) {
	id := uuid.New().String()
	nodes := []taxonomy.TreeNode{{
		ID: id, Name: "Root", Slug: "root",
		Children: []taxonomy.TreeNode{{
			ID: uuid.New().String(), Name: "Child", Slug: "child",
		}},
	}}
	if err := taxonomy.ValidateTree(nodes, taxonomy.ValidateTreeOpts{}); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidateTree_duplicateSlug(t *testing.T) {
	slug := "dup"
	nodes := []taxonomy.TreeNode{
		{ID: uuid.New().String(), Name: "A", Slug: slug},
		{ID: uuid.New().String(), Name: "B", Slug: slug},
	}
	if err := taxonomy.ValidateTree(nodes, taxonomy.ValidateTreeOpts{}); err == nil {
		t.Fatal("expected duplicate slug error")
	}
}

func TestValidateTree_maxDepth12_OK(t *testing.T) {
	nodes := []taxonomy.TreeNode{buildTreeNodeWithDepth(1, taxonomy.DefaultMaxTreeDepth)}
	if err := taxonomy.ValidateTree(nodes, taxonomy.ValidateTreeOpts{}); err != nil {
		t.Fatalf("expected max depth %d to be valid, got %v", taxonomy.DefaultMaxTreeDepth, err)
	}
}

func TestValidateTree_depth13_Fail(t *testing.T) {
	nodes := []taxonomy.TreeNode{buildTreeNodeWithDepth(1, taxonomy.DefaultMaxTreeDepth+1)}
	if err := taxonomy.ValidateTree(nodes, taxonomy.ValidateTreeOpts{}); err == nil {
		t.Fatalf("expected depth %d to fail validation", taxonomy.DefaultMaxTreeDepth+1)
	}
}

func TestValidateTree_translationNameTooLong(t *testing.T) {
	long := ""
	for i := 0; i < taxonomy.DefaultMaxNameLen+1; i++ {
		long += "x"
	}
	nodes := []taxonomy.TreeNode{{
		ID:   uuid.New().String(),
		Name: "Root",
		Slug: "root",
		Translations: map[string]taxonomy.NodeTranslation{
			"vi": {Name: long},
		},
	}}
	if err := taxonomy.ValidateTree(nodes, taxonomy.ValidateTreeOpts{}); err == nil {
		t.Fatal("expected long translation name to fail")
	}
}

func buildTreeNodeWithDepth(level, maxDepth int) taxonomy.TreeNode {
	node := taxonomy.TreeNode{
		ID:   uuid.New().String(),
		Name: fmt.Sprintf("Node %d", level),
		Slug: fmt.Sprintf("node-%d", level),
	}
	if level < maxDepth {
		node.Children = []taxonomy.TreeNode{buildTreeNodeWithDepth(level+1, maxDepth)}
	}
	return node
}
