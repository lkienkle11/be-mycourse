## Context

See `proposal.md - Why` for motivation and `specs/user/slug-management/spec.md` for requirements. Everything below was verified by reading the code (2026-10-15). GitNexus was 4 commits behind HEAD and resolved **zero callers** for every slug symbol (generics and cross-file calls are not captured), so all call sites come from direct reading and grep. Task 0 re-indexes and re-runs `impact` before any edit.

### Current state (facts)

- `users` (`migrations/000001_schema.up.sql:27`) has no slug. `email` and `user_code` are plain unique constraints; `deleted_at` is BIGINT (000011). Latest migration is `000042`; `migrations/README.md`'s chain table stops at `000034`.
- Shared slug primitives (pure, domain-agnostic): `internal/shared/slug/{autobase,format,suffix}.go` — `GenerateAutoSlugBase(name, maxLen, kind) (base, mustSuffix)` (gosimple/slug, fallback = the `Kind` string), `ValidateManualFormat`, `RetryWithSuffix` (7 chars x7 attempts, then 8, 9, ...), `TruncateToLen`, `TruncateForSuffix`, closed `Kind` enum (course, level, topic, skill, tag, node). Also `internal/shared/validate/optional.go`, `internal/shared/gormx/retry_savepoint.go` (`RetryInSavepoint`), `apperrors.SlugConflict = 3007`.
- Write paths: all user inserts end at `GormUserRepository.CreateWithDB` (`internal/auth/infra/user_repo.go:70`). `registerNewPending` (`auth/application/service.go:178`) calls `Create` with no transaction. OAuth `createOAuthUser` (`service_oauth.go:272-299`) → `authOAuthAccountWriter.CreateUserWithIdentityAndLearnerRole` (`internal/server/wire_auth_oauth.go:71-86`, one transaction: user, identity, learner role). `SaveWithDB` (`user_repo.go:81`) is a **full-row** `Save(toUserRow(u))`, used by confirm (`wire_auth_confirm.go`), link/merge (`wire_auth_oauth.go:88-103`) and resend (`service.go:205`).
- `UpdateMe` (`service.go:319`) returns early when `avatarFileID == nil`; `PATCH /me` accepts only `avatar_file_id`. `display_name` has no production writer except `registerResendPending`.
- `/me` is cached (`mycourse:user:me:{id}`, 1 min, `auth/application/service_cache.go`, invalidated by `delCachedMe` / `InvalidateUserMeCache`; instructor and rbac already reach it through their own consumer-owned ports).

### Duplicated capabilities found (Dedup / "single owner" rule)

| # | Capability | Implementations (file:line) | Same semantics? | Decision |
|---|---|---|---|---|
| 1 | suffix candidate builder | `course/infra/slug.go:18 buildSuffixedCandidate`; `taxonomy/infra/slug.go:81 buildTaxonomySuffixedCandidate`; inline in `shared/taxonomy/tree_write.go:100-110 resolveNodeSlugSuffix` | Identical (truncate base for suffix, error when no room); only max length differs | Consolidate → `slug.BuildSuffixedCandidate(base, suffix, maxLen)` |
| 2 | slug availability query | `course/infra/slug.go:33 courseSlugAvailable`; `taxonomy/infra/slug.go:35 taxonomySlugAvailable[R]` | Identical body (taxonomy is the generic form) | Consolidate → `gormslug.AvailableIn` |
| 3 | duplicate-key detection | `course/infra/repo_helpers.go:291`; `taxonomy/infra/slug.go:55`; `auth/application/service_oauth.go:337 isOAuthUniqueViolation`; `rbac/application/service.go:303 isDuplicateError` | Course/taxonomy: index-scoped. OAuth/RBAC: any unique violation (RBAC also matches SQLite wording) | Consolidate → `gormx.IsUniqueViolation(err, constraintNames ...string)`; no names = any, names = index-scoped. All four call it. |
| 4 | create-slug resolver | `course/infra/repo_instructor.go:150 resolveCreateSlug`; `taxonomy/infra/slug.go:96 resolveTaxonomyCreateSlug[R]` | Same algorithm | Consolidate → `slug.ResolveCreate` |
| 5 | update-slug resolver | `course/infra/repo_instructor.go:311 resolveUpdateSlug`; `taxonomy/infra/slug.go:151 resolveTaxonomyUpdateSlug[R]` | Same algorithm | Consolidate → `slug.ResolveUpdate` |
| 6 | retry attempt count | `repo_instructor.go:23` (`courseSlugCreateRetry`), `:295` (inline `updateSlugRetry`); `taxonomy/infra/slug.go:27` | Same value 5 | Consolidate → `gormslug.RetryAttempts` |
| 7 | conflict error type | `course/domain/errors.go:45`; `taxonomy/domain/errors.go:17` | Same fields; only `Error()` text differs | Consolidate → `slug.ConflictError`; both become `type SlugConflictError = slug.ConflictError` |
| 8 | 409 mapping | `course/delivery/handler_base.go:33-37`; `taxonomy/delivery/handler_helpers.go:134-138` | Identical | Consolidate → one shared mapper (see D1) |
| 9 | manual slug input parsing | `course/delivery/handler_instructor.go:51-62`; `course/application/service.go:44-49`; `taxonomy/application/slug.go:16 validateManualTaxonomySlug`, `:34 resolveTaxonomySlugInput`; `taxonomy/delivery/handler_helpers.go:35-47` | Same logic, different error sentinel | Consolidate → `slug.ParseManual` / `slug.ParseUpdateInput`; each feature maps sentinels to its own error |
| 10 | user + avatar picker projection | `shared/userpicker/list.go:60 UserPickerSelectSQL`; `course/infra/repo_collaborators.go:78 collaboratorProfileSelectSQL` | Same columns and media join (verify exact equality in task 3.4) | Consolidate → collaborators uses `userpicker`; keep separate and record why if any column semantics differ |
| 11 | course owner name fragments | `course/infra/repos.go:234-236` consts; hand-written copy in `repo_learner.go:61` | Same module, same semantics | Consolidate inside course: `continueLearningQuery` uses the shared consts |
| 12 | /me invalidation ports | `rbac/delivery/handler.go:26`; `instructor/ports.go:32`; `server/wire_instructor_adapters.go:93` | Same behavior | **Keep**: each is a consumer-owned one-method port (interface segregation), all delegate to `AuthService.InvalidateUserMeCache`; merging would create a sibling dependency |
| 13 | random generators | `slug.GenerateRandomSuffix` vs `utils.GenerateRandomDigits` | Different (digits, panics, biased) | Keep separate; slugs use the former only |
| 14 | tree slug uniqueness | `shared/taxonomy/tree_write.go:50` | In-memory, no DB | Keep separate |
| 15 | media owner / instructor admin identity selects | `media/infra/repos.go:56`; `instructor/infra/repos.go:78,145,428,459,528,550` | Different (admin/private identity, media admin) | Keep separate; slug not added (not public surfaces) |

Capabilities with **no** existing implementation (so new, single-owner code): reserved-word list, slug history, keyset batch backfill loop, `user_slug_history` repository code.

### Tooling constraints that shape the design (must not be changed or bypassed)

- `make test-all` = `go test ./...`, `CGO_ENABLED=1 go test ./...`, `go vet`, `golangci-lint run`, `check-layout`, `check-architecture`, `check-dupl` (`dupl -t 60 internal`, must print `Found total 0 clone groups`). `make check-all` = `go fmt`, `test-all`, `build-nocgo`, `build`.
- `.go-arch-lint.yml`: `domain` may depend only on `shared`; `application` on domain/shared/pkg; `infra` on application/domain/shared/pkg; `delivery` on application/domain/jobs/shared/pkg; `shared/**` is a common component (importing vendors such as gorm is allowed, importing feature packages is not). `.golangci.yml` depguard forbids gorm, `encoding/json`, gin, uuid in `domain`. funlen 60 lines/40 statements, gocyclo 15, revive file-length 600 / argument-limit 8, nolintlint (no suppressions; also forbidden by AGENTS.md).
- `.husky/` and commitlint config do **not** exist in this repo; `make check-all` is run manually. No commit is created unless the user asks.

## Goals / Non-Goals

**Goals:**
- Exactly one implementation of each capability in the table above (rows 1-11), consumed by course, taxonomy and users.
- A slug assigned inside the same transaction as every user insert; no raw DB error leaks.
- Self-service slug change with permanent reservation of former slugs.
- Slug returned on every response surface listed in D7.
- A rollout that never fails a migration on un-backfilled rows.

**Non-Goals:**
- Public lookup / redirect / availability endpoints; dropping `user_id`; throttling slug changes.
- Any change to course or taxonomy requirements (refactor is behavior-preserving; no delta spec).
- Slug on instructor tickets and media owner (not public surfaces).

## Decisions

### D1 — Shared modules (ownership and exact API)

**`internal/shared/slug`** (pure, no gorm; owner = "slug format and generation"):

| File | Add |
|---|---|
| `autobase.go` | `KindUser Kind = "user"` |
| `errors.go` (new) | `type ConflictError struct{ RecommendedSlug string }` + `Error() string`; `var ErrInvalidFormat, ErrEmpty, ErrReserved error` |
| `suffix.go` | `BuildSuffixedCandidate(base, suffix string, maxLen int) (string, error)` — body of the two existing builders, built on `TruncateForSuffix` |
| `input.go` (new) | `ParseManual(raw string, maxLen int) (string, error)` (create; empty → `("", nil)` meaning auto); `ParseUpdateInput(set bool, value string, maxLen int) (*string, error)` (takes the `validate.Optional` pair as plain values because `shared/slug` must not import `shared/validate`: validate → utils → response → slug would be an import cycle; omitted → `(nil, nil)`; set + empty/whitespace/null → `ErrEmpty`; bad format → `ErrInvalidFormat`) |
| `reserved.go` (new) | `IsReserved(slug string) bool` over an unexported immutable set |

**`internal/shared/slug/resolve.go`** (pure; added because the repo has no sqlmock/sqlite, so DB-bound resolvers are untestable; a callback-based algorithm is unit-testable with fakes):
```go
type AvailableFunc func(candidate string) (bool, error)
// ResolveCreate: manual taken -> ("", *ConflictError{recommended}, nil); manual free -> manual;
// auto: GenerateAutoSlugBase then free base or RetryWithSuffix.
func ResolveCreate(manual, name string, kind Kind, maxLen int, avail AvailableFunc) (string, *ConflictError, error)
// ResolveUpdate: free -> newSlug, else RetryWithSuffix (never a conflict error).
func ResolveUpdate(newSlug string, maxLen int, avail AvailableFunc) (string, error)
```
Reserved words and history are folded into the caller's `AvailableFunc` (a reserved or historical candidate simply reports "not available"), so no extra hooks are needed.

**`internal/shared/slug/gormslug`** (imports gorm + `slug` + `gormx`; owner = "table-backed availability and retry policy"):
```go
// AvailableIn builds an AvailableFunc: active rows (deleted_at IS NULL) of the model's table with slug = ?,
// optionally excluding one id (update self-collision).
func AvailableIn[R any](ctx context.Context, db *gorm.DB, newModel func() *R, excludeID *string) slug.AvailableFunc
const RetryAttempts = 5
func IsDuplicate(err error, dupIndexes ...string) bool // gormx.IsUniqueViolation(err, dupIndexes...)
```
Course, taxonomy (4 tables) and users each keep a tiny local spec (model constructor, max length, kind, index names) and call `slug.ResolveCreate`/`slug.ResolveUpdate` with `gormslug.AvailableIn`.

**`internal/shared/gormx`**: `IsUniqueViolation(err error, constraintNames ...string) bool` — case-insensitive; prefers `*pgconn.PgError` (`Code == "23505"`, `ConstraintName`) when present, falls back to the existing "duplicate key" / "unique constraint" substring match so SQLite-based tests and the RBAC path keep working.

**Shared 409 mapper**: `response.FailIfSlugConflict(c *gin.Context, err error) bool` in `internal/shared/response/slug_conflict.go` (the package that owns `response.Fail`).

Alternatives rejected: importing taxonomy helpers from users (sibling dependency); a third copy (violates Dedup and fails `check-dupl`); putting gorm into the pure `slug` package (forces every importer, including `domain` aliases, to pull gorm).

### D2 — Schema and two-step rollout

```
000043_user_slug (release 1)
  ALTER TABLE users ADD COLUMN slug VARCHAR(255)
  ALTER TABLE users ADD CONSTRAINT chk_users_slug_format
    CHECK (slug IS NULL OR (slug <> '' AND slug ~ '^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$'))
  CREATE UNIQUE INDEX uix_users_slug_active ON users (slug) WHERE deleted_at IS NULL
  CREATE TABLE user_slug_history (
      id UUID PRIMARY KEY,
      user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
      slug VARCHAR(255) NOT NULL,
      created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
      CONSTRAINT uq_user_slug_history_slug UNIQUE (slug),
      CONSTRAINT chk_user_slug_history_slug_format CHECK (slug <> '' AND slug ~ '<same regex>'))
  CREATE INDEX idx_user_slug_history_user_id ON user_slug_history (user_id)
000044_user_slug_not_null (release 2, only after backfill count = 0)
  ALTER TABLE users ALTER COLUMN slug SET NOT NULL
```
- NULL never conflicts in a unique index and passes a CHECK, so `000043` always applies; no SQL placeholder slug (product decision: every slug comes from the one Go resolver).
- The partial unique index mirrors `uix_courses_slug_active` (000016) / taxonomy (000012). History uniqueness is global and permanent. `ON DELETE CASCADE` means a **hard** delete (`DELETE /me/hard`) frees that user's former slugs; a soft delete keeps them reserved.
- Migration runner splits on every `;` (`migrations/README.md`): no semicolons in comments or strings, no `DO` blocks. Audit timestamps are BIGINT epoch seconds.
- `.down.sql`: drop history table, index, constraint, column (000044 down: `DROP NOT NULL`).
- Index review for `postgresql-optimization`: lookups by slug use `uix_users_slug_active`; owner joins use the PK so no extra index; `EXPLAIN` evidence is a task.

### D3 — Row mapping must not wipe slug
`userRow.Slug *string` (`gorm:"column:slug;size:255"`), `domain.User.Slug string`. `toUserDomain`: nil → `""`. `toUserRow`: `""` → nil (never write `''`; the CHECK rejects it). Because `SaveWithDB` is a full-row save and every `FindBy*` goes through `toUserDomain`, both directions are mandatory and covered by a Find→Save round-trip test. `SaveWithDB` additionally uses `Omit("slug")`, so a full-row save from a stale in-memory user can never revert or wipe a slug that changed meanwhile; slug is written only by create, `ChangeSlug` and the backfill.

### D4 — Create and change flows
**Create** — `CreateWithDB` resolves the slug before the INSERT with `slug.ResolveCreate("", displayName, slug.KindUser, 255, avail)` where `avail` = `gormslug.AvailableIn` AND not in `user_slug_history` AND not `slug.IsReserved`; duplicate detection uses `gormslug.IsDuplicate(err, "uix_users_slug_active", "uq_user_slug_history_slug")`; no manual slug.
- Register (no tx today): the repository wraps each attempt in a fresh transaction and retries up to `RetryAttempts` while `IsDuplicate` (same shape as taxonomy `createWithNameTranslations`; no savepoint needed because each attempt is a new tx).
- OAuth (inside the writer's transaction): the INSERT is wrapped in `gormx.RetryInSavepoint` (savepoint name `user_slug_create_retry`) so identity and role writes survive a retry. `isOAuthUniqueViolation` becomes a call to `gormx.IsUniqueViolation(err)` **after** a slug-index check, so a slug collision is never reported as `ErrOAuthIdentityConflict`.
- `registerResendPending`, confirm, link and merge keep the stored slug (stability is intentional; slug does not follow `display_name`).

**Change** — `UpdateMe(ctx, userID, domain.UpdateMeParams{AvatarFileID *string, Slug *string})`:
1. Delivery parses `validate.Optional[string]` with `slug.ParseUpdateInput` (course pattern at `handler_instructor.go:51-62`) and maps `ErrEmpty`/`ErrInvalidFormat` to `ErrUserInvalidSlug` (400).
2. Service rejects `slug.IsReserved` with `ErrUserSlugReserved` (400); equal to current → no-op; otherwise `userRepo.ChangeSlug`.
3. `ChangeSlug` (one transaction): lock the user row, re-read the current slug, `slug.ResolveUpdate` (availability excludes self and also checks history and reserved) inside `gormx.RetryInSavepoint`; delete a history row equal to the new slug **if it belongs to the same user**; insert the old slug into `user_slug_history`; `UPDATE users SET slug, updated_at`.
4. After the transaction: `delCachedMe`. The early-return on nil avatar is replaced by per-field "skip when not set"; neither set → return `GetMe`.
- A conflicting request is auto-suffixed (never `409`), same as course/taxonomy update.

### D5 — Reserved words
Immutable set in `internal/shared/slug/reserved.go`, enabled only through `Spec.Reserved` (users). Initial set (to be reviewed by the product owner before apply): `admin, administrator, api, auth, catalog, course, courses, edit, help, instructor, instructors, login, logout, me, new, null, register, root, settings, support, system, undefined, user, users`. Course/taxonomy do not enable it (different semantics; behavior unchanged). `user` is reserved bare, consistent with the fallback always being `user-{suffix}`.

### D6 — Backfill command (batch, respects `logic-n-1-optimize`)
New `internal/appcli/backfill_user_slugs.go`: `MaybeRunBackfillUserSlugs(db *gorm.DB) bool`, gated by `CLI_BACKFILL_USER_SLUGS` via `parsebool.EnvEnabled`, `guardCLIOperation(ctx, constants.CLIOpBackfillUserSlugs)`, registered in `main.go` next to `MaybeRunImportLegacyData`. It calls `GormUserRepository.BackfillMissingSlugs(ctx, batchSize=500)`:
1. One query per batch: `SELECT id, display_name FROM users WHERE slug IS NULL AND id > $cursor ORDER BY id LIMIT $n` (includes soft-deleted rows).
2. Compute bases in Go (`GenerateAutoSlugBase`, reserved → suffix), de-duplicate within the batch in memory.
3. One availability query for the whole batch against `users` and `user_slug_history` (`slug IN (...)`); suffix only the taken ones via `RetryWithSuffix`.
4. One set-based `UPDATE ... FROM (VALUES ...)`/`unnest` per batch with `WHERE slug IS NULL` (idempotent). On a unique violation the batch falls back to row-by-row `ResolveCreate`.
5. Prints processed / updated / remaining counts; `ANALYZE users` is a runbook step afterwards.

### D7 — Read surfaces (additive; `user_id` kept)
Exact fields and files are enumerated in the **Change inventory** below. Rules: user-facing field is `slug`, course-owner field is `owner_slug` (`omitempty` on list structs that already `omitempty` the owner name); a `NULL` slug is returned as `""`.

## Change inventory (what is added, modified, deleted)

### Database
| Object | Action |
|---|---|
| `users.slug VARCHAR(255) NULL` | add (000043), `SET NOT NULL` (000044) |
| `chk_users_slug_format` | add (CHECK tolerates NULL) |
| `uix_users_slug_active` (partial unique) | add |
| `user_slug_history` + `uq_user_slug_history_slug`, `chk_user_slug_history_slug_format`, `idx_user_slug_history_user_id` | add |
| any existing table/column/index | none modified or removed |

### API
| Endpoint | Action | Detail |
|---|---|---|
| `PATCH /api/v1/me` | reuse + extend | request gains optional `slug` (omit / null→400 / same→no-op / new→auto-suffix); response `MeResponse` gains `slug` |
| `GET /api/v1/me` | reuse + extend | `slug` added to `MeResponse` (cached payload tolerant of old entries) |
| `GET /api/v1/catalog/instructors/popular` | reuse + extend | `slug` per item, `user_id` kept |
| `GET/PUT /instructor-applications/me`, `GET /instructor-applications`, `GET /instructor-applications/:id` | reuse + extend | `slug` in application response |
| `GET /instructor-profiles`, `/me`, `/:id` | reuse + extend | `slug` via `identityProjection` |
| `GET /instructors`, `GET /instructors/roster-candidates` | reuse + extend | `slug` on roster member / candidate |
| `GET /catalog/courses/trending`, `GET /learner-courses/continue`, `/courses/my`, `/course-admin/courses`, `/course-reviews/pending` (+ trash lists sharing `courseListScanRow`) | reuse + extend | `owner_slug` next to `owner_display_name` |
| course collaborators + instructor-candidates | reuse + extend | `slug` per person |
| `POST /auth/register`, OAuth callbacks | reuse, behavior extended | slug assigned internally, not in request/response |
| new routes | **none** | public get-by-slug is a follow-up change |

### Modules and code (per file)
| File | Symbols | Change |
|---|---|---|
| `internal/shared/slug/{autobase,suffix}.go` | `Kind`, new builder | add `KindUser`, `BuildSuffixedCandidate` |
| `internal/shared/slug/{errors,input,reserved}.go` | new | add `ConflictError`, sentinels, `ParseManual`, `ParseUpdateInput`, `IsReserved` (+ tests) |
| `internal/shared/slug/resolve.go` | new | `AvailableFunc`, `ResolveCreate`, `ResolveUpdate` (+ fake-driven unit tests) |
| `internal/shared/slug/gormslug/*.go` | new | `AvailableIn[R]`, `RetryAttempts`, `IsDuplicate` |
| `internal/shared/gormx/unique_violation.go` | new | `IsUniqueViolation` (+ test table: pg error, wrapped, substring, SQLite wording, scoped names) |
| shared response helper pkg | new func | `MapSlugConflict` |
| `internal/course/infra/slug.go` | delete | replaced by `slug.ResolveCreate/ResolveUpdate` + `gormslug.AvailableIn` called from `repo_instructor.go` |
| `internal/course/infra/repo_helpers.go:291` | delete `isCourseSlugDuplicateKey` | callers use `gormslug.IsDuplicate(err, "uix_courses_slug_active", "courses_slug")` |
| `internal/course/infra/repo_instructor.go:23,93-95,150,296,311` | modify | create/update use spec; drop `courseSlugCreateRetry`, inline `updateSlugRetry` |
| `internal/course/domain/errors.go:45` | modify | `SlugConflictError` becomes alias of `slug.ConflictError` |
| `internal/course/delivery/{handler_base.go:33-37,handler_instructor.go:51-62}` | modify | shared mapper, `ParseUpdateInput` |
| `internal/course/application/service.go:44-49` | modify | `ParseManual` |
| `internal/taxonomy/infra/slug.go` | delete | replaced by `slug_specs.go` (4 specs) |
| `internal/taxonomy/infra/{repos.go:211,362,514; repos_mappers.go:199,215-217,233}` | modify | use specs and `RetryAttempts` |
| `internal/taxonomy/domain/errors.go:17` | modify | alias |
| `internal/taxonomy/application/slug.go`, `service.go:77,142,346,387`, `service_helpers.go:72,129` | modify | thin wrappers over `ParseManual` / `ParseUpdateInput` |
| `internal/taxonomy/delivery/handler_helpers.go:35-47,134-138` | modify | shared parsing and mapper |
| `internal/shared/taxonomy/tree_write.go:100-110` | modify | `BuildSuffixedCandidate` |
| `internal/rbac/application/service.go:303` | modify | `IsUniqueViolation(err)` |
| `internal/auth/domain/user.go` | `User`, `MeProfile`; new `UpdateMeParams` | add `Slug` fields and params struct |
| `internal/auth/domain/repository.go` | `UserRepository` | add `ChangeSlug`, `BackfillMissingSlugs` (no `UpdateSlug`/`FindBySlug`: no consumer until a public lookup endpoint exists) |
| `internal/auth/domain/errors.go` | new | `ErrUserInvalidSlug`, `ErrUserSlugReserved` |
| `internal/auth/infra/user_model.go:11-78` | `userRow`, `toUserDomain`, `toUserRow` | add `Slug *string`, two-way mapping; add `userSlugHistoryRow` |
| `internal/auth/infra/user_slug.go` (new) | `userSlugAvailable`, history lookup | availability callback for users |
| `internal/auth/infra/user_repo.go`, `user_slug.go`, `user_slug_backfill.go` | `CreateWithDB`, `SaveWithDB`; new `ChangeSlug`, `BackfillMissingSlugs` | slug resolution + savepoint retry on create, `Omit("slug")` on save, change with history, batch backfill |
| `internal/auth/application/service.go:178,319,490` | `registerNewPending`, `UpdateMe`, `buildMeProfile` | unchanged create call; new `UpdateMe` flow; map `Slug` |
| `internal/auth/application/service_oauth.go:337` | `isOAuthUniqueViolation` | delegate to `gormx.IsUniqueViolation`, slug-aware ordering |
| `internal/auth/delivery/{dto.go,handler.go}` | `UpdateMeRequest`, `MeResponse`, `PatchMe`, `toMeResponse`, `writeMeAccessError`, `AuthUseCase.UpdateMe` | add `slug`, parse, map 400 |
| `internal/rbac/delivery/me_cache_integration_test.go:25-57` | fake repo | add new methods |
| `internal/shared/constants/{dbschema_name.go,ratelimit.go}` | consts | `TableUserSlugHistory`, `CLIOpBackfillUserSlugs` |
| `internal/appcli/backfill_user_slugs.go`, `main.go:36-46` | new / modify | command and wiring |
| `internal/shared/userpicker/list.go:12-18,59-70` | `Row`, `UserPickerSelectSQL` | add `Slug` / `COALESCE(u.slug,'') AS slug` |
| `internal/course/infra/repo_collaborators.go:70-89,132-137,240-244` | scan row, select, mappers | use `userpicker` projection; add `Slug` |
| `internal/course/infra/repos.go:231-257,525-545` | consts, `courseListScanRow`, `toCourseListItem` | add `courseListOwnerSlugColumn`, `OwnerSlug` |
| `internal/course/infra/{repo_admin.go:15-30,repo_review.go:46-62,repo_catalog.go:12-50}` | selects, scan rows | append owner slug column, map |
| `internal/course/infra/repo_learner.go:61,90-116` | `continueLearningQuery`, scan row, `toDomain` | use shared consts, add `owner_slug` |
| `internal/course/domain/course.go:47-62,94-101,203-209,260-285` | `CourseListItem`, `Collaborator`, `InstructorCandidate`, `TrendingCourseItem`, `ContinueLearningItem` | add `OwnerSlug` / `Slug` |
| `internal/instructor/infra/{repo_catalog.go:17-52,repos_load.go:14-105,repos.go:62-160,repo_roster_list.go:40-100,repo_roster_bulk.go:25-40,158-166}` | queries, scan rows, mappers | add slug column and mapping |
| `internal/instructor/domain/instructor.go:96-175,243-249` | `Application`, `Profile`, `RosterMember`, `PopularInstructor`, `RosterCandidate` | add `Slug` |
| `internal/instructor/delivery/dto.go:149-213,226-322` | response structs and mappers | add `slug` |

### Tests to add or modify
`migrations/user_slug_test.go` (static SQL assertions like `drop_course_collaborators_test.go`); `internal/shared/slug/{errors,input,reserved,suffix}_test.go`; `internal/shared/slug/gormslug/*_test.go`; `internal/shared/gormx/unique_violation_test.go`; merge `course/infra/repo_slug_test.go` and `taxonomy/infra/slug_test.go` assertions into the shared tests; `course/delivery/handler_base_test.go` (`TestMapCourseErrorSlugConflict`) stays green; `auth/infra/user_repo_slug_test.go`; `auth/application/service_slug_test.go`; `auth/application/service_roles_test.go` (`MeProfile.Slug`); `auth/delivery/me_roles_test.go` + new `handler_patch_me_test.go`; `appcli/backfill_user_slugs_test.go`; query-string tests for `popularInstructorsQuery`, `trendingCoursesQuery`, `continueLearningQuery` (extend `repo_catalog_test.go`, `repo_continue_learning_test.go`), `repos_list_scan_test.go` (identity columns assert `slug`).

## Documentation to update (all English; every file verified as English; replace, never append)

| File | Section / lines | Update |
|---|---|---|
| `docs/database.md` | users table ~350-375; migration table ~815-842 | `slug` column, `chk_users_slug_format`, `uix_users_slug_active`, `user_slug_history` section, rows for `000043` and `000044` |
| `migrations/README.md` | chain table (stops at 000034) | add `000043`, `000044`; fill 000035-000042 if missing |
| `docs/api_swagger.yaml` | `MeProfile` ~495-517; `UpdateMeRequest` ~800; `InstructorApplicationMeResponse` ~340; descriptions of `/catalog/courses/trending`, `/catalog/instructors/popular`, `/learner-courses/continue`, `/course-admin/courses`, `/course-reviews/pending`, `/courses/my`, roster and candidate paths | add `slug` / `owner_slug`, PATCH `slug` semantics (omit / null / conflict) |
| `docs/api-dog-import.json` | generated | regenerate with `ruby scripts/generate-apidog-postman.rb` (AGENTS.md omits the `scripts/` prefix) |
| `docs/return_types.md` | ~115,139,156 (me/auth), 470,784 (instructor), 488,503 (`CourseListItem`), 514,523 (trending / continue), 556-565 (popular), 841-880 | add `slug` / `owner_slug` fields and population notes |
| `docs/curl_api.md` | `GET /me` ~568-615; admin / trash / pending ~1988-2034; catalog ~2048-2100 | sample JSON `slug` / `owner_slug`; new `PATCH /me` slug section |
| `docs/modules/auth.md` | `/me`, register, OAuth sections | `slug` in `MeProfile`, generation at create, PATCH `slug`, history |
| `docs/modules/user.md` | whole file (40 lines; notes there is no `internal/user`) | user slug ownership lives in `internal/auth` |
| `docs/modules/instructor.md` | ~55, 254, 280-291 | `slug` on popular / identity / roster; `users.slug` in joined columns |
| `docs/modules/course.md` | 3, 41, 233-259, 322 | `owner_slug`, `Collaborator.slug`, rewrite "slug is course-only" at line 41, audit line |
| `docs/modules/enrollment.md` | continue-learning row | `owner_slug` |
| `docs/modules/taxonomy.md` | 13, 63, 94 | cross-reference shared slug resolver |
| `docs/api-overview.md` | 51, 89, 124 | `/me` slug, `owner_slug` |
| `docs/router.md` | 78-79, 196-206 | `owner_slug`, PATCH `/me` slug row |
| `docs/requirements.md` | FR-2.1 (~57,117,166-170), FR-12 (~467-470) | add slug requirements |
| `docs/security-public-seo-notes.md` | B8 (27), B1 (20), B10 (29) | user slug exposure, `/me` cache, history reservation |
| `docs/patterns.md` | 218-237, 354 | generalize "Slug uniqueness (taxonomy)" to the shared resolver (course, taxonomy, user) |
| `docs/reusable-assets.md` | 578-604, 778-781, cache keys ~850 | rewrite slug assets (no longer "course/taxonomy only"; fix line 586 "future instructor slug"), add `slug.ResolveCreate`, `gormslug.AvailableIn`, `slug.ConflictError`, `gormx.IsUniqueViolation`, `userpicker.Row.Slug` |
| `docs/architecture.md` (162, 215), `docs/modules.md` (183), `docs/data-flow.md` (115-132), `docs/sequence_diagrams.md` (97-102, 1055), `docs/folder-structure.md`, `docs/logic-flow.md` | slug and `/me` flows | update or confirm unchanged; add a user-slug create/change sequence |
| `.context/session-<SESSION_ID>.md` | current session file only | learning summary (git-ignored; other sessions untouched) |
| GitNexus | `npx gitnexus analyze --embeddings` | index has 3348 embeddings; running without the flag deletes them |

A final scan for the stale statements "slug only for course/taxonomy" (`reusable-assets.md:578-604`, `patterns.md:218-237`, `taxonomy.md:13`, `course.md:41`, `security-public-seo-notes.md:27`) must return zero hits.

## Risks / Trade-offs

- [Missed mapping makes `Save` wipe slug] → D3 two-way mapping + round-trip test + targeted updates.
- [OAuth slug collision misreported as identity conflict] → slug-index check before `IsUniqueViolation(err)`; forced-violation test.
- [Refactor touches shipped course/taxonomy write paths that have **no** resolver tests today] → characterization tests first (task 1), `gitnexus_impact` per symbol, `RetryInSavepoint` semantics untouched.
- [`IsUniqueViolation` consolidation changes RBAC/OAuth matching] → variadic no-name form keeps the exact "any unique violation" behavior; table test includes SQLite wording.
- [Alias `SlugConflictError` drops per-feature `Error()` text] → grep for string matches before the change.
- [`courseListScanRow` shared by many queries; one missing column silently scans `""`] → per-query tests asserting `owner_slug`.
- [History race between two users] → `uq_user_slug_history_slug` is the safety net; violation retries like a duplicate.
- [Stale `/me` up to 60 s on other devices] → `delCachedMe` on change; old payload decodes `slug` as `""`.
- [`000044` fails if backfill incomplete] → runbook count check, separate release.
- [`dupl -t 60` flags near-identical new specs/tests] → specs are data (config structs), tests are table-driven.
- [Unlimited slug changes grow history] → accepted, follow-up.
- [`check-arch`: `gormslug` under `shared/**` importing `gormx`] → shared→shared allowed; verified by `make check-architecture`.

## Migration Plan

1. Release 1: `000043` + code. New users get slugs. Operator runs `CLI_BACKFILL_USER_SLUGS=1 go run .` per environment, then `ANALYZE users`, then verifies `SELECT count(*) FROM users WHERE slug IS NULL` = 0.
2. Release 2: `000044`.
3. Rollback: release 1 is additive; `000043.down` removes every new object; `000044.down` drops `NOT NULL`.

## Confirmed product decisions

- Reserved-word list in D5 approved as written.
- Hard delete releases former slugs (`ON DELETE CASCADE`); soft delete keeps them reserved.
- No rate limit on slug changes in this change (follow-up).
- Slug never follows `display_name` changes; it changes only through an explicit `PATCH /me` slug.
