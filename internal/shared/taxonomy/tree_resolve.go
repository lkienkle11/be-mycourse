package taxonomy

import (
	"strings"

	"mycourse-io-be/internal/shared/i18n"
)

// ResolveNodeName applies locale fallback on a single tree node (exact → base → en → canonical name).
func ResolveNodeName(node TreeNode, requestedLocale string) (name string, resolvedLocale string) {
	tr := make(map[string]string, len(node.Translations))
	for loc, nt := range node.Translations {
		tr[loc] = nt.Name
	}
	return i18n.ResolveText(requestedLocale, tr, node.Name)
}

// ResolveTreeNames returns a deep copy of the tree with Name fields resolved for display.
// Canonical Name is preserved in the resolved output as the display name; Translations are cleared
// on the public/localized shape (callers that need edit DTO should keep the original tree).
func ResolveTreeNames(nodes []TreeNode, requestedLocale string) []TreeNode {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		name, _ := ResolveNodeName(n, requestedLocale)
		out[i] = TreeNode{
			ID:       n.ID,
			Name:     name,
			Slug:     n.Slug,
			Children: ResolveTreeNames(n.Children, requestedLocale),
		}
	}
	return out
}

// EnsureEnTranslation mirrors canonical Name into translations.en when en is missing/empty.
// Does not overwrite an existing non-empty translations.en.name.
func EnsureEnTranslation(nodes []TreeNode) []TreeNode {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		tr := cloneNodeTranslations(n.Translations)
		en := strings.TrimSpace(tr[i18n.DefaultLocale].Name)
		if en == "" {
			tr[i18n.DefaultLocale] = NodeTranslation{Name: strings.TrimSpace(n.Name)}
		}
		out[i] = TreeNode{
			ID:           n.ID,
			Name:         n.Name,
			Slug:         n.Slug,
			Translations: tr,
			Children:     EnsureEnTranslation(n.Children),
		}
	}
	return out
}

// SyncCanonicalAndEn applies write sync rules for one node:
// - both set and different → conflict=true
// - only canonical → mirror to en
// - only en → mirror to canonical
// - both equal → keep
func SyncCanonicalAndEn(canonicalName string, translations map[string]NodeTranslation) (name string, out map[string]NodeTranslation, conflict bool) {
	out = cloneNodeTranslations(translations)
	canon := strings.TrimSpace(canonicalName)
	en := strings.TrimSpace(out[i18n.DefaultLocale].Name)
	switch {
	case canon != "" && en != "" && canon != en:
		return canon, out, true
	case canon != "" && en == "":
		out[i18n.DefaultLocale] = NodeTranslation{Name: canon}
		return canon, out, false
	case canon == "" && en != "":
		out[i18n.DefaultLocale] = NodeTranslation{Name: en}
		return en, out, false
	default:
		if canon != "" {
			out[i18n.DefaultLocale] = NodeTranslation{Name: canon}
		}
		return canon, out, false
	}
}

// PrepareTreeForWrite syncs each node's canonical↔en, ensures en translation, then slugifies.
// Returns conflict=true when any node has canonical/en mismatch.
func PrepareTreeForWrite(nodes []TreeNode) (out []TreeNode, conflict bool) {
	if nodes == nil {
		return nil, false
	}
	synced, conflict := syncTreeCanonicalAndEn(nodes)
	if conflict {
		return nil, true
	}
	return NormalizeTreeSlugs(EnsureEnTranslation(synced)), false
}

func syncTreeCanonicalAndEn(nodes []TreeNode) ([]TreeNode, bool) {
	if len(nodes) == 0 {
		return nodes, false
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		name, tr, conflict := SyncCanonicalAndEn(n.Name, n.Translations)
		if conflict {
			return nil, true
		}
		children, conflict := syncTreeCanonicalAndEn(n.Children)
		if conflict {
			return nil, true
		}
		out[i] = TreeNode{
			ID:           strings.TrimSpace(n.ID),
			Name:         name,
			Slug:         n.Slug,
			Translations: tr,
			Children:     children,
		}
	}
	return out, false
}

func cloneNodeTranslations(in map[string]NodeTranslation) map[string]NodeTranslation {
	if len(in) == 0 {
		return map[string]NodeTranslation{}
	}
	out := make(map[string]NodeTranslation, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
