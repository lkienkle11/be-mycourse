package slug

import gosimpleslug "github.com/gosimple/slug"

// Kind identifies which entity an auto-generated slug base is for, selecting
// the literal fallback prefix GenerateAutoSlugBase uses when a name
// transliterates to nothing usable.
type Kind string

const (
	KindCourse      Kind = "course"
	KindCourseLevel Kind = "level"
	KindCourseTopic Kind = "topic"
	KindCourseSkill Kind = "skill"
	KindTag         Kind = "tag"
	KindTreeNode    Kind = "node"
)

// GenerateAutoSlugBase derives a base slug from name via gosimple/slug
// (backed by gosimple/unidecode, which transliterates CJK, Cyrillic, Thai,
// Vietnamese, and most other scripts to ASCII rather than dropping them),
// truncated to maxLen BEFORE format validation — a long or multi-byte name
// can transliterate to something far longer than the original. An empty or
// still-format-invalid result after truncation (e.g. an emoji-only name,
// which gosimple/unidecode has no transliteration table for) falls back to
// kind's literal prefix and signals the caller to always suffix it.
//
// Single shared primitive for every entity this codebase derives an
// auto-generated slug for — course, the 4 taxonomy root tables, and tree
// nodes — replacing what were 3 separate near-identical copies
// (course/infra/slug.go's former generateAutoSlugBase,
// taxonomy/infra/slug.go's former generateAutoTaxonomySlugBase,
// shared/taxonomy/tree_write.go's former generateAutoNodeSlugBase), two of
// which wrapped a different name-to-slug mechanism
// (internal/shared/utils.SlugifyName plus an ASCII-fallback) than course's
// own gosimple/slug. Unifying every kind onto gosimple/slug removes that
// divergence; internal/shared/utils.SlugifyName was deleted as dead code
// once taxonomy stopped being its only caller.
func GenerateAutoSlugBase(name string, maxLen int, kind Kind) (base string, mustSuffix bool) {
	candidate := gosimpleslug.Make(name)
	candidate = TruncateToLen(candidate, maxLen)
	if trimmed, ok := ValidateManualFormat(candidate, maxLen); ok {
		return trimmed, false
	}
	return string(kind), true
}
