package taxonomy_test

import (
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
