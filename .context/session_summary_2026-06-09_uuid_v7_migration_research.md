# Session summary — UUID v7 + ULID migration research (2026-06-09)

## Scope of discovery

Discovery-only phase for UUID v7 + ULID migration in `be-mycourse`:

- Reviewed `.context` history (auth/course/instructor/int64 audits)
- Reviewed BE docs and migration docs
- Reviewed git baseline and branch delta
- Ran GitNexus research via subagent + main thread (`query`, `context`, `impact`)
- Cross-checked source files for shared ID parsing/JWT/current user identity

No code, migration, or docs implementation edits were made in this discovery step.

## Git baseline

- Current branch: `fix/course-version-row-version-backfill`
- Base branch in this repo: `master` (not `main`)
- `git diff master...HEAD --stat` is large because this branch already contains broad refactors and doc sync before this UUID task.

## GitNexus status

- Repo: `be-mycourse`
- Index status: **STALE** (indexed commit behind current HEAD)
- `gitnexus analyze` refresh is required before close-out gate.

## Symbols reuse vs change

| Category | Symbols | Notes |
|---|---|---|
| Reuse | `Claims`, `GenerateAccess`, `ParseUintPathParam`, `ParseUintParam`, `CurrentUserID` | Shared anchors already used across many handlers. Should be extended, not duplicated. |
| Change | `ParseUintPathParam` family | Add UUID parser path without removing numeric parsing immediately where RBAC still numeric. |
| Change | `Claims.UserID` usage path | Migration target is string UUID IDs; current is `uint`. Requires coordinated middleware + delivery updates. |
| Reuse | `uuid.NewV7()` pattern from auth register | Existing v7 generation exists and should be centralized for entity IDs. |
| Change | media/course helper generators using `uuid.NewString()` | Upgrade entity-id generation to v7 where required by migration scope. |

## Symbol-level impact (upstream)

| Symbol | Risk | Blast radius summary |
|---|---|---|
| `CurrentUserID` | **CRITICAL** | Broad authenticated handler dependency surface (auth/course/instructor/taxonomy). |
| `ParseUintPathParam` / `ParseUintParam` | **CRITICAL** | Directly consumed by many route handlers and helper wrappers. |
| `Claims` | LOW→HIGH (semantic) | Struct itself has limited direct callers but claim type change affects middleware contract globally. |

## Process findings

Key execution flows confirmed during discovery:

1. Auth login/refresh path derives user context from JWT `user_id` numeric claim.
2. Course/instructor/taxonomy HTTP handlers parse numeric path params (`ParseUintPathParam`).
3. Media entity IDs are string UUID-based, but generation currently uses v4-style `uuid.NewString()` in several places.
4. RBAC role IDs remain numeric and must stay numeric per migration scope.

## Docs gap list (docs vs UUID-v7 target)

Major mismatches discovered against migration target:

1. `docs/database.md`
   - `users.id` documented as `BIGSERIAL`; target is `UUID` v7.
   - Many domain tables documented with numeric PK/FK (`BIGSERIAL`/`BIGINT`) while target is UUID FKs.
   - `user_roles.user_id` and `user_permissions.user_id` still documented as `BIGINT`.
2. `docs/return_types.md`
   - Many shapes still use numeric `id`/`user_id` in examples and type snippets.
3. `docs/api_swagger.yaml`
   - Multiple path params (`courseId`, taxonomy IDs, instructor IDs, etc.) are typed as integer.
4. `docs/curl_api.md`
   - Examples assume numeric IDs for entity routes.
5. `docs/modules.md` + module docs
   - Mixed assumptions on numeric identifiers across bounded contexts.
6. `migrations/README.md`
   - Current chain documents numeric PK strategy, not UUID-v7+ULID target model.

## PK/FK migration matrix (target)

| Area | Current | Target |
|---|---|---|
| `users.id` | BIGSERIAL | UUID v7 PK |
| `users.user_code` | UUID | ULID (`CHAR(26)`) |
| RBAC `roles.id` | BIGSERIAL | Keep numeric (unchanged) |
| RBAC `permissions.permission_id` | VARCHAR | Keep unchanged |
| RBAC junctions (`user_roles`, `user_permissions`) | `user_id BIGINT` | `user_id UUID` FK to `users.id` |
| Taxonomy tables | BIGSERIAL PK + BIGINT FKs | UUID v7 PK/FK |
| Instructor tables | BIGSERIAL PK + BIGINT FKs | UUID v7 PK/FK |
| Course tables | BIGSERIAL PK + BIGINT FKs | UUID v7 PK/FK |
| Media (entity IDs) | UUID/v4 generation style | UUID v7 generation style |
| `system_privileged_users` | BIGSERIAL | UUID v7 PK |

## Planned touch scope for implementation phases

- Migrations `000001`…`000020` rewritten in place for UUID-v7 schema target (RBAC exceptions preserved)
- Shared parsing/JWT identity helpers in `internal/shared`
- Domain/infra/delivery IDs in auth/rbac/taxonomy/instructor/media/course
- App CLI import path for legacy data restore mapping (`old bigint -> new uuid`)
- Docs + Swagger + curl examples + Postman regeneration in close-out

## Notes from source baseline cross-check

- `internal/shared/utils/params.go`: numeric path parser only.
- `internal/shared/utils/requestutil.go`: `CurrentUserID` returns `uint`.
- `internal/shared/token/jwt.go`: `Claims.UserID` is `uint`; `UserCode` string already exists.
- `internal/shared/middleware/auth_jwt.go`: stores numeric `ContextUserID` in request context.
- `internal/auth/application/service.go`: currently uses `uuid.NewV7()` for `user_code` only.
- `internal/media/infra/helpers.go`: local UUID helper uses `uuid.NewString()`.

## Discovery pass checklist

- [x] `.context` and required docs reviewed
- [x] git baseline reviewed (`log`, `diff` against `master`)
- [x] GitNexus subagent report reviewed
- [x] Main-thread `query` + `context` + `impact` executed
- [x] Research note and PK/FK target matrix recorded
- [x] Source baseline cross-checked
- [x] No implementation code change done during discovery stage
