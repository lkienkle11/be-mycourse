## Context

See `proposal.md - Why` for motivation. Verified directly in code during `/opsx:explore` (this repo's prior session, `.context/session-*` history), not assumed:

- `internal/course/infra/repo_instructor.go:229-329` (current file): `UpdateBasicInfo` wraps its whole body in **one** `r.db.WithContext(ctx).Transaction(...)`. Inside it, `applyUpdateSlugWithRetry` (lines 279-305) loops up to 5 times calling `resolveUpdateSlug` (a SELECT) then `tx.Model(&courseRow{}).Updates(...)` (a write) **on that same `tx`**, with no `SavePoint`/`RollbackTo` anywhere in the file or the whole repo (`grep -rn "SavePoint" .` → 0 hits). PostgreSQL aborts the **entire** transaction on any statement error (a unique-violation included) — every subsequent statement on that same transaction, including the "retry," then fails with `current transaction is aborted, commands ignored until end of transaction block` (`25P02`) instead of running at all. Net effect: on a real update-time slug collision, the retry loop's 2nd iteration's `resolveUpdateSlug` SELECT itself fails with this generic error (not a duplicate-key error `isCourseSlugDuplicateKey` recognizes), `applyUpdateSlugWithRetry` returns it immediately, and the whole request — including unrelated title/description/thumbnail changes made earlier in the same request — rolls back and surfaces as a generic `500`, instead of the documented "auto-resolved via random suffix, no confirmation step" behavior `specs/course/slug-management/spec.md` already requires. `CreateCourse`'s own retry loop (`repo_instructor.go:892-904`) does **not** have this bug — each retry attempt (`createCourseOnce`) opens a **fresh** `r.db.Transaction(...)`, not a retry on an already-open one, so it never hits an aborted-transaction state.
- `internal/taxonomy/application/service.go` (current file): `CreateTopic` (line 92), `applyCourseTopicUpdate` (line 142), `CreateCourseSkill` (line 345), `UpdateCourseSkill` (line 377), and `service_helpers.go`'s `applySlugStatusFields` (line 133, shared by Tag/CourseLevel) all call `slugFromName(name) = strings.TrimSpace(utils.SlugifyName(name))` **unconditionally** on every create and update — there is no manual-slug input field in any DTO (`internal/taxonomy/delivery/dto.go:42-122`), and update recomputes slug from `name`/`translations.en.name` every time (`internal/shared/taxonomy/tree_resolve.go`'s `SyncCanonicalAndEn` keeps `name` and `translations.en.name` mirrored, so there is no way to change one without the other today — see D3 below for why this matters for the omit-preserves-slug requirement).
- `internal/taxonomy/infra/repos.go`/`repos_mappers.go`/`repos_crud_helper.go` (current files): no function anywhere in `internal/taxonomy` recognizes a Postgres duplicate-key error (`grep -rn "duplicate key\|unique_violation\|23505" internal/taxonomy/` → 0 hits). A raw duplicate-key insert/update bubbles up through `mapTaxonomyMutationError`'s final fallback (`handler_helpers.go:107`) as **HTTP 400 with the raw Postgres driver message text exposed to the client** — a real information-leak gap independent of whether manual slugs are added, since taxonomy's existing auto-derive-from-name path can already race today.
- `internal/shared/taxonomy/tree_slug.go:11-27` (`NormalizeTreeSlugs`) unconditionally recomputes **every** node's slug from its `Name` on every write, including the "tree omitted from the PATCH body" branches (`internal/taxonomy/application/service.go:153-165` for topics, `:394-404` for skills) which reload the **stored** tree from the DB and re-run it through the exact same normalize step — today this is a no-op only because no manual node slug can ever exist to lose. `internal/shared/taxonomy/tree_validate.go:57-63,108-111` already tracks a `seenSlug` map across the **whole tree** (all depths, one shared map — not per-parent-array, not global-across-rows) and rejects a duplicate with a bare `"duplicate tree node slug"` error that does not identify which node.
- `courses.slug`'s reference implementation (fully shipped, `openspec/changes/rework-course-slug-management`, archived-in-place but not yet moved to a main `openspec/specs/` — there is no `openspec/specs/` directory in this repo yet, confirmed by `ls`): `internal/shared/slug/{format.go,suffix.go}` (`ValidateManualFormat(raw string, maxLen int) (string, bool)`, `GenerateRandomSuffix`, `RetryWithSuffix`, `TruncateToLen`, `TruncateForSuffix`) and `internal/shared/validate.Optional[T]` (`internal/shared/validate/optional.go`) are domain-agnostic and **used unchanged** by this design — no edits to either package. `internal/course/domain/errors.go:38-49`'s `SlugConflictError{RecommendedSlug string}` + `internal/shared/errors/errcode_codes.go:37`'s `SlugConflict = 3007` are reused as-is (no new app error code needed).
- The 4 taxonomy root tables already have the correct partial unique index shape (`uix_course_levels_slug_active`, `uix_course_topics_slug_active`, `uix_course_skills_slug_active`, `uix_tags_slug_active`, all `WHERE deleted_at IS NULL`, added by `migrations/000012_soft_delete_taxonomy_users_ban.up.sql`) — this part needs **no** migration. What is missing, confirmed by `grep -rniE "CHECK" migrations/*.up.sql` (only hit: `migrations/000040_course_slug_check_constraint.up.sql:7`, `courses` only), is a format `CHECK` constraint on any of the 4 tables.
- Taxonomy's `created_at`/`updated_at` columns are `TIMESTAMPTZ` (`migrations/000002_taxonomy_domain.up.sql:9-10,22-23,34-35`, `migrations/000009_taxonomy_topics_outcomes_skills.up.sql:22-23,36-37`) — **not** `BIGINT` epoch seconds like `courses.updated_at` (`timex.NowUnix()`). `deleted_at` **is** `BIGINT` epoch (added by `migrations/000012...up.sql`). The backfill migration below uses `NOW()`, not course's `extract(epoch FROM now())::bigint` — a real, easy-to-miss difference between the two modules' schemas.
- `internal/taxonomy/domain/repository.go:6-16`: all 5 taxonomy repositories (`CourseTopicRepository`, `CourseOutcomeRepository`, `CourseSkillRepository`, `TagRepository`, `CourseLevelRepository`) are **type aliases of one shared generic interface** `taxonomyRepository[T]`, with `Create(ctx, t *T) error` / `Save(ctx, t *T, expectedRowVersion int64) error` as single-parameter methods. `CourseOutcome` has **no** `Slug` field at all. This constrains the design below: the repository **interface signatures cannot change** without also touching `CourseOutcome` (which must stay untouched, it has no slug) — see Decision D4 for how this is worked around with zero interface changes.

## Goals / Non-Goals

**Goals:**
- `course_levels`/`course_topics`/`course_skills`/`tags` create accepts an optional manual `slug` (omit/empty → auto-generate from `name`); update accepts an independent, omittable `slug` (omit → unchanged even if `name` changes) — matching course's contract shape exactly, reusing `internal/shared/slug` and `internal/shared/validate.Optional[T]` without modification.
- Tree nodes inside `child_topics`/`children` get the same manual-or-auto + omit-preserves-by-ID slug semantics, without changing the existing `TreeNode` read/response shape. (`NormalizeTreeSlugs`/`PrepareTreeForWrite` themselves were later deleted as dead code once a code review confirmed zero remaining callers — see D11 — rather than kept unused "for compatibility.")
- Close the raw-duplicate-key-error leak in taxonomy's mutation error mapping.
- Fix course's update-slug retry-in-an-aborted-transaction bug via one new, reusable, generic helper — used by both the course fix and taxonomy's own new update-retry logic, so the savepoint-retry pattern is written exactly once.
- Zero new database tables/columns beyond a `CHECK` constraint per taxonomy root table; zero new API routes; zero exported repository interface signature changes (see D4).

**Non-Goals:**
- Instructor slug ("vấn đề số 3") — a separate, later change, per the user's own framing of this change as its premise.
- `course_outcomes`, media object keys, OAuth `provider` strings — confirmed (during `/opsx:explore`) to have no user-facing content slug; untouched.
- Changing `utils.SlugifyName` itself, or making taxonomy switch to `github.com/gosimple/slug` like course did — taxonomy keeps its own existing transliteration utility; only a new ASCII-fallback wrapper is added around it (D2), matching the reasoning `rework-course-slug-management`'s own design.md already recorded for why the two modules use different base transliteration.
- A `course_slug_history` / redirect-on-old-slug mechanism — out of scope here exactly as it was for course's own change; no route in this backend resolves anything by slug today (confirmed: `grep -rn "func.*BySlug\|GetBySlug\|FindBySlug" internal/` → 0 hits), so there is nothing to redirect yet.
- Node-level slug uniqueness across different entities' trees — confirmed by reading `tree_validate.go` that today's scope is "within one entity's own tree," and this design keeps that scope unchanged (see `specs/taxonomy/tree-slug-management/spec.md`).

## Decisions

### D1 — New shared `internal/shared/gormx.RetryInSavepoint`; course bugfix built on it

Full new file `internal/shared/gormx/retry_savepoint.go`:
```go
package gormx

import "gorm.io/gorm"

// RetryInSavepoint runs attempt up to attempts times inside a SAVEPOINT taken
// on tx, rolling back to that savepoint (not the whole surrounding
// transaction) whenever attempt's error satisfies isRetryable — most
// commonly a Postgres unique-constraint violation the caller is about to
// retry with a freshly generated candidate.
//
// This exists because Postgres aborts an ENTIRE transaction on any statement
// error (unique violations included): every subsequent statement on that
// same transaction, including a naive "just try the write again" retry,
// then fails immediately with "current transaction is aborted" instead of
// running at all. A SAVEPOINT is the only way to recover from one failed
// statement without discarding everything already done earlier in the same
// outer transaction (e.g. other fields already updated in the same
// request). Found the hard way: internal/course/infra/repo_instructor.go's
// original applyUpdateSlugWithRetry retried a write on the same *gorm.DB
// without a savepoint and could never actually succeed past the first real
// collision — see design.md Context for the full account.
//
// attempt is responsible for doing its own work (e.g. re-resolving a
// candidate value before writing) on every call, including the first;
// RetryInSavepoint does not cache or reuse anything between attempts.
// Requires a driver that implements gorm.SavePointerDialectorInterface —
// gorm.io/driver/postgres does.
func RetryInSavepoint(tx *gorm.DB, name string, attempts int, isRetryable func(error) bool, attempt func() error) error {
	if err := tx.SavePoint(name).Error; err != nil {
		return err
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		err := attempt()
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		if rbErr := tx.RollbackTo(name).Error; rbErr != nil {
			return rbErr
		}
		lastErr = err
	}
	return lastErr
}
```

`internal/course/infra/repo_instructor.go`'s `applyUpdateSlugWithRetry` — full new content (replaces the current lines 279-305 exactly; `resolveUpdateSlug` at lines 307-329 is unchanged, not repeated here; `gormx` is already imported in this file, no new import needed):
```go
// applyUpdateSlugWithRetry closes a pre-existing race-safety gap:
// UpdateBasicInfo previously resolved a new slug and wrote it exactly once
// with no retry-on-duplicate-key, unlike CreateCourse. Scoped to only the
// slug write sub-step (not the whole surrounding transaction) so a slug-only
// collision retry does not re-validate/re-write the unrelated metadata
// fields already committed earlier in the same transaction. Uses
// gormx.RetryInSavepoint (see design.md D1) — an earlier version of this
// function retried on the same *gorm.DB without a SAVEPOINT and could not
// actually recover from a real collision; fixed here.
func (r *GormRepository) applyUpdateSlugWithRetry(ctx context.Context, tx *gorm.DB, access *courseAccess, newSlugInput *string) error {
	if newSlugInput == nil {
		return nil // omitted — do not touch courses.slug at all
	}
	newSlug := *newSlugInput
	if newSlug == access.Slug {
		return nil // no-op: identical to current slug, never a self-conflict
	}
	const updateSlugRetry = 5 // mirrors courseSlugCreateRetry; same tuning rationale
	return gormx.RetryInSavepoint(tx, "course_slug_update_retry", updateSlugRetry, isCourseSlugDuplicateKey, func() error {
		finalSlug, err := resolveUpdateSlug(ctx, tx, newSlug, access.ID)
		if err != nil {
			return err
		}
		return tx.Model(&courseRow{}).Where("id = ? AND deleted_at IS NULL", access.ID).
			Updates(map[string]any{"slug": finalSlug, "updated_at": timex.NowUnix()}).Error
	})
}
```
No other line in this file changes. No spec-level behavior changes — `specs/course/slug-management/spec.md`'s "Update slug conflict is auto-resolved, not reported" requirement already describes this exact contract; this is a pure implementation fix.

### D2 — New `internal/taxonomy/infra/slug.go`

Mirrors `internal/course/infra/slug.go`, generalized over the 4 taxonomy row types via a Go type parameter (this codebase already uses generics for taxonomy repo code — `slugStatusEntityRepo[T]`, `listHydratedWithImagesArgs[R,D]` — so this is consistent style, not a new pattern):
```go
package infra

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	sharedslug "mycourse-io-be/internal/shared/slug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	sharedutils "mycourse-io-be/internal/shared/utils"
	"mycourse-io-be/internal/taxonomy/domain"
)

// maxTaxonomySlugLen mirrors every taxonomy root table's slug VARCHAR(255)
// column limit (migrations/000002_taxonomy_domain.up.sql,
// migrations/000009_taxonomy_topics_outcomes_skills.up.sql) and reuses the
// same 255 already relied on by tree node slug validation
// (internal/shared/taxonomy/tree_validate.go's DefaultMaxSlugLen), so root
// and tree-node slug length limits cannot drift apart.
const maxTaxonomySlugLen = taxpkg.DefaultMaxSlugLen

// taxonomySlugRetryAttempts mirrors internal/course/infra/repo_instructor.go's
// courseSlugCreateRetry (create) / updateSlugRetry (update) — same tuning
// rationale (random-suffix collisions can need more attempts under
// contention than a single pre-check allows).
const taxonomySlugRetryAttempts = 5

// taxonomySlugAvailable reports whether slug is free among active rows of
// model's table, optionally excluding one row id (used on update so
// resubmitting the current slug is never treated as a self-collision).
// Generalizes internal/course/infra/slug.go's courseSlugAvailable over the
// GORM row type, since 4 taxonomy tables share the same shape
// (slug + deleted_at) instead of course's single table.
func taxonomySlugAvailable[R any](ctx context.Context, db *gorm.DB, model *R, slug string, excludeID *string) (bool, error) {
	q := db.WithContext(ctx).Model(model).Where("deleted_at IS NULL").Where("slug = ?", slug)
	if excludeID != nil {
		if id := strings.TrimSpace(*excludeID); id != "" {
			q = q.Where("id != ?", id)
		}
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// isTaxonomySlugDuplicateKey reports whether err is a Postgres
// unique-violation on one of the 4 taxonomy slug partial unique indexes
// (migrations/000012_soft_delete_taxonomy_users_ban.up.sql). Mirrors
// internal/course/infra/repo_helpers.go's isCourseSlugDuplicateKey exactly
// (same message-substring detection — this repo's established way to
// recognize a duplicate-key error without a driver-specific type assertion).
func isTaxonomySlugDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "duplicate key") {
		return false
	}
	for _, idx := range []string{
		"uix_course_levels_slug_active",
		"uix_course_topics_slug_active",
		"uix_course_skills_slug_active",
		"uix_tags_slug_active",
	} {
		if strings.Contains(msg, idx) {
			return true
		}
	}
	return false
}

// generateAutoTaxonomySlugBase derives a base slug from name via this
// module's existing utils.SlugifyName (unchanged — course intentionally
// uses a different library, github.com/gosimple/slug, for its own
// auto-generation; that divergence is untouched by this change), truncated
// to maxTaxonomySlugLen BEFORE format validation. An empty or still-invalid
// result after truncation (e.g. a name using a script SlugifyName cannot
// fully transliterate to ASCII) falls back to fallbackPrefix and signals the
// caller to always suffix it — mirrors internal/course/infra/slug.go's
// generateAutoSlugBase "course" fallback.
func generateAutoTaxonomySlugBase(name, fallbackPrefix string) (base string, mustSuffix bool) {
	candidate := sharedslug.TruncateToLen(sharedutils.SlugifyName(name), maxTaxonomySlugLen)
	if trimmed, ok := sharedslug.ValidateManualFormat(candidate, maxTaxonomySlugLen); ok {
		return trimmed, false
	}
	return fallbackPrefix, true
}

// buildTaxonomySuffixedCandidate truncates base to leave room for
// "-"+suffix within maxTaxonomySlugLen, erroring if no room remains at all —
// mirrors internal/course/infra/slug.go's buildSuffixedCandidate exactly
// (same guard against an unreachable leading-hyphen candidate, found by
// code review during course's own rework).
func buildTaxonomySuffixedCandidate(base, suffix string) (string, error) {
	truncatedBase := sharedslug.TruncateForSuffix(base, maxTaxonomySlugLen, len(suffix))
	if truncatedBase == "" {
		return "", fmt.Errorf("slug base %q leaves no room for a %d-character suffix within %d characters", base, len(suffix), maxTaxonomySlugLen)
	}
	return truncatedBase + "-" + suffix, nil
}

// resolveTaxonomyCreateSlug implements the create-time slug branch, shared
// by all 4 taxonomy root entities via the R type parameter for the target
// GORM row — same role as internal/course/infra/repo_instructor.go's
// resolveCreateSlug, generalized since taxonomy's 4 tables share one shape.
// manualSlug is "" for auto-generate; a non-empty value has already been
// trimmed+format-validated by the application layer (mirrors course's own
// division of responsibility — see D5).
func resolveTaxonomyCreateSlug[R any](
	ctx context.Context, tx *gorm.DB, model *R,
	manualSlug, name, fallbackPrefix string,
) (string, *domain.SlugConflictError, error) {
	if manualSlug != "" {
		available, err := taxonomySlugAvailable(ctx, tx, model, manualSlug, nil)
		if err != nil {
			return "", nil, err
		}
		if available {
			return manualSlug, nil, nil
		}
		recommended, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
			candidate, err := buildTaxonomySuffixedCandidate(manualSlug, suffix)
			if err != nil {
				return "", false, err
			}
			ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, nil)
			return candidate, ok, err
		})
		if err != nil {
			return "", nil, err
		}
		return "", &domain.SlugConflictError{RecommendedSlug: recommended}, nil
	}
	base, mustSuffix := generateAutoTaxonomySlugBase(name, fallbackPrefix)
	if !mustSuffix {
		available, err := taxonomySlugAvailable(ctx, tx, model, base, nil)
		if err != nil {
			return "", nil, err
		}
		if available {
			return base, nil, nil
		}
	}
	final, err := sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildTaxonomySuffixedCandidate(base, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, nil)
		return candidate, ok, err
	})
	return final, nil, err
}

// resolveTaxonomyUpdateSlug implements the update-time slug branch — no
// conflict-error path exists here (update auto-resolves directly via random
// suffix, per the product decision that taxonomy update mirrors course's
// UPDATE behavior, not its CREATE behavior — see proposal.md). Also used for
// the "slug omitted" case: the caller passes the entity's own
// already-loaded, unchanged slug as newSlug, which this function's
// excludeID-scoped availability check always finds available (it belongs to
// this same row), returning it unchanged — a deliberate simplification so
// no separate "was slug touched" branch is needed at this layer (see D5).
// Mirrors internal/course/infra/repo_instructor.go's resolveUpdateSlug.
func resolveTaxonomyUpdateSlug[R any](ctx context.Context, tx *gorm.DB, model *R, newSlug, excludeID string) (string, error) {
	available, err := taxonomySlugAvailable(ctx, tx, model, newSlug, &excludeID)
	if err != nil {
		return "", err
	}
	if available {
		return newSlug, nil
	}
	return sharedslug.RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := buildTaxonomySuffixedCandidate(newSlug, suffix)
		if err != nil {
			return "", false, err
		}
		ok, err := taxonomySlugAvailable(ctx, tx, model, candidate, &excludeID)
		return candidate, ok, err
	})
}
```

**Why "always re-resolve on update, even when slug is unchanged" is deliberate, not an oversight**: an alternative design would thread a separate `bool touched` flag through every `Save` call so unchanged slugs skip the availability check entirely. This design instead always runs `resolveTaxonomyUpdateSlug` against whatever `t.Slug` holds at Save time (either the untouched original value, when the application layer chose not to overwrite it, or a new validated value). Since the availability check excludes the row's own ID, the untouched-value case always resolves to "available" and returns the same value — correct, harmless, and idempotent, at the cost of one extra `SELECT ... COUNT(*)` per update regardless of whether slug changed. Taxonomy writes are low-frequency admin operations (not a learner-facing hot path), so this trade-off buys real simplification: **zero** repository interface signature changes (see D4) instead of threading a new parameter through 4 repositories × 2 methods.

### D3 — Taxonomy domain changes

`internal/taxonomy/domain/errors.go` — full new content (adds one typed error, mirrors `internal/course/domain/errors.go:38-49` exactly):
```go
package domain

import "errors"

var (
	// ErrTaxonomyOptimisticLock is returned when expected_row_version does not match.
	ErrTaxonomyOptimisticLock = errors.New("taxonomy resource was modified by another request; refresh and retry")
	// ErrTaxonomyCanonicalConflict is returned when canonical text disagrees with translations.en.
	ErrTaxonomyCanonicalConflict = errors.New("canonical field conflicts with translations.en")
)

// SlugConflictError is returned when a manually supplied slug on create
// already belongs to another active row of the same taxonomy entity type.
// Mirrors internal/course/domain.SlugConflictError exactly: RecommendedSlug
// is confirmed available at construction time, but every write path
// re-checks at write time regardless (see internal/taxonomy/infra/slug.go).
type SlugConflictError struct {
	RecommendedSlug string
}

func (e *SlugConflictError) Error() string { return "taxonomy slug already exists" }
```
No new `ErrTaxonomyInvalidSlug`-style sentinel is added: taxonomy already has a generic, message-carrying, 400-mapped validation sentinel (`application.ErrTaxonomyValidation`, wrapped via `errors.Join`, mapped at `delivery/handler_helpers.go:103-106`) that already does exactly what a dedicated slug-invalid sentinel would do. Every "slug is blank/malformed" case in this design reuses `errors.Join(ErrTaxonomyValidation, errors.New("..."))`, the same pattern `validateOutcomePayload` already uses in this same file — introducing a second, functionally-identical sentinel would itself be the kind of duplication this project's rules forbid.

`internal/taxonomy/domain/taxonomy.go` — only the 6 input structs shown change (everything else in this file — `CourseTopic`, `CourseOutcome`, `CourseSkill`, `Tag`, `CourseLevel`, `TaxonomyFilter`, `CreateCourseOutcomeInput`, `UpdateCourseOutcomeInput` — is unchanged, not repeated here); new import `"mycourse-io-be/internal/shared/validate"` added:
```go
import (
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/validate" // NEW
)

// CreateCourseTopicInput carries data for creating a new course topic.
type CreateCourseTopicInput struct {
	ActorID      string
	Name         string
	Slug         string // NEW: "" means auto-generate from Name; a non-empty value is the caller's manual choice (already trimmed+format-validated by the application layer)
	Status       string
	ImageFileID  string
	ChildTopics  []taxpkg.TreeNode
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateCourseTopicInput carries partial-update data for a course topic.
type UpdateCourseTopicInput struct {
	Name               *string
	Slug               validate.Optional[string] // NEW: not Set means "do not change slug"; Set+non-empty is a new value already trimmed+format-validated by the application layer; Set+empty is rejected before this type is even populated (see D5) — no longer derived from Name
	Status             *string
	ImageFileID        *string
	ChildTopics        *[]taxpkg.TreeNode
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateCourseSkillInput carries data for creating a course skill.
type CreateCourseSkillInput struct {
	ActorID      string
	Name         string
	Slug         string // NEW — same semantics as CreateCourseTopicInput.Slug
	Status       string
	Children     []taxpkg.TreeNode
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateCourseSkillInput carries partial-update data for a course skill.
type UpdateCourseSkillInput struct {
	Name               *string
	Slug               validate.Optional[string] // NEW — same semantics as UpdateCourseTopicInput.Slug
	Status             *string
	Children           *[]taxpkg.TreeNode
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateTagInput carries data for creating a new tag.
type CreateTagInput struct {
	ActorID      string
	Name         string
	Slug         string // NEW — same semantics as CreateCourseTopicInput.Slug
	Status       string
	Translations map[string]taxpkg.NodeTranslation
}

// UpdateTagInput carries partial-update data for a tag.
type UpdateTagInput struct {
	Name               *string
	Slug               validate.Optional[string] // NEW — same semantics as UpdateCourseTopicInput.Slug
	Status             *string
	Translations       map[string]taxpkg.NodeTranslation
	ExpectedRowVersion int64
}

// CreateCourseLevelInput carries data for creating a new course level.
type CreateCourseLevelInput = CreateTagInput

// UpdateCourseLevelInput carries partial-update data for a course level.
type UpdateCourseLevelInput = UpdateTagInput
```
(`CreateCourseLevelInput`/`UpdateCourseLevelInput` remain type aliases — the new `Slug` fields apply to `CourseLevel` automatically, no separate edit.) `internal/taxonomy/domain/repository.go` is **unchanged** — see D4 for why.

### D4 — Zero repository interface changes (deliberate)

`internal/taxonomy/domain/repository.go:6-16`'s `taxonomyRepository[T]` generic interface — including `Create(ctx, t *T) error` and `Save(ctx, t *T, expectedRowVersion int64) error` — is used by all 5 taxonomy repositories, **including `CourseOutcomeRepository`, whose `CourseOutcome` domain struct has no `Slug` field at all**. Adding a `manualSlug string` (or similar) parameter to `Create`/`Save` in this shared interface would force `CourseOutcome`'s otherwise-untouched Create/Save to also accept a meaningless slug argument.

Instead, this design reuses the **existing** `Slug string` field already on `CourseTopic`/`CourseSkill`/`Tag`/`CourseLevel` as the single carrier for both directions:
- **Before** calling `Create`/`Save`: the application layer sets `entity.Slug` to either `""` (auto-generate) or an already-format-validated manual/new value (D5) — exactly the same field it already set today (previously always via `slugFromName(name)`; now via the new manual-or-omit logic).
- **Inside** `Create`/`Save` (infra layer, D6): the concrete implementation reads `entity.Slug` as the **candidate**, resolves it via `resolveTaxonomyCreateSlug`/`resolveTaxonomyUpdateSlug` (D2), and overwrites `entity.Slug` with the **final** resolved value before the actual INSERT/UPDATE.

This closes the loop with **zero** changes to `domain/repository.go`, `CourseTopicRepository`, `CourseSkillRepository`, `TagRepository`, or `CourseLevelRepository`'s method signatures — `CourseOutcomeRepository` (and `GormCourseOutcomeRepository`'s own, separate `Create`/`Save` implementations, which do not go through any of the shared helpers touched below) is completely unaffected, confirmed by inspection: `GormCourseOutcomeRepository.Create`/`Save` (`internal/taxonomy/infra/repos.go:256-296`) already have their own inline transaction bodies, never call `createWithNameTranslations` or `saveSlugStatusEntity` (D6's targets).

### D5 — Taxonomy application layer changes

New file `internal/taxonomy/application/slug.go`:
```go
package application

import (
	"errors"
	"strings"

	sharedslug "mycourse-io-be/internal/shared/slug"
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/validate"
)

// validateManualTaxonomySlug trims raw and format-validates it, mirroring
// internal/course/application/service.go's manual-slug handling inside
// CreateCourse. An empty/whitespace-only input (after trimming) returns ""
// with no error — the caller's repo.Create resolves "" as "auto-generate."
func validateManualTaxonomySlug(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	validated, ok := sharedslug.ValidateManualFormat(trimmed, taxpkg.DefaultMaxSlugLen)
	if !ok {
		return "", errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return validated, nil
}

// resolveTaxonomySlugInput interprets an update request's Optional[string]
// slug field, mirroring internal/course/delivery/handler_instructor.go's
// updateBasicInfo slug block (D6 of rework-course-slug-management),
// relocated to the application layer here because taxonomy's delivery layer
// is generic/shared across all 5 entity types (updateTaxonomyMutation) while
// course's is not. Returns (nil, nil) for "omitted — leave slug unchanged."
func resolveTaxonomySlugInput(in validate.Optional[string]) (*string, error) {
	if !in.Set {
		return nil, nil
	}
	trimmed := strings.TrimSpace(in.Value)
	if trimmed == "" {
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug must not be null, empty, or whitespace-only"))
	}
	validated, ok := sharedslug.ValidateManualFormat(trimmed, taxpkg.DefaultMaxSlugLen)
	if !ok {
		return nil, errors.Join(ErrTaxonomyValidation, errors.New("slug format is invalid"))
	}
	return &validated, nil
}
```

`internal/taxonomy/application/service.go` — every changed function shown in full; every function not listed (`ListTopics`, `GetTopic`, `DeleteTopic`, all of CourseOutcome's functions, `DeleteCourseSkill`, `DeleteTag`, `DeleteCourseLevel`, etc.) is unchanged, not repeated:

```go
func (s *TaxonomyService) CreateTopic(ctx context.Context, in domain.CreateCourseTopicInput) (*domain.CourseTopic, error) {
	name, tr, err := syncNameCanonicalAndTranslations(in.Name, in.Translations)
	if err != nil {
		return nil, err
	}
	manualSlug, err := validateManualTaxonomySlug(in.Slug) // NEW
	if err != nil {
		return nil, err
	}
	childTopics, err := resolveTreeForCreate(in.ChildTopics) // NEW — replaces prepareTreeWrite+validateChildTopics, see below
	if err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(in.ImageFileID)
	if fileID != "" && s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, fileID); err != nil {
			return nil, err
		}
	}
	_, st := trimmedNameAndStatus(name, in.Status) // MODIFIED: trimmedTaxonomyFields renamed, no longer returns a slug — see below
	t := &domain.CourseTopic{
		Name: name, Slug: manualSlug, Status: st, ChildTopics: childTopics, // MODIFIED: Slug: manualSlug (was slugFromName(name))
		Translations: tr, RowVersion: 1,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if fileID != "" {
		t.ImageFileID = &fileID
	}
	if err := s.topicRepo.Create(ctx, t); err != nil {
		return nil, err // may be *domain.SlugConflictError — propagated as-is, mapped at delivery (D6)
	}
	return s.topicRepo.GetDetail(ctx, t.ID, "", true)
}

func (s *TaxonomyService) UpdateTopic(ctx context.Context, id string, in domain.UpdateCourseTopicInput) (*domain.CourseTopic, error) {
	row, err := s.topicRepo.GetDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	if err := applyCourseTopicUpdate(row, in); err != nil {
		return nil, err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateImageFileID(ctx, &row.ImageFileID, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.topicRepo.Save(ctx, row, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	enqueueReplacedImageCleanup(ctx, s.orphanEnqueuer, in.ImageFileID, prevFileID, imageFileIDStr(row.ImageFileID))
	return s.topicRepo.GetDetail(ctx, id, "", true)
}

func applyCourseTopicUpdate(row *domain.CourseTopic, in domain.UpdateCourseTopicInput) error {
	canonical := row.Name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := row.Translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return err
		}
		merged = replaceNameTranslations(patch)
	}
	name, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return err
	}
	slugInput, err := resolveTaxonomySlugInput(in.Slug) // NEW
	if err != nil {
		return err
	}
	row.Name = name
	// MODIFIED: row.Slug = slugFromName(name) REMOVED — slug is now
	// independent of name; only touched when slugInput is non-nil.
	if slugInput != nil {
		row.Slug = *slugInput
	}
	row.Translations = tr
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	return applyCourseTopicChildren(row, in.ChildTopics)
}

func applyCourseTopicChildren(row *domain.CourseTopic, childTopics *[]taxpkg.TreeNode) error {
	oldByID := taxpkg.BuildNodeIndexByID(row.ChildTopics) // NEW
	source := row.ChildTopics
	if childTopics != nil {
		source = *childTopics
	}
	resolved, err := resolveTreeForUpdate(source, oldByID) // NEW — replaces prepareTreeWrite+validateChildTopics
	if err != nil {
		return err
	}
	row.ChildTopics = resolved
	return nil
}

func (s *TaxonomyService) CreateCourseSkill(ctx context.Context, in domain.CreateCourseSkillInput) (*domain.CourseSkill, error) {
	name, tr, err := syncNameCanonicalAndTranslations(in.Name, in.Translations)
	if err != nil {
		return nil, err
	}
	manualSlug, err := validateManualTaxonomySlug(in.Slug) // NEW
	if err != nil {
		return nil, err
	}
	children, err := resolveTreeForCreate(in.Children) // NEW
	if err != nil {
		return nil, err
	}
	_, st := trimmedNameAndStatus(name, in.Status)
	sk := &domain.CourseSkill{
		Name: name, Slug: manualSlug, Status: st, Children: children, // MODIFIED: Slug: manualSlug
		Translations: tr, RowVersion: 1,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if err := s.skillRepo.Create(ctx, sk); err != nil {
		return nil, err
	}
	return s.skillRepo.GetDetail(ctx, sk.ID, "", true)
}

func (s *TaxonomyService) UpdateCourseSkill(ctx context.Context, id string, in domain.UpdateCourseSkillInput) (*domain.CourseSkill, error) {
	row, err := s.skillRepo.GetDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	canonical := row.Name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := row.Translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return nil, err
		}
		merged = replaceNameTranslations(patch)
	}
	name, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return nil, err
	}
	slugInput, err := resolveTaxonomySlugInput(in.Slug) // NEW
	if err != nil {
		return nil, err
	}
	row.Name = name
	if slugInput != nil { // MODIFIED — was unconditional row.Slug = slugFromName(name)
		row.Slug = *slugInput
	}
	row.Translations = tr
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	oldByID := taxpkg.BuildNodeIndexByID(row.Children) // NEW
	source := row.Children
	if in.Children != nil {
		source = *in.Children
	}
	resolved, err := resolveTreeForUpdate(source, oldByID) // NEW — replaces both branches' prepareTreeWrite+validateChildren
	if err != nil {
		return nil, err
	}
	row.Children = resolved
	if err := s.skillRepo.Save(ctx, row, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	return s.skillRepo.GetDetail(ctx, id, "", true)
}
```

`--- internal helpers ---` section changes (same file): `validateChildTopics`/`validateChildren` are **removed** (their job — normalize then validate — is now split between `resolveTreeForCreate`/`resolveTreeForUpdate`, D7, and a plain validate call, since normalization now happens once inside those functions instead of being redundantly re-run — see the found-cleanup note below):
```go
// NEW — replaces validateChildTopics/validateChildren (removed): both were
// identical wrappers that re-ran taxpkg.NormalizeTreeSlugs internally on
// already-normalized input from prepareTreeWrite before validating — a
// harmless but wasted double-normalize, confirmed by reading both call
// sites (the reassigned `nodes` from that internal re-normalize was never
// actually used; callers used prepareTreeWrite's own output instead). Now
// that resolveTreeForCreate/resolveTreeForUpdate produce final,
// already-resolved slugs directly, no second normalize pass is needed.
func validateResolvedTree(nodes []taxpkg.TreeNode) error {
	if nodes == nil {
		return nil
	}
	if err := taxpkg.ValidateTree(nodes, taxpkg.ValidateTreeOpts{}); err != nil {
		return errors.Join(ErrTaxonomyValidation, err)
	}
	return nil
}

// resolveTreeForCreate resolves slugs for a brand-new tree (create path —
// no prior tree exists, so every node is "new" for preserve-by-id purposes).
func resolveTreeForCreate(nodes []taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	return resolveTreeForWrite(nodes, nil)
}

// resolveTreeForUpdate resolves slugs for an update, given the entity's
// previously-stored tree (oldByID) for preserve-by-id lookups.
func resolveTreeForUpdate(nodes []taxpkg.TreeNode, oldByID map[string]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	return resolveTreeForWrite(nodes, oldByID)
}

func resolveTreeForWrite(nodes []taxpkg.TreeNode, oldByID map[string]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	canonized, err := canonicalizeTreeLocales(nodes)
	if err != nil {
		return nil, err
	}
	resolved, conflict, err := taxpkg.PrepareTreeForWriteWithSlugPolicy(canonized, oldByID, "node")
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, domain.ErrTaxonomyCanonicalConflict
	}
	if err := validateResolvedTree(resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}
```

`trimmedTaxonomyFields` (removed) → `trimmedNameAndStatus` (new, simplified — drops the slug computation entirely, since no caller needs a synchronously-computed slug anymore):
```go
// MODIFIED — was trimmedTaxonomyFields(name, status string) (string, string, string)
// returning (n, sl, st); the middle "sl" return is deleted since slug
// resolution now requires DB access and lives in the infra layer (D2/D6).
func trimmedNameAndStatus(name, status string) (string, string) {
	n := strings.TrimSpace(name)
	st := strings.ToUpper(strings.TrimSpace(status))
	if st == "" {
		st = "ACTIVE"
	}
	return n, st
}
```
`slugFromName` (removed — its one remaining caller after this change is `internal/taxonomy/infra/slug.go`'s `generateAutoTaxonomySlugBase`, which calls `sharedutils.SlugifyName` directly instead of going through this application-layer wrapper, since infra cannot import `application`).

`internal/taxonomy/application/service_helpers.go` — changed functions shown in full (`deleteWithOrphanImage`, `imageIDLoader*` unchanged, not repeated):
```go
func createSlugStatusEntity[T any](
	ctx context.Context,
	name, status, manualSlug string, // MODIFIED: manualSlug param added
	actorID string,
	translations map[string]taxpkg.NodeTranslation,
	repo slugStatusEntityRepo[T],
) (*T, error) {
	n, tr, err := syncNameCanonicalAndTranslations(name, translations)
	if err != nil {
		return nil, err
	}
	_, st := trimmedNameAndStatus(n, status) // MODIFIED: trimmedTaxonomyFields -> trimmedNameAndStatus, no slug return
	entity := repo.build(n, manualSlug, st, stringPtrIfNotBlank(actorID), tr) // MODIFIED: manualSlug passed as the "sl" build param — repo.Create (D6) resolves it into the final slug
	if err := repo.create(ctx, entity); err != nil {
		return nil, err
	}
	return repo.getDetail(ctx, repo.idOf(entity), "", true)
}

// newTagFromFields/newCourseLevelFromFields/tagID/courseLevelID: UNCHANGED —
// build's signature (n, sl, st string, createdBy *string, tr map[...]) *T
// already accepts a slug positionally; only the VALUE flowing into it
// changes (manualSlug instead of a pre-computed final slug).

func createSlugStatusFromInput[T any](ctx context.Context, in domain.CreateTagInput, c slugStatusCreator[T]) (*T, error) {
	manualSlug, err := validateManualTaxonomySlug(in.Slug) // NEW
	if err != nil {
		return nil, err
	}
	return createSlugStatusEntity(ctx, in.Name, in.Status, manualSlug, in.ActorID, in.Translations, c)
}

func applySlugStatusFields(
	name, slug, status *string,
	translations *map[string]taxpkg.NodeTranslation,
	in domain.UpdateTagInput,
) error {
	canonical := *name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := *translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return err
		}
		merged = replaceNameTranslations(patch)
	}
	syncedName, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return err
	}
	slugInput, err := resolveTaxonomySlugInput(in.Slug) // NEW
	if err != nil {
		return err
	}
	nextStatus := *status
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			nextStatus = v
		}
	}
	*name = syncedName
	if slugInput != nil { // MODIFIED — was unconditional: *slug = slugFromName(syncedName)
		*slug = *slugInput
	}
	*status = nextStatus
	*translations = tr
	return nil
}

// applyTagUpdate/applyCourseLevelUpdate: UNCHANGED (still call
// applySlugStatusFields(&e.Name, &e.Slug, &e.Status, &e.Translations, in)).
```

`internal/taxonomy/application/translation_sync.go`: `prepareTreeWrite` (old lines 234-247) is **removed** — every call site now goes through D7's `taxpkg.PrepareTreeForWriteWithSlugPolicy` via `resolveTreeForWrite` above instead. `canonicalizeTreeLocales` (lines 249-272) is **unchanged** and still called directly by `resolveTreeForWrite`. Every other function in this file (`canonicalizeNameTranslations`, `syncNameCanonicalAndTranslations`, outcome-translation helpers) is unchanged, not repeated.

### D6 — Taxonomy delivery layer changes

`internal/taxonomy/delivery/dto.go` — new `TreeNodeInput` type, and every Create*/Update* request DTO gains a `slug` field (only the changed structs are shown; `TaxonomyBaseFilter`, `TaxonomyGetQuery`, response DTOs (D6b below), `OutcomeTranslationDTO`, `CreateCourseOutcomeRequest`/`UpdateCourseOutcomeRequest` are unchanged, not repeated):
```go
import (
	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/validate" // NEW
)

// TreeNodeInput is the write-path counterpart of taxpkg.TreeNode: Slug uses
// validate.Optional[string] instead of a plain string so the delivery layer
// can distinguish "omitted for this node" (Set=false — preserve by id on
// update, or auto-generate for a new node) from "explicit null/empty for
// this node" (Set=true, Value=="" — rejected, see toDomainTreeNodes below)
// from "explicit manual value" (Set=true, Value!="") — something a plain
// string on a JSON array element cannot express. Used ONLY in Create/Update
// request bodies; taxpkg.TreeNode (unchanged) remains the read/response
// type — see docs/reusable-assets.md's updated asset entry for why these
// are deliberately two different types instead of one Optional-carrying
// TreeNode used everywhere (a MarshalJSON complication for the response
// shape, found and avoided during design).
type TreeNodeInput struct {
	ID           string                            `json:"id"`
	Name         string                            `json:"name"`
	Slug         validate.Optional[string]         `json:"slug"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
	Children     []TreeNodeInput                   `json:"children"`
}

// CreateCourseTopicRequest is the JSON body for creating a course topic.
type CreateCourseTopicRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	Slug         string                            `json:"slug" validate:"omitempty,max=255"` // NEW
	ImageFileID  string                            `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics  []TreeNodeInput                   `json:"child_topics"` // MODIFIED: []taxpkg.TreeNode -> []TreeNodeInput
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateCourseTopicRequest is the JSON body for updating a course topic.
type UpdateCourseTopicRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	Slug               validate.Optional[string]         `json:"slug"` // NEW — no validate tag, same omitempty-on-pointer/Optional gotcha reasoning as course's own D6
	ImageFileID        *string                           `json:"image_file_id" validate:"omitempty,uuid"`
	ChildTopics        *[]TreeNodeInput                  `json:"child_topics"` // MODIFIED: *[]taxpkg.TreeNode -> *[]TreeNodeInput
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// CreateCourseSkillRequest is the JSON body for creating a course skill.
type CreateCourseSkillRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	Slug         string                            `json:"slug" validate:"omitempty,max=255"` // NEW
	Children     []TreeNodeInput                   `json:"children"` // MODIFIED
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateCourseSkillRequest is the JSON body for updating a course skill.
type UpdateCourseSkillRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	Slug               validate.Optional[string]         `json:"slug"` // NEW
	Children           *[]TreeNodeInput                  `json:"children"` // MODIFIED
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// CreateTagRequest is the JSON body for creating a tag.
type CreateTagRequest struct {
	Name         string                            `json:"name" validate:"omitempty,min=1,max=255"`
	Slug         string                            `json:"slug" validate:"omitempty,max=255"` // NEW
	Status       string                            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations map[string]taxpkg.NodeTranslation `json:"translations"`
}

// UpdateTagRequest is the JSON body for updating a tag.
type UpdateTagRequest struct {
	Name               *string                           `json:"name" validate:"omitempty,min=1,max=255"`
	Slug               validate.Optional[string]         `json:"slug"` // NEW
	Status             *string                           `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
	Translations       map[string]taxpkg.NodeTranslation `json:"translations"`
	ExpectedRowVersion int64                             `json:"expected_row_version" validate:"required,min=1"`
}

// CreateCourseLevelRequest/UpdateCourseLevelRequest remain type aliases of
// CreateTagRequest/UpdateTagRequest — no separate edit needed.
```

`internal/taxonomy/delivery/handler_helpers.go` — changed functions (everything else — `listTaxonomyItems*`, `getTaxonomyByID`, `deleteTaxonomyByID`, `slugStatusResponse*`, `editRowVersion`, `mapRowsToResponses`, outcome-translation mapping — unchanged, not repeated):
```go
// NEW: toDomainTreeNodes converts the write-path DTO tree into the shared
// read/write taxpkg.TreeNode shape, resolving each node's Optional[string]
// Slug into a plain string per this design's carrier convention (D4): ""
// means "no manual slug for this node" (auto-generate or preserve-by-id,
// resolved later by taxpkg.ResolveTreeSlugs), a non-empty value is an
// already-trimmed+format-validated manual value. An explicit null/empty
// node slug is rejected immediately, identifying the offending node by id
// (or "(new node)" when id is blank), matching the root-level explicit
// empty-slug-on-update rejection.
func toDomainTreeNodes(in []TreeNodeInput) ([]taxpkg.TreeNode, error) {
	out := make([]taxpkg.TreeNode, len(in))
	for i, n := range in {
		slug := ""
		if n.Slug.Set {
			trimmed := strings.TrimSpace(n.Slug.Value)
			if trimmed == "" {
				return nil, errors.Join(application.ErrTaxonomyValidation,
					fmt.Errorf("tree node %q has an explicit empty slug", nodeLabel(n.ID)))
			}
			validated, ok := sharedslug.ValidateManualFormat(trimmed, taxpkg.DefaultMaxSlugLen)
			if !ok {
				return nil, errors.Join(application.ErrTaxonomyValidation,
					fmt.Errorf("tree node %q has an invalid slug format", nodeLabel(n.ID)))
			}
			slug = validated
		}
		children, err := toDomainTreeNodes(n.Children)
		if err != nil {
			return nil, err
		}
		out[i] = taxpkg.TreeNode{
			ID: n.ID, Name: n.Name, Slug: slug,
			Translations: n.Translations, Children: children,
		}
	}
	return out, nil
}

func nodeLabel(id string) string {
	if strings.TrimSpace(id) == "" {
		return "(new node)"
	}
	return id
}

// mapTaxonomyMutationError — NEW branch inserted first (typed-error check,
// mirrors internal/course/delivery/handler_base.go's mapCourseError exactly):
func mapTaxonomyMutationError(c *gin.Context, err error, withProfileMedia bool) error {
	if err == nil {
		return nil
	}
	var slugConflict *domain.SlugConflictError // NEW
	if stderrors.As(err, &slugConflict) {
		response.Fail(c, http.StatusConflict, apperrors.SlugConflict,
			apperrors.DefaultMessage(apperrors.SlugConflict),
			gin.H{"recommended_slug": slugConflict.RecommendedSlug})
		return err
	}
	if stderrors.Is(err, apperrors.ErrNotFound) {
		response.Fail(c, http.StatusNotFound, apperrors.NotFound, "not found", nil)
		return err
	}
	if stderrors.Is(err, domain.ErrTaxonomyOptimisticLock) {
		response.Fail(c, http.StatusConflict, apperrors.Conflict, err.Error(), nil)
		return err
	}
	if stderrors.Is(err, domain.ErrTaxonomyCanonicalConflict) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	if withProfileMedia && stderrors.Is(err, apperrors.ErrInvalidProfileMediaFile) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	if stderrors.Is(err, application.ErrTaxonomyValidation) {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return err
	}
	response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, err.Error(), nil)
	return err
}

// courseSkillInputFromCreate/courseSkillInputFromUpdate — MODIFIED to pass
// Slug through and convert Children via toDomainTreeNodes:
func courseSkillInputFromCreate(req CreateCourseSkillRequest, actorID string) (domain.CreateCourseSkillInput, error) {
	children, err := toDomainTreeNodes(req.Children)
	if err != nil {
		return domain.CreateCourseSkillInput{}, err
	}
	return domain.CreateCourseSkillInput{
		ActorID: actorID, Name: req.Name, Slug: req.Slug, Status: req.Status, Children: children,
		Translations: req.Translations,
	}, nil
}

func courseSkillInputFromUpdate(req UpdateCourseSkillRequest) (domain.UpdateCourseSkillInput, error) {
	var children *[]taxpkg.TreeNode
	if req.Children != nil {
		converted, err := toDomainTreeNodes(*req.Children)
		if err != nil {
			return domain.UpdateCourseSkillInput{}, err
		}
		children = &converted
	}
	return domain.UpdateCourseSkillInput{
		Name: req.Name, Slug: req.Slug, Status: req.Status, Children: children,
		Translations: req.Translations, ExpectedRowVersion: req.ExpectedRowVersion,
	}, nil
}

// slugStatusInputFromCreate/slugStatusInputFromUpdate — MODIFIED to pass Slug through:
func slugStatusInputFromCreate(req CreateTagRequest, actorID string) domain.CreateTagInput {
	return domain.CreateTagInput{ActorID: actorID, Name: req.Name, Slug: req.Slug, Status: req.Status, Translations: req.Translations}
}

func slugStatusInputFromUpdate(req UpdateTagRequest) domain.UpdateTagInput {
	return domain.UpdateTagInput{
		Name: req.Name, Slug: req.Slug, Status: req.Status, Translations: req.Translations,
		ExpectedRowVersion: req.ExpectedRowVersion,
	}
}
```
(`createTaxonomyMutation`/`updateTaxonomyMutation` generic wrappers are unchanged in shape, but `courseSkillInputFromCreate`/`courseSkillInputFromUpdate` now return `(In, error)` instead of `In` — their generic call sites in `handler.go`'s `createCourseSkill`/`updateCourseSkill` need the one-line adjustment shown in D6b below; `createTaxonomyMutation[Req,In,Row,Resp]`'s `toInput func(Req, string) In` parameter type itself would need widening to `func(Req, string) (In, error)` — shown fully in D6b since it changes the generic helper signature used by every entity type, not just skill.)

**Widening `createTaxonomyMutation`/`updateTaxonomyMutation` to propagate a mapping error** (needed because `courseSkillInputFromCreate`/`courseSkillInputFromUpdate` can now fail on an invalid tree node slug, and `createTopic`/`updateTopic` in `handler.go` need the same for `child_topics`) — full new content of both generic functions:
```go
func updateTaxonomyMutation[Req any, In any, Row any, Resp any](
	c *gin.Context,
	updateFn func(context.Context, string, In) (*Row, error),
	toInput func(Req) (In, error), // MODIFIED: returns error
	toResponse func(Row) Resp,
	withProfileMedia bool,
) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in, err := toInput(req) // NEW
	if err != nil {
		mapTaxonomyMutationError(c, err, withProfileMedia)
		return
	}
	row, err := updateFn(c.Request.Context(), id, in)
	if err := mapTaxonomyMutationError(c, err, withProfileMedia); err != nil {
		return
	}
	response.OK(c, "updated", toResponse(*row))
}

func createTaxonomyMutation[Req any, In any, Row any, Resp any](
	c *gin.Context,
	createFn func(context.Context, In) (*Row, error),
	toInput func(Req, string) (In, error), // MODIFIED: returns error
	toResponse func(Row) Resp,
	withProfileMedia bool,
) {
	var req Req
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	in, err := toInput(req, utils.CurrentUserID(c)) // NEW
	if err != nil {
		mapTaxonomyMutationError(c, err, withProfileMedia)
		return
	}
	row, err := createFn(c.Request.Context(), in)
	if err := mapTaxonomyMutationError(c, err, withProfileMedia); err != nil {
		return
	}
	response.Created(c, "created", toResponse(*row))
}
```
Every existing call site that passes a mapper NOT returning an error (`courseOutcomeInputFromCreate`, `courseOutcomeInputFromUpdate`, `slugStatusInputFromCreate`, `slugStatusInputFromUpdate` — none of these can fail) needs a **trivial 1-line wrapper** added to satisfy the new `(In, error)` shape, e.g.:
```go
func courseOutcomeInputFromCreateOK(req CreateCourseOutcomeRequest, actorID string) (domain.CreateCourseOutcomeInput, error) {
	return courseOutcomeInputFromCreate(req, actorID), nil
}
```
(and the 3 analogous `...OK` wrappers for `courseOutcomeInputFromUpdate`, `slugStatusInputFromCreate`, `slugStatusInputFromUpdate` — used only at their `handler.go` call sites, D6b, so the original never-fails mapper functions themselves stay exactly as they are today and are not otherwise touched).

`internal/taxonomy/delivery/handler.go` — only `createTopic`, `updateTopic`, `createCourseSkill`, `updateCourseSkill`, `createTag`, `updateTag`, `createCourseLevel`, `updateCourseLevel` call sites change (to route through the now-fallible mappers); every other handler (`listTopics`, `getTopic`, `deleteTopic`, all CourseOutcome handlers, `deleteCourseSkill`, `hardDelete*`, etc.) is unchanged, not repeated:
```go
func (h *Handler) createTopic(c *gin.Context) {
	var req CreateCourseTopicRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	childTopics, err := toDomainTreeNodes(req.ChildTopics) // NEW
	if err != nil {
		mapTaxonomyMutationError(c, err, true)
		return
	}
	in := domain.CreateCourseTopicInput{
		ActorID: utils.CurrentUserID(c), Name: req.Name, Slug: req.Slug, // MODIFIED: Slug added
		Status: req.Status, ImageFileID: req.ImageFileID, ChildTopics: childTopics, // MODIFIED: childTopics (converted)
		Translations: req.Translations,
	}
	row, err := h.svc.CreateTopic(c.Request.Context(), in)
	if err := mapTaxonomyMutationError(c, err, true); err != nil {
		return
	}
	response.Created(c, "created", toCourseTopicResponse(*row))
}

func (h *Handler) updateTopic(c *gin.Context) {
	id, ok := utils.ParseUUIDParam(c, "id")
	if !ok {
		response.Fail(c, http.StatusBadRequest, apperrors.BadRequest, "invalid id", nil)
		return
	}
	var req UpdateCourseTopicRequest
	if err := validate.BindJSON(c, &req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	var childTopics *[]taxpkg.TreeNode // NEW
	if req.ChildTopics != nil {
		converted, err := toDomainTreeNodes(*req.ChildTopics)
		if err != nil {
			mapTaxonomyMutationError(c, err, true)
			return
		}
		childTopics = &converted
	}
	in := domain.UpdateCourseTopicInput{
		Name: req.Name, Slug: req.Slug, Status: req.Status, // MODIFIED: Slug added
		ImageFileID: req.ImageFileID, ChildTopics: childTopics, // MODIFIED
		Translations: req.Translations, ExpectedRowVersion: req.ExpectedRowVersion,
	}
	row, err := h.svc.UpdateTopic(c.Request.Context(), id, in)
	if err := mapTaxonomyMutationError(c, err, true); err != nil {
		return
	}
	response.OK(c, "updated", toCourseTopicResponse(*row))
}

func (h *Handler) createCourseSkill(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateCourseSkill, courseSkillInputFromCreate, toCourseSkillResponse, false)
}

func (h *Handler) updateCourseSkill(c *gin.Context) {
	updateTaxonomyMutation(c, h.svc.UpdateCourseSkill, courseSkillInputFromUpdate, toCourseSkillResponse, false)
}

func (h *Handler) createTag(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateTag, slugStatusInputFromCreateOK, slugStatusResponseFromTag, false)
}

func (h *Handler) updateTag(c *gin.Context) {
	updateSlugStatusTaxonomy[UpdateTagRequest, domain.Tag](c, h.svc.UpdateTag, slugStatusInputFromUpdateOK, slugStatusResponseFromTag)
}

func (h *Handler) createCourseLevel(c *gin.Context) {
	createTaxonomyMutation(c, h.svc.CreateCourseLevel, slugStatusInputFromCreateOK, slugStatusResponseFromCourseLevel, false)
}

func (h *Handler) updateCourseLevel(c *gin.Context) {
	updateSlugStatusTaxonomy[UpdateCourseLevelRequest, domain.CourseLevel](c, h.svc.UpdateCourseLevel, slugStatusInputFromUpdateOK, slugStatusResponseFromCourseLevel)
}
```
(`createCourseOutcome`/`updateCourseOutcome` similarly switch to the `...OK` wrapper names; `updateSlugStatusTaxonomy`'s own `toInput func(Req) domain.UpdateTagInput` parameter type in `handler_helpers.go:111-118` widens to `func(Req) (domain.UpdateTagInput, error)` to match `updateTaxonomyMutation`'s new signature — a one-line change to that wrapper's own signature, not repeated here in full since only the type annotation changes.)

Response DTOs (`CourseTopicResponse.ChildTopics []taxpkg.TreeNode`, `CourseSkillResponse.Children []taxpkg.TreeNode`, `SlugStatusResponse.Slug string`, `TagResponse`/`CourseLevelResponse` aliases) are **unchanged** — they already return `Slug string`/`slug`-bearing `TreeNode`, satisfying "every slug-affecting response returns the final stored slug" for free, exactly as course's own D6 noted for itself.

### D7 — Shared tree-write slug resolution

**SUPERSEDED in part by D11**: `PrepareTreeForWrite` was removed entirely during the code-review round (confirmed zero remaining callers), not kept "unchanged" as this section originally described. Shown here for historical context only — see D11 for the final file content.

`internal/shared/taxonomy/tree_resolve.go` — one new function appended (everything else in this file — `ResolveNodeName`, `ResolveTreeNames`, `EnsureEnTranslation`, `SyncCanonicalAndEn`, `PrepareTreeForWrite`, `syncTreeCanonicalAndEn`, `cloneNodeTranslations` — is unchanged, not repeated):
```go
// PrepareTreeForWriteWithSlugPolicy mirrors PrepareTreeForWrite's
// locale-sync steps (SyncCanonicalAndEn + EnsureEnTranslation) but resolves
// slugs via ResolveTreeSlugs (tree_write.go — preserve-by-id / manual /
// auto-generate / in-tree collision auto-resolve) instead of
// NormalizeTreeSlugs's unconditional recompute-from-name. PrepareTreeForWrite
// itself is unchanged and still available for any caller wanting the old
// always-recompute behavior (none exist in this codebase after this change,
// but the function is not removed — see docs/reusable-assets.md).
func PrepareTreeForWriteWithSlugPolicy(nodes []TreeNode, oldByID map[string]TreeNode, fallbackPrefix string) (out []TreeNode, conflict bool, err error) {
	if nodes == nil {
		return nil, false, nil
	}
	synced, conflict := syncTreeCanonicalAndEn(nodes)
	if conflict {
		return nil, true, nil
	}
	resolved, err := ResolveTreeSlugs(EnsureEnTranslation(synced), oldByID, fallbackPrefix)
	if err != nil {
		return nil, false, err
	}
	return resolved, false, nil
}
```

New file `internal/shared/taxonomy/tree_write.go`:
```go
package taxonomy

import (
	"fmt"
	"strings"

	sharedslug "mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/utils"
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
//     via utils.SlugifyName, with an ASCII-fallback to a random-suffixed
//     literal prefix. This mirrors internal/taxonomy/infra/slug.go's
//     generateAutoTaxonomySlugBase logic as a small, separate, dependency-free
//     copy — internal/shared/taxonomy must not import internal/taxonomy/infra
//     (would invert this repo's module dependency direction), so this one
//     small piece of derivation logic exists in two places rather than
//     being literally shared; both copies wrap the same two underlying
//     primitives (utils.SlugifyName, internal/shared/slug), so there is no
//     duplicated *algorithm*, only a duplicated 4-line call sequence.
//
// Collisions within the same tree (any two nodes at any depth sharing a
// slug) are auto-resolved: whichever node is walked second gets a random
// suffix appended, escalating length on repeated collision, using
// sharedslug.RetryWithSuffix — the same crypto/rand-based mechanism as
// every other slug feature in this codebase. Walk order is depth-first,
// matching tree_validate.go's walkTree, so behavior is deterministic for a
// given input order.
func ResolveTreeSlugs(nodes []TreeNode, oldByID map[string]TreeNode, fallbackPrefix string) ([]TreeNode, error) {
	seenSlug := map[string]struct{}{}
	return resolveTreeSlugsWalk(nodes, oldByID, fallbackPrefix, seenSlug)
}

func resolveTreeSlugsWalk(nodes []TreeNode, oldByID map[string]TreeNode, fallbackPrefix string, seenSlug map[string]struct{}) ([]TreeNode, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}
	out := make([]TreeNode, len(nodes))
	for i, n := range nodes {
		slug, err := resolveOneNodeSlug(n, oldByID, fallbackPrefix, seenSlug)
		if err != nil {
			return nil, err
		}
		children, err := resolveTreeSlugsWalk(n.Children, oldByID, fallbackPrefix, seenSlug)
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

func resolveOneNodeSlug(n TreeNode, oldByID map[string]TreeNode, fallbackPrefix string, seenSlug map[string]struct{}) (string, error) {
	candidate := strings.TrimSpace(n.Slug)
	if candidate == "" {
		if old, ok := oldByID[strings.TrimSpace(n.ID)]; ok && strings.TrimSpace(old.Slug) != "" {
			candidate = strings.TrimSpace(old.Slug)
		} else {
			base, mustSuffix := generateAutoNodeSlugBase(n.Name, fallbackPrefix)
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

func generateAutoNodeSlugBase(name, fallbackPrefix string) (base string, mustSuffix bool) {
	candidate := sharedslug.TruncateToLen(utils.SlugifyName(strings.TrimSpace(name)), DefaultMaxSlugLen)
	if trimmed, ok := sharedslug.ValidateManualFormat(candidate, DefaultMaxSlugLen); ok {
		return trimmed, false
	}
	return fallbackPrefix, true
}
```
`DefaultMaxSlugLen` is the existing constant already declared in `internal/shared/taxonomy/tree_validate.go:16` (value `255`) — reused as-is, not redeclared.

### D8 — Taxonomy infra: repos.go / repos_mappers.go / repos_crud_helper.go

`internal/taxonomy/infra/repos_mappers.go`'s `createNameTranslationArgs`/`createWithNameTranslations` (lines 183-210) — full new content, gains 5 new fields and an outer create-retry loop (mirrors `internal/course/infra/repo_instructor.go`'s `CreateCourse`/`createCourseOnce` split exactly — each retry attempt is a **fresh** transaction, avoiding D1's aborted-transaction bug entirely by construction):
```go
type createNameTranslationArgs[Row any, Domain any] struct {
	Entity       *Domain
	RowVersion   *int64
	ToRow        func(*Domain) *Row
	EntityID     *string
	CreatedAt    *int64
	UpdatedAt    *int64
	Meta         func(*Row) (string, int64, int64)
	Table        string
	FKCol        string
	Translations map[string]taxpkg.NodeTranslation
	ManualSlug   string                 // NEW: "" = auto-generate from Name
	Name         string                 // NEW: base for auto-generate
	SlugPrefix   string                 // NEW: fallback prefix ("topic"/"skill"/"tag"/"level")
	SetSlug      func(*Domain, string)  // NEW: writes the resolved slug onto Entity before ToRow runs
	NewRow       func() *Row            // NEW: a fresh zero-value Row, used only as taxonomySlugAvailable's Model argument (GORM needs a typed pointer to infer the table)
}

func createWithNameTranslations[Row any, Domain any](
	ctx context.Context,
	db *gorm.DB,
	args createNameTranslationArgs[Row, Domain],
) error {
	var lastErr error
	for i := 0; i < taxonomySlugRetryAttempts; i++ {
		lastErr = createWithNameTranslationsOnce(ctx, db, args)
		if lastErr == nil || !isTaxonomySlugDuplicateKey(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func createWithNameTranslationsOnce[Row any, Domain any](
	ctx context.Context,
	db *gorm.DB,
	args createNameTranslationArgs[Row, Domain],
) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if *args.RowVersion == 0 {
			*args.RowVersion = 1
		}
		slug, conflict, err := resolveTaxonomyCreateSlug(ctx, tx, args.NewRow(), args.ManualSlug, args.Name, args.SlugPrefix)
		if err != nil {
			return err
		}
		if conflict != nil {
			return conflict // *domain.SlugConflictError — propagated as-is, mapped at delivery (D6)
		}
		args.SetSlug(args.Entity, slug)
		if err := createTaxonomyDomain(ctx, tx, args.Entity, args.ToRow, args.EntityID, args.CreatedAt, args.UpdatedAt, args.Meta); err != nil {
			return err
		}
		return upsertNameTranslations(ctx, tx, args.Table, args.FKCol, *args.EntityID, args.Translations)
	})
}
```

`internal/taxonomy/infra/repos.go` — every call site of `createNameTranslationArgs`/`Save` that changes, shown in full (everything else in this file — `List`, `GetByID`, `SoftDelete`, `HardDelete` for every repository, all of `GormCourseOutcomeRepository`, row types, filter helpers — is unchanged, not repeated):
```go
func (r *GormCourseTopicRepository) Create(ctx context.Context, t *domain.CourseTopic) error {
	return createWithNameTranslations(ctx, r.db, createNameTranslationArgs[courseTopicRow, domain.CourseTopic]{
		Entity: t, RowVersion: &t.RowVersion, ToRow: courseTopicToRow,
		EntityID: &t.ID, CreatedAt: &t.CreatedAt, UpdatedAt: &t.UpdatedAt, Meta: metaCourseTopicRow,
		Table: constants.TableTaxonomyCourseTopicTranslations, FKCol: "topic_id", Translations: t.Translations,
		ManualSlug: t.Slug, Name: t.Name, SlugPrefix: "topic", // NEW
		SetSlug: func(e *domain.CourseTopic, s string) { e.Slug = s },   // NEW
		NewRow:  func() *courseTopicRow { return &courseTopicRow{} },    // NEW
	})
}

func (r *GormCourseTopicRepository) Save(ctx context.Context, t *domain.CourseTopic, expectedRowVersion int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child := t.ChildTopics
		if child == nil {
			child = []taxpkg.TreeNode{}
		}
		childVal, err := treeNodesJSONB(child).Value()
		if err != nil {
			return err
		}
		model := &courseTopicRow{}
		return gormx.RetryInSavepoint(tx, "course_topic_slug_retry", taxonomySlugRetryAttempts, isTaxonomySlugDuplicateKey, func() error {
			finalSlug, err := resolveTaxonomyUpdateSlug(ctx, tx, model, t.Slug, t.ID) // NEW
			if err != nil {
				return err
			}
			updates := map[string]any{
				"name": t.Name, "slug": finalSlug, "status": t.Status, // MODIFIED: "slug": finalSlug (was t.Slug directly)
				"image_file_id": t.ImageFileID, "child_topics": childVal,
			}
			if err := gormx.OptimisticUpdate(ctx, tx, model, t.ID, expectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
				return err
			}
			if err := upsertNameTranslations(ctx, tx, constants.TableTaxonomyCourseTopicTranslations, "topic_id", t.ID, t.Translations); err != nil {
				return err
			}
			t.Slug = finalSlug // NEW
			t.RowVersion = expectedRowVersion + 1
			if v, ok := updates["updated_at"].(int64); ok {
				t.UpdatedAt = v
			}
			return nil
		})
	})
}

func (r *GormCourseSkillRepository) Create(ctx context.Context, s *domain.CourseSkill) error {
	args := createNameTranslationArgs[courseSkillRow, domain.CourseSkill]{
		Entity: s, RowVersion: &s.RowVersion, ToRow: courseSkillToRow,
		EntityID: &s.ID, CreatedAt: &s.CreatedAt, UpdatedAt: &s.UpdatedAt, Meta: metaCourseSkillRow,
		Table: constants.TableTaxonomyCourseSkillTranslations, FKCol: "skill_id", Translations: s.Translations,
		ManualSlug: s.Slug, Name: s.Name, SlugPrefix: "skill", // NEW
		SetSlug: func(e *domain.CourseSkill, sl string) { e.Slug = sl }, // NEW
		NewRow:  func() *courseSkillRow { return &courseSkillRow{} },    // NEW
	}
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormCourseSkillRepository) Save(ctx context.Context, s *domain.CourseSkill, expectedRowVersion int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child := s.Children
		if child == nil {
			child = []taxpkg.TreeNode{}
		}
		childVal, err := treeNodesJSONB(child).Value()
		if err != nil {
			return err
		}
		model := &courseSkillRow{}
		return gormx.RetryInSavepoint(tx, "course_skill_slug_retry", taxonomySlugRetryAttempts, isTaxonomySlugDuplicateKey, func() error {
			finalSlug, err := resolveTaxonomyUpdateSlug(ctx, tx, model, s.Slug, s.ID) // NEW
			if err != nil {
				return err
			}
			updates := map[string]any{
				"name": s.Name, "slug": finalSlug, "status": s.Status, // MODIFIED
				"children": childVal,
			}
			if err := gormx.OptimisticUpdate(ctx, tx, model, s.ID, expectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
				return err
			}
			if err := upsertNameTranslations(ctx, tx, constants.TableTaxonomyCourseSkillTranslations, "skill_id", s.ID, s.Translations); err != nil {
				return err
			}
			s.Slug = finalSlug // NEW
			s.RowVersion = expectedRowVersion + 1
			if v, ok := updates["updated_at"].(int64); ok {
				s.UpdatedAt = v
			}
			return nil
		})
	})
}

func (r *GormTagRepository) Create(ctx context.Context, t *domain.Tag) error {
	args := createNameTranslationArgs[tagRow, domain.Tag]{Entity: t, RowVersion: &t.RowVersion}
	args.ToRow = tagToRow
	args.EntityID = &t.ID
	args.CreatedAt = &t.CreatedAt
	args.UpdatedAt = &t.UpdatedAt
	args.Meta = metaTagRow
	args.Table = constants.TableTaxonomyTagTranslations
	args.FKCol = "tag_id"
	args.Translations = t.Translations
	args.ManualSlug = t.Slug // NEW
	args.Name = t.Name       // NEW
	args.SlugPrefix = "tag"  // NEW
	args.SetSlug = func(e *domain.Tag, s string) { e.Slug = s } // NEW
	args.NewRow = func() *tagRow { return &tagRow{} }           // NEW
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormTagRepository) Save(ctx context.Context, t *domain.Tag, expectedRowVersion int64) error {
	return saveSlugStatusEntity(ctx, r.db, slugStatusSaveArgs[tagRow]{
		Model: &tagRow{}, ID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
		ExpectedRowVersion: expectedRowVersion, RowVersion: &t.RowVersion, UpdatedAt: &t.UpdatedAt,
		Table: constants.TableTaxonomyTagTranslations, FKCol: "tag_id", Translations: t.Translations,
	})
}

func (r *GormCourseLevelRepository) Create(ctx context.Context, cl *domain.CourseLevel) error {
	args := createNameTranslationArgs[courseLevelRow, domain.CourseLevel]{Entity: cl}
	args.RowVersion = &cl.RowVersion
	args.ToRow = courseLevelToRow
	args.EntityID, args.CreatedAt, args.UpdatedAt = &cl.ID, &cl.CreatedAt, &cl.UpdatedAt
	args.Meta = metaCourseLevelRow
	args.Table, args.FKCol = constants.TableTaxonomyCourseLevelTranslations, "course_level_id"
	args.Translations = cl.Translations
	args.ManualSlug = cl.Slug // NEW
	args.Name = cl.Name       // NEW
	args.SlugPrefix = "level" // NEW
	args.SetSlug = func(e *domain.CourseLevel, s string) { e.Slug = s } // NEW
	args.NewRow = func() *courseLevelRow { return &courseLevelRow{} }  // NEW
	return createWithNameTranslations(ctx, r.db, args)
}

func (r *GormCourseLevelRepository) Save(ctx context.Context, cl *domain.CourseLevel, expectedRowVersion int64) error {
	args := slugStatusSaveArgs[courseLevelRow]{
		Model: &courseLevelRow{}, ID: cl.ID, Name: cl.Name, Slug: cl.Slug, Status: cl.Status,
		ExpectedRowVersion: expectedRowVersion, RowVersion: &cl.RowVersion, UpdatedAt: &cl.UpdatedAt,
		Table: constants.TableTaxonomyCourseLevelTranslations, FKCol: "course_level_id", Translations: cl.Translations,
	}
	return saveSlugStatusEntity(ctx, r.db, args)
}

// saveSlugStatusEntity — MODIFIED (shared by Tag.Save and CourseLevel.Save
// above, both unchanged themselves — only this shared function's body changes):
func saveSlugStatusEntity[R any](ctx context.Context, db *gorm.DB, args slugStatusSaveArgs[R]) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return gormx.RetryInSavepoint(tx, "taxonomy_slug_status_retry", taxonomySlugRetryAttempts, isTaxonomySlugDuplicateKey, func() error {
			finalSlug, err := resolveTaxonomyUpdateSlug(ctx, tx, args.Model, args.Slug, args.ID) // NEW
			if err != nil {
				return err
			}
			updates := map[string]any{"name": args.Name, "slug": finalSlug, "status": args.Status} // MODIFIED
			if err := gormx.OptimisticUpdate(ctx, tx, args.Model, args.ID, args.ExpectedRowVersion, updates, domain.ErrTaxonomyOptimisticLock); err != nil {
				return err
			}
			if err := upsertNameTranslations(ctx, tx, args.Table, args.FKCol, args.ID, args.Translations); err != nil {
				return err
			}
			*args.RowVersion = args.ExpectedRowVersion + 1
			if v, ok := updates["updated_at"].(int64); ok {
				*args.UpdatedAt = v
			}
			return nil
		})
	})
}
```
`slugStatusSaveArgs[R]` struct itself is **unchanged** (`Slug string` field already existed as a plain write-input carrier). New import needed in `repos.go`: `"mycourse-io-be/internal/shared/gormx"` is **already imported** (confirmed: `"mycourse-io-be/internal/shared/gormx"` at the top of the existing file) — no new import required for `RetryInSavepoint`.

### D9 — Database migrations

Two new migrations, next available numbers confirmed at `/opsx:apply` time by `ls migrations/*.up.sql | sed -E 's#.*/([0-9]+)_.*#\1#' | sort -n | tail -1` (currently `000040`, so `000041`/`000042` — **must be re-checked at apply time**, not hardcoded from this design pass, in case another change lands a migration first).

`migrations/000041_backfill_taxonomy_slug_ascii.up.sql`:
```sql
-- Regenerate any existing taxonomy root slug value that does not conform to
-- the new format (non-ASCII passthrough from utils.SlugifyName, or any
-- other pre-existing malformed value) across all 4 tables, so the CHECK
-- constraints added in 000042 cannot fail to apply. Scans EVERY row
-- regardless of deleted_at (the CHECK constraint applies table-wide, unlike
-- the partial unique indexes which only cover active rows) -- mirrors
-- courses' own 000039 backfill exactly, same reasoning.
--
-- Plain UPDATE statements, not a DO block: this repo's migration runner
-- splits files on every literal semicolon character, including inside
-- comments -- see migrations/README.md and 000039's own history of finding
-- this out twice. No comment line in this file may contain that character.
--
-- created_at/updated_at on these 4 tables are TIMESTAMPTZ (unlike courses'
-- BIGINT epoch seconds) -- NOW(), not extract(epoch FROM now())::bigint.
--
-- The replacement slug uses a 16-character lowercase hex suffix from md5()
-- over enough entropy (row id, wall-clock, and random()) that a collision
-- is astronomically unlikely, and md5 output is already confined to
-- [0-9a-f], so no further sanitizing is needed to satisfy the slug format.
UPDATE course_levels
SET slug = 'level-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = NOW()
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE course_topics
SET slug = 'topic-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = NOW()
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE course_skills
SET slug = 'skill-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = NOW()
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';

UPDATE tags
SET slug = 'tag-' || substr(md5(id::text || clock_timestamp()::text || random()::text), 1, 16),
    updated_at = NOW()
WHERE slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$';
```
`migrations/000041_backfill_taxonomy_slug_ascii.down.sql`:
```sql
-- Data backfill is not safely reversible: the original non-conforming slug
-- values are not recoverable once regenerated. Matches courses' own
-- 000039_backfill_course_slug_ascii.down.sql precedent.
```

`migrations/000042_taxonomy_slug_check_constraints.up.sql`:
```sql
ALTER TABLE course_levels
    ADD CONSTRAINT chk_course_levels_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE course_topics
    ADD CONSTRAINT chk_course_topics_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE course_skills
    ADD CONSTRAINT chk_course_skills_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');

ALTER TABLE tags
    ADD CONSTRAINT chk_tags_slug_format
    CHECK (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$');
```
`migrations/000042_taxonomy_slug_check_constraints.down.sql`:
```sql
ALTER TABLE course_levels DROP CONSTRAINT IF EXISTS chk_course_levels_slug_format;
ALTER TABLE course_topics DROP CONSTRAINT IF EXISTS chk_course_topics_slug_format;
ALTER TABLE course_skills DROP CONSTRAINT IF EXISTS chk_course_skills_slug_format;
ALTER TABLE tags DROP CONSTRAINT IF EXISTS chk_tags_slug_format;
```
No new table, no new column — all 4 `slug` columns are already `NOT NULL`; the `CHECK` only adds format enforcement. `uix_*_slug_active` (uniqueness, `migrations/000012...up.sql`) is untouched.

**Per the same practice course's own change used**: before writing/applying `000042`, run `SELECT id, slug FROM course_levels WHERE deleted_at IS NULL AND slug !~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'` (and the equivalent for the other 3 tables) against the real dev DB during `/opsx:apply` to confirm the actual dirty-row count — record it, do not assume.

### D10 — Module map (packages touched)

| Package | Change |
|---|---|
| `internal/shared/gormx` | **new** `retry_savepoint.go` (`RetryInSavepoint`) |
| `internal/shared/taxonomy` | **new** `tree_write.go` (`BuildNodeIndexByID`, `ResolveTreeSlugs`); `tree_resolve.go` gains `PrepareTreeForWriteWithSlugPolicy` (additive) |
| `internal/shared/slug`, `internal/shared/validate`, `internal/shared/errors` | **untouched** — reused as-is |
| `internal/course/infra` | `repo_instructor.go`'s `applyUpdateSlugWithRetry` rewritten (bugfix only) |
| `internal/course/*` (everything else) | **untouched** |
| `internal/taxonomy/domain` | `errors.go` gains `SlugConflictError`; `taxonomy.go`'s 6 input structs gain `Slug`/`Slug validate.Optional[string]`; `repository.go` **untouched** (D4) |
| `internal/taxonomy/application` | **new** `slug.go`; `service.go` (Create/Update Topic/Skill, internal helpers) and `service_helpers.go` (generic Tag/CourseLevel path) rewritten; `translation_sync.go` loses `prepareTreeWrite` (dead after this change) |
| `internal/taxonomy/delivery` | `dto.go` (new `TreeNodeInput`, `slug` fields); `handler_helpers.go` (new `toDomainTreeNodes`, widened generic mutation helpers, new `SlugConflictError` mapping); `handler.go` (8 handler functions' call sites) |
| `internal/taxonomy/infra` | **new** `slug.go`; `repos.go` (4 repositories' Create/Save + `saveSlugStatusEntity`); `repos_mappers.go` (`createNameTranslationArgs`/`createWithNameTranslations`) |
| `internal/instructor` | **untouched** (explicitly deferred, per Non-Goals) |
| `migrations` | 2 new files (D9) |
| `go.mod`/`go.sum` | **unchanged** — no new dependency (unlike course's `github.com/gosimple/slug`; taxonomy keeps `utils.SlugifyName`) |

### D11 — Code review round: 2 real findings fixed after `/opsx:apply` landed the code

`/code-review` found 2 real, verified findings in the implementation D1-D10 describe — both reuse/cleanup issues, not correctness bugs (confirmed: `go build`/`go vet`/`go test` all passed cleanly before and after, across every touched package).

**Finding 1 — `UpdateCourseSkill` duplicated `applyCourseTopicChildren`'s tree-resolve-for-update block inline instead of reusing a shared helper.** Both `applyCourseTopicChildren` (`CourseTopic.ChildTopics`) and `UpdateCourseSkill`'s inline tree handling (`CourseSkill.Children`) built an identical `oldByID := taxpkg.BuildNodeIndexByID(current); source := current; if incoming != nil { source = *incoming }; resolveTreeForUpdate(source, oldByID)` sequence — a future change to this logic (e.g. a new preserve-by-id edge case) applied to one would silently not apply to the other, letting topics and skills diverge in tree-slug handling despite every doc in this change claiming they "share the same node-slug rules."

Fix: extracted `resolveTreeFieldForUpdate(current []taxpkg.TreeNode, incoming *[]taxpkg.TreeNode) ([]taxpkg.TreeNode, error)` in `internal/taxonomy/application/service.go`, used by both `applyCourseTopicChildren` and `UpdateCourseSkill`:
```go
func applyCourseTopicChildren(row *domain.CourseTopic, childTopics *[]taxpkg.TreeNode) error {
	resolved, err := resolveTreeFieldForUpdate(row.ChildTopics, childTopics)
	if err != nil {
		return err
	}
	row.ChildTopics = resolved
	return nil
}

// resolveTreeFieldForUpdate resolves a tree field's node slugs for update,
// shared by applyCourseTopicChildren (CourseTopic.ChildTopics) and
// UpdateCourseSkill (CourseSkill.Children) — both entities have an
// independent tree field with identical "omitted → preserve from current,
// provided → use incoming" semantics, differing only in which struct field
// holds the tree, which the caller already resolves before calling in.
func resolveTreeFieldForUpdate(current []taxpkg.TreeNode, incoming *[]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	oldByID := taxpkg.BuildNodeIndexByID(current)
	source := current
	if incoming != nil {
		source = *incoming
	}
	return resolveTreeForUpdate(source, oldByID)
}
```
`UpdateCourseSkill`'s own inline block (SUPERSEDES the version shown earlier in D5) is replaced by a single `resolveTreeFieldForUpdate(row.Children, in.Children)` call.

**Finding 2 — `PrepareTreeForWrite` (and the `NormalizeTreeSlugs` always-recompute-from-name path it exclusively called) became dead code on every write path after this change, but were left in place "for any caller wanting the old behavior."** `grep` confirmed zero callers outside their own file once `resolveTreeForCreate`/`resolveTreeForUpdate` switched to `PrepareTreeForWriteWithSlugPolicy`. Keeping an exported function whose entire purpose is the exact "slug always tied to name" bug this change fixes invites a future contributor to call it by mistake and reintroduce that bug.

Fix: **deleted**, not deprecated in place — `internal/shared/taxonomy/tree_slug.go` (the whole file, `NormalizeTreeSlugs`) removed entirely; `PrepareTreeForWrite` removed from `tree_resolve.go`, leaving `PrepareTreeForWriteWithSlugPolicy` as the sole write-path entry point in that file. No test referenced either removed symbol by name (confirmed before deleting). `docs/reusable-assets.md` and `docs/modules/taxonomy.md` updated to match (no mention of the removed functions as "kept for compatibility").

**Verification**: both fixes covered by the existing test suite (no behavior change for any currently-exercised path — `resolveTreeFieldForUpdate` is a pure refactor, and the dead-code removal has no runtime effect by definition). Full gate chain (`gofmt`, `go vet`, `go test` ×full suite, `golangci-lint`, `check-layout`, `check-architecture`, `check-dupl`, `build-nocgo`, `build`) re-run clean after both fixes — `check-dupl` specifically confirms the `dupl` clone pair the review found is gone (0 clone groups).

### Documentation updates

Full file-by-file audit performed via a dedicated research pass reading every file live (not from memory), covering every doc file that matched a repo-wide `taxonomy`/`slug` grep. Summary of disposition (exact literal OLD/NEW text for every file needing a change is tracked in `tasks.md`'s documentation section, reproduced there in full so `/opsx:apply` can apply each block directly):

- **Needs changes**: `docs/reusable-assets.md` (5 blocks: stale "taxonomy never accepts client slug" claim on the `SlugifyName` asset; add taxonomy as a second `internal/shared/slug` consumer; note the course-orchestration asset now points at a parallel taxonomy asset; add a new "Taxonomy slug orchestration" asset; add a new "RetryInSavepoint" asset; update the "Taxonomy tree + description validators" asset for `ResolveTreeForWrite`), `docs/modules/taxonomy.md` (7 blocks: reuse-map row, "regenerate only from canonical name" sentence, create/update body sentence, tree nodes sentence, mutation-errors sentence, directory layout ×2, migrations table), `docs/database.md` (5 blocks: soft-delete sentence, 4× per-table slug column row, migrations table append — exact migration numbers confirmed at apply time), `docs/api_swagger.yaml` (6 blocks: `CreateCourseLevelRequest`/`UpdateCourseLevelRequest`, `TreeNode`, `CreateCourseTopicRequest`/`UpdateCourseTopicRequest`, `CreateCourseSkillRequest`/`UpdateCourseSkillRequest`, `CreateTagRequest`/`UpdateTagRequest`, 4× new `409` response blocks on create endpoints reusing the existing `SlugConflictResponse` schema — no new schema), `docs/curl_api.md` (6 blocks: levels/topics/skills/tags create-body sentences + example bodies, update-body sentences), `docs/return_types.md` (1 block), `docs/patterns.md` (2 blocks: slug table row, tree-omit-on-PATCH row), `docs/security-public-seo-notes.md` (1 block: B8 row), `docs/sequence_diagrams.md` (1 block — a **pre-existing** staleness found opportunistically: the "Create Course" diagram still shows `SlugifyName(title)`, which course itself stopped using in `rework-course-slug-management`; corrected here since this change's doc sweep already touches this file's neighborhood).
- **No change needed** (confirmed by reading, not assumed): `docs/modules/course.md` (already correctly describes the *intended* update-retry contract this change's D1 makes actually work — the fix doesn't change documented behavior, only makes broken code match it; one optional clarifying clause recommended, not required), `docs/architecture.md`, `docs/api-overview.md`, `docs/requirements.md`, `docs/dependencies.md`, `docs/modules.md`, `docs/router.md`, `docs/folder-structure.md`, `docs/deploy.md`, `docs/modules/media.md`, `docs/data-flow.md`, `docs/modules/rbac.md`, `docs/modules/auth.md`, `docs/modules/instructor.md` (no "taxonomy"/"slug" match relevant to this change, or matches confirmed unrelated — e.g. OAuth `provider` "slug", instructor `current_job_title_id` legacy backfill format).
- **Regenerate, do not hand-edit**: `docs/api-dog-import.json` via `ruby scripts/generate-apidog-postman.rb` after `docs/api_swagger.yaml` lands — same established convention as course's own change.

## Risks / Trade-offs

- **[Risk] Always-resolve-on-update runs one extra SELECT per taxonomy update, even when slug is untouched** → **Mitigation**: accepted deliberately (D2) — taxonomy writes are low-frequency admin operations, and the alternative (threading a "was slug touched" flag through repository interfaces shared with slug-less `CourseOutcome`) is a larger, more invasive change for a negligible performance cost.
- **[Risk] `ResolveTreeSlugs`'s auto-generate fallback (`generateAutoNodeSlugBase`) duplicates 4 lines of derivation logic already in `internal/taxonomy/infra/slug.go`'s `generateAutoTaxonomySlugBase`** → **Mitigation**: `internal/shared/taxonomy` cannot import `internal/taxonomy/infra` (would invert the module dependency direction this codebase already enforces — confirmed via `docs/reusable-assets.md`'s own architecture note on bounded-context layering). Both copies call the same two underlying primitives (`utils.SlugifyName`, `internal/shared/slug`) — the *algorithm* is not duplicated, only a short call sequence gluing two existing primitives together. Documented explicitly in D7's code comment so a future reader does not mistake this for careless copy-paste.
- **[Risk] In-tree collision auto-resolution changes a manually-chosen slug without telling the client, for BOTH a genuinely-new node and an explicitly-renamed existing node** → **Mitigation**: this is the explicit, confirmed product decision (matches taxonomy root's own update-conflict behavior); the response always returns the actual final slug (spec requirement), so the client can detect and display the change even without an error.
- **[Risk] The 2 new migrations' exact numbers (`000041`/`000042` as drafted here) may be stale by the time `/opsx:apply` runs**, if another change lands a migration first → **Mitigation**: `tasks.md` requires re-running the `ls migrations/*.up.sql | ... | tail -1` check immediately before creating the migration files, not trusting this design doc's hardcoded numbers.
- **[Risk] `docs/sequence_diagrams.md`'s `SlugifyName(title)` fix is a pre-existing staleness from a prior change (`rework-course-slug-management`), not something this change's own design introduced** → **Mitigation**: flagged explicitly as opportunistic (found while doing this change's own doc sweep of the same file), not silently bundled in as if it were new; safe to include since it's a pure doc correction with no code/behavior implication.

## Migration Plan

1. Land D1 (shared `gormx.RetryInSavepoint` + course bugfix) first and independently — it has no dependency on anything taxonomy-specific and fixes a real, currently-shipping bug.
2. Land D2-D9 (taxonomy root + tree node slug) as a single unit — the taxonomy changes are too interdependent (domain input structs, infra resolution, application interpretation, delivery conversion, migrations) to safely split further within this change.
3. Run the 2 new migrations against the real dev DB (per D9's per-table dirty-row `SELECT` check) before writing the `CHECK`-constraint migration's final form, exactly as course's own change did.
4. Full doc sync (per "Documentation updates" above) after code lands, verified against the actually-shipped code (not this design's draft) — mirroring course's own 3-round documentation-correctness history.
5. No rollback strategy beyond each migration's own `down.sql` — the `CHECK` constraint's `down.sql` is a simple `DROP CONSTRAINT IF EXISTS`; the backfill's `down.sql` is intentionally a no-op (data changes are not safely reversible, same posture as course's `000039`).
