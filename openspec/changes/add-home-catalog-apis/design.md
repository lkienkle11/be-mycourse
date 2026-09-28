## Context

See `proposal.md` for motivation. This design covers the "how": exact route wiring, exact SQL, exact caching mechanism, the exact code for every new/changed symbol, the exact GitNexus impact-analysis results (per `CLAUDE.md`'s mandate), and the exhaustive documentation-sync content required by this repo's own conventions.

This repo already anticipated this exact feature in several "take-note" markers written in a prior planning pass (2026-07-25), which this design follows rather than re-deciding from scratch:
- `docs/security-public-seo-notes.md` (B1–B10): public GETs must be **published-only projections**, never the full `CourseDetail`/`Profile` shape; must stay **cookie/Bearer-free**; must **reuse existing rate-limit tiers**, not invent a parallel quota system; must **reuse the existing auth cache-aside pattern** rather than a new cache mechanism.
- `docs/patterns.md` ("Public SEO DTO pattern" take-note): "Prefer published-only projections and existing rate-limit tiers over inventing parallel auth/quota stacks."
- `docs/requirements.md` NFR-1.1 (rate limiting) and NFR-1.3 (caching) are the authoritative tables for these tiers/keys and must gain new rows, not new tiers.
- `openspec/changes/*` house style (5 prior changes) confirms the `proposal.md` / `design.md` / `tasks.md` / `specs/<domain>/<capability-slug>/spec.md` structure and `domain/kebab-case-name` capability IDs used here.

This design was also checked against every skill under `.ai/skills/` (mandatory per `AGENTS.md`) — see "Skill Compliance" below.

## Goals / Non-Goals

**Goals:**
- Ship 3 endpoints (2 public, 1 authenticated) using only existing schema (plus index-only additions), existing middleware, and existing cache infrastructure.
- Every new public response is a narrow, hand-picked projection — never the existing `CourseListItem`/`CourseDetail`/`Profile`/`RosterMember` structs, so no internal/workflow field can leak by accident and no unrelated endpoint's JSON shape changes.
- Update every documentation file this repo's own take-notes and `patterns.md`'s "Documentation Sync" checklist point at, with the exact content to insert (not just a pointer to "add a row").
- Keep every new list query free of N+1 access patterns (per `.ai/skills/logic-n-1-optimize`) and properly indexed (per `.ai/skills/postgresql-optimization`).

**Non-Goals (explicitly deferred, already agreed with the user):**
- Price, discount, "Best Seller" badges — no payment/pricing schema exists anywhere in this codebase (`docs/modules/enrollment.md`: "Payment and commercial enrollment flows are still outside this implementation"); out of scope.
- Learner star rating / review count — no such table exists; the only existing "review" concept (`internal/course/infra/repo_review.go`) is the admin content-approval workflow, unrelated; out of scope.
- Reviving the deprecated `instructor_profiles.headline` column — the instructor subtitle uses `current_job_title` instead (already agreed).
- Cache invalidation on course publish — `TopicCoursePublished` is not emitted yet (`docs/security-public-seo-notes.md` B4); this design accepts up to the cache TTL's staleness window instead of building that event hook now.
- Any `fe-mycourse` change.

## Skill Compliance (`.ai/skills/`)

- **`logic-n-1-optimize`** (batch API / avoid N+1): none of the 3 endpoints are multi-item mutations, so the batch-endpoint-shape rule doesn't apply directly; the relevant part is "avoid N+1 database queries." All 3 list endpoints are single aggregate SQL statements — the continue-learning sub-lesson-completion counts are computed with `GROUP BY`/subqueries across all of a learner's enrollments in one query (see Decision #5 below), never one query per course.
- **`postgresql-optimization`**: audited existing indexes on every touched table (`migrations/000016_course_management.up.sql`) — `courses` has no index on `created_at` at all, and `course_enrollments` has no index on `user_id` alone. This design adds a new index-only migration (Decision #7) rather than accepting sequential scans as the tables grow.
- **`cache-strategy-implementer`** (generic checklist): layering = Redis in front of Postgres for the 2 public lists only; invalidation = TTL-based (5 min), not event-based, given `TopicCoursePublished` doesn't exist yet; rollback = the cache is purely additive and fails open (`cache.RedisAvailable()` guard), so disabling/removing it never breaks correctness.

## Decisions

### 1. New `/api/v1/catalog/...` URL namespace for the 2 public endpoints
**Chosen:** `GET /catalog/courses/trending`, `GET /catalog/instructors/popular`, mounted as `catalog := notAuthen.Group("/catalog")`.
**Why:** The existing authenticated trees already use `GET /courses/:courseId` and `GET /instructors/:id` — a public `GET /courses/trending` would sit on the same Gin radix tree as a wildcard `:courseId` segment at the same depth. Gin resolves static-vs-param unambiguously in practice, but a dedicated `/catalog` prefix removes any doubt, and gives FE/CDN one unambiguous prefix to point ISR/edge-cache rules at.
**Alternative considered:** Reuse `/courses/trending` directly under the public group. Rejected only for the clarity/CDN-prefix reason above.

### 2. Reuse the existing `notAuthen` rate-limit tier — no new tier
**Chosen:** `catalog` is a plain sub-group of `notAuthen`; it inherits `middleware.RateLimitLocal(60, 1)` (`router.go:80`, documented as NFR-1.1) with no additional `.Use(...)`.
**Why:** `docs/requirements.md` NFR-1.1 and `docs/patterns.md`'s take-note both explicitly say to extend the existing tier rather than add a parallel one.
**Alternative considered:** A dedicated, more generous tier (e.g. 90/min). Rejected — the doc trail is explicit that a new tier should not be invented.

### 3. Cache-aside via the existing Redis client as a package-level global, not new DI
**Chosen:** New file `internal/shared/cache/json_cache.go`:
```go
package cache

import (
	"context"
	"encoding/json"
	"time"
)

// GetJSON reads a JSON-encoded value from Redis. Returns (zero, false) on a
// cache miss, decode error, or when Redis is unavailable — callers must
// always be able to fall through to the database.
func GetJSON[T any](ctx context.Context, key string) (T, bool) {
	var out T
	if !RedisAvailable() {
		return out, false
	}
	raw, err := Redis.Get(ctx, key).Bytes()
	if err != nil {
		return out, false
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, false
	}
	return out, true
}

// SetJSON writes a JSON-encoded value with the given TTL. No-op (not an
// error) when Redis is unavailable.
func SetJSON[T any](ctx context.Context, key string, ttl time.Duration, value T) {
	if !RedisAvailable() {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	Redis.Set(ctx, key, raw, ttl)
}
```
Course/instructor's new `service_catalog.go` files call these directly:
```go
// internal/course/application/service_catalog.go
package application

import (
	"context"
	"fmt"
	"time"

	"mycourse-io-be/internal/course/domain"
	"mycourse-io-be/internal/shared/cache"
)

const trendingCoursesCacheTTL = 5 * time.Minute

func (s *CourseService) ListTrendingCourses(ctx context.Context, limit int) ([]domain.TrendingCourseItem, error) {
	key := fmt.Sprintf("mycourse:catalog:trending_courses:limit:%d", limit)
	if cached, ok := cache.GetJSON[[]domain.TrendingCourseItem](ctx, key); ok {
		return cached, nil
	}
	items, err := s.repo.ListTrendingCourses(ctx, limit)
	if err != nil {
		return nil, err
	}
	cache.SetJSON(ctx, key, trendingCoursesCacheTTL, items)
	return items, nil
}
```
(Instructor's `ListPopularInstructors` mirrors this exactly, key `mycourse:catalog:popular_instructors:limit:%d`.)
**Why:** Two existing precedents already coexist in this codebase: `internal/auth/application/service_cache.go` injects a redis client via DI, while `internal/shared/middleware/circuitbreaker.go` / `internal/shared/resilience/circuitbreaker.go` read the global `cache.Redis`/`cache.RedisAvailable()` directly for cross-instance state. The global-access style is lower blast radius here (`CourseService`/`InstructorService` constructors and `wire_course.go`/`wire_instructor_adapters.go` stay untouched) and this cache has no per-request validation logic (unlike auth's `/me` cache, which validates `UserID`/`Roles` on read).
**Alternative considered:** Thread a redis client through `NewCourseService`/`NewInstructorService` like auth does. Rejected to avoid touching 2 more wiring files and 2 constructor signatures for no behavioral benefit.
**New cache keys (added to `docs/requirements.md` NFR-1.3):**
| Key | TTL | Source |
|---|---|---|
| `mycourse:catalog:trending_courses:limit:{n}` | 5 min | trending courses list |
| `mycourse:catalog:popular_instructors:limit:{n}` | 5 min | popular instructors list |

### 4. New, narrow DTOs instead of extending shared structs
**Chosen — domain layer** (`internal/course/domain/course.go`, appended near `CourseListItem`):
```go
// TrendingCourseItem is the published-only projection served by the public
// trending-courses catalog endpoint (see design.md Decision #4 — deliberately
// not CourseListItem, to avoid leaking fields into unrelated endpoints).
type TrendingCourseItem struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Title            string `json:"title"`
	ShortDescription string `json:"short_description"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
	OwnerDisplayName string `json:"owner_display_name,omitempty"`
	CreatedAt        int64  `json:"created_at"`
}

// ContinueLearningItem is the authenticated learner's own in-progress course
// projection served by the continue-learning endpoint. Counts every
// sub-lesson kind (VIDEO/QUIZ/TEXT) — corrected during apply after user
// review: the Figma mock's "X/Y Videos Completed" label only showed
// video-only example courses, but this project's real courses mix content
// kinds under one course (Section -> Lesson -> Sub-lesson), so progress
// must count all of them, not just VIDEO.
type ContinueLearningItem struct {
	CourseID            string `json:"course_id"`
	Slug                string `json:"slug"`
	Title               string `json:"title"`
	ThumbnailURL        string `json:"thumbnail_url,omitempty"`
	OwnerDisplayName    string `json:"owner_display_name,omitempty"`
	CompletedSubLessons int    `json:"completed_sub_lessons"`
	TotalSubLessons     int    `json:"total_sub_lessons"`
	LastActivityAt      int64  `json:"last_activity_at"`
}
```
`Repository` interface additions (appended after `ListPublishedCourses`, `course.go:283`):
```go
ListTrendingCourses(ctx context.Context, limit int) ([]TrendingCourseItem, error)
ListContinueLearning(ctx context.Context, userID string, limit int) ([]ContinueLearningItem, error)
```
**Instructor domain** (`internal/instructor/domain/instructor.go`, appended near `RosterMember`):
```go
// PopularInstructor is the public projection served by the popular
// instructors catalog endpoint.
type PopularInstructor struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar,omitempty"`
	Subtitle    string `json:"subtitle,omitempty"` // instructor_profiles.current_job_title
	CourseCount int64  `json:"course_count"`
}
```
`internal/instructor/domain/repository.go` — new sub-interface, composed alongside the existing 5:
```go
type CatalogRepository interface {
	ListPopularInstructors(ctx context.Context, limit int) ([]PopularInstructor, error)
}

type Repository interface {
	ApplicationRepository
	ProfileRepository
	RosterRepository
	ExpertiseRepository
	TicketRepository
	InstructorDataRepository
	CatalogRepository // new
}
```
**Why narrow, separate structs:** `CourseListItem` is embedded across admin catalog, trash, review-queue, and "my courses" responses; `RosterMember`/`Profile` are used across roster/application/profile admin responses. Adding trending/popular-only fields to those would leak new fields into unrelated authenticated responses and widen the impact radius of a single struct edit across endpoints this change has no business touching — and directly satisfies `docs/security-public-seo-notes.md`'s "published-only projection" requirement.
**Alternative considered:** Add `ShortDescription`/`CourseCount` fields to `CourseListItem`/`RosterMember`. Rejected for the leakage/blast-radius reason above.

### 5. Exact SQL for the 3 new repository methods

**`internal/course/infra/repo_catalog.go` (new file) — trending courses.** Built by literally reusing the existing Go string constants from `repos.go:232-238` (`courseListOwnerDisplayNameColumn`, `courseListOwnerUserJoin`) via concatenation — not re-typing equivalent SQL — exactly the pattern `ListPublishedCourses` (`repo_learner.go:15-45`) already uses for `courseListBaseColumns`:
```go
const trendingCoursesQuery = `
SELECT c.id, c.slug, c.created_at,
    pv.title, pv.short_description,
    COALESCE(pm.url, '') AS thumbnail_url,
    ` + courseListOwnerDisplayNameColumn + `
FROM courses c
INNER JOIN course_versions pv
    ON pv.id = c.current_published_version_id AND pv.deleted_at IS NULL
LEFT JOIN media_files pm
    ON pm.id = pv.thumbnail_file_id AND pm.deleted_at IS NULL
` + courseListOwnerUserJoin + `
WHERE c.deleted_at IS NULL AND c.trashed_at IS NULL AND c.current_published_version_id IS NOT NULL
ORDER BY c.created_at DESC
LIMIT ?`

func (r *GormRepository) ListTrendingCourses(ctx context.Context, limit int) ([]domain.TrendingCourseItem, error) {
	var rows []trendingCourseScanRow
	if err := r.db.WithContext(ctx).Raw(trendingCoursesQuery, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.TrendingCourseItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}
```
(Served by the new `idx_courses_published_created_at` partial index — Decision #7. `pv.short_description` already exists on `course_versions`, unused by today's `ListPublishedCourses`. Matches `ListPublishedCourses`'s exact `r.db.WithContext(ctx).Raw(q).Scan(&rows)` call shape, verified by direct read of `repo_learner.go:15-45` this review pass. **`current_published_version_id IS NOT NULL` is stated explicitly in `WHERE`** — added after a complexity review found that Postgres cannot reliably prove a partial index's predicate is satisfied when it's only implied by an `INNER JOIN` condition; an explicit clause is required to guarantee the planner can use this partial index instead of falling back to a full sort. See "Complexity analysis" below.)

**`internal/course/infra/repo_learner.go` (append method) — continue learning.** Two corrections made along the way, both after actual user review of the design/output (not guessed):
1. **Counting unit** — the user pointed out (against the Figma mock's "X/Y Videos Completed" label) that this project's real course outline is `Section -> Lesson -> Sub-lesson`, and a course legitimately mixes `VIDEO`/`QUIZ`/`TEXT` sub-lessons under one course; Figma's video-only wording only happened to describe video-only example courses. Progress therefore counts **every sub-lesson, any kind** — confirmed via `AskUserQuestion` against the 3 real schema tiers (all sub-lessons / whole lessons / whole sections); "all sub-lessons regardless of kind" was chosen, matching the Figma numbers' magnitude (e.g. 40) better than counting whole sections (which would typically be a much smaller number per course).
2. **Completion source** — `course_progress_items.content_type` is **client-supplied and not server-validated** against the real sub-lesson kind — `internal/course/delivery/dto.go:139` only has `validate:"required"` on it (non-empty), and `CourseService.SaveProgress` (`application/service.go:203-205`) is a pure passthrough to the repo with no cross-check against `course_sub_lessons.kind`. So completion is derived from `course_sub_lessons` (the authoritative outline) joined by `stable_content_id = course_sub_lessons.stable_id` via a `LEFT JOIN LATERAL`, scoped to the enrollment's own `current_version_id` — not from trusting `content_type`:
```sql
SELECT c.id, c.slug, cv.title,
       COALESCE(pm.url, '') AS thumbnail_url,
       COALESCE(ou.display_name, '') AS owner_display_name,
       COALESCE(prog2.completed_sub_lessons, 0) AS completed_sub_lessons,
       COALESCE(prog2.total_sub_lessons, 0) AS total_sub_lessons,
       COALESCE(prog.last_interacted_at, e.created_at) AS last_activity_at
FROM course_enrollments e
INNER JOIN courses c ON c.id = e.course_id AND c.deleted_at IS NULL
INNER JOIN course_versions cv ON cv.id = e.current_version_id AND cv.deleted_at IS NULL
LEFT JOIN media_files pm ON pm.id = cv.thumbnail_file_id AND pm.deleted_at IS NULL
LEFT JOIN users ou ON ou.id = c.owner_user_id AND ou.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT MAX(last_interacted_at) AS last_interacted_at
    FROM course_progress_items
    WHERE enrollment_id = e.id AND deleted_at IS NULL
) prog ON TRUE
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) AS total_sub_lessons,
        COUNT(*) FILTER (WHERE pi.status = 'COMPLETED') AS completed_sub_lessons
    FROM course_sub_lessons sl
    LEFT JOIN course_progress_items pi
        ON pi.enrollment_id = e.id
       AND pi.stable_content_id = sl.stable_id
       AND pi.deleted_at IS NULL
    WHERE sl.course_version_id = e.current_version_id
      AND sl.deleted_at IS NULL
) prog2 ON TRUE
WHERE e.user_id = ? AND e.deleted_at IS NULL
ORDER BY COALESCE(prog.last_interacted_at, e.created_at) DESC
LIMIT ?
```
**Complexity correction found by Big-O review (post-apply):** `prog` was originally a plain (non-`LATERAL`) `GROUP BY enrollment_id` derived table joined on `prog.enrollment_id = e.id`. Postgres cannot push an outer equality predicate into a `GROUP BY` subquery before aggregating — it would aggregate `MAX(last_interacted_at)` over **every** `course_progress_items` row on the entire platform (`O(P)`, P = platform-wide progress-row count) just to read one learner's own last-activity timestamp. Rewritten as `LATERAL` with `WHERE enrollment_id = e.id` inside the subquery, so it is provably bounded per outer row and uses `uix_course_progress_items_active (enrollment_id, stable_content_id)` for an index-scoped lookup — cost now depends only on this learner's own enrollment count, not platform size. See "Complexity analysis" section below for the full per-query Big-O breakdown.
(No `kind` filter — every sub-lesson counts, regardless of `VIDEO`/`QUIZ`/`TEXT`. Still a single SQL statement — no N+1 network round-trips, per `.ai/skills/logic-n-1-optimize`: `LEFT JOIN LATERAL` is evaluated once per outer row by the Postgres planner within the same query, not as a separate query per enrollment from the Go application. Served by the new `idx_course_enrollments_user_active` index — Decision #7.)

**`internal/instructor/infra/repo_catalog.go` (new file) — popular instructors.** Verified this review pass directly against `internal/instructor/infra/repo_roster_list.go:39-51`'s `ListRoster` — same "is this user a currently-active instructor" check, reusing `userpicker.ActiveUserWhereClause()` (`internal/shared/userpicker/eligible_clause.go:5-8`: `u.is_disable = FALSE AND (u.banned_until IS NULL OR u.banned_until <= @now)`) instead of hand-rolling a weaker `is_disable`-only check that would miss actively-banned-but-not-disabled accounts:
```go
const popularInstructorsQuery = `
SELECT u.id AS user_id, u.display_name,
    COALESCE(m.url, '') AS avatar_url,
    COALESCE(ip.current_job_title, '') AS subtitle,
    agg.course_count
FROM (
    SELECT c.owner_user_id, COUNT(*) AS course_count, MAX(c.created_at) AS latest_course_at
    FROM ` + constants.TableCourses + ` c
    WHERE c.deleted_at IS NULL AND c.trashed_at IS NULL AND c.current_published_version_id IS NOT NULL
    GROUP BY c.owner_user_id
) agg
INNER JOIN ` + constants.TableAppUsers + ` u ON u.id = agg.owner_user_id AND u.deleted_at IS NULL
INNER JOIN ` + constants.TableRBACUserRoles + ` ur ON ur.user_id = u.id
INNER JOIN ` + constants.TableRBACRoles + ` r ON r.id = ur.role_id AND r.name = @role_name
LEFT JOIN ` + constants.TableInstructorProfiles + ` ip ON ip.user_id = u.id AND ip.deleted_at IS NULL
LEFT JOIN ` + constants.TableMediaFiles + ` m ON m.id = u.avatar_file_id AND m.deleted_at IS NULL
WHERE 1=1 ` + userpicker.ActiveUserWhereClause() + `
ORDER BY agg.course_count DESC, agg.latest_course_at DESC
LIMIT @limit`

func (r *GormRepository) ListPopularInstructors(ctx context.Context, limit int) ([]domain.PopularInstructor, error) {
	var rows []popularInstructorScanRow
	err := r.db.WithContext(ctx).Raw(popularInstructorsQuery, map[string]any{
		"role_name": instdomain.RoleNameInstructor,
		"now":       timex.NowUnix(),
		"limit":     limit,
	}).Scan(&rows).Error
	// ... map rows -> []domain.PopularInstructor, same shape as ListRoster's row-mapping loop
}
```
(The `courses` aggregate is computed once in a subquery, then joined to identity/profile data — not per-instructor N+1, per `.ai/skills/logic-n-1-optimize`. Named-param style (`@role_name`, `@now`, `@limit`) matches `ListRoster`'s exact GORM `Raw(sql, map[string]any{...})` convention, since this query — like `ListRoster` — has 3 bound values; the simpler positional-`?` style used for trending/continue-learning above is equally valid GORM usage for 1-2 params, matching `ListPublishedCourses`'s no-param style at the low end of that same spectrum. Served by the new `idx_courses_owner_published_created_at` partial index — Decision #7.)

**Note on table-name style:** the trending/continue-learning queries above use literal table names (`courses`, `course_versions`, ...), matching their immediate neighbor `ListPublishedCourses` in the same file (`repo_learner.go`), which also uses literals, not `constants.Table*`. The popular-instructors query uses `constants.Table*` because its neighbor `ListRoster` (`repo_roster_list.go`) does. This is deliberate — each new query matches its own file's existing local convention rather than picking one style repo-wide, since neither convention is declared canonical anywhere in `docs/patterns.md`.

### 6. `RegisterRoutes` signature change mirrors the existing `auth` module exactly
**Chosen:**
```go
// internal/course/delivery/routes.go
func RegisterRoutes(authen, notAuthen *gin.RouterGroup, h *Handler, pc middleware.PermissionChecker) {
	if authen != nil {
		// ... existing body unchanged, verbatim ...
		learner.GET("/continue", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.getContinueLearning)
		learner.GET("/:courseId", utils.RoutePermission(pc, constants.AllPermissions.CourseRead), h.getLearningCourse)
		// ... rest unchanged ...
	}
	if notAuthen != nil {
		catalog := notAuthen.Group("/catalog/courses")
		catalog.GET("/trending", h.listTrendingCourses)
	}
}
```
(identical shape for `internal/instructor/delivery/routes.go`, with `catalog := notAuthen.Group("/catalog/instructors"); catalog.GET("/popular", h.listPopularInstructors)`.)
**Router call sites** (`internal/server/router.go:92-93`):
```go
coursedelivery.RegisterRoutes(authen, notAuthen, h.Course, svc.RBAC)
instdelivery.RegisterRoutes(authen, notAuthen, h.Instructor, svc.RBAC)
```
**Why:** copied verbatim from `internal/auth/delivery/routes.go:16-53`, the only existing delivery package in this repo that already splits public/authenticated routes in one file — reusing the one existing convention rather than inventing a second (e.g. a separate `RegisterPublicRoutes` function).
**Impact — verified, not assumed** (see "GitNexus Impact Analysis" below): 1 caller each.

### 7. New index-only migration (`migrations/000038_home_catalog_indexes`)
Required by `.ai/skills/postgresql-optimization` — see proposal.md's revised "What Changes". `up.sql`:
```sql
-- Speeds up "trending courses": ORDER BY created_at DESC over published, non-trashed courses.
CREATE INDEX idx_courses_published_created_at
    ON courses (created_at DESC)
    WHERE deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL;

-- Speeds up "popular instructors": GROUP BY owner_user_id + MAX(created_at) tie-break, same filter.
CREATE INDEX idx_courses_owner_published_created_at
    ON courses (owner_user_id, created_at DESC)
    WHERE deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL;

-- Speeds up "continue learning": WHERE user_id = ? across a learner's enrollments.
CREATE INDEX idx_course_enrollments_user_active
    ON course_enrollments (user_id)
    WHERE deleted_at IS NULL;
```
`down.sql`:
```sql
DROP INDEX IF EXISTS idx_course_enrollments_user_active;
DROP INDEX IF EXISTS idx_courses_owner_published_created_at;
DROP INDEX IF EXISTS idx_courses_published_created_at;
```
**Why partial indexes:** every one of the 3 new queries filters on the same `deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL` (or `deleted_at IS NULL`) predicate — a partial index matching that exact predicate is smaller and faster than a full-table index, and Postgres will use it automatically when the query's `WHERE` clause matches or implies the index predicate.
**No existing index is modified or dropped** — purely additive, zero risk to any existing query plan.

## Delivery layer: exact handlers (no separate response DTOs — corrected during apply)

Verified directly against `internal/course/delivery/handler_learner.go`, `handler_review.go`, `handler_admin.go`, and `handler_base.go:124-138`: **every existing no-path-param list handler in this module returns its domain slice directly** via `response.OK(c, "ok", rows)` (`listPublishedCourses`, `listPendingReviews`, `listAdminCourses` all do this) — no separate response-struct/mapper layer. An earlier draft of this design added `trendingCourseResponse`/`continueLearningResponse` structs that were byte-for-byte identical to the domain types — pure duplication, corrected here: no response DTOs for `TrendingCourseItem`/`ContinueLearningItem`. The one field that needed deriving (`progress_percent`) was moved onto the domain type itself (`ContinueLearningItem.ProgressPercent`, computed once in the repo's row-mapping — see Decision #5). **Re-corrected during apply:** the instructor module's own convention is different from course's — every instructor domain struct (`RosterMember`, `Profile`, etc.) is deliberately kept **tag-free**, with JSON tags living only on delivery-layer response DTOs, with no exception. So `PopularInstructor` was made tag-free to match, and it **does** get a `popularInstructorResponse` DTO + `toPopularInstructorResponse` mapper in `internal/instructor/delivery/dto.go` (a trivial field-for-field copy, unlike `rosterResponse`'s rename, but still following the module's real "domain is JSON-tag-free" rule rather than course's "domain already carries tags" rule). Each module's new code matches its own existing convention rather than a single repo-wide rule — this file was itself wrong about this point on an earlier pass, caught only once the code was actually written.

**`internal/shared/utils/query_limit.go` (new file — shared from the start, not duplicated per module):** confirmed via grep that no `clamp`/`ParseLimit`/query-int-clamping helper exists anywhere in `internal/shared/utils` today:
```go
package utils

import "strconv"

// ClampQueryLimit parses raw (a "limit" query param) as a positive int,
// falling back to def on empty/invalid/non-positive input, and capping at max.
func ClampQueryLimit(raw string, def, max int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
```

**`internal/course/delivery/handler_catalog.go` (new file):**
```go
package delivery

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/utils"
)

const (
	trendingCoursesDefaultLimit = 8
	trendingCoursesMaxLimit     = 24
)

func (h *Handler) listTrendingCourses(c *gin.Context) {
	limit := utils.ClampQueryLimit(c.DefaultQuery("limit", ""), trendingCoursesDefaultLimit, trendingCoursesMaxLimit)
	rows, err := h.svc.ListTrendingCourses(c.Request.Context(), limit)
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}
```

**`internal/course/delivery/handler_learner.go` (append):**
```go
const (
	continueLearningDefaultLimit = 4
	continueLearningMaxLimit     = 10
)

func (h *Handler) getContinueLearning(c *gin.Context) {
	limit := utils.ClampQueryLimit(c.DefaultQuery("limit", ""), continueLearningDefaultLimit, continueLearningMaxLimit)
	rows, err := h.svc.ListContinueLearning(c.Request.Context(), utils.CurrentUserID(c), limit)
	if mapCourseError(c, err) {
		return
	}
	response.OK(c, "ok", rows)
}
```

**Instructor side — `internal/instructor/delivery/handler_public.go` (new file), no DTO:**
```go
const (
	popularInstructorsDefaultLimit = 4
	popularInstructorsMaxLimit     = 12
)

func (h *Handler) listPopularInstructors(c *gin.Context) {
	limit := utils.ClampQueryLimit(c.DefaultQuery("limit", ""), popularInstructorsDefaultLimit, popularInstructorsMaxLimit)
	rows, err := h.svc.ListPopularInstructors(c.Request.Context(), limit)
	if mapInstructorError(c, err) { // internal/instructor/delivery/handler_helpers.go:18 — existing convention, mirrors mapCourseError
		return
	}
	out := make([]popularInstructorResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPopularInstructorResponse(row))
	}
	response.OK(c, "ok", out)
}
```
(`utils.ClampQueryLimit` is used by both `internal/course/delivery/handler_catalog.go` and `internal/instructor/delivery/handler_public.go` — added once, directly, to `internal/shared/utils` rather than duplicated per module, since neither module had this helper before this change.)

## GitNexus Impact Analysis (run this session, per `CLAUDE.md`)

`npx gitnexus analyze` was re-run first (index was 2 commits behind; now current at commit `4c5feeb` → refreshed, 7,956 nodes / 23,287 edges / 300 flows). `gitnexus_impact` was then run on the 3 highest-blast-radius symbols this change touches:

| Symbol | GitNexus result | Manual verification (grep) |
|---|---|---|
| `RegisterRoutes` (course + instructor delivery) | `impactedCount: 0`, `risk: UNKNOWN` — tool notes "No callers resolved... confirm with a text search" (Go call-graph edges through package-qualified function calls aren't fully resolved by this index) | Exactly 1 caller each: `internal/server/router.go:92` (course), `:93` (instructor). No other file references either symbol. |
| `Repository` (course + instructor domain interfaces) | Same `UNKNOWN`/0 result, same tool limitation | **Correction found during `/opsx:apply` (the grep-only check below missed this):** `internal/course/application` has no fake full-interface implementer (only `CourseService`/`GormRepository`), but **`internal/instructor/application` has 2**: `appTestRepo` (`service_application_test.go`, also used by `validate_profile_test.go`) and `rosterBulkTestRepo` (`service_roster_test.go`) both implement `domain.Repository` in full for unit tests. Adding `CatalogRepository`/`ListPopularInstructors` to the interface broke both at compile time (caught immediately by the IDE's Go diagnostics, not by the grep-based check originally planned) — fixed by adding a trivial `ListPopularInstructors(context.Context, int) ([]domain.PopularInstructor, error) { return nil, nil }` stub to each. **Lesson: a plain-text grep for "mock"/"fake" in filenames is not sufficient to find fake interface implementers — a real compiler/build check is required, which is why this was still caught before merge.** |
| `ListPublishedCourses` (repo + service) | Same `UNKNOWN`/0 result | Left untouched by this change (trending uses a new, separate method) — confirmed no other caller depends on its current `ORDER BY c.id DESC` behavior changing. |

**Conclusion:** all 3 symbols are low blast-radius (single-caller or additive-only); GitNexus's Go call-graph resolution under-reports here (documented tool limitation, not a false "safe"), so this table records the actual verified caller counts rather than relying on the tool's `UNKNOWN` risk label alone.

## Complexity analysis (post-apply, user-requested Big-O review)

No live DB this session (dev server down), so this is reasoned from the schema/index facts already verified (not a captured `EXPLAIN ANALYZE`) — must be confirmed with a real `EXPLAIN (ANALYZE, BUFFERS)` once the dev DB is up (see Migration Plan / deferred items).

**1. Trending courses (`trendingCoursesQuery`)** — N = number of published, non-trashed courses; K = `limit` (≤24).
- With `idx_courses_published_created_at (created_at DESC) WHERE deleted_at IS NULL AND trashed_at IS NULL AND current_published_version_id IS NOT NULL`, and the query's `WHERE` now stating that predicate explicitly (fixed this pass — see Decision #5): Postgres can do an **Index Scan** in `created_at DESC` order and stop after `K` rows — **O(log N + K)**, i.e. effectively constant relative to `N` for a fixed `K`. Each of the 3 joins (`course_versions`, `media_files`, `users`) is a primary-key lookup per matched row: `O(K · log(table size))`, negligible.
- Without the explicit predicate (the bug just fixed), Postgres cannot prove the partial index applies from the `INNER JOIN` alone and may fall back to a Seq Scan + Sort of every qualifying course — **O(N log N)**.
- Rough magnitude: at N in the low thousands, the fixed version should be low-single-digit milliseconds; the buggy version would grow with catalog size (still fast at N=10k, but scales the wrong way as the catalog grows — exactly the class of bug that looks fine in dev and slows down in production).

**2. Popular instructors (`popularInstructorsQuery`)** — N = number of published, non-trashed courses; G = number of distinct instructors among them (G ≤ N); K = `limit` (≤12).
- This query has a `GROUP BY owner_user_id` **before** the `ORDER BY … LIMIT`. A `LIMIT` cannot short-circuit an aggregate: every qualifying course row must be read once to know each instructor's true count. The partial index `idx_courses_owner_published_created_at (owner_user_id, created_at DESC)` lets Postgres do this as a single-pass **GroupAggregate** (sorted-by-key streaming) instead of a Seq Scan + HashAggregate, and skips non-published/trashed/deleted rows — but the asymptotic class stays **O(N)** (linear in the courses table), not O(log N) or O(1). No index can make "count every row per group" sub-linear without a maintained rollup table, which this design does not add (out of scope — see Non-Goals).
- After aggregation, sorting the `G` groups by `course_count DESC` with a `LIMIT` uses Postgres's Top-N heapsort — **O(G log K)**, small.
- The `users`/`user_roles`/`roles`/`instructor_profiles`/`media_files` joins run once per one of the `G` groups (not per course row), each an indexed lookup: **O(G log M)**.
- **Total: O(N + G log M)**, dominated by **O(N)** — this is the one query of the 3 that is genuinely linear in a growing table, not a bug to "fix" (it's inherent to "rank by count over the whole table"), but worth knowing: **the 5-minute Redis cache-aside (Decision #3) converts this from "O(N) on every HTTP request" to "O(N) once per 5-minute window, O(1) cache hit for every other request in that window"** — this is the concrete reason this endpoint needed a cache more than the other two, not just as blanket "public-endpoint hygiene."
- Rough magnitude: at N in the low-to-mid thousands, a `COUNT`/`MAX` GroupAggregate over narrow int/UUID columns is typically single-digit milliseconds on modest hardware; this will grow roughly linearly as the platform's total published-course count grows, but is only paid once per cache TTL window, not per request.

**3. Continue learning (`continueLearningQuery`)** — e = this one learner's own active-enrollment count (small, bounded — realistically a handful to a few dozen courses per learner, never proportional to platform size); p = average sub-lesson/progress-row count per course (bounded by course size).
- The outer `course_enrollments` scan uses `idx_course_enrollments_user_active (user_id) WHERE deleted_at IS NULL` — **O(log E + e)**, E = total enrollments table size.
- For each of the `e` enrollment rows: 3 more PK-indexed joins (`courses`, `course_versions`, plus `media_files`/`users`), and **2 `LATERAL` subqueries** (`prog`, `prog2`), each explicitly filtered by `enrollment_id = e.id` / `course_version_id = e.current_version_id` inside the subquery — each is an indexed, per-row lookup bounded by that one enrollment's own data (`O(log P + p)`, not the whole platform).
- **Total: O(e · (log P + p))** — bounded by this learner's own small enrollment count, independent of total platform size. This is the query with a genuine, already-fixed bug (see Decision #5): an earlier draft's `prog` subquery was a plain (non-`LATERAL`) `GROUP BY enrollment_id` derived table — Postgres cannot push the outer `e.id` predicate into a `GROUP BY` subquery before aggregating, so that plan would have aggregated `MAX(last_interacted_at)` over **every** `course_progress_items` row on the **entire platform** (`O(P)`, P = platform-wide progress-row count, unbounded and growing forever) just to answer one learner's request — a correctness-preserving but severely mis-scaling bug that would not show up in dev/staging with little data, then degrade badly in production as the platform accumulates learner activity. Fixed by making `prog` `LATERAL` with the filter inside it, matching `prog2`'s already-correct shape.
- Rough magnitude: with the fix, expect low-single-digit milliseconds regardless of platform size (bounded by one learner's own small data); the buggy version's cost would have scaled with total platform-wide progress-row count — likely fine at launch, a real production incident within a few months of real usage.

**Go-side application logic** (handlers/services, all 3 endpoints): no nested loops, no query-in-a-loop (confirmed `.ai/skills/logic-n-1-optimize` compliance earlier), just a single `for _, row := range rows { ... }` mapping loop bounded by the returned row count (≤ `limit`, a small constant ≤24). This is **O(K)**, linear in the (small, bounded) result set — never quadratic, and not a function of table/platform size at all. `utils.ClampQueryLimit` is O(1) (one `strconv.Atoi` + 2 comparisons). `cache.GetJSON`/`SetJSON` are O(1) Redis round-trips plus O(payload size) JSON marshal/unmarshal, where payload size is itself O(K). **Conclusion: all complexity risk in this change lives in the SQL layer, not the Go logic layer** — which is exactly why this review focused there.

## Risks / Trade-offs

- **[Risk] Up to 5-minute staleness on public lists after a new course/instructor changes rank** → **Mitigation:** acceptable per product decision (simple `created_at`/count-based ranking, not real-time trending); documented as a known limitation in `security-public-seo-notes.md`, not silently hidden.
- **[Risk] `current_job_title` can theoretically be blanked by an admin `PATCH /instructor-profiles/:id` (no non-empty check on that path, only on submit)** → **Mitigation:** response mapping treats an empty value as "no subtitle" (`omitempty`) rather than erroring; does not block the instructor from appearing in the list.
- **[Risk] Interface changes to `course.domain.Repository` / `instructor.domain.Repository` are broad-surface symbols** → **Mitigation:** see "GitNexus Impact Analysis" above — course has a single concrete implementer; instructor additionally has 2 test fakes, both updated to keep compiling.
- **[Risk] Gin route-tree ambiguity between a new static segment and an existing `:id`/`:courseId` wildcard at the same depth** → **Mitigation:** avoided entirely by the `/catalog` namespace decision (#1) for the 2 public routes; the 1 authenticated addition (`/learner-courses/continue`) uses the well-established Gin pattern of a static sibling next to a param route.
- **[Risk] New partial indexes add write overhead to `courses`/`course_enrollments` inserts/updates** → **Mitigation:** negligible — 3 small partial indexes on low-write-frequency tables (courses are created/updated far less often than read), and all 3 already have at least one index each, so this is an incremental, not first-time, write cost.
- **[Risk] `course_progress_items.content_type` is client-supplied and not server-validated against the real sub-lesson kind** (pre-existing gap, not introduced by this change) → **Mitigation:** the continue-learning sub-lesson-count query (Decision #5) derives completion from `course_sub_lessons` (the authoritative outline row existing) joined by `stable_id`, instead of trusting `content_type`, so this endpoint's correctness does not depend on that pre-existing gap. This gap otherwise remains out of scope for this change (fixing `SaveProgress` validation is a separate concern).

## Migration Plan

1. Apply `migrations/000038_home_catalog_indexes.up.sql` (index-only, no lock beyond Postgres's standard `CREATE INDEX` — acceptable for this table's current size; use `CREATE INDEX CONCURRENTLY` instead if the production `courses` table has grown large enough that a brief lock is a concern at deploy time).
2. Merge code → standard CI (`make check-all`, tests, build).
3. Deploy — new routes are purely additive; no existing endpoint's request/response shape changes, so this is safe to roll forward or back at any time.
4. Rollback: `migrations/000038_home_catalog_indexes.down.sql` drops the 3 new indexes; code rollback is a plain revert (no data migration to reverse).
5. Post-deploy: run `npx gitnexus analyze` (per `CLAUDE.md`/`AGENTS.md`) and `gitnexus_detect_changes({scope: "all"})` to confirm only the expected symbols/files changed.
