// Package taxonomy provides shared taxonomy helpers (tree JSONB, description arrays).
package taxonomy

// TreeNode is a recursive node stored in JSONB (child_topics, children).
type TreeNode struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	Children []TreeNode `json:"children,omitempty"`
}
