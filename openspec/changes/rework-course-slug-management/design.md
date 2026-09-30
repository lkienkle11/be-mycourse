## Context

See `proposal.md - Why` for motivation. Current state, verified directly in code (not from docs, which turned out to be stale in places — see "Documentation updates" below):

- `internal/course/application/service.go:courseTitleAndSlug` derives `slug` from `title` via `utils.SlugifyName` (`internal/shared/utils/slug.go`) on **every** create and **every** update where `title` is present — and `title` is `required` on every `PATCH .../basic-info` call (`internal/course/delivery/dto.go:updateBasicInfoRequest`, no pointer fields except `PreviewVideoFileID`), so slug is unconditionally recomputed on every metadata save today.
- `internal/course/infra/repo_helpers.go:ensureUniqueCourseSlug` (+ `listCollidingCourseSlugs`, `nextFreeCourseSlug`, `courseSlugCandidate`, `courseSlugNumericSuffix`, `isCourseSlugNumericSuffixVariant`) implements a numeric-suffix collision strategy (`base`, `base-2`, `base-3`, …) with a single indexed pre-query (`slug = ? OR slug LIKE 'base-%'`) and in-memory suffix selection. `CreateCourse` (`internal/course/infra/repo_instructor.go:81-93`) wraps this in a `courseSlugCreateRetry = 3` loop that catches `isCourseSlugDuplicateKey` and retries the whole transaction. `UpdateBasicInfo` (`repo_instructor.go:154-204`) calls `ensureUniqueCourseSlug` once, with **no** retry-on-duplicate-key wrapper — a real race-safety gap this change closes.
- `utils.SlugifyName` keeps any Unicode letter surviving NFD diacritic-stripping — ASCII-safe for Vietnamese (its diacritics decompose to combining marks that get stripped) but **not** for Chinese/Japanese/Thai/Korean/Russian, which pass through untouched. Verified against the actual `gosimple/unidecode` table data (not assumed): CJK ideographs (21,246 table entries, e.g. `影`→`"Ying "`, `師`→`"Shi "`), Hiragana/Katakana (e.g. `あ`→`"a"`), Thai (e.g. `ก`→`"k"`), and precomposed Hangul (11,263 entries, e.g. `가`→`"ga"`) all have real transliteration entries in `gosimple/slug`'s backing table — so switching to it, rather than patching `SlugifyName`, preserves meaningful non-Latin slugs instead of collapsing every such title to `course-{random}`.
- `courses.slug VARCHAR(255) NOT NULL` + unique index `uix_courses_slug_active` (partial, `WHERE deleted_at IS NULL`) — from `migrations/000016_course_management.up.sql`. No format `CHECK` constraint exists anywhere in this codebase's migrations today (verified: `grep -n "CHECK" migrations/*.up.sql` only matches `000001` (unrelated) and `000034`/`000035` (authorization wildcard, not slug/regex)).
- `internal/shared/response.Fail(c, httpStatus, appCode, message, data any, ...)` already carries an arbitrary `data` payload on every error response — no envelope change needed to carry `recommended_slug`.
- `internal/shared/errors` (import alias `apperrors`) is a flat numeric app-code registry (`Success=0`, 1xxx transport, 2xxx validation, 3xxx client/HTTP-shaped, 4xxx auth, 9xxx server) with a `DefaultMessage(code) string` lookup — adding one constant + one map entry is the established way to add a new error kind; no per-domain namespacing exists.
- `internal/shared/validate` wraps `go-playground/validator/v10` with 2 registered custom tags (`nonwhitespace_min`, `delta_nonwhitespace_min`) — an earlier draft of this design planned a third tag (`course_slug`) here, but D2 below explains why it was dropped (an `omitempty`-on-plain-string bug plus redundancy with `sharedslug.ValidateManualFormat` already called directly in Go code); this package is **not modified** by this change.
- `docs/modules/taxonomy.md:58` documents an explicit reuse-map rule: "Slug → `utils.SlugifyName` + `tree_slug.go` — Do not create: Second slug util." Adding `github.com/gosimple/slug` appears to conflict with this rule on its face; **Decision D3** below explains why it doesn't (taxonomy keeps using `SlugifyName` unchanged — this rule is about not forking taxonomy's own slug derivation, not a blanket ban on any new slug library anywhere in the repo — but the doc update must make this explicit so a future reader doesn't see two slug mechanisms and assume one is a mistake).
- `internal/taxonomy/*` was investigated as a potential source of reusable slug machinery and found to offer **less** than course already has: name-only auto-derive (same `SlugifyName`), zero manual-input path, zero collision retry (raw DB unique-violation error only, no numeric-suffix fallback even), and no dedicated format validator. There is nothing in taxonomy to reuse; the new `internal/shared/slug` package is written from scratch, informed only by course's own (superior) numeric-suffix precedent and this proposal's random-suffix requirement.

## Goals / Non-Goals

**Goals:**
- Course slug becomes independently settable by the caller on create (optional) and update (optional-but-strict), while keeping the DB the final authority on uniqueness and format.
- Extract the domain-agnostic parts of slug handling (format validation, random-suffix generation, escalating-retry orchestration) into `internal/shared/slug`, usable without modification by a future taxonomy/media/instructor slug feature.
- Close the update-path race-safety gap (`UpdateBasicInfo` currently has no retry-on-duplicate-key).
- Guarantee `courses.slug` is never `NULL`/empty/malformed even under an application-layer bug, via a DB `CHECK` constraint.

**Non-Goals:**
- Instructor slug, taxonomy/media/level/outcome slug rework — explicitly deferred by the user; this change only builds the shared primitive they will later depend on, and wires it into `course` only.
- `course_slug_history` / 301-redirect-on-old-slug — optional per the user's own original spec; tracked as a candidate follow-up change, not built here.
- Any frontend change (explicitly backend-only per the user's instruction for this change).
- Changing taxonomy's own slug behavior (`SlugifyName`, `tree_slug.go`) — untouched.

## Decisions

### D1 — New package `internal/shared/slug`

Holds only domain-agnostic primitives; no course/GORM/HTTP imports. Two files:

`internal/shared/slug/format.go`:
```go
package slug

import (
	"regexp"
	"strings"
)

var formatPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// SUPERSEDED by D10 (code review Bug 2): this signature is missing a maxLen
// parameter — see D10 for the corrected `ValidateManualFormat(raw string,
// maxLen int)`. Shown here only for historical context of the original design.
//
// ValidateManualFormat trims raw and checks it against the shared slug format
// (lowercase ASCII a-z0-9-, no leading/trailing hyphen). It does not check
// uniqueness — callers own that against their own table. An empty/whitespace-
// only input after trimming is reported the same as a malformed input: the
// caller decides whether "no slug supplied" is valid for its own flow (course
// create treats it as "generate one"; course update treats it as an error).
func ValidateManualFormat(raw string) (trimmed string, ok bool) {
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" || !formatPattern.MatchString(trimmed) {
		return "", false
	}
	return trimmed, true
}
```

`internal/shared/slug/suffix.go`:
```go
package slug

import (
	"crypto/rand"
	"io"
	"strings"
	"unicode/utf8"
)

const randomSuffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateRandomSuffix returns n random lowercase-ASCII-letter/digit
// characters using crypto/rand (never math/rand — mirrors the existing
// internal/shared/utils.GenerateRandomDigits precedent). It returns an error
// instead of panicking on RNG failure — a web backend must not crash a
// request just because the entropy source had a transient problem — and uses
// rejection sampling instead of a plain `% len(alphabet)` to avoid the small
// modulo bias that `256 % 36 != 0` would otherwise introduce.
func GenerateRandomSuffix(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	const alphabetLen = byte(len(randomSuffixAlphabet)) // 36
	const maxUnbiased = 252                              // largest multiple of 36 that fits in a byte (36*7)
	out := make([]byte, 0, n)
	buf := make([]byte, 1)
	for len(out) < n {
		if _, err := io.ReadFull(rand.Reader, buf); err != nil {
			return "", err
		}
		if buf[0] >= maxUnbiased {
			continue // reject and redraw — keeps the distribution uniform over the 36-letter alphabet
		}
		out = append(out, randomSuffixAlphabet[buf[0]%alphabetLen])
	}
	return string(out), nil
}

const (
	startSuffixLen    = 7
	attemptsPerLength = 7
)

// RetryWithSuffix generates random suffixes of increasing length (7 chars x7
// attempts, then 8x7, 9x7, ... — no upper bound) and calls build with each
// freshly generated suffix. build is responsible for constructing the actual
// candidate string — so it can apply any domain-specific rule such as a
// maximum total length by truncating its own base before concatenating (see
// TruncateForSuffix) — and for reporting whether that candidate is usable.
// This package never concatenates base+suffix itself and has no opinion on
// column length limits, keeping it reusable for any future table.
func RetryWithSuffix(build func(suffix string) (candidate string, accepted bool, err error)) (string, error) {
	for length := startSuffixLen; ; length++ {
		for attempt := 0; attempt < attemptsPerLength; attempt++ {
			suffix, err := GenerateRandomSuffix(length)
			if err != nil {
				return "", err
			}
			candidate, ok, err := build(suffix)
			if err != nil {
				return "", err
			}
			if ok {
				return candidate, nil
			}
		}
	}
}

// TruncateToLen trims s to at most maxLen bytes, never splitting a UTF-8
// codepoint, and trims any trailing hyphen the cut may have left (so the
// result still satisfies the "no trailing hyphen" slug format rule after a
// caller re-validates it).
func TruncateToLen(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	cut := s[:maxLen]
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, "-")
}

// TruncateForSuffix trims base so that base + "-" + a suffix of length
// suffixLen fits within maxTotalLen bytes total. Callers pass their own
// column limit (e.g. 255 for courses.slug) — this package has no opinion on
// it, keeping it reusable for a future table with a different limit.
func TruncateForSuffix(base string, maxTotalLen, suffixLen int) string {
	return TruncateToLen(base, maxTotalLen-1-suffixLen) // -1 for the hyphen
}
```

Rationale for the `build func(suffix string) (candidate string, accepted bool, err error)` shape (changed from an earlier draft that took a fixed `base string` and only an `accept` callback): letting the caller build the full candidate — not just accept/reject a pre-built one — is what makes length-safe truncation possible at all, since the safe truncation budget for `base` depends on `suffixLen`, which only `RetryWithSuffix` knows at each iteration (it grows every 7 attempts). A `base string` parameter fixed for the whole call could never express "truncate differently as the suffix gets longer." This also still lets the same function serve three different call sites with three different truth sources (SELECT-based pre-check for the create-conflict *recommendation*; actual write-and-catch for create-confirm and update-resolve) without the shared package knowing about GORM, transactions, `courses`, or its column length at all.

### D2 — (removed) `course_slug` validator tag — reconsidered, not implemented

An earlier draft of this design registered a custom `go-playground/validator` tag (`course_slug`) on `createCourseRequest.Slug`. Two problems surfaced during review, both traced back to the same root cause — this tag was redundant from the start:

1. **The bug that triggered this reconsideration**: `validate:"omitempty,course_slug,max=255"` on a plain (non-pointer) `string` field does not actually skip a whitespace-only value — `go-playground/validator`'s `omitempty` only skips when the field is the Go zero value (`""` exactly); `"   "` is not `""`, so `course_slug` still ran, trimmed it internally to `""`, failed the format regex, and rejected the request with `400` — even though the spec (`specs/course/slug-management/spec.md`, "Course creation accepts an optional manual slug") already correctly requires whitespace-only to be treated as "no manual slug supplied" → auto-generate, not an error.
2. **Once fixing that**, the tag became provably unnecessary: `internal/course/application/service.go:CreateCourse` (D7) already calls `sharedslug.ValidateManualFormat` directly on the trimmed value as its own independent check — the validator-tag layer was duplicating logic that already existed one layer down, and `updateBasicInfoRequest.Slug` never used this tag at all (D6 always did its own explicit Go check, for the same `omitempty`-on-pointer reason).

**Resolution**: `internal/shared/validate/text_rules.go` and `internal/shared/validate/validate.go` are **not modified by this change at all** — remove this file pair from the touched-files list entirely (see updated D9). Format validation for both create and update slugs lives in exactly one place, `internal/shared/slug.ValidateManualFormat`, called directly from Go code: `internal/course/application/service.go` (create) and `internal/course/delivery/handler_instructor.go` (update) — see D6/D7's actual code, which already did this correctly for update and now does the same for create (D6's `createCourseRequest.Slug` tag is simplified to `validate:"omitempty,max=255"` — a harmless DTO-level length ceiling only; `max` does not care about whitespace content, so it never blocks the whitespace-is-auto-generate case).

**Update (D10, unrelated to the tag question above)**: the `internal/shared/validate` *package* did end up gaining one new file, `optional.go` — but for a completely different reason (JSON PATCH-omit-vs-explicit-null decoding, found necessary by code review, not slug format validation). The two specific files this D2 resolution is about remain untouched exactly as stated.

### D3 — `github.com/gosimple/slug` for title→slug auto-generation only

Used exclusively inside the new course auto-generation path (D7); `internal/taxonomy` and `internal/shared/utils.SlugifyName` are **not** touched or replaced. Verified capability (not assumed) via `gosimple/unidecode`'s actual data table: CJK, Kana, Thai, and Hangul all have real substitution entries (see Context). Empty/invalid transliteration output (e.g. an emoji-only title) triggers the `course-{randomSuffix}` fallback per the user's original spec §10 — implemented as: if `slug.Make(title) == ""` or fails `ValidateManualFormat`, treat the base as `"course"` and go straight to `RetryWithSuffix` (skip trying the bare base, since `"course"` alone is a near-certain collision magnet and the spec explicitly says not to try saving it bare).

`docs/modules/taxonomy.md:58`'s "do not create a second slug util" rule is about taxonomy not forking its own derivation logic into a second competing implementation — it does not block an unrelated module (course) from choosing a different, purpose-fit library for its own derivation. The docs update (below) makes this explicit so the two mechanisms don't read as an unresolved inconsistency to a future maintainer.

### D4 — Random-suffix collision algorithm replaces numeric suffixes

`ensureUniqueCourseSlug`, `nextFreeCourseSlug`, `courseSlugCandidate`, `courseSlugNumericSuffix`, `isCourseSlugNumericSuffixVariant`, `listCollidingCourseSlugs`, `isCourseSlugNumericSuffixVariant` in `internal/course/infra/repo_helpers.go` are **removed** (not deprecated in place — nothing else in the codebase calls them per the caller list already enumerated in `proposal.md - Impact`, and keeping dead numeric-suffix code alongside the new random-suffix code would violate this project's own "no duplicate logic" rule). `isCourseSlugDuplicateKey` (the duplicate-key **error detector**, distinct from the suffix algorithm) is kept as-is — it is still needed to recognize a unique-violation on `courses.slug` regardless of which suffix algorithm produced the candidate.

**SUPERSEDED by D10 (code review Bugs 2 & 3)**: the local `maxCourseSlugLen` constant shown below was moved to `domain.MaxSlugLen`, and D10 adds a new `buildSuffixedCandidate` helper to this same file. Shown here for historical context only — see D10 for the final file content.

New `internal/course/infra/slug.go` (replaces the removed functions):
```go
package infra

import (
	"context"
	"strings"

	gosimpleslug "github.com/gosimple/slug"
	"gorm.io/gorm"

	sharedslug "mycourse-io-be/internal/shared/slug"
)

// maxCourseSlugLen mirrors the courses.slug VARCHAR(255) column limit
// (migrations/000016_course_management.up.sql). Kept here, not in
// internal/shared/slug, because that package deliberately has no opinion on
// any specific table's column length (see D1's TruncateForSuffix rationale).
const maxCourseSlugLen = 255

// generateAutoSlugBase derives a base slug from title via gosimple/slug,
// truncated to maxCourseSlugLen BEFORE format validation (a long or
// multi-byte title — e.g. Chinese/Japanese, where each character can
// transliterate to several ASCII characters plus a separating hyphen — can
// produce a transliteration far longer than the original title). An empty or
// still-format-invalid result after truncation falls back to the literal
// base "course" and signals the caller to skip trying it bare (see
// resolveCreateSlug in repo_instructor.go).
func generateAutoSlugBase(title string) (base string, mustSuffix bool) {
	candidate := gosimpleslug.Make(title)
	candidate = sharedslug.TruncateToLen(candidate, maxCourseSlugLen)
	if trimmed, ok := sharedslug.ValidateManualFormat(candidate); ok {
		return trimmed, false
	}
	return "course", true
}

// courseSlugAvailable reports whether slug is free among active courses,
// excluding excludeCourseID (used on update to allow "no-op" resubmission of
// the current slug without treating it as a self-collision).
func courseSlugAvailable(ctx context.Context, db *gorm.DB, slug string, excludeCourseID *string) (bool, error) {
	q := db.WithContext(ctx).Model(&courseRow{}).Where("deleted_at IS NULL").Where("slug = ?", slug)
	if excludeCourseID != nil {
		if id := strings.TrimSpace(*excludeCourseID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}
```
(`domain` import dropped from this file versus an earlier draft — nothing in `slug.go` itself references `domain` anymore now that `ErrCourseInvalidSlug`-on-empty-base handling moved to `resolveCreateSlug`/the service layer; double check during `/opsx:apply` whether it's still needed once the full file is assembled.)

`isCourseSlugDuplicateKey` stays in `repo_helpers.go` unchanged. Exact block to delete from `internal/course/infra/repo_helpers.go` (everything from the `maxCourseSlugLen`/`courseSlugCreateRetry` const block through `truncateCourseSlugBase`, immediately followed in the current file by `isCourseSlugDuplicateKey`, which is KEPT — only stop deleting right before that function):
```go
const (
	maxCourseSlugLen      = 255
	courseSlugCreateRetry = 3
)

// ensureUniqueCourseSlug picks the first available slug among base, base-2, base-3, …
// among active courses. One indexed query loads sibling slugs; suffix selection is in-memory.
// excludeCourseID skips the current course on title/slug updates.
func ensureUniqueCourseSlug(ctx context.Context, db *gorm.DB, baseSlug string, excludeCourseID *string) (string, error) {
	baseSlug = strings.TrimSpace(baseSlug)
	if baseSlug == "" {
		return "", domain.ErrCourseInvalidSlug
	}
	taken, err := listCollidingCourseSlugs(ctx, db, baseSlug, excludeCourseID)
	if err != nil {
		return "", err
	}
	return nextFreeCourseSlug(baseSlug, taken), nil
}

// listCollidingCourseSlugs returns active slugs equal to base or base-<digits> only.
// LIKE base-% is a broad prefilter; non-numeric tails (e.g. base-advanced) are dropped in Go.
func listCollidingCourseSlugs(ctx context.Context, db *gorm.DB, baseSlug string, excludeCourseID *string) (map[string]struct{}, error) {
	var rows []string
	q := db.WithContext(ctx).Model(&courseRow{}).
		Where("deleted_at IS NULL").
		Where("slug = ? OR slug LIKE ?", baseSlug, baseSlug+"-%")
	if excludeCourseID != nil {
		if id := strings.TrimSpace(*excludeCourseID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	if err := q.Pluck("slug", &rows).Error; err != nil {
		return nil, err
	}
	taken := make(map[string]struct{}, len(rows))
	for _, slug := range rows {
		if isCourseSlugNumericSuffixVariant(slug, baseSlug) {
			taken[slug] = struct{}{}
		}
	}
	return taken, nil
}

func isCourseSlugNumericSuffixVariant(slug, base string) bool {
	if slug == base {
		return true
	}
	prefix := base + "-"
	if !strings.HasPrefix(slug, prefix) {
		return false
	}
	suffix := slug[len(prefix):]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func nextFreeCourseSlug(base string, taken map[string]struct{}) string {
	if _, exists := taken[base]; !exists {
		return courseSlugCandidate(base, 1)
	}
	maxSuffix := 1
	for slug := range taken {
		if n, ok := courseSlugNumericSuffix(slug, base); ok && n > maxSuffix {
			maxSuffix = n
		}
	}
	for i := 2; i <= maxSuffix+1; i++ {
		candidate := courseSlugCandidate(base, i)
		if _, exists := taken[candidate]; !exists {
			return candidate
		}
	}
	return courseSlugCandidate(base, maxSuffix+1)
}

// courseSlugNumericSuffix returns n when slug is base-n (n >= 2).
func courseSlugNumericSuffix(slug, base string) (int, bool) {
	prefix := base + "-"
	if !strings.HasPrefix(slug, prefix) {
		return 0, false
	}
	suffix := slug[len(prefix):]
	if suffix == "" {
		return 0, false
	}
	n := 0
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	if n < 2 {
		return 0, false
	}
	return n, true
}

func courseSlugCandidate(base string, attempt int) string {
	if attempt <= 1 {
		return truncateCourseSlugBase(base, maxCourseSlugLen)
	}
	suffix := fmt.Sprintf("-%d", attempt)
	maxBase := maxCourseSlugLen - len(suffix)
	return truncateCourseSlugBase(base, maxBase) + suffix
}

func truncateCourseSlugBase(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return strings.TrimRight(s, "-")
	}
	cut := s[:maxBytes]
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimRight(cut, "-")
}
```
After deletion, verify `fmt` and `unicode/utf8` are still used elsewhere in `repo_helpers.go` before removing their imports — if `courseSlugCandidate`/`truncateCourseSlugBase` were their only callers in this file, the import lines must be removed too or `go build`/`goimports` will fail on an unused import. The old `maxCourseSlugLen = 255` constant in this file is deleted along with the rest of this block — it is **re-declared in the new `internal/course/infra/slug.go`** (see above), not left orphaned; `go build ./internal/course/...` will fail with a duplicate-constant error if both copies are accidentally left in place at once, which is an easy way to self-check this deletion was done correctly.

### D5 — `SLUG_CONFLICT` error contract (create-time manual conflict only)

`internal/course/domain/errors.go` — full file shown (current content is the single `var (...)` block below with 26 sentinels; the new `SlugConflictError` type is appended after it, in the same file, since Go convention in this repo already mixes sentinel `var` blocks and typed error `struct`s in one `errors.go` per bounded context — see `internal/shared/errors`'s `RegistrationEmailRateLimitedError` precedent, in its own small file but same package):
```go
package domain

import "errors"

var (
	ErrCourseNotFound                        = errors.New("course not found")
	ErrCourseVersionNotFound                 = errors.New("course version not found")
	ErrCourseDraftRequired                   = errors.New("course draft is required")
	ErrCourseDraftInReview                   = errors.New("course draft is in review")
	ErrCourseDraftRejectedOnly               = errors.New("only a rejected draft can be reopened")
	ErrCoursePublishedRequired               = errors.New("published course version is required")
	ErrCourseOwnerOnly                       = errors.New("only the course owner can perform this action")
	ErrCourseCollaboratorAccess              = errors.New("course collaborator access is required")
	ErrCourseOptimisticLock                  = errors.New("course resource was modified by another request; refresh and retry")
	ErrCourseLeaseHeldByOtherUser            = errors.New("course resource is currently locked by another user")
	ErrCourseLeaseTokenInvalid               = errors.New("course lease token is invalid")
	ErrCourseInvalidSubLessonKind            = errors.New("invalid course sub-lesson kind")
	ErrCourseInvalidReviewState              = errors.New("course draft is not in a valid state for this review action")
	ErrCourseInvalidOrdering                 = errors.New("ordered stable ids do not match the existing resource set")
	ErrCourseInstructorRequired              = errors.New("course collaborator must be an instructor")
	ErrCourseOwnerCannotBeRemoved            = errors.New("course owner cannot be removed from collaborators")
	ErrCourseEnrollmentNotFound              = errors.New("course enrollment not found")
	ErrCourseProgressVersionAbsent           = errors.New("course version for learner progress is not available")
	ErrCourseInvalidSlug                     = errors.New("course slug is invalid") // MODIFIED wording — see note below
	ErrCourseTitleTooShort                   = errors.New("course title must contain at least 5 non-whitespace characters")
	ErrCoursePreviewNotAllowedForQuiz        = errors.New("quiz lesson items cannot be marked as preview")
	ErrCourseQuizSingleChoiceMultipleCorrect = errors.New("single-choice quiz must have exactly one correct answer")
	ErrCourseSubmitBasicInfoIncomplete       = errors.New("course submit blocked: basic info is incomplete")
	ErrCourseSubmitOutlineIncomplete         = errors.New("course submit blocked: outline must contain at least one section, one lesson, and one item")
	ErrCourseSubmitInvalidSubLesson          = errors.New("course submit blocked: one or more lesson items are invalid")
	ErrCourseSubmitCollaboratorRequired      = errors.New("course submit blocked: at least one collaborator is required")
	ErrCourseCollaboratorInactive            = errors.New("course submit blocked: collaborator must be active and have instructor role")
	ErrCourseTrashed                         = errors.New("course is in trash and cannot be edited or learned")
	ErrCourseNotTrashed                      = errors.New("course is not in trash")
	ErrCourseTrashNotEligible                = errors.New("only approved courses that are not rejected in the current version can be moved to trash")
)

// SlugConflictError is returned when a manually supplied slug on create
// already belongs to another active course. RecommendedSlug is confirmed
// available at the time this error is constructed — but the update flow
// (which never returns this error type at all, see specs/course/slug-management)
// and the create-confirm resubmission both re-check availability at write
// time, never trusting a SELECT result as still valid forever.
type SlugConflictError struct {
	RecommendedSlug string
}

func (e *SlugConflictError) Error() string { return "course slug already exists" }
```
**Note on `ErrCourseInvalidSlug`'s wording**: its old text ("course title must produce a non-empty slug") described only the old title-derived-slug failure mode. It is now used for BOTH a malformed manual slug on create/update AND an explicit empty/whitespace slug on update (see D6/D7), so its message is generalized. This is a **wording-only** change to an existing sentinel — every existing call site that compares by identity (`==`/`errors.Is`) is unaffected; only the literal string changes, which only matters where `.Error()` text is surfaced to a user (`mapCourseError` already sends `err.Error()` as the HTTP message for this case, per `handler_base.go` below, so the user-facing text does change — expected and desired, not a bug).

`internal/shared/errors/errcode_codes.go` — the existing "Client / HTTP-shaped (3xxx)" block gains one new line (full block shown, only `SlugConflict` is new):
```go
	// Client / HTTP-shaped (3xxx) — align loosely with HTTP family
	BadRequest      = 3001
	Unauthorized    = 3002
	Forbidden       = 3003
	NotFound        = 3004
	Conflict        = 3005
	TooManyRequests = 3006
	SlugConflict    = 3007 // NEW — manual slug already exists on create (see domain.SlugConflictError)
```

`internal/shared/errors/errcode_messages.go` — the existing block in the `defaultMessages` map gains one new line (full block shown, only `SlugConflict` is new):
```go
	BadRequest:      "Bad request",
	Unauthorized:    "Unauthorized",
	Forbidden:       "Forbidden",
	NotFound:        "Resource not found",
	Conflict:        "Conflict",
	TooManyRequests: "Too many requests",
	SlugConflict:    "Slug already exists", // NEW
```

`internal/course/delivery/handler_base.go` — full new content of `mapCourseError` (existing function; only the `errors` import and the new `if errors.As(...)` block right after the nil-check are new — every existing `case` line is unchanged verbatim):
```go
package delivery

import (
	"context"
	"errors" // NEW
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/shared/validate"
)

type Handler struct {
	svc *application.CourseService
}

func NewHandler(svc *application.CourseService) *Handler { return &Handler{svc: svc} }

func mapCourseError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	// NEW: typed-error check first — SlugConflictError carries a dynamic
	// RecommendedSlug field that a sentinel-identity switch (below) cannot
	// express. Mirrors the existing internal/shared/errors.RegistrationEmailRateLimitedError
	// precedent (also a typed error checked with errors.As at its own call site).
	var slugConflict *domain.SlugConflictError
	if errors.As(err, &slugConflict) {
		response.Fail(c, http.StatusConflict, apperrors.SlugConflict,
			apperrors.DefaultMessage(apperrors.SlugConflict),
			gin.H{"recommended_slug": slugConflict.RecommendedSlug})
		return true
	}
	switch err {
	case domain.ErrCourseNotFound, domain.ErrCourseVersionNotFound, domain.ErrCourseEnrollmentNotFound:
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, err.Error(), nil)
	case domain.ErrCourseOwnerOnly, domain.ErrCourseCollaboratorAccess:
		response.Fail(c, http.StatusForbidden, apperrors.Forbidden, err.Error(), nil)
	case domain.ErrCourseOptimisticLock, domain.ErrCourseLeaseHeldByOtherUser:
		response.Fail(c, http.StatusConflict, apperrors.Conflict, err.Error(), nil)
	case domain.ErrCourseDraftRequired, domain.ErrCourseDraftInReview, domain.ErrCourseDraftRejectedOnly,
		domain.ErrCourseInvalidSubLessonKind, domain.ErrCourseInvalidReviewState, domain.ErrCourseInvalidOrdering,
		domain.ErrCourseInstructorRequired, domain.ErrCourseOwnerCannotBeRemoved, domain.ErrCoursePublishedRequired,
		domain.ErrCourseLeaseTokenInvalid, domain.ErrCourseInvalidSlug, domain.ErrCourseTitleTooShort,
		domain.ErrCoursePreviewNotAllowedForQuiz, domain.ErrCourseQuizSingleChoiceMultipleCorrect,
		domain.ErrCourseSubmitBasicInfoIncomplete,
		domain.ErrCourseSubmitOutlineIncomplete, domain.ErrCourseSubmitInvalidSubLesson,
		domain.ErrCourseSubmitCollaboratorRequired, domain.ErrCourseCollaboratorInactive,
		domain.ErrCourseTrashed, domain.ErrCourseNotTrashed, domain.ErrCourseTrashNotEligible:
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	case apperrors.ErrNotFound, apperrors.ErrInvalidProfileMediaFile:
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
	default:
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
	}
	return true
}

func badParam(c *gin.Context, msg string) {
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, msg, nil)
}

func bindJSON[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return nil, false
	}
	return &req, true
}

func withCourseID(c *gin.Context, fn func(courseID string)) {
	courseID, ok := utils.ParseUUIDParam(c, "courseId")
	if !ok {
		badParam(c, "invalid course id")
		return
	}
	fn(courseID)
}

func withBody[T any](c *gin.Context, fn func(req *T)) {
	req, ok := bindJSON[T](c)
	if !ok {
		return
	}
	fn(req)
}

func withCourseAndBody[T any](c *gin.Context, fn func(courseID string, req *T)) {
	withCourseID(c, func(courseID string) {
		withBody(c, func(req *T) {
			fn(courseID, req)
		})
	})
}
```

This mirrors the existing typed-error precedent in this codebase (`internal/shared/errors.RegistrationEmailRateLimitedError`, already carries a dynamic `RetryAfterSeconds int64` field and is mapped the same way at its call site) — not a new pattern.

### D6 — Request/response contract changes

**SUPERSEDED by D10 (code review Bug 1)**: `updateBasicInfoRequest.Slug *string` shown below cannot actually distinguish an omitted field from an explicit JSON `null` (encoding/json sets `*string` to nil for both) — the whole "explicit `""`/whitespace → hard 400" analysis in this section still holds, but the mechanism is wrong. See D10 for the corrected `validate.Optional[string]` field and handler code. Shown here for historical context only.

`internal/course/delivery/dto.go`:
```go
type createCourseRequest struct {
	Title string `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	Slug  string `json:"slug" validate:"omitempty,max=255"` // MODIFIED: no course_slug tag — see D2 (the fixed bug: omitempty on a plain string does not skip a whitespace-only value, which the spec requires to be treated as "no slug supplied"); format validation happens once, in Go, in CourseService.CreateCourse
}

type updateBasicInfoRequest struct {
	ExpectedRowVersion int64    `json:"expected_row_version" validate:"required,min=1"`
	Title              string   `json:"title" validate:"required,nonwhitespace_min=5,max=255"`
	Slug               *string  `json:"slug"` // intentionally NO validate tag — see below
	ShortDescription   string   `json:"short_description" validate:"required,nonwhitespace_min=20,max=500"`
	AboutCourse        string   `json:"about_course" validate:"required,delta_nonwhitespace_min=30"`
	ThumbnailFileID    string   `json:"thumbnail_file_id" validate:"required,uuid"`
	PreviewVideoFileID *string  `json:"preview_video_file_id" validate:"omitempty,uuid"`
	CourseLevelID      string   `json:"course_level_id" validate:"required,uuid"`
	CourseTopicID      string   `json:"course_topic_id" validate:"required,uuid"`
	TagIDs             []string `json:"tag_ids" validate:"required,min=1,dive,uuid"`
	SkillIDs           []string `json:"skill_ids" validate:"required,min=1,dive,uuid"`
	OutcomeIDs         []string `json:"outcome_ids" validate:"required,len=1,dive,uuid"`
}
```

**Why `updateBasicInfoRequest.Slug` has no `validate` tag** (correcting the earlier "same as `PreviewVideoFileID`" assumption made during exploration): `go-playground/validator`'s `omitempty` on a pointer field does not distinguish "field omitted → pointer is `nil`" from "field explicitly sent as `""` → pointer is non-nil but points to the zero value" — both are treated as "empty" and skip the rest of the tag's rules. Since the spec requires **omitted → keep current slug** but **explicit `""`/whitespace → hard 400**, relying on `omitempty,course_slug` would silently let an explicit empty string through unvalidated, which is exactly the invariant this change must not violate. `PreviewVideoFileID` doesn't need this distinction (there is no "explicit-empty-is-an-error" rule for it), so its existing pattern is not actually a safe template to copy for slug specifically. Instead, `handler_instructor.go`'s update handler does the check explicitly in Go, where `nil` vs. non-nil is unambiguous.

Full new content of `internal/course/delivery/handler_instructor.go` (existing file; every line is unchanged from the current file EXCEPT the two functions `createCourse` and `updateBasicInfo`, and one new import — `optionalTrimmedStringPtr`, `deleteCourse`, `coursePaginatedUserList`, `listCollaborators`, `listInstructorCandidates`, `addCollaboratorsBulk`, `removeCollaborator`, `listEditableCourses`, `getCourseDetail`, `prepareDraft` are shown for exact placement/context but are **byte-for-byte identical** to today):
```go
package delivery

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/response"
	sharedslug "mycourse-io-be/internal/shared/slug" // NEW
	"mycourse-io-be/internal/shared/utils"
)

func (h *Handler) listEditableCourses(c *gin.Context) {
	rows, err := h.svc.ListEditableCourses(c.Request.Context(), utils.CurrentUserID(c))
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}

// MODIFIED: createCourse now passes req.Slug through (empty string means
// "auto-generate", enforced by the service layer — see design.md D7).
func (h *Handler) createCourse(c *gin.Context) {
	withBody(c, func(req *createCourseRequest) {
		row, err := h.svc.CreateCourse(c.Request.Context(), domain.CreateCourseInput{
			ActorUserID: utils.CurrentUserID(c),
			Title:       req.Title,
			Slug:        req.Slug, // NEW
		})
		if mapCourseError(c, err) {
			return
		}
		response.Created(c, "created", row)
	})
}

func (h *Handler) getCourseDetail(c *gin.Context) {
	includeOutline := c.DefaultQuery("include_outline", "true") != "false"
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.GetCourseDetail(c.Request.Context(), courseID, utils.CurrentUserID(c), true, includeOutline)
	})
}

func (h *Handler) prepareDraft(c *gin.Context) {
	h.courseOK(c, "ok", func(courseID string) (any, error) {
		return h.svc.PrepareDraft(c.Request.Context(), courseID, utils.CurrentUserID(c))
	})
}

// MODIFIED: updateBasicInfo now explicitly validates req.Slug in Go (see D6's
// omitempty-on-pointer rationale) instead of always deriving it from title.
func (h *Handler) updateBasicInfo(c *gin.Context) {
	courseBodyOK(h, c, "updated", func(courseID string, req *updateBasicInfoRequest) (any, error) {
		var slugInput *string
		if req.Slug != nil {
			trimmed := strings.TrimSpace(*req.Slug)
			if trimmed == "" {
				return nil, domain.ErrCourseInvalidSlug // explicit empty/whitespace → reject
			}
			validated, ok := sharedslug.ValidateManualFormat(trimmed)
			if !ok {
				return nil, domain.ErrCourseInvalidSlug
			}
			slugInput = &validated
		}
		title := strings.TrimSpace(req.Title)
		shortDescription := strings.TrimSpace(req.ShortDescription)
		aboutCourse := strings.TrimSpace(req.AboutCourse)
		thumbnailFileID := strings.TrimSpace(req.ThumbnailFileID)
		courseLevelID := strings.TrimSpace(req.CourseLevelID)
		courseTopicID := strings.TrimSpace(req.CourseTopicID)
		return h.svc.UpdateBasicInfo(c.Request.Context(), courseID, utils.CurrentUserID(c), domain.UpdateBasicInfoInput{
			ActorUserID:        utils.CurrentUserID(c),
			ExpectedRowVersion: req.ExpectedRowVersion,
			Title:              &title,
			Slug:               slugInput, // NEW — nil unless client sent a new value
			ShortDescription:   &shortDescription,
			AboutCourse:        &aboutCourse,
			ThumbnailFileID:    &thumbnailFileID,
			PreviewVideoFileID: optionalTrimmedStringPtr(req.PreviewVideoFileID),
			CourseLevelID:      &courseLevelID,
			CourseTopicID:      &courseTopicID,
			TagIDs:             req.TagIDs,
			SkillIDs:           req.SkillIDs,
			OutcomeIDs:         req.OutcomeIDs,
		})
	})
}

func optionalTrimmedStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (h *Handler) deleteCourse(c *gin.Context) {
	withCourseID(c, func(courseID string) {
		if err := h.svc.DeleteCourse(c.Request.Context(), courseID, utils.CurrentUserID(c)); mapCourseError(c, err) {
			return
		}
		response.OK(c, "deleted", gin.H{"course_id": courseID})
	})
}

func (h *Handler) coursePaginatedUserList(c *gin.Context, mode string) {
	h.listCoursePaginatedSearch(c, func(ctx context.Context, courseID, actorUserID string, page, perPage int, search string) (any, int64, error) {
		switch mode {
		case "collaborators":
			return h.svc.ListCollaborators(ctx, courseID, actorUserID, domain.CollaboratorListFilter{
				Page: page, PerPage: perPage, Search: search,
			})
		case "instructor-candidates":
			return h.svc.ListInstructorCandidates(ctx, courseID, actorUserID, domain.InstructorCandidateFilter{
				Page: page, PerPage: perPage, Search: search,
			})
		default:
			return nil, 0, domain.ErrCourseNotFound
		}
	})
}

func (h *Handler) listCollaborators(c *gin.Context) {
	h.coursePaginatedUserList(c, "collaborators")
}

func (h *Handler) listInstructorCandidates(c *gin.Context) {
	h.coursePaginatedUserList(c, "instructor-candidates")
}

func (h *Handler) addCollaboratorsBulk(c *gin.Context) {
	courseBodyOK(h, c, "ok", func(courseID string, req *addCollaboratorsBulkRequest) (any, error) {
		result, err := h.svc.AddCollaboratorsBulk(
			c.Request.Context(),
			courseID,
			utils.CurrentUserID(c),
			req.UserIDs,
			req.Role,
		)
		if err != nil {
			return nil, err
		}
		return toCollaboratorBulkResponse(result), nil
	})
}

func (h *Handler) removeCollaborator(c *gin.Context) {
	h.courseUUIDParamOK(c, "userId", "invalid user id", "updated", func(courseID string, userID string) (any, error) {
		return h.svc.RemoveCollaborator(c.Request.Context(), courseID, utils.CurrentUserID(c), userID)
	})
}
```

Response bodies need no shape change: `CourseDetail`/`CourseListItem` already embed `Course.Slug json:"slug"` (`internal/course/domain/course.go:31`), satisfying "every slug-affecting response returns the final stored slug" for free.

### D7 — Domain & repository orchestration

**SUPERSEDED in part by D10 (code review Bugs 2 & 3)**: every `sharedslug.ValidateManualFormat(trimmed)` call below is missing the `maxLen` argument, and every inline `sharedslug.TruncateForSuffix(...) + "-" + suffix` below is missing the empty-base guard. See D10 for the corrected calls (`ValidateManualFormat(trimmed, domain.MaxSlugLen)` and `buildSuffixedCandidate(base, suffix)`). Shown here for historical context only.

`internal/course/domain/course.go` — `CreateCourseInput` and `UpdateBasicInfoInput` full structs (only the doc comments on `Slug` change; field types/names are unchanged, so this is a comment-only diff on this file):
```go
type CreateCourseInput struct {
	ActorUserID string
	Slug        string // MODIFIED comment: "" means auto-generate from Title; a non-empty value is the caller's manual choice (already trimmed+format-validated by the application layer)
	Title       string
}

type UpdateBasicInfoInput struct {
	ActorUserID        string
	ExpectedRowVersion int64
	Title              *string
	Slug               *string // MODIFIED comment: nil means "do not change courses.slug"; non-nil is a new value already trimmed+format-validated by the delivery layer — no longer derived from Title
	ShortDescription   *string
	AboutCourse        *string
	ThumbnailFileID    *string
	PreviewVideoFileID *string
	CourseLevelID      *string
	CourseTopicID      *string
	TagIDs             []string
	SkillIDs           []string
	OutcomeIDs         []string
}
```

Full new content of `internal/course/application/service.go`'s top section, from the package/import block through `UpdateBasicInfo` (`courseTitleAndSlug` is **removed**, replaced by `validateCourseTitle`; every other function in this file below `UpdateBasicInfo` — `DeleteCourse`, `ListCollaborators`, etc. — is unchanged and not repeated here):
```go
package application

import (
	"context"
	"strings"

	"mycourse-io-be/internal/course/domain"
	sharedslug "mycourse-io-be/internal/shared/slug" // NEW
	"mycourse-io-be/internal/shared/utils"
)

type CourseService struct {
	repo domain.Repository
}

func NewCourseService(repo domain.Repository) *CourseService {
	return &CourseService{repo: repo}
}

// REPLACES courseTitleAndSlug: title validation only now — slug is no longer
// derived from title at all (manual-or-empty, handled separately below).
func validateCourseTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if utils.CountNonWhitespace(title) < 5 {
		return "", domain.ErrCourseTitleTooShort
	}
	return title, nil
}

func (s *CourseService) ListEditableCourses(ctx context.Context, userID string) ([]domain.CourseListItem, error) {
	return s.repo.ListEditableCourses(ctx, userID)
}

// MODIFIED: no longer derives Slug from Title. An optional manual Slug is
// format-validated here (fast fail, no DB call); "" is passed through
// unchanged to the repo, which treats it as "auto-generate" (see D7 infra).
func (s *CourseService) CreateCourse(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	title, err := validateCourseTitle(in.Title)
	if err != nil {
		return nil, err
	}
	manualSlug := ""
	if trimmed := strings.TrimSpace(in.Slug); trimmed != "" {
		validated, ok := sharedslug.ValidateManualFormat(trimmed)
		if !ok {
			return nil, domain.ErrCourseInvalidSlug
		}
		manualSlug = validated
	}
	return s.repo.CreateCourse(ctx, domain.CreateCourseInput{ActorUserID: in.ActorUserID, Title: title, Slug: manualSlug})
}

func (s *CourseService) GetCourseDetail(ctx context.Context, courseID string, userID string, includeDraft bool, includeOutline bool) (*domain.CourseDetail, error) {
	return s.repo.GetCourseDetail(ctx, courseID, userID, includeDraft, includeOutline)
}

func (s *CourseService) PrepareDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	return s.repo.PrepareDraft(ctx, courseID, actorUserID)
}

// MODIFIED: validates Title only when present; Slug is passed through
// untouched (already validated by the delivery layer, see D6) — the old
// "recompute slug whenever title is set" behavior is removed entirely.
func (s *CourseService) UpdateBasicInfo(ctx context.Context, courseID string, actorUserID string, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
	if in.Title != nil {
		title, err := validateCourseTitle(*in.Title)
		if err != nil {
			return nil, err
		}
		in.Title = &title
	}
	return s.repo.UpdateBasicInfo(ctx, courseID, actorUserID, in)
}

func (s *CourseService) DeleteCourse(ctx context.Context, courseID string, actorUserID string) error {
	return s.repo.DeleteCourse(ctx, courseID, actorUserID)
}

// ... every remaining function in this file (ListCollaborators onward) is
// unchanged — not repeated here, see the current file for the rest.
```

Full new content of the relevant section of `internal/course/infra/repo_instructor.go` — from the import block through `UpdateBasicInfo` (everything from `ListEditableCourses` down to (not including) `DeleteCourse` is shown; `DeleteCourse` onward, further down the file, is unchanged and not repeated):
```go
package infra

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	authzdomain "mycourse-io-be/internal/authorization/domain"
	courseapp "mycourse-io-be/internal/course/application"
	"mycourse-io-be/internal/course/domain"
	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/gormx"
	sharedslug "mycourse-io-be/internal/shared/slug" // NEW
	"mycourse-io-be/internal/shared/timex"
)

// ListEditableCourses — UNCHANGED, not repeated here (see current file).

// MODIFIED: courseSlugCreateRetry raised 3 -> 5 (confirmed with user; see
// design.md "Resolved decisions" — random-suffix collisions can need more
// attempts under contention than the old deterministic numeric scheme).
func (r *GormRepository) CreateCourse(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	var (
		detail *domain.CourseDetail
		err    error
	)
	for range courseSlugCreateRetry {
		detail, err = r.createCourseOnce(ctx, in)
		if err == nil || !isCourseSlugDuplicateKey(err) {
			return detail, err
		}
	}
	return nil, err
}

// MODIFIED: slug resolution is now branch: a non-empty in.Slug (manual,
// already format-validated by the service layer) is checked for availability
// and used as-is, or the whole operation fails with *domain.SlugConflictError
// (no silent auto-suffix on a caller's own explicit choice); an empty in.Slug
// (auto-generate) derives a base from the title via gosimple/slug and
// resolves any collision with a random suffix via sharedslug.RetryWithSuffix.
func (r *GormRepository) createCourseOnce(ctx context.Context, in domain.CreateCourseInput) (*domain.CourseDetail, error) {
	var courseID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		finalSlug, err := resolveCreateSlug(ctx, tx, in)
		if err != nil {
			return err // may be *domain.SlugConflictError — propagated as-is, not wrapped
		}
		course := &courseRow{OwnerUserID: in.ActorUserID, Slug: finalSlug}
		if err := touchCreateCourseEntity(ctx, tx, &course.CreatedAt, &course.UpdatedAt, course); err != nil {
			return err
		}
		courseID = course.ID

		version := &courseVersionRow{
			CourseID: course.ID, VersionNo: 1, Status: domain.VersionStatusDraft,
			Title: strings.TrimSpace(in.Title), RowVersion: 1,
		}
		if err := touchCreateCourseEntity(ctx, tx, &version.CreatedAt, &version.UpdatedAt, version); err != nil {
			return err
		}
		// No course_collaborators row for the owner: ownership is courses.owner_user_id
		// itself, synthesized by CoursePolicyProvider and read directly by
		// collaboratorsSelectSQL/instructorCandidatesBaseSQL/ListEditableCourses — never a
		// stored role binding or collaborator row.
		return tx.Model(&courseRow{}).Where("id = ?", course.ID).
			Updates(map[string]any{"current_draft_version_id": version.ID, "updated_at": timex.NowUnix()}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, in.ActorUserID, true, true)
}

// NEW: resolveCreateSlug implements design.md D7's create-time branch. Lives
// in repo_instructor.go (not slug.go) because it orchestrates courseRow
// writes/reads directly; slug.go holds only the two smaller, more reusable
// primitives (generateAutoSlugBase, courseSlugAvailable). Every RetryWithSuffix
// call truncates its base via sharedslug.TruncateForSuffix before
// concatenating, so base+"-"+suffix can never exceed maxCourseSlugLen (255) —
// closing the VARCHAR(255) overflow gap a near-max-length manual slug could
// otherwise hit once a suffix is appended.
func resolveCreateSlug(ctx context.Context, tx *gorm.DB, in domain.CreateCourseInput) (string, error) {
	if in.Slug != "" {
		available, err := courseSlugAvailable(ctx, tx, in.Slug, nil)
		if err != nil {
			return "", err
		}
		if available {
			return in.Slug, nil
		}
		// Treated exactly like any other manual-slug conflict below — including
		// on the round trip where the caller resubmits create with this exact
		// recommended value: if a concurrent writer took it in the meantime
		// (astronomically rare given the suffix's keyspace), the resubmission is
		// itself just another manual slug, and gets its own fresh
		// SlugConflictError + a new recommendation (e.g.
		// "golang-course-x7k92ab-k39mz81") — no special-cased "never conflict
		// twice" bypass. See specs/course/slug-management/spec.md and design.md
		// "Resolved decisions" for why this uniform handling was chosen over
		// adding a silent-resolve special case.
		recommended, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
			candidate := sharedslug.TruncateForSuffix(in.Slug, maxCourseSlugLen, len(suffix)) + "-" + suffix
			ok, err := courseSlugAvailable(ctx, tx, candidate, nil)
			return candidate, ok, err
		})
		if err != nil {
			return "", err
		}
		return "", &domain.SlugConflictError{RecommendedSlug: recommended}
	}
	base, mustSuffix := generateAutoSlugBase(in.Title) // already truncated to maxCourseSlugLen internally
	if !mustSuffix {
		available, err := courseSlugAvailable(ctx, tx, base, nil)
		if err != nil {
			return "", err
		}
		if available {
			return base, nil
		}
	}
	// write-attempt-based collision resolution below relies on the OUTER
	// CreateCourse retry loop + isCourseSlugDuplicateKey to catch a race this
	// SELECT-based accept callback missed — this accept is only the
	// UX-fast-path, not the safety net (see proposal.md "Slug uniqueness is
	// enforced at the database").
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate := sharedslug.TruncateForSuffix(base, maxCourseSlugLen, len(suffix)) + "-" + suffix
		ok, err := courseSlugAvailable(ctx, tx, candidate, nil)
		return candidate, ok, err
	})
}

func (r *GormRepository) GetCourseDetail(ctx context.Context, courseID string, userID string, includeDraft bool, includeOutline bool) (*domain.CourseDetail, error) {
	return r.loadCourseDetail(ctx, r.db.WithContext(ctx), courseID, userID, includeDraft, includeOutline)
}

func (r *GormRepository) PrepareDraft(ctx context.Context, courseID string, actorUserID string) (*domain.CourseDetail, error) {
	// UNCHANGED — not repeated here, see current file.
}

// MODIFIED: slug handling replaced entirely (no-op-on-same-value check, direct
// random-suffix auto-resolve on conflict, and — closing the race-safety gap
// flagged in Context — wrapped in its own retry-on-duplicate-key loop, which
// the pre-existing code never had).
func (r *GormRepository) UpdateBasicInfo(ctx context.Context, courseID string, actorUserID string, in domain.UpdateBasicInfoInput) (*domain.CourseDetail, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access, err := r.ensureEditableDraft(ctx, tx, courseID, actorUserID)
		if err != nil {
			return err
		}
		version, err := r.loadVersionRow(ctx, tx, *access.CurrentDraftVersionID)
		if err != nil {
			return err
		}
		if version.RowVersion != in.ExpectedRowVersion {
			return apperrors.ErrMediaOptimisticLock
		}
		if err := r.validateVersionRefs(ctx, tx, versionRefValidationInput{
			ThumbnailFileID:    in.ThumbnailFileID,
			PreviewVideoFileID: in.PreviewVideoFileID,
			CourseLevelID:      in.CourseLevelID,
			CourseTopicID:      in.CourseTopicID,
			TagIDs:             in.TagIDs,
			SkillIDs:           in.SkillIDs,
			OutcomeIDs:         in.OutcomeIDs,
		}); err != nil {
			return err
		}
		updates := buildBasicInfoUpdates(in)
		if err := tx.Model(&courseVersionRow{}).
			Where("id = ? AND row_version = ? AND deleted_at IS NULL", version.ID, in.ExpectedRowVersion).
			Updates(updates).Error; err != nil {
			return err
		}
		if err := r.replaceVersionRefs(ctx, tx, version.ID, in.TagIDs, in.SkillIDs, in.OutcomeIDs); err != nil {
			return err
		}
		return r.applyUpdateSlugWithRetry(ctx, tx, access, in.Slug) // NEW — was: single ensureUniqueCourseSlug call, no retry
	})
	if stderrors.Is(err, apperrors.ErrMediaOptimisticLock) {
		return nil, domain.ErrCourseOptimisticLock
	}
	if err != nil {
		return nil, err
	}
	return r.loadCourseDetail(ctx, r.db, courseID, actorUserID, true, true)
}

// NEW: applyUpdateSlugWithRetry closes the pre-existing race-safety gap
// (UpdateBasicInfo previously called ensureUniqueCourseSlug exactly once with
// no retry-on-duplicate-key, unlike CreateCourse). Scoped to only the slug
// write sub-step (not the whole surrounding transaction) so a slug-only
// collision retry does not re-validate/re-write the unrelated metadata fields
// already committed earlier in the same transaction.
func (r *GormRepository) applyUpdateSlugWithRetry(ctx context.Context, tx *gorm.DB, access *courseAccess, newSlugInput *string) error {
	if newSlugInput == nil {
		return nil // omitted — do not touch courses.slug at all
	}
	newSlug := *newSlugInput
	if newSlug == access.Slug {
		return nil // no-op: identical to current slug, never a self-conflict
	}
	const updateSlugRetry = 5 // mirrors courseSlugCreateRetry; same tuning rationale
	var lastErr error
	for attempt := 0; attempt < updateSlugRetry; attempt++ {
		finalSlug, err := resolveUpdateSlug(ctx, tx, newSlug, access.ID)
		if err != nil {
			return err
		}
		writeErr := tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", access.ID).
			Updates(map[string]any{"slug": finalSlug, "updated_at": timex.NowUnix()}).Error
		if writeErr == nil {
			return nil
		}
		if !isCourseSlugDuplicateKey(writeErr) {
			return writeErr
		}
		lastErr = writeErr // a concurrent writer won the race on finalSlug — retry with a fresh suffix
	}
	return lastErr
}

// NEW: resolveUpdateSlug implements design.md D7's update-time branch — no
// SlugConflictError path exists here (update auto-resolves directly, unlike
// create's manual-conflict-returns-a-recommendation behavior). Truncates via
// sharedslug.TruncateForSuffix for the same VARCHAR(255)-overflow reason as
// resolveCreateSlug above.
func resolveUpdateSlug(ctx context.Context, tx *gorm.DB, newSlug string, excludeCourseID string) (string, error) {
	available, err := courseSlugAvailable(ctx, tx, newSlug, &excludeCourseID)
	if err != nil {
		return "", err
	}
	if available {
		return newSlug, nil
	}
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate := sharedslug.TruncateForSuffix(newSlug, maxCourseSlugLen, len(suffix)) + "-" + suffix
		ok, err := courseSlugAvailable(ctx, tx, candidate, &excludeCourseID)
		return candidate, ok, err
	})
}
```
(`courseSlugAvailable`'s signature takes `excludeCourseID *string`; `resolveUpdateSlug` takes `access.ID` — a plain `string` — so it takes its address, `&excludeCourseID`, when calling through; implementers should double check this pointer plumbing compiles cleanly against the exact `slug.go` signature in D4 during `/opsx:apply`, since this design doc is not compiler-checked.)

### D8 — Database changes

Two new migrations (`000039`, `000040` — next available numbers after `000038_home_catalog_indexes`):

**`migrations/000039_backfill_course_slug_ascii.up.sql`** (data-safety step, runs first, MUST run before D8's `CHECK` migration can safely apply — see Migration Plan and the "CHECK constraint risk" decision already agreed with the user):

**Corrected twice during `/opsx:apply`** — two real bugs, both found only by actually running this migration, not by reading the design:
1. Originally drafted as a `DO $$ ... $$` PL/pgSQL block with a nested collision-retry loop. `migrations/README.md` states the runner splits files by every literal semicolon character and explicitly warns against `DO $$` blocks for this reason (citing `000015`/`000032` as prior casualties). Rewritten as a single plain `UPDATE` statement (matching `000020_course_version_row_version_backfill`'s precedent) — caught during design review, before ever touching a DB.
2. **Found only by actually running `CGO_ENABLED=1 MIGRATE=1 go run .` against a real dev DB** (not caught by the section-1 fix above, nor by any code review): the rewritten comment block *itself* still contained two literal semicolon characters in its prose (one inside a quoted `";"` example, one in "...astronomically unlikely; md5 output...") — and the runner's `;`-splitting is byte-level, applying inside `--` comment lines too, not just inside executable SQL. The real failure: `migrate database failed: ... unterminated quoted identifier ...`. Fixed by rewording the comment to avoid the semicolon character entirely (verified live: the migration then applied cleanly, `schema_migrations` reached version 40 clean, and `chk_courses_slug_format`/`uix_courses_slug_active`/`NOT NULL` were all confirmed present via `\d courses`). **Lesson generalized in the session context file: this runner's constraint is "no semicolon character anywhere in the file," not "no semicolon inside executable SQL" — comments are not exempt.**

```sql
-- Regenerate any existing courses.slug value that does not conform to the
-- new format (non-ASCII passthrough from the pre-existing utils.SlugifyName,
-- or any other pre-existing malformed value) so the CHECK constraint added
-- in 000040 cannot fail to apply. Scans EVERY row regardless of deleted_at
-- (the CHECK constraint applies table-wide, unlike the partial unique index
-- uix_courses_slug_active which only covers active rows).
--
-- Plain single-statement UPDATE, not a DO block or PL/pgSQL body: this
-- repo's migration runner splits files on every literal semicolon character
-- (see migrations/README.md), even one appearing inside a comment like this
-- one, so no comment line in this file may contain that character at all.
-- The replacement slug uses a 16-character lowercase hex suffix from md5()
-- over enough entropy (row id, wall-clock, and random()) that a collision is
-- astronomically unlikely, and md5 output is already confined to [0-9a-f],
-- so no further sanitizing is needed to satisfy the slug format. If a
-- collision with an active row's slug ever did occur, this UPDATE would fail
-- on uix_courses_slug_active and could simply be re-run.
UPDATE courses
SET slug = 'course-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = extract(epoch FROM now())::bigint
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';
```
`migrations/000039_backfill_course_slug_ascii.down.sql`: intentionally a no-op (`-- Data backfill is not safely reversible: the original non-conforming slug values are not recoverable once regenerated.`) — matches `000020_course_version_row_version_backfill.down.sql`'s identical posture, the closest existing precedent in this repo for an irreversible data-only backfill.

**`migrations/000040_course_slug_check_constraint.up.sql`**:
```sql
ALTER TABLE courses
    ADD CONSTRAINT chk_courses_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');
```
`migrations/000040_course_slug_check_constraint.down.sql`:
```sql
ALTER TABLE courses DROP CONSTRAINT IF EXISTS chk_courses_slug_format;
```
No new table, no new column — `slug` is already `NOT NULL`; the `CHECK` only adds format enforcement. `uix_courses_slug_active` (uniqueness) is untouched.

**Per the user's explicit decision**: before writing `000040`, run `SELECT id, slug FROM courses WHERE deleted_at IS NULL AND slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';` against the real dev DB during `/opsx:apply` to confirm whether `000039` actually finds any rows (it is safe/idempotent either way — a 0-row backfill is a no-op — but the task list should record the actual count found, not assume it).

### D9 — Module map (packages touched)

| Package | Change |
|---|---|
| `internal/shared/slug` (**new**) | format validation, random suffix, escalating retry (D1) |
| `internal/shared/utils` | none (no change to `SlugifyName` — taxonomy still uses it) |
| `internal/shared/validate` | **new** `Optional[T]` type (D10 — code review found a second, unrelated need for this package: JSON PATCH-omit-vs-null decoding, not the format-validation tag D2 dropped) |
| `internal/shared/errors` | new `SlugConflict` code + message (D5) |
| `internal/shared/response` | none (already generic) |
| `internal/course/domain` | `SlugConflictError` type; `UpdateBasicInfoInput.Slug` semantics change (D5, D7) |
| `internal/course/application` | `courseTitleAndSlug` split/removed; slug validation moved (D7) |
| `internal/course/delivery` | DTO + handler changes (D6) |
| `internal/course/infra` | `repo_helpers.go` numeric-suffix functions removed; new `slug.go`; `repo_instructor.go` create/update flows rewritten (D4, D7) |
| `internal/taxonomy`, `internal/shared/taxonomy` | **untouched** |
| `internal/instructor` | **untouched** (explicitly deferred, per Non-Goals) |
| `migrations` | 2 new files (D8) |
| `go.mod`/`go.sum` | + `github.com/gosimple/slug` |

### D10 — Code review round: 3 real bugs found and fixed after `/opsx:apply` landed the code

`/code-review` found 3 real, empirically-verified bugs in the implementation D1-D9 describe. **This section is the authoritative final code for every piece it touches — D1, D4, D6, and D7 above describe the pre-review versions and are superseded where they conflict with this section** (each superseded block above has been annotated in place; this section is not duplicated maintenance, it is the corrected reference).

**Bug 1 (most severe) — `updateBasicInfoRequest.Slug *string` could not distinguish an omitted field from an explicit JSON `null`.** Verified directly against `encoding/json` (not assumed): both `{}` and `{"slug":null}` decode to `Slug == nil` for a plain `*string` field. This meant the entire "explicit null → 400" contract in `specs/course/slug-management/spec.md` ("Requirement: Course update accepts an independent, omittable slug field") never actually fired — a client sending `{"slug": null, ...}` silently succeeded with the slug left unchanged instead of getting the documented `400`.

Fix: new generic type in `internal/shared/validate/optional.go` (full file):
```go
package validate

import (
	"bytes"
	"encoding/json"
)

// Optional distinguishes "field omitted from the request body" from "field
// explicitly present" (including an explicit JSON null) when decoding a
// PATCH-style request — something a plain pointer field cannot do:
// encoding/json leaves a *T field nil for both an omitted key and an
// explicit `null` value (verified directly against encoding/json, not
// assumed), so any endpoint needing "omitted = leave unchanged, explicit
// null = reject" semantics must use this instead of *T.
type Optional[T any] struct {
	Value T
	Set   bool
}

// UnmarshalJSON is only invoked by encoding/json (and anything built on it,
// e.g. Gin's ShouldBindJSON) when the field's key is present in the source
// object — including when its value is the literal null. An absent key never
// calls this at all, leaving Set at its zero value (false).
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		var zero T
		o.Value = zero
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}
```
`internal/course/delivery/dto.go`'s `updateBasicInfoRequest.Slug` field changes from `*string` (SUPERSEDES D6) to:
```go
Slug validate.Optional[string] `json:"slug"` // NOT *string: encoding/json sets a *string field to nil for BOTH an omitted key and an explicit JSON null (verified empirically) — validate.Optional distinguishes them via UnmarshalJSON, which only runs when the key is present at all. No validate tag: the explicit nil/empty/format check happens in Go in updateBasicInfo (handler_instructor.go).
```
`internal/course/delivery/handler_instructor.go`'s `updateBasicInfo` slug block changes (SUPERSEDES D6's handler snippet) from `if req.Slug != nil { trimmed := strings.TrimSpace(*req.Slug); ... }` to:
```go
var slugInput *string
if req.Slug.Set {
	trimmed := strings.TrimSpace(req.Slug.Value)
	if trimmed == "" {
		return nil, domain.ErrCourseInvalidSlug // explicit null/empty/whitespace on update -> reject
	}
	validated, ok := sharedslug.ValidateManualFormat(trimmed, domain.MaxSlugLen)
	if !ok {
		return nil, domain.ErrCourseInvalidSlug
	}
	slugInput = &validated
}
// slugInput stays nil when req.Slug.Set is false (field omitted) — the repo
// layer treats nil as "do not touch courses.slug". req.Slug.Set is true for
// BOTH an explicit JSON null and an explicit value (Optional distinguishes
// "omitted" from "present", not "null" from "non-null" — this endpoint
// treats explicit null the same as explicit empty/whitespace, both rejected
// above, so no further distinction is needed here).
```
No API wire-format change (the JSON contract was always documented as `slug: string, optional, null rejected` — this fix makes the Go code actually match what `docs/api_swagger.yaml`/`docs/curl_api.md` already said, it doesn't change either doc).

**Bug 2 — `updateBasicInfoRequest.Slug` (and `ValidateManualFormat` itself) had no maximum-length check, unlike `createCourseRequest.Slug`'s `validate:"omitempty,max=255"` tag.** A format-valid manual slug longer than `courses.slug VARCHAR(255)` would pass all application-layer validation on update, reach the DB, and fail with a raw Postgres "value too long for type character varying(255)" error that `isCourseSlugDuplicateKey` does not recognize — propagating to `mapCourseError`'s `default` case as a generic `500` instead of a clean `400`.

Fix: moved the length limit to a single shared constant, `internal/course/domain/course.go` (added to the existing `const (...)` block, SUPERSEDES D4's locally-declared `maxCourseSlugLen` in `slug.go`, which is deleted):
```go
	// MaxSlugLen mirrors the courses.slug VARCHAR(255) column limit
	// (migrations/000016_course_management.up.sql). Single source of truth
	// shared by the create DTO's validate tag, the manual-slug format check
	// (application + delivery layers), and the infra-layer suffix truncation
	// — code-review found create and update enforcing this inconsistently
	// (update had no length check at all) when it was duplicated ad hoc.
	MaxSlugLen = 255
```
`internal/shared/slug/format.go`'s `ValidateManualFormat` signature changes (SUPERSEDES D1) from `ValidateManualFormat(raw string) (trimmed string, ok bool)` to:
```go
func ValidateManualFormat(raw string, maxLen int) (trimmed string, ok bool) {
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > maxLen || !formatPattern.MatchString(trimmed) {
		return "", false
	}
	return trimmed, true
}
```
Every call site updated to pass `domain.MaxSlugLen`: `internal/course/application/service.go`'s `CreateCourse` (SUPERSEDES D7), `internal/course/delivery/handler_instructor.go`'s `updateBasicInfo` (shown above, Bug 1's fix), and `internal/course/infra/slug.go`'s `generateAutoSlugBase` (SUPERSEDES D4).

**Bug 3 — `TruncateForSuffix`'s caller in `repo_instructor.go` had no guard against the truncated base becoming empty.** `maxTotalLen - 1 - suffixLen` can reach zero or negative once escalating collisions push `suffixLen` close to `domain.MaxSlugLen`; `TruncateToLen` then returns `""`, and the un-guarded candidate becomes `"-"+suffix` — a value starting with a hyphen, violating the slug format the `chk_courses_slug_format` CHECK constraint enforces. Astronomically unlikely in practice (requires ~1700+ sequential collisions against the same base), but unguarded.

Fix: new shared helper in `internal/course/infra/slug.go` (full new addition to that file, used by all 3 `RetryWithSuffix` call sites in `repo_instructor.go`, SUPERSEDES the inline `sharedslug.TruncateForSuffix(...) + "-" + suffix` shown 3× in D7):
```go
// buildSuffixedCandidate truncates base to leave room for "-"+suffix within
// domain.MaxSlugLen and returns an error if no room remains at all (base
// truncates to empty). Code review found the untruncated/unguarded version
// used by an earlier draft could otherwise produce a leading-hyphen candidate
// (e.g. "-x7k92ab") once escalating suffix length consumed the entire column
// budget — a value that violates the slug format itself, not just a
// collision. Shared by all 3 RetryWithSuffix call sites in this file so the
// guard exists exactly once.
func buildSuffixedCandidate(base, suffix string) (string, error) {
	truncatedBase := sharedslug.TruncateForSuffix(base, domain.MaxSlugLen, len(suffix))
	if truncatedBase == "" {
		return "", fmt.Errorf("slug base %q leaves no room for a %d-character suffix within %d characters", base, len(suffix), domain.MaxSlugLen)
	}
	return truncatedBase + "-" + suffix, nil
}
```
Each of the 3 `RetryWithSuffix` closures in `resolveCreateSlug`/`resolveUpdateSlug` (D7) now calls `candidate, err := buildSuffixedCandidate(base, suffix); if err != nil { return "", false, err }` instead of inlining the truncate+concat — `RetryWithSuffix`'s own `if err != nil { return "", err }` (D1) then propagates this cleanly as a real error (surfacing as `500` via `mapCourseError`'s default case) instead of ever attempting to write an invalid candidate.

**Verification**: all 3 fixes are covered by new/updated unit tests (`internal/shared/validate/optional_test.go`, `internal/shared/slug/format_test.go`'s `TestValidateManualFormatRejectsOverMaxLen`, `internal/course/infra/repo_slug_test.go`'s `TestBuildSuffixedCandidateErrorsWhenNoRoomForBase`/`TestBuildSuffixedCandidateNeverExceedsMaxSlugLen`, `internal/course/delivery/handler_update_slug_test.go`'s 4 new tests including `TestUpdateBasicInfoExplicitNullSlugIsRejected` — this last one would have panicked on a nil `h.svc` before the fix, since the pre-fix code path fell through toward calling it instead of stopping at the reject). Full gate chain (`gofmt`, `go test` ×2, `go vet`, `golangci-lint`, `check-layout`/`check-architecture`/`check-dupl`, `build-nocgo`/`build`) re-run clean after all 3 fixes. No API wire-format or documented behavior changed by any of the 3 fixes — every fix makes the code match what `specs/course/slug-management/spec.md` and `docs/api_swagger.yaml`/`docs/curl_api.md` already said; none of the `docs/*.md` files needed further edits.

## API contract (reuse vs. extend vs. new)

| Endpoint | Status | What changes |
|---|---|---|
| `POST /api/v1/courses` | **Reused, extended** (no new route) | Request gains optional `slug`; response unchanged shape (already returns final slug); new possible `409 SlugConflict` response with `data.recommended_slug` |
| `PATCH /api/v1/courses/:courseId/basic-info` | **Reused, extended** (no new route) | Request gains optional `slug` (`*string`, PATCH-omit semantics); response unchanged shape |
| Every other course endpoint | **Reused as-is** | No change — they already return `Course.Slug` verbatim |

No new endpoint is introduced anywhere in this change, confirming the user's own hypothesis from the exploration phase.

## Documentation updates (English; EXACT literal before/after text for every file, re-verified against the live files — not paraphrase. `tasks.md` §9 references these by number.)

### Doc-1. `docs/reusable-assets.md`

**Doc-1a. Replace the `SlugifyName` asset block:**

OLD:
```
### Asset: SlugifyName (utils)
- Name: `SlugifyName`
- Type: Function (util)
- Path: `internal/shared/utils/slug.go`
- Purpose: Build URL slug from display name — mirrors FE `slugifyName` / `generateSlug` (trim, lowercase, strip accents, `đ/Đ -> d`, spaces/underscores → `-`, Unicode letters/numbers only, collapse dashes).
- Scope: Taxonomy create/update (root slug + tree nodes via `NormalizeTreeSlugs`), course create/update (`title` → `courses.slug`).
- Dependencies: `golang.org/x/text/unicode/norm`, Go `unicode`.
- Current Usage: `internal/taxonomy/application/service.go`, `internal/shared/taxonomy/tree_slug.go`, `internal/course/application/service.go`.
- Reuse: Never accept client-provided slug on write — always derive with `SlugifyName`.
```

NEW:
```
### Asset: SlugifyName (utils)
- Name: `SlugifyName`
- Type: Function (util)
- Path: `internal/shared/utils/slug.go`
- Purpose: Build URL slug from display name — mirrors FE `slugifyName` / `generateSlug` (trim, lowercase, strip accents, `đ/Đ -> d`, spaces/underscores → `-`, Unicode letters/numbers only, collapse dashes).
- Scope: Taxonomy create/update only (root slug + tree nodes via `NormalizeTreeSlugs`). **No longer used by course** (see `openspec/changes/rework-course-slug-management`) — course now uses `github.com/gosimple/slug` for auto-generation from title, because `SlugifyName` keeps any surviving Unicode letter after diacritic-stripping (ASCII-safe for Vietnamese, not for Chinese/Japanese/Thai/Korean/Russian), while `gosimple/slug` has real transliteration tables for those scripts. Taxonomy is unaffected and intentionally still never accepts a client-supplied slug — this divergence is scoped to course only, not a repo-wide policy change.
- Dependencies: `golang.org/x/text/unicode/norm`, Go `unicode`.
- Current Usage: `internal/taxonomy/application/service.go`, `internal/shared/taxonomy/tree_slug.go`.
- Reuse: Taxonomy — never accept client-provided slug on write, always derive with `SlugifyName`. Course — see the new `internal/shared/slug` + `github.com/gosimple/slug` asset entries below instead; do not call `SlugifyName` from course code.
```

**Doc-1b. Fix the stale cross-reference:**

OLD:
```
- Current Usage: `internal/course/delivery/dto.go`, `courseTitleAndSlug` in `internal/course/application/service.go`.
```

NEW:
```
- Current Usage: `internal/course/delivery/dto.go`, `validateCourseTitle` in `internal/course/application/service.go` (renamed from `courseTitleAndSlug` — slug derivation moved out of this function, see `openspec/changes/rework-course-slug-management`).
```

**Doc-1c. Replace the `ensureUniqueCourseSlug` asset block with two new entries:**

OLD:
```
### Asset: ensureUniqueCourseSlug (course infra)
- Name: `ensureUniqueCourseSlug`
- Type: Function (repo helper)
- Path: `internal/course/infra/repo_helpers.go`
- Purpose: After `SlugifyName`, allocate the first free slug among `base`, `base-2`, `base-3`, … for active `courses` rows (`uix_courses_slug_active`). **One** indexed query loads sibling slugs (`slug = base OR slug LIKE base-%`, filtered to numeric suffixes in Go); suffix pick is in-memory (not per-candidate DB round-trips). Used on create and when `title` changes on `PATCH /basic-info` (excludes current course id on update).
- Scope: Course module only — do not duplicate slug-collision logic in handlers or FE.
- Dependencies: GORM, `courses` table, `domain.ErrCourseInvalidSlug`.
- Current Usage: `CreateCourse`, `UpdateBasicInfo` in `internal/course/infra/repo_instructor.go`.
```

NEW:
```
### Asset: slug (shared)
- Name: `ValidateManualFormat`, `GenerateRandomSuffix`, `RetryWithSuffix`
- Type: Package (`internal/shared/slug`)
- Path: `internal/shared/slug/format.go`, `internal/shared/slug/suffix.go`
- Purpose: Domain-agnostic slug primitives — format validation (`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`, no leading/trailing hyphen), `crypto/rand`-based random alphanumeric suffix generation, and an escalating-retry helper (`RetryWithSuffix`) that tries a 7-char suffix ×7 attempts, then 8×7, 9×7, … until a caller-supplied `accept` callback reports success. No GORM/HTTP/course imports — `accept` is the caller's own source of truth (a SELECT-based pre-check or the actual write attempt), so this package has no opinion on how uniqueness is actually enforced.
- Scope: Any domain needing a validated, collision-safe slug. Introduced by, and currently used only by, course (`openspec/changes/rework-course-slug-management`) — written generically so a future taxonomy/media/instructor slug feature can reuse it without modification.
- Dependencies: Go stdlib only (`regexp`, `strings`, `crypto/rand`, `io`).
- Current Usage: `internal/course/infra/slug.go`, `internal/course/application/service.go`, `internal/course/delivery/handler_instructor.go`.
- Reuse: Prefer this over writing a new domain-specific slug-collision algorithm; this is the shared primitive layer taxonomy/media/instructor should build on when they add their own slug features.

### Asset: Course slug orchestration (course infra)
- Name: `generateAutoSlugBase`, `courseSlugAvailable`
- Type: Function (repo helper)
- Path: `internal/course/infra/slug.go`
- Purpose: Course-specific slug orchestration built on top of `internal/shared/slug` and `github.com/gosimple/slug`. `generateAutoSlugBase(title)` transliterates `title` via `gosimple/slug` for auto-generation (falls back to base `"course"`, always suffixed, when transliteration yields nothing usable — e.g. an emoji-only title). `courseSlugAvailable(ctx, db, slug, excludeCourseID)` checks uniqueness among active `courses` rows, optionally excluding one course id (used on update so resubmitting the current slug is never treated as a self-collision). Collision resolution uses `sharedslug.RetryWithSuffix` (random suffix, escalating length: 7×7, 8×7, 9×7, …) — replaces the old `base`, `base-2`, `base-3`, … numeric-suffix algorithm entirely (`ensureUniqueCourseSlug` and its helpers were removed, not deprecated in place).
- Scope: Course module only — do not duplicate slug-collision logic in handlers or FE.
- Dependencies: `internal/shared/slug`, `github.com/gosimple/slug`, GORM, `courses` table.
- Current Usage: `createCourseOnce`/`resolveCreateSlug`, `UpdateBasicInfo`/`applyUpdateSlugWithRetry`/`resolveUpdateSlug` in `internal/course/infra/repo_instructor.go`.
```

### Doc-2. `docs/curl_api.md`

**Doc-2a. §14.1 request body description:**

OLD:
```
Request body: `{ "title": string }` only. Slug is computed server-side from `title` via `utils.SlugifyName` (not accepted from clients).
```

NEW:
```
Request body: `title` (required, `nonwhitespace_min=5`, max 255) plus an optional `slug` (string, `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`, max 255). Omit `slug` (or send `""`/whitespace) to auto-generate one from `title` via `github.com/gosimple/slug` (falls back to `course-{randomSuffix}` if the title has no transliterable characters, e.g. emoji-only). A manually supplied `slug` that is already taken by another active course is rejected with `409`/`3007` and a `data.recommended_slug` suggestion instead of being created — see the response table below.
```

**Doc-2b. §14.1 example — append a manual-slug + conflict example after the existing curl block (existing block unchanged):**

APPEND after the existing `curl -sS -X POST ".../courses" -d '{"title":"Introduction to Go"}'` example:
```
With a manual slug:

​```bash
curl -sS -X POST "{{BASE_URL}}/api/v1/courses" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Introduction to Go","slug":"golang-course"}'
​```

On conflict (`golang-course` already taken):

​```json
{
  "code": 3007,
  "message": "Slug already exists",
  "data": {
    "recommended_slug": "golang-course-x7k92ab"
  }
}
​```
```
(remove the zero-width-space-escaped ​``` markers above when actually pasting — they're only here so this fenced example doesn't prematurely close this design.md code block; use plain ``` in the real file.)

**Doc-2c. §14.1 response code table:**

OLD:
```
| HTTP | `code` | Notes |
|------|--------|-------|
| 201 | 0 | `data` = `CourseDetail` (draft v1, owner collaborator, empty outline) |
| 400 | 3001 | Empty slug after slugify (`invalid slug`) |
| 401 | 3002 | Missing/invalid JWT |
| 403 | 3003 | Missing `course:create` |
| 409 | 3005 | Duplicate slug (unique constraint) |
| 500 | 9001 | Internal error |
```

NEW:
```
| HTTP | `code` | Notes |
|------|--------|-------|
| 201 | 0 | `data` = `CourseDetail` (draft v1, owner collaborator, empty outline); `data.course.slug` is the actual final slug stored — always trust the response, not the request |
| 400 | 3001 | Manually supplied `slug` fails format validation (not `a-z0-9-`, or starts/ends with `-`) |
| 401 | 3002 | Missing/invalid JWT |
| 403 | 3003 | Missing `course:create` |
| 409 | 3007 | Manually supplied `slug` already exists — `data.recommended_slug` suggests an available alternative (requested slug + random suffix); resubmit create with the accepted or a different slug |
| 500 | 9001 | Internal error |
```
(the old `409 | 3005` and `400 | 3001 | Empty slug after slugify` rows are both retired — see design.md Context: a bare unique-constraint 409 without `recommended_slug` should no longer normally reach a client, and empty/omitted slug now means "auto-generate," not "error.")

**Doc-2d. §14.1 example JSON response** — no change needed; `"slug": "introduction-to-go"` remains valid for the no-manual-slug case.

**Doc-2e. §14.4 request body description:**

OLD:
```
Request body: `expected_row_version` (required, `>= 1`) plus required metadata fields (`title` ≥5 non-whitespace, server slugify; `short_description`, `about_course`, `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids`, `skill_ids`, `outcome_ids`). `preview_video_file_id` is optional.
```

NEW:
```
Request body: `expected_row_version` (required, `>= 1`) plus required metadata fields (`title` ≥5 non-whitespace; `short_description`, `about_course`, `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids`, `skill_ids`, `outcome_ids`). `preview_video_file_id` and `slug` are optional: `slug` uses PATCH-omit semantics — omit the field entirely to leave the course's slug unchanged (even when `title` changes; `title` no longer affects `slug` at all), send a new valid value to change it, or send `null`/`""`/whitespace to get a `400` validation error (explicit empty is invalid on update, unlike on create). A new slug identical to the current one is a no-op, not a conflict. A new slug that collides with another active course is auto-resolved server-side with a random suffix (no confirmation step, unlike create) — the response's `slug` field always reflects the actual final value.
```

**Doc-2f. §14.4 example — append a slug-update example after the existing curl block (existing block unchanged):**
```
Changing the slug in the same call:

​```bash
curl -sS -X PATCH "{{BASE_URL}}/api/v1/courses/{{courseId}}/basic-info" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expected_row_version":1,"title":"Introduction to Go","short_description":"Short blurb","slug":"advanced-golang"}'
​```
```

### Doc-3. `docs/api_swagger.yaml`

**Doc-3a. New component schema — insert after `EnvelopeOK` (before `EnvelopeHealth`), matching this file's existing structured-response convention (`LoginSessionTokensResponse`):**

INSERT:
```yaml
    SlugConflictResponse:
      type: object
      properties:
        code: { type: integer, example: 3007 }
        message: { type: string, example: Slug already exists }
        data:
          type: object
          properties:
            recommended_slug: { type: string, example: golang-course-x7k92ab }
```

**Doc-3b. `POST /api/v1/courses` block:**

OLD:
```yaml
  /api/v1/courses:
    post:
      tags: [Course]
      summary: Create course root
      security:
        - bearerJwt: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title]
              properties:
                title: { type: string, minLength: 1, maxLength: 255 }
              description: Course slug is computed server-side from title (utils.SlugifyName). Request fails only when slugify yields an empty string.
      responses:
        "201":
          description: Created
```

NEW:
```yaml
  /api/v1/courses:
    post:
      tags: [Course]
      summary: Create course root
      security:
        - bearerJwt: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title]
              properties:
                title: { type: string, minLength: 1, maxLength: 255 }
                slug:
                  type: string
                  pattern: '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'
                  maxLength: 255
                  description: Optional. Omit or send empty to auto-generate from title. A value that collides with another active course's slug is rejected (see 409) instead of being created.
              description: slug is optional — when omitted or empty, one is auto-generated from title (transliterated to ASCII, falling back to a random suffix if the title has no transliterable characters). A manually supplied slug must match the pattern and must not already be taken by another active course.
      responses:
        "201":
          description: Created
        "409":
          description: Manually supplied slug already exists (app code 3007)
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SlugConflictResponse"
```

**Doc-3c. `PATCH /api/v1/courses/{courseId}/basic-info` block** (also fixes the pre-existing missing `title` property, confirmed to fix in this same hunk):

OLD:
```yaml
  /api/v1/courses/{courseId}/basic-info:
    patch:
      tags: [Course]
      summary: Update draft basic info
      security:
        - bearerJwt: []
      parameters:
        - name: courseId
          in: path
          required: true
          schema: { type: integer }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [expected_row_version]
              properties:
                expected_row_version:
                  { type: integer, format: int64, minimum: 1 }
                short_description: { type: string }
                about_course: { type: string }
                thumbnail_file_id: { type: string, format: uuid }
                preview_video_file_id: { type: string, format: uuid }
                course_level_id: { type: integer }
                course_topic_id: { type: integer }
                tag_ids:
                  type: array
                  items: { type: integer }
                skill_ids:
                  type: array
                  items: { type: integer }
                outcome_ids:
                  type: array
                  items: { type: integer }
      responses:
        "200":
          description: Updated
```

NEW:
```yaml
  /api/v1/courses/{courseId}/basic-info:
    patch:
      tags: [Course]
      summary: Update draft basic info
      security:
        - bearerJwt: []
      parameters:
        - name: courseId
          in: path
          required: true
          schema: { type: integer }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [expected_row_version, title]
              properties:
                expected_row_version:
                  { type: integer, format: int64, minimum: 1 }
                title: { type: string, minLength: 1, maxLength: 255 }
                slug:
                  type: string
                  pattern: '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'
                  maxLength: 255
                  description: Optional, PATCH-omit semantics — deliberately NOT `nullable` (this field must never accept a JSON null as a valid value, unlike a true optional/nullable field). Omit this property entirely from the request body to leave the current slug unchanged (title no longer affects slug). Sending it as null, empty, or whitespace-only is a 400 validation error, not an accepted value. A value equal to the current slug is a no-op. A value colliding with another active course's slug is auto-resolved with a random suffix (no confirmation step) — the response always reflects the actual final slug.
                short_description: { type: string }
                about_course: { type: string }
                thumbnail_file_id: { type: string, format: uuid }
                preview_video_file_id: { type: string, format: uuid }
                course_level_id: { type: integer }
                course_topic_id: { type: integer }
                tag_ids:
                  type: array
                  items: { type: integer }
                skill_ids:
                  type: array
                  items: { type: integer }
                outcome_ids:
                  type: array
                  items: { type: integer }
      responses:
        "200":
          description: Updated
```

### Doc-4. `docs/router.md`

OLD:
```
| PATCH | `/api/v1/courses/:courseId/basic-info` | `course:update` | Update draft basic info (`title` → server slugify updates `courses.slug`) |
```

NEW:
```
| PATCH | `/api/v1/courses/:courseId/basic-info` | `course:update` | Update draft basic info (optional independent `slug` field, PATCH-omit semantics — `title` no longer affects `slug`) |
```

### Doc-5. `docs/return_types.md`

**Doc-5a. `CreateCourse` error-list row:**

OLD:
```
| `CreateCourse` | `CreateCourse(ctx, CreateCourseInput) (*CourseDetail, error)` | `*CourseDetail` on success; `ErrCourseInvalidSlug`; repo errors |
```

NEW:
```
| `CreateCourse` | `CreateCourse(ctx, CreateCourseInput) (*CourseDetail, error)` | `*CourseDetail` on success; `ErrCourseTitleTooShort`, `ErrCourseInvalidSlug` (manual slug fails format); `*domain.SlugConflictError` (manual slug taken — carries `RecommendedSlug`); repo errors |
```

**Doc-5b. "Create input" prose:**

OLD:
```
**Create input:** service layer accepts `{ title }`, slugifies title, passes `CreateCourseInput{ ActorUserID, Title, Slug }` to repository. Repository calls `ensureUniqueCourseSlug` (`base`, `base-2`, …) then assigns UUID v7 ids via `gormx.EnsureStringID` before inserting `courses` and `course_versions`. No collaborator row is inserted: ownership is `courses.owner_user_id` itself, synthesized by `CoursePolicyProvider` (see `docs/modules/authorization.md`), never a stored `authorization_role_bindings` row.
```

NEW:
```
**Create input:** service layer accepts `{ title, slug? }` and passes `CreateCourseInput{ ActorUserID, Title, Slug }` to the repository — `Slug` is either the caller's manually supplied, already format-validated value, or `""` meaning "auto-generate." Repository logic (`internal/course/infra/slug.go`, `repo_instructor.go`): a non-empty manual slug is checked for availability and used as-is if free, or returns `*domain.SlugConflictError{RecommendedSlug}` if taken (never silently substituted); an empty slug derives a base from `title` via `github.com/gosimple/slug` (falling back to `course-{randomSuffix}` if transliteration yields nothing usable) and resolves any collision with `internal/shared/slug.RetryWithSuffix` (random alphanumeric suffix, escalating length: 7 chars ×7 attempts, then 8×7, 9×7, …) — replacing the old `base`, `base-2`, `base-3`, … numeric-suffix algorithm. UUID v7 ids are then assigned via `gormx.EnsureStringID` before inserting `courses` and `course_versions`. No collaborator row is inserted: ownership is `courses.owner_user_id` itself, synthesized by `CoursePolicyProvider` (see `docs/modules/authorization.md`), never a stored `authorization_role_bindings` row.
```

**Doc-5c. "Update basic info input" prose:**

OLD:
```
**Update basic info input:** `UpdateBasicInfoInput` carries `expected_row_version` and draft metadata fields. Delivery layer requires all basic-info fields on PATCH (except optional `preview_video_file_id`); handler passes trimmed pointers. When `title` is set, service slugifies via `courseTitleAndSlug` (≥5 non-whitespace) and `ensureUniqueCourseSlug` (excluding current course).
```

NEW:
```
**Update basic info input:** `UpdateBasicInfoInput` carries `expected_row_version` and draft metadata fields. Delivery layer requires all basic-info fields on PATCH except `preview_video_file_id` and `slug`, both optional; handler passes trimmed pointers. `title` is validated (≥5 non-whitespace via `validateCourseTitle`) but no longer affects `slug` in any way. `Slug *string` is independent: `nil` (field omitted) means leave the stored slug unchanged; a non-nil value has already been trimmed and format-validated by the delivery handler (an explicit empty/whitespace value is rejected as `400` before reaching the service). The repository skips all slug work when the new value equals the current slug (no false self-conflict), otherwise resolves any collision directly via `internal/shared/slug.RetryWithSuffix` (random suffix, no confirmation step) and persists the actual final slug, which the response reflects.
```

**Doc-5d. §POST /api/v1/courses request/response section:**

OLD:
```
**Request body:** `{ "title": string }` (required, 1–255 chars). Slug is server-computed; not in request.

| Status | `code` | `data` |
|--------|--------|--------|
| 201 | 0 | `domain.CourseDetail` — `slug` is globally unique among active courses (`uix_courses_slug_active`; duplicate titles get `-2`, `-3`, … suffixes) |
| 400 | 3001 | `null` — empty slug after slugify |
| 401 | 3002 | `null` |
| 403 | 3003 | `null` — missing permission |
| 500 | 9001 | `null` |
```

NEW:
```
**Request body:** `title` (required, 1–255 chars, `nonwhitespace_min=5`) plus optional `slug` (string, `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`, max 255). Omit/empty `slug` to auto-generate from `title`.

| Status | `code` | `data` |
|--------|--------|--------|
| 201 | 0 | `domain.CourseDetail` — `slug` is globally unique among active courses (`uix_courses_slug_active`); collisions on an auto-generated or accepted-recommendation slug are resolved with a random suffix, not `-2`, `-3`, … |
| 400 | 3001 | `null` — manually supplied `slug` fails format validation |
| 401 | 3002 | `null` |
| 403 | 3003 | `null` — missing permission |
| 409 | 3007 | `{ "recommended_slug": string }` — manually supplied `slug` already exists |
| 500 | 9001 | `null` |
```

### Doc-6. `docs/database.md`

**Doc-6a. New `### courses` section — insert immediately after the `## Course management tables (000016)` summary table's last row:**

INSERT:
```markdown
### `courses`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | `UUID` | PK | |
| `owner_user_id` | `UUID` | NOT NULL, FK → `users(id)` | Canonical ownership; synthesized as the `OWNER` role by `CoursePolicyProvider`, never a stored `authorization_role_bindings` row |
| `slug` | `VARCHAR(255)` | NOT NULL, UNIQUE (`uix_courses_slug_active`, partial where `deleted_at IS NULL`), CHECK `chk_courses_slug_format` (`slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'`, added in migration `000040`) | URL-safe identifier; optional manual input on create (auto-generated from title if omitted) or update (PATCH-omittable — unchanged if the field is absent from the request); see `openspec/changes/rework-course-slug-management` |
| `current_published_version_id` | `UUID` | nullable, FK → `course_versions(id)` | Learner-facing published snapshot |
| `current_draft_version_id` | `UUID` | nullable, FK → `course_versions(id)` | Instructor-facing editable snapshot |
| `trashed_at` | `BIGINT` | nullable | Unix epoch seconds; in trash when set (distinct from `deleted_at`) |
| `created_at` | `BIGINT` | NOT NULL DEFAULT `EXTRACT(EPOCH FROM NOW())::BIGINT` | Unix epoch seconds |
| `updated_at` | `BIGINT` | NOT NULL DEFAULT `EXTRACT(EPOCH FROM NOW())::BIGINT` | Unix epoch seconds |
| `deleted_at` | `BIGINT` | nullable | Soft delete (Unix epoch seconds) |

**Indexes:** `uix_courses_slug_active` (partial unique on `slug` where `deleted_at IS NULL`); see `migrations/000038_home_catalog_indexes` for the additional `idx_courses_published_created_at`/`idx_courses_owner_published_created_at` pair.

**`chk_courses_slug_format` (migration `000040`):** database-level backstop so `courses.slug` can never be `NULL`, empty, or malformed even if application-layer validation has a bug. Migration `000039_backfill_course_slug_ascii` runs first to regenerate any pre-existing non-conforming row (e.g. a non-ASCII slug from the pre-`000040` version of `utils.SlugifyName`) so `000040` cannot fail to apply.
```

**Doc-6b. Append both new migrations to the migrations table, right after the `000038` row:**

APPEND (same table, same row format):
```
| 000039 | `backfill_course_slug_ascii` | Data-only, no schema change: regenerates any pre-existing `courses.slug` value that would violate the new format check (idempotent; no-op `down.sql` — original non-conforming values are not recoverable), see `openspec/changes/rework-course-slug-management` |
| 000040 | `course_slug_check_constraint` | Adds `chk_courses_slug_format` CHECK constraint on `courses.slug` (`slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'`); `uix_courses_slug_active`/`NOT NULL` unchanged, see `openspec/changes/rework-course-slug-management` |
```

### Doc-7. `docs/security-public-seo-notes.md`

OLD:
```
| B8 | Slug uniqueness | `ensureUniqueCourseSlug` | Canonical slug future; no public resolve-by-slug yet. |
```

NEW:
```
| B8 | Slug uniqueness | `internal/shared/slug` (`RetryWithSuffix`) + `internal/course/infra/slug.go` (`courseSlugAvailable`) | Canonical slug future; no public resolve-by-slug yet. `ensureUniqueCourseSlug` (numeric-suffix algorithm) was removed and replaced by a random-suffix algorithm in `openspec/changes/rework-course-slug-management`. |
```

### Doc-8. `docs/modules/course.md`

**Doc-8a. Core model slug bullet:**

OLD:
```
  - `slug` (derived from `title` via `utils.SlugifyName` on create and when `title` changes on `PATCH /basic-info`; not accepted from clients; globally unique among active rows — colliding base slugs get `-2`, `-3`, … suffixes)
```

NEW:
```
  - `slug` (globally unique among active rows, `chk_courses_slug_format` CHECK-enforced ASCII `a-z0-9-` format). On create: optional client input — a manual value is validated and used if free, or rejected with a `SlugConflictError` + `recommended_slug` if taken; an omitted/empty value auto-generates from `title` via `github.com/gosimple/slug`, falling back to `course-{randomSuffix}` when transliteration yields nothing usable. On update (`PATCH /basic-info`): an independent, PATCH-omittable field — omitted leaves it unchanged (title no longer touches slug at all), a new value equal to the current one is a no-op, and a new value colliding with another course is auto-resolved directly with a random suffix (no confirmation step). Collision suffixes are always random alphanumeric (`crypto/rand`, 7 chars ×7 attempts, then 8×7, 9×7, …), never the old numeric `-2`, `-3`, … scheme. See `openspec/changes/rework-course-slug-management`.
```

**Doc-8b. Optimistic locking bullet:**

OLD:
```
  - `PATCH /basic-info` requires `expected_row_version >= 1` and increments `row_version` on success; accepts `title` (server recomputes `courses.slug` with the same uniqueness rules as create)
```

NEW:
```
  - `PATCH /basic-info` requires `expected_row_version >= 1` and increments `row_version` on success; accepts `title` (no longer touches `slug`) and an independent, optional `slug` field (omitted = unchanged; explicit empty/whitespace = `400`; new value auto-resolved on collision with a random suffix, no confirmation step)
```

**Doc-8c. HTTP routes bullets:**

OLD:
```
- `POST /api/v1/courses` — body `{ "title" }` only (`nonwhitespace_min=5`, max 255); slug is computed server-side from `title` via `SlugifyName`. When the base slug is already used by another active course, the server allocates the next free variant (`base`, then `base-2`, `base-3`, …) so each active course has a globally unique slug (`uix_courses_slug_active`).
- `PATCH /api/v1/courses/:courseId/basic-info` — all listed fields required on save except `preview_video_file_id` (optional UUID): `title` (≥5 non-whitespace, server slugify), `short_description` (≥20), `about_course` (Delta JSON, ≥30 non-whitespace text), `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids` (≥1), `skill_ids` (≥1), `outcome_ids` (exactly 1), `expected_row_version`.
```

NEW:
```
- `POST /api/v1/courses` — body `title` (required, `nonwhitespace_min=5`, max 255) plus optional `slug` (`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`, max 255). Omitted/empty `slug` auto-generates from `title` via `github.com/gosimple/slug` (falls back to `course-{randomSuffix}` if nothing transliterates). A manually supplied `slug` already used by another active course is rejected (`409`/`SlugConflict`) with a `recommended_slug` suggestion — never silently auto-suffixed. `uix_courses_slug_active` and `chk_courses_slug_format` remain the DB-level source of truth.
- `PATCH /api/v1/courses/:courseId/basic-info` — all listed fields required on save except `preview_video_file_id` and `slug` (both optional): `title` (≥5 non-whitespace; no longer affects `slug`), `short_description` (≥20), `about_course` (Delta JSON, ≥30 non-whitespace text), `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids` (≥1), `skill_ids` (≥1), `outcome_ids` (exactly 1), `expected_row_version`. `slug`, when present, is independent of `title`: omitted from the request leaves it unchanged; `null`/`""`/whitespace is a `400`; a new value equal to the current slug is a no-op; a new value colliding with another course is auto-resolved server-side with a random suffix (no confirmation step) and the response reflects the actual final value.
```

### Doc-9. No change needed

`docs/modules.md`, `docs/patterns.md`, `docs/modules/taxonomy.md` — confirmed taxonomy-scoped or generic and unaffected by two independent audits; `docs/modules/taxonomy.md`'s "do not create a second slug util" line is addressed by explanation in this design (D3), not by editing that file, since taxonomy's own behavior does not change.

## Risks / Trade-offs

- **[Risk] `go-playground/validator`'s `omitempty` cannot distinguish a nil pointer from a pointer-to-empty-string** → Mitigated by D6: no `validate` tag on `updateBasicInfoRequest.Slug`; the nil-check happens explicitly in Go in the handler before any struct-tag validation runs.
- **[Risk] Removing `ensureUniqueCourseSlug` et al. outright (vs. keeping them dead) could look like an accidental behavior loss to a reviewer scanning the diff** → Mitigated by this design doc + `docs/reusable-assets.md` update explicitly recording the old algorithm was replaced, not lost; `check-dupl`/lint would flag leftover dead code if kept, so removal is also required by this repo's own quality gates.
- **[Risk, found and fixed during `/opsx:apply`] The backfill migration (D8) was originally drafted as a `DO $$ ... $$` PL/pgSQL block with a nested collision-retry loop** — this repo's migration runner splits files on every literal `;` (`migrations/README.md`), which would have corrupted a procedural block with internal semicolons, exactly the trap `000015`/`000032` already had to work around → Fixed by rewriting as a single plain `UPDATE` statement using a 16-hex-character `md5()`-derived suffix (astronomically low collision probability, no retry loop needed), matching the precedent of `000020_course_version_row_version_backfill`.
- **[Risk] No dev database was reachable inside the sandbox this change was implemented in** (no local Postgres running, Docker unavailable, `.env`'s `DB_HOST`/`DB_USER`/`DB_NAME` empty) → The migration SQL was verified by direct inspection and by confirming `//go:embed *.sql` picks up both new files cleanly (`go build ./migrations/...`), but tasks 3.1-3.5 and 8.10 (the actual `SELECT`/apply/rollback/curl verification against a live DB) could not be executed in this session and are deferred to the user's own dev DB — this is reported honestly in the final task status, not claimed as done.
- **[Trade-off] `courseSlugCreateRetry` increasing from 3 to a larger constant** trades a small amount of worst-case latency under heavy concurrent-create contention for correctness; still bounded and rare (only matters when many callers race on the exact same base slug simultaneously).
- **[Risk] `github.com/gosimple/slug` is a new third-party dependency** — mitigated: its transliteration behavior was verified against real table data (not the README's claims alone) before this decision, and it is used in a narrow, isolated call site (`generateAutoSlugBase`) that is easy to swap later if needed.

## Migration Plan

1. Add `internal/shared/slug` (D1) — no callers yet, safe to land alone, fully unit-testable in isolation.
2. Add `SlugConflict` error code (D5) — additive, no behavior change yet since nothing calls it. (No validator-tag step here — D2 explains why `internal/shared/validate` is untouched.)
3. Add `go.mod` dependency on `github.com/gosimple/slug` (D3).
4. Rewrite `internal/course/infra` (D4, D7) and `internal/course/application`/`domain`/`delivery` (D6, D7) together, since they change as one coherent unit (cannot ship the DTO change without the service/repo change it depends on, or vice versa).
5. Run the DB row-count check from D8 against the dev DB; write `000039` (only if rows are found — still write it as documentation of the check either way) then `000040`; apply both, in that order, before/alongside step 4 landing (the app code and the CHECK constraint should ship together so there is never a window where invalid app-layer output could violate a not-yet-added constraint, nor a constraint active before app code guarantees conformance — sequencing this is a `tasks.md` ordering concern).
6. Update all docs listed above.
7. Run `gitnexus_detect_changes({scope: "all"})` and the full quality-gate set (`gofmt`, `golangci-lint`, `make test-all`, `make check-all`) per this repo's standing workflow.
8. Rollback strategy: both migrations have real `down.sql` (D8); the CHECK constraint (`000040`) can be dropped independently of the backfill (`000039`) if only the constraint needs reverting; application-code rollback is a normal git revert of steps 1-4 (additive package + swapped call sites, no partial-deploy compatibility shims needed since this is a single backend service, not a rolling multi-version API contract).

## Resolved decisions (confirmed with user after initial drafting)

- `courseSlugCreateRetry` is raised from `3` to **`5`** (confirmed, not just a default).
- The `PATCH .../basic-info` Swagger schema's pre-existing missing `title` property (found during the docs audit, unrelated to slug) **is fixed** in this change, as its own reviewable diff hunk within `docs/api_swagger.yaml` (confirmed — not deferred to a separate change).
- The backfill migration's (`000039`) no-op `down.sql` (irreversible — original non-conforming slug values are not recoverable once regenerated) **is accepted as-is** (confirmed — no backup table/column added for old slug values, since a non-ASCII slug has no legitimate use in a public URL going forward anyway).

## Open Questions

(none remaining — the three items above were the only open questions and are now resolved)
