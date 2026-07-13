package taxonomy

import (
	"strings"

	"mycourse-io-be/internal/shared/utils"
)

// NormalizeTreeSlugs derives slug from each node name (recursive).
// Translations maps are preserved (cloned) so write paths can keep locale payloads.
func NormalizeTreeSlugs(nodes []TreeNode) []TreeNode {
	if nodes == nil {
		return nil
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		name := strings.TrimSpace(n.Name)
		out[i] = TreeNode{
			ID:           strings.TrimSpace(n.ID),
			Name:         name,
			Slug:         utils.SlugifyName(name),
			Translations: cloneNodeTranslations(n.Translations),
			Children:     NormalizeTreeSlugs(n.Children),
		}
	}
	return out
}
