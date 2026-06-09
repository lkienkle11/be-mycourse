package utils

import (
	"sort"
	"strings"
)

// SameStringSet reports whether two string slices contain the same values.
// Order is ignored.
func SameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

// UniqueUint removes duplicated ids while preserving first-seen order.
func UniqueUint(ids []uint) []uint {
	return uniqueBy(ids, func(id uint) uint { return id })
}

// UniqueString removes duplicated ids while preserving first-seen order.
func UniqueString(ids []string) []string {
	return uniqueBy(ids, func(id string) string { return id })
}

func uniqueBy[T any, K comparable](items []T, keyFn func(T) K) []T {
	set := make(map[K]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		key := keyFn(item)
		if _, ok := set[key]; ok {
			continue
		}
		set[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

// NilIfBlank returns nil for blank values; otherwise the trimmed string.
func NilIfBlank(v string) any {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// NilIfZeroUint returns nil when pointer is nil or points to zero.
func NilIfZeroUint(v *uint) any {
	if v == nil || *v == 0 {
		return nil
	}
	return *v
}

// NormalizeJSON returns fallback when the json-ish input is blank.
func NormalizeJSON(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
