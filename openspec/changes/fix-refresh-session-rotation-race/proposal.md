## Why

`POST /api/v1/auth/refresh` reads the session map, checks the refresh UUID in Go (`service_session.go:35`), then writes the new UUID with an unconditional `jsonb_set(..., true)` (`internal/auth/infra/user_repo.go:205`). Two overlapping refreshes carrying the same refresh token can both pass the check; the later write wins in the database while the browser may keep the tokens of the other request. The next refresh then sends a UUID the database no longer holds and gets 401 `InvalidSession`, the FE clears its cookies, and a healthy user is logged out.

The same lost-update class exists in every other writer that rewrites the whole `users.refresh_token_session` map from a stale read: login (`AddSession`), logout (`RemoveSession` + `SaveSessions`), and the GORM full-row user save (`SaveWithDB`, used by register-resend, email confirmation and OAuth identity link). A rotation done by one device can therefore be erased by a login, logout or OAuth link on another device, with the same user-visible symptom.

Source: FE handoff `Handoff - fe-mycourse to be-mycourse (rotateSession compare-and-swap)` (FE change `fix-stale-auth-session-cookies`). The race was found by reading code and is not yet reproduced by a test. The handoff's SQL names the table `app_users`; the real table is `users` (`constants.TableAppUsers`).

## What Changes

1. **Conditional rotation (core fix).** Add `GormRefreshSessionRepository.RotateSession`: one `UPDATE` that replaces the session entry only if the stored `refresh_token_uuid` equals the UUID that `RefreshSession` verified, with `jsonb_set(..., create_missing=false)` so a revoked session is never re-created. 0 rows updated returns `domain.ErrInvalidSession`, which the existing handler maps to 401 `InvalidSession`.
2. **Replace, do not duplicate.** `rotateSession` calls `RotateSession`; the unconditional `SaveSession` is deleted (its only caller was `rotateSession`). The application-local `sessionRepo` interface swaps `SaveSession` for `RotateSession`.
3. **Login-like flows lock the user row.** `AddSession` appends `FOR UPDATE` to its `SELECT` (inside its existing transaction) and adds `AND deleted_at IS NULL` to its `UPDATE`. Reached by `POST /auth/login`, `POST /auth/confirm` and all OAuth login endpoints through `issueTokenPair`.
4. **Logout deletes atomically.** `RemoveSession` becomes one `UPDATE ... refresh_token_session - key` statement (missing key or user stays a no-op). `SaveSessions`, whose only caller was `RemoveSession`, is deleted from the Gorm repository and from `domain.RefreshSessionRepository`.
5. **Full-row user saves stop touching sessions.** `GormUserRepository.SaveWithDB` changes `Omit("slug")` to `Omit("slug", "refresh_token_session")` and its comment explains why. No code mutates `User.RefreshTokenSession` before a save (only `RefreshSession` reads it), so no intended behavior changes. `CreateWithDB` is untouched, so new rows still get the `'{}'` default.
6. **Tests, documentation and the API contract note** as listed under Impact.
7. **Not changed:** token and cookie formats, `MaxActiveSessions` eviction rule, request/response shapes, status codes and error codes of every endpoint, `RequireActiveUser`, Redis `/me` cache, any migration.

## Capabilities

### New Capabilities
- `auth/session-rotation`: single-winner refresh-token rotation and lost-update-free persistence of the per-user session map (rotate, add, remove, full-row user save).

### Modified Capabilities
(none: `openspec/specs/` has no archived auth capability yet, so this is additive only)

## Impact

### Modules
Only `internal/auth` (`domain`, `application`, `infra`). Wiring files `internal/server/wire_core.go`, `wire_auth_confirm.go`, `wire_auth_oauth.go` are read-only for this change: they construct `GormRefreshSessionRepository` / call `SaveWithDB` and need no edit. No change in `rbac`, `course`, `taxonomy`, `media`, `instructor`, `authorization`, `internal/shared/*`, middleware, routes or delivery.

### Code (exact; details and snippets in `design.md`)

| File | Symbol | Action |
|---|---|---|
| `internal/auth/infra/user_repo.go` | `RotateSession` | **Add** (new method, 4 new SQL consts near the top) |
| `internal/auth/infra/user_repo.go:158-183` | `AddSession` | **Modify** (`FOR UPDATE`, `deleted_at IS NULL`) |
| `internal/auth/infra/user_repo.go:185-198` | `RemoveSession` | **Modify** (single atomic statement) |
| `internal/auth/infra/user_repo.go:149-156` | `SaveSessions` | **Delete** |
| `internal/auth/infra/user_repo.go:200-207` | `SaveSession` | **Delete** |
| `internal/auth/infra/user_repo.go:98-107` | `SaveWithDB` | **Modify** (`Omit("slug","refresh_token_session")` + comment) |
| `internal/auth/application/service_session.go:100` | `rotateSession` | **Modify** (call `RotateSession`, pass `entry.RefreshTokenUUID`) |
| `internal/auth/application/service.go:79-85` | `sessionRepo` interface + comment | **Modify** (`SaveSession` -> `RotateSession`) |
| `internal/auth/domain/repository.go:32-35` | `RefreshSessionRepository` | **Modify** (delete `SaveSessions`, reword `RemoveSession` doc) |
| `internal/auth/application/service_session_test.go` | session fakes + rotation tests | **Add** (new file) |
| `internal/auth/infra/user_repo_session_test.go` | SQL-shape tests | **Add** (new file) |
| `internal/auth/delivery/handler_refresh_test.go` | refresh status-mapping test | **Add** (new file) |

Unchanged on purpose: `LoadSessions`, `Logout`, `RefreshSession`, `issueTokenPair`, `pickOldestSessionKey`, `MaxActiveSessions`, `toUserRow`, `CreateWithDB`, `IncrementEmailSendCount`, `delivery/handler.go`, `routes.go`.

### APIs (all reused; none created, none with a contract change)

| Endpoint | Treatment |
|---|---|
| `POST /api/v1/auth/refresh` | **Reused and extended**: same request/response and codes; only its persistence step becomes conditional. Swagger gains the already-existing 400/401/403 responses (documentation only). |
| `POST /api/v1/auth/logout` | **Reused as is**; internal `RemoveSession` becomes atomic. |
| `POST /api/v1/auth/login`, `POST /api/v1/auth/confirm`, `POST /api/v1/auth/{google,google/onetap,google/mobile,x,discord}` | **Reused as is**; internal `AddSession` takes a row lock. |
| `POST /api/v1/auth/register` (resend path) and the OAuth identity-link path | **Reused as is**; internal full-row save no longer writes sessions. |

### Database
No table, column, index, constraint, trigger, seed or migration is added, changed or removed. `users.refresh_token_session` (`JSONB NOT NULL DEFAULT '{}'`, `migrations/000001_schema.up.sql:38`) keeps its shape. SQL text changes only: a conditional `UPDATE`, a `FOR UPDATE` lock, a `jsonb - text` delete, and one column omitted from the GORM full-row `UPDATE`. Rollback is a plain revert.

### Documentation (all English; replace outdated text, never append)

| File | Entry | What changes |
|---|---|---|
| `docs/DATA-MODEL.md` | line 280 `GORM model` paragraph | full-row saves also omit `refresh_token_session` |
| `docs/DATA-MODEL.md` | lines 296-301 session rules table | rewrite add/rotate rows, add remove row and a concurrency rule row |
| `docs/FEATURES.md` | `## auth` Rules line 22 | session eviction + rotation + logout + full-row-save rule |
| `docs/FEATURES.md` | `## auth` Flow line 31 | refresh wording |
| `docs/FEATURES.md` | refresh diagram line 114 | `RotateSession` + 0-row branch -> 401 4007 |
| `docs/FEATURES.md` | logout diagram line 134 | atomic key delete |
| `docs/ARCHITECTURE.md` | error table line 307 | meaning of 4007 |
| `docs/SHARED.md` | lines 129-137 `issueTokenPair / RefreshSession` asset | replace stale, non-existent paths with the real ones and name `RotateSession` |
| `docs/ALL-SPRINTS.md` | In progress table | new row |
| `docs/api_swagger.yaml` | `/api/v1/auth/refresh` responses (line ~1352) | add 400, 401, 403 |
| `docs/api-dog-import.json` | generated | regenerate with `ruby scripts/generate-apidog-postman.rb`, never hand-edit |
| `AGENTS.md`, `CLAUDE.md` | GitNexus index stats line | changed only by the mandatory `npx gitnexus analyze`; no rule text is edited |

Checked and not changed: `docs/FEATURES.md` line 29 `API:` (endpoint paths unchanged), `docs/FEATURES.md` line 142 `DB:`, `docs/DATA-MODEL.md` lines 271 and 282-294 (column and JSONB shape unchanged) and 860, `docs/deploy.md`, `docs/docker.md`, `docs/security-public-seo-notes.md`, `docs/observability/*`, dated handoff notes.

### Reuse and duplication
Every candidate was read first; see the table in `design.md`. Nothing is duplicated: no existing helper covers row locking of this column, JSONB key compare/delete, or CAS on a JSONB sub-key; `gormx.OptimisticUpdate` has different semantics (integer `row_version`) and is reused only as an idiom. Pre-existing hand-written copies of `OptimisticUpdate` in `media` and `course` are unrelated to sessions and are recorded as a follow-up, not touched.

### Risks
- Strict compare-and-swap means a client retrying with an already-rotated token (for example after a lost response) now gets 401 instead of a second success. Accepted (no grace window).
- SQL is verified by statement-shape tests and an optional live two-request smoke test; there is no Postgres integration harness in the repo.
- Research limitation: the `.context/` scan by the research subagent was grep-based, not a full read of all session files. GitNexus `impact` returned UNKNOWN for the Go methods; callers came from `context` and grep.
