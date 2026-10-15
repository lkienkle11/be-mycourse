package taxonomy

import (
	"fmt"
	"strings"

	sharedslug "mycourse-io-be/internal/shared/slug"
)

// BuildNodeIndexByID flattens a tree (recursively, all depths) into a map
// keyed by node ID, for write-path preserve-by-id lookups. A nil/empty
// input tree (e.g. a brand-new topic/skill being created) yields an empty
// map, so every node in the incoming write is treated as new.
func BuildNodeIndexByID(nodes []TreeNode) map[string]TreeNode {
	out := map[string]TreeNode{}
	var walk func([]TreeNode)
	walk = func(list []TreeNode) {
		for _, n := range list {
			if id := strings.TrimSpace(n.ID); id != "" {
				out[id] = n
			}
			walk(n.Children)
		}
	}
	walk(nodes)
	return out
}

// ResolveTreeSlugs walks nodes (already locale-synced by the caller — see
// PrepareTreeForWriteWithSlugPolicy) and resolves each node's final slug:
//   - Slug already non-empty (a manual value, pre-validated for format by
//     the delivery layer before this function ever runs — see
//     internal/taxonomy/delivery/handler_helpers.go's toDomainTreeNodes) →
//     used as-is, subject to the in-tree collision check below.
//   - Slug empty AND node.ID matches a node in oldByID with a non-empty
//     slug → preserved unchanged (the existing node's slug, found by id).
//   - Otherwise (empty, new/unrecognized node) → auto-generated from Name
//     via sharedslug.GenerateAutoSlugBase (kind=sharedslug.KindTreeNode) —
//     the same shared primitive course and taxonomy root entities use, kept
//     here as a single call rather than wrapped in a local function, since
//     there is no longer any kind-specific logic left to wrap.
//
// Collisions within the same tree (any two nodes at any depth sharing a
// slug) are auto-resolved: whichever node is walked second gets a random
// suffix appended, escalating length on repeated collision, using
// sharedslug.RetryWithSuffix — the same crypto/rand-based mechanism as
// every other slug feature in this codebase. Walk order is depth-first,
// matching tree_validate.go's walkTree, so behavior is deterministic for a
// given input order.
func ResolveTreeSlugs(nodes []TreeNode, oldByID map[string]TreeNode, kind sharedslug.Kind) ([]TreeNode, error) {
	seenSlug := map[string]struct{}{}
	return resolveTreeSlugsWalk(nodes, oldByID, kind, seenSlug)
}

func resolveTreeSlugsWalk(nodes []TreeNode, oldByID map[string]TreeNode, kind sharedslug.Kind, seenSlug map[string]struct{}) ([]TreeNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		slug, err := resolveOneNodeSlug(n, oldByID, kind, seenSlug)
		if err != nil {
			return nil, err
		}
		children, err := resolveTreeSlugsWalk(n.Children, oldByID, kind, seenSlug)
		if err != nil {
			return nil, err
		}
		out[i] = TreeNode{
			ID:           strings.TrimSpace(n.ID),
			Name:         strings.TrimSpace(n.Name),
			Slug:         slug,
			Translations: cloneNodeTranslations(n.Translations),
			Children:     children,
		}
	}
	return out, nil
}

func resolveOneNodeSlug(n TreeNode, oldByID map[string]TreeNode, kind sharedslug.Kind, seenSlug map[string]struct{}) (string, error) {
	candidate := strings.TrimSpace(n.Slug)
	if candidate == "" {
		if old, ok := oldByID[strings.TrimSpace(n.ID)]; ok && strings.TrimSpace(old.Slug) != "" {
			candidate = strings.TrimSpace(old.Slug)
		} else {
			base, mustSuffix := sharedslug.GenerateAutoSlugBase(n.Name, DefaultMaxSlugLen, kind)
			if mustSuffix {
				return resolveNodeSlugSuffix(base, seenSlug)
			}
			candidate = base
		}
	}
	if _, taken := seenSlug[candidate]; !taken {
		seenSlug[candidate] = struct{}{}
		return candidate, nil
	}
	return resolveNodeSlugSuffix(candidate, seenSlug)
}

func resolveNodeSlugSuffix(base string, seenSlug map[string]struct{}) (string, error) {
	final, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		truncatedBase := sharedslug.TruncateForSuffix(base, DefaultMaxSlugLen, len(suffix))
		if truncatedBase == "" {
			return "", false, fmt.Errorf("tree node slug base %q leaves no room for a %d-character suffix within %d characters", base, len(suffix), DefaultMaxSlugLen)
		}
		candidate := truncatedBase + "-" + suffix
		_, taken := seenSlug[candidate]
		return candidate, !taken, nil
	})
	if err != nil {
		return "", err
	}
	seenSlug[final] = struct{}{}
	return final, nil
}
