// Package taxonomy provides shared taxonomy helpers (tree JSONB, description arrays).
package taxonomy

// NodeTranslation is the per-locale payload stored under TreeNode.Translations.
type NodeTranslation struct {
	Name string `json:"name"`
}

// TreeNode is a recursive node stored in JSONB (child_topics, children).
// Canonical Name/Slug remain identity fields; Translations holds localized names.
type TreeNode struct {
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	Slug         string                     `json:"slug"`
	Translations map[string]NodeTranslation `json:"translations,omitempty"`
	Children     []TreeNode                 `json:"children,omitempty"`
}
