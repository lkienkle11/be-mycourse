package taxonomy_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"mycourse-io-be/pkg/taxonomy"
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
