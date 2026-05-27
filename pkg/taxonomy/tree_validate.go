package taxonomy

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	DefaultMaxTreeDepth = 12
	DefaultMaxTreeNodes = 100
	DefaultMaxNameLen   = 255
	DefaultMaxSlugLen   = 255
)

// ValidateTreeOpts tunes tree validation limits.
type ValidateTreeOpts struct {
	MaxDepth   int
	MaxNodes   int
	MaxNameLen int
	MaxSlugLen int
}

type resolvedTreeOpts struct {
	maxDepth   int
	maxNodes   int
	maxNameLen int
	maxSlugLen int
}

func resolveTreeOpts(opts ValidateTreeOpts) resolvedTreeOpts {
	o := resolvedTreeOpts{
		maxDepth:   opts.MaxDepth,
		maxNodes:   opts.MaxNodes,
		maxNameLen: opts.MaxNameLen,
		maxSlugLen: opts.MaxSlugLen,
	}
	if o.maxDepth <= 0 {
		o.maxDepth = DefaultMaxTreeDepth
	}
	if o.maxNodes <= 0 {
		o.maxNodes = DefaultMaxTreeNodes
	}
	if o.maxNameLen <= 0 {
		o.maxNameLen = DefaultMaxNameLen
	}
	if o.maxSlugLen <= 0 {
		o.maxSlugLen = DefaultMaxSlugLen
	}
	return o
}

// ValidateTree checks depth, node count, UUID ids, and duplicate id/slug within the tree.
func ValidateTree(nodes []TreeNode, opts ValidateTreeOpts) error {
	o := resolveTreeOpts(opts)
	seenID := make(map[string]struct{})
	seenSlug := make(map[string]struct{})
	count := 0
	return walkTree(nodes, 1, o, seenID, seenSlug, &count)
}

func walkTree(list []TreeNode, depth int, o resolvedTreeOpts, seenID, seenSlug map[string]struct{}, count *int) error {
	if len(list) == 0 {
		return nil
	}
	if depth > o.maxDepth {
		return errors.New("tree exceeds max depth")
	}
	for _, n := range list {
		*count++
		if *count > o.maxNodes {
			return errors.New("tree exceeds max node count")
		}
		if err := validateTreeNode(n, o, seenID, seenSlug); err != nil {
			return err
		}
		if err := walkTree(n.Children, depth+1, o, seenID, seenSlug, count); err != nil {
			return err
		}
	}
	return nil
}

func validateTreeNode(n TreeNode, o resolvedTreeOpts, seenID, seenSlug map[string]struct{}) error {
	id := strings.TrimSpace(n.ID)
	if id == "" {
		return errors.New("tree node id is required")
	}
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("tree node id must be a valid uuid")
	}
	if _, ok := seenID[id]; ok {
		return errors.New("duplicate tree node id")
	}
	seenID[id] = struct{}{}

	name := strings.TrimSpace(n.Name)
	if name == "" || utf8.RuneCountInString(name) > o.maxNameLen {
		return errors.New("tree node name is invalid")
	}
	slug := strings.TrimSpace(n.Slug)
	if slug == "" || utf8.RuneCountInString(slug) > o.maxSlugLen {
		return errors.New("tree node slug is invalid")
	}
	if _, ok := seenSlug[slug]; ok {
		return errors.New("duplicate tree node slug")
	}
	seenSlug[slug] = struct{}{}
	return nil
}
