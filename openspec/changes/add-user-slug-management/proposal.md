## Why

`courses.slug` (`rework-course-slug-management`) and the taxonomy slugs (`unify-taxonomy-slug-management`) are managed by one consistent contract. Users have no slug at all, so a public instructor page, "become instructor" flow or course card can only reference a person by UUID (`user_id`). The product decision is to give **every user** a slug (not an instructor-only slug), so any later feature (instructor profile page, SEO links, share links) can reuse it. This change builds the data and write path only; no new public lookup endpoint is added here.

While researching, the slug resolution code was found duplicated between `internal/course/infra` and `internal/taxonomy/infra` (~150 lines: availability check, suffix candidate builder, duplicate-key detector, create/update resolvers, `SlugConflictError`, 409 mapping, manual-input parsing). A third copy for users is forbidden by the project's Dedup rule, so the shared capability is extracted first and course, taxonomy and user all consume it.

## What Changes

- **New `users.slug`** column: auto-generated at every user creation (register and OAuth) from `display_name` via the existing `GenerateAutoSlugBase` (gosimple/slug); an empty or untransliterable name falls back to `user-{randomSuffix}`, a collision appends a random suffix (7 chars x7 attempts, then 8, 9, ...) via the existing `RetryWithSuffix`. Format is the existing `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`, max 255.
- **User may change their own slug** through `PATCH /api/v1/me` (new optional `slug`, `validate.Optional[string]` omit-vs-null semantics, same as course/taxonomy update). A conflicting value is silently auto-suffixed (same decision as course/taxonomy *update*). Old slugs are kept forever in a new `user_slug_history` table and nobody else can claim them.
- **Reserved words** (e.g. `admin`, `me`, `api`, `instructors`) are rejected for manual slugs and never produced by auto-generation (a reserved base is suffixed).
- **Backfill** existing users from `display_name` using the same Go resolver through a new batch CLI command (SQL cannot transliterate); two-step rollout: `000043` adds the nullable column plus constraints that tolerate `NULL`, `000044` (a later release, after the backfill has run) sets `NOT NULL`.
- **Expose `slug` (additive, `user_id` retained)** in `GET/PATCH /me`, popular instructors, instructor application/profile/roster responses, and `owner_slug` in course list/trending/continue-learning/collaborator responses.
- **Refactor (behavior-preserving)**: lift the duplicated slug resolution out of course and taxonomy into shared modules; both features and users call it. Also consolidated: the four unique-violation detectors (course, taxonomy, OAuth, RBAC) into one `gormx.IsUniqueViolation`, the duplicate manual-slug parsing and 409 mapping, the course owner-name SQL fragments, and the user+avatar projection shared by `userpicker` and course collaborators. Capabilities that look similar but differ (media owner select, instructor admin identity selects, `/me` invalidation ports, tree-node in-memory uniqueness, digit generator) are intentionally kept separate; reasons are in `design.md`. No course/taxonomy requirement changes.
- **Out of scope**: public "get user/instructor by slug" endpoint, slug-availability endpoint, redirect-by-history endpoint, dropping `user_id` from public responses (breaking, deferred to the frontend), rate limiting of slug changes.

## Capabilities

### New Capabilities
- `user/slug-management`: format, auto-generation at creation, self-service change via `/me`, conflict handling, reserved words, permanent history reservation, never-null-after-rollout guarantee, and read exposure of the slug.

### Modified Capabilities
(none — the course/taxonomy refactor preserves their existing requirements verbatim)

## Impact

- **Code**: new `internal/shared/slug` additions (`ConflictError`, `BuildSuffixedCandidate`, `ParseManual`/`ParseUpdateInput`, `Reserved`, `KindUser`), new `internal/shared/slug/gormslug`; `internal/course/{infra,domain,delivery}`, `internal/taxonomy/{infra,application,domain,delivery}`, `internal/shared/taxonomy/tree_write.go` switch to them; `internal/auth/{domain,infra,application,delivery}` and `internal/server/wire_auth_oauth.go` gain slug generation/update; `internal/instructor/{infra,domain,delivery}` and `internal/course/{infra,domain}` gain read fields; new `internal/appcli/backfill_user_slugs.go`, `main.go`, `internal/shared/constants/ratelimit.go`.
- **DB**: `000043` (nullable `users.slug`, `chk_users_slug_format`, `uix_users_slug_active`, new `user_slug_history`), `000044` (`SET NOT NULL`, later release).
- **API**: `PATCH /me` request gains `slug`; `slug`/`owner_slug` added to responses listed in `design.md`; no new routes; `user_id` kept.
- **Docs**: full list in `design.md` (Documentation) — written in English.
- **Risks**: `Save` round-trip can wipe the slug if mapping is missed; OAuth `isOAuthUniqueViolation` would misreport a slug collision as an identity conflict; `courseListScanRow` is shared so every list query must select the new column; `/me` cache must be invalidated.
