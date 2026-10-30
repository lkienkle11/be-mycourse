## Context

See `proposal.md` (Why, Impact) and `specs/auth/session-rotation/spec.md` (behavior). Current state, verified by reading the code (line numbers are from branch `docs/restructure-docs-and-shrink-agents-md` at `c1e707a`):

- Sessions live in `users.refresh_token_session` (`JSONB NOT NULL DEFAULT '{}'`, `migrations/000001_schema.up.sql:38`). Table constant: `constants.TableAppUsers = "users"` (`internal/shared/constants/dbschema_name.go:51`). No index or trigger on the column.
- **Refresh**: `RefreshSession` (`internal/auth/application/service_session.go:15-52`) calls `userRepo.FindByID`, runs `checkUserAccessible`, compares `entry.RefreshTokenUUID != refreshClaims.UUID` in Go (line 35), checks expiry, then `rotateSession` (line 80-110) generates tokens and calls `sessionRepo.SaveSession` (line 100): unconditional `jsonb_set(..., true)` (`internal/auth/infra/user_repo.go:205`). Sole caller of `SaveSession`.
- **Other writers of the column**, none conditional or locked: `AddSession` (`user_repo.go:158-183`, plain `SELECT`, whole-map `UPDATE` without `deleted_at IS NULL`, sole caller `issueTokenPair` at `service_session.go:136`, reached from `Login` `service.go:148`, `ConfirmEmail` `service.go:292`, `LoginOrCreateFromExternal` `service_oauth.go:138`); `RemoveSession` (`user_repo.go:185-198`, `LoadSessions` then `SaveSessions`; sole caller `Logout` `service_session.go:73`); `SaveSessions` (`user_repo.go:149-156`, sole caller `RemoveSession`); GORM full-row `SaveWithDB` (`user_repo.go:99-107`, `Omit("slug").Save(row)`; callers `Save` <- `registerResendPending` `service.go:217`, `ConfirmEmailWithLearnerRole` `internal/server/wire_auth_confirm.go:35`, `LinkIdentityAndUpdateUser` `internal/server/wire_auth_oauth.go:94`). `toUserRow` copies `User.RefreshTokenSession` when non-nil (`user_model.go:77-79`), so a stale or nil map is written back.
- **Delivery** already maps `domain.ErrInvalidSession` to 401 `InvalidSession` and `ErrRefreshTokenExpired` to 401 `RefreshTokenExpired` (`internal/auth/delivery/handler.go:169-173`). No delivery change.
- **Interfaces**: `domain.RefreshSessionRepository` (`internal/auth/domain/repository.go:28-38`: `LoadSessions`, `SaveSessions`, `RemoveSession`, `IncrementEmailSendCount`); application-local `sessionRepo` (`internal/auth/application/service.go:79-85`) embeds it and adds `AddSession`, `SaveSession`. Only implementation: `GormRefreshSessionRepository`, wired at `internal/server/wire_core.go:69`. No fake of either interface exists in any `*_test.go`.
- GitNexus (index at `c1e707a`, fresh): `impact` answered UNKNOWN for every Go method; callers above come from `context` and grep. No HIGH or CRITICAL result.

## Goals / Non-Goals

**Goals:**
- Exactly one of N concurrent refreshes with the same token succeeds; the others get `ErrInvalidSession`.
- No other session writer can overwrite a concurrently rotated entry.
- No schema change, no new endpoint, no new shared package, no duplicated or dead code.

**Non-Goals:**
- Grace window or reuse detection for old refresh tokens (decided: strict compare-and-swap).
- Refactoring the hand-written optimistic-lock copies in `media` and `course` (follow-up only).
- Changing token or cookie formats, `MaxActiveSessions`, `RequireActiveUser`, the Redis `/me` cache, or returning the session expiry to the FE.
- A Postgres integration-test harness (the repo has none).

## Decisions

### D1. One conditional `UPDATE` instead of transaction + `SELECT ... FOR UPDATE`
```sql
UPDATE users
SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, false),
    updated_at = ?
WHERE id = ? AND deleted_at IS NULL
  AND refresh_token_session -> ?::text ->> 'refresh_token_uuid' = ?
```
Bind order: `sessionStr`, entry JSON, `timex.NowUnix()`, `userID`, `sessionStr`, `expectedUUID`.
- `create_missing=false`: a session removed meanwhile (logout) is not re-created.
- Under READ COMMITTED a concurrent second `UPDATE` waits for the row lock, re-evaluates the `WHERE` on the new row version, sees a different UUID and updates 0 rows: the write is the arbiter, no explicit lock needed.
- `?::text` avoids overload ambiguity (`jsonb -> text` vs `jsonb -> int`) for an untyped bind parameter. The jsonb `?` operator is never used because `?` is the GORM placeholder.
- Rejected: transaction + `FOR UPDATE` + write (more code and a longer lock for no extra correctness here).

### D2. `RotateSession` returns `domain.ErrInvalidSession` on 0 rows
Signature: `RotateSession(ctx context.Context, userID, sessionStr, expectedUUID string, entry domain.RefreshSessionEntry) error`. The repository returns the sentinel itself, the same idiom as `gormx.OptimisticUpdate` returning its `lockErr`; `rotateSession` only propagates it. This is smaller than the `(bool, error)` shape in the FE handoff (no mapping branch). 0 rows covers rotated, revoked and user-gone, which are one outcome (401 `InvalidSession`). `expectedUUID` is `entry.RefreshTokenUUID` of the entry `RefreshSession` already verified.

### D3. `AddSession` locks the user row
Same transaction, two SQL edits: the `SELECT` gets `FOR UPDATE`; the `UPDATE` gets `AND deleted_at IS NULL` (every sibling write filters it; `AddSession` and `SaveSessions` were the exceptions). A concurrent rotate then queues behind the lock instead of being overwritten by the whole-map write. Raw SQL stays raw, like the rest of the file.

### D4. `RemoveSession` is one atomic statement
```sql
UPDATE users
SET refresh_token_session = refresh_token_session - ?::text, updated_at = ?
WHERE id = ? AND deleted_at IS NULL AND refresh_token_session -> ?::text IS NOT NULL
```
Bind order: `sessionStr`, `timex.NowUnix()`, `userID`, `sessionStr`. A missing key or user matches 0 rows and returns nil (idempotent logout). `Logout` is unchanged: it still loads the map to compare the UUID in Go; a logout racing a rotation may delete the rotated entry, which is the intended outcome (the user asked to leave). The delete is deliberately not conditioned on the UUID.

### D5. `SaveSessions` and `SaveSession` are deleted
After D2 and D4 neither has a caller. They are removed from the Gorm repository; `SaveSessions` is also removed from `domain.RefreshSessionRepository`. `LoadSessions` stays (used by `Logout`).

### D6. `SaveWithDB` omits `refresh_token_session`
`Omit("slug", "refresh_token_session")`. Nothing mutates `User.RefreshTokenSession` before a save (grep: only `service_session.go:35` reads it), so no intended behavior changes; the silent overwrite disappears. `CreateWithDB` is untouched (new users still get `'{}'`), and `toUserRow` stays as is because `CreateWithDB` needs the field.

### D7. No grace window
A client retrying with an already rotated token (for example after a lost response) now gets 401. Accepted by the product owner. A later grace window would add `previous_refresh_token_uuid` and `rotated_at` inside the JSONB (still no migration) at a security cost.

### D8. SQL text lives in named constants so tests can assert it without a database
`AddSession` runs in `db.Transaction`, which needs a live connection, so its statements cannot be captured with a GORM dry-run handle. The four new or changed statements become unexported constants in `user_repo.go` (concatenated with `constants.TableAppUsers`, a constant), asserted by plain string tests; `SaveWithDB` is asserted through a dry-run handle plus an `Update` callback (the pattern of `internal/shared/gormx/hard_delete_test.go`). Unchanged statements (`LoadSessions`, `IncrementEmailSendCount`) keep their inline `fmt.Sprintf`.

## Change inventory (exact)

### `internal/auth/infra/user_repo.go`
Add after the import block (new consts; names final):
```go
const (
	sqlRotateSession = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = jsonb_set(refresh_token_session, ARRAY[?], ?::jsonb, false), updated_at = ?" +
		" WHERE id = ? AND deleted_at IS NULL AND refresh_token_session -> ?::text ->> 'refresh_token_uuid' = ?"
	sqlSelectSessionsForUpdate = "SELECT refresh_token_session FROM " + constants.TableAppUsers +
		" WHERE id = ? AND deleted_at IS NULL FOR UPDATE"
	sqlStoreSessions = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = ?::jsonb, updated_at = ? WHERE id = ? AND deleted_at IS NULL"
	sqlRemoveSession = "UPDATE " + constants.TableAppUsers +
		" SET refresh_token_session = refresh_token_session - ?::text, updated_at = ?" +
		" WHERE id = ? AND deleted_at IS NULL AND refresh_token_session -> ?::text IS NOT NULL"
)
```
- **Modify `AddSession`** (lines 158-183): replace the two `fmt.Sprintf` queries (`q` line 163, `uq` line 180) by `sqlSelectSessionsForUpdate` and `sqlStoreSessions`; the eviction logic (lines 167-174), `json.Marshal` and `MaxActiveSessions` are unchanged.
- **Modify `RemoveSession`** (lines 185-198) to:
  ```go
  func (r *GormRefreshSessionRepository) RemoveSession(ctx context.Context, userID string, sessionStr string) error {
  	return r.db.WithContext(ctx).Exec(sqlRemoveSession, sessionStr, timex.NowUnix(), userID, sessionStr).Error
  }
  ```
- **Delete `SaveSessions`** (lines 149-156) and **`SaveSession`** (lines 200-207).
- **Add `RotateSession`** where `SaveSession` was:
  ```go
  // RotateSession replaces the session entry only while its stored refresh_token_uuid equals
  // expectedUUID. It returns domain.ErrInvalidSession when no row matched (the session was
  // rotated, revoked or the user is gone), so exactly one of several concurrent refreshes wins.
  func (r *GormRefreshSessionRepository) RotateSession(ctx context.Context, userID, sessionStr, expectedUUID string, entry domain.RefreshSessionEntry) error {
  	data, err := json.Marshal(entry)
  	if err != nil {
  		return err
  	}
  	res := r.db.WithContext(ctx).Exec(sqlRotateSession, sessionStr, string(data), timex.NowUnix(), userID, sessionStr, expectedUUID)
  	if res.Error != nil {
  		return res.Error
  	}
  	if res.RowsAffected == 0 {
  		return domain.ErrInvalidSession
  	}
  	return nil
  }
  ```
- **Modify `SaveWithDB`** (lines 99-107): `Omit("slug", "refresh_token_session").Save(row)`; comment becomes: slug is owned by `CreateWithDB`, `ChangeSlug` and `BackfillMissingSlugs`, and `refresh_token_session` by `GormRefreshSessionRepository`; a full-row save never writes either, so a stale in-memory user can not revert a slug changed meanwhile or erase a session rotated meanwhile.
- The `fmt` import stays (still used by `LoadSessions`, `IncrementEmailSendCount`, user queries); verify at build.

### `internal/auth/application/service_session.go`
- `rotateSession` line 100 becomes `if err := s.sessionRepo.RotateSession(ctx, user.ID, sessionStr, entry.RefreshTokenUUID, updatedEntry); err != nil {`. The return of the error is unchanged (no tokens are returned on error). Nothing else in the file changes.

### `internal/auth/application/service.go` (lines 79-85)
- Comment: `(AddSession, RotateSession beyond the domain interface)`. Interface: replace `SaveSession(...)` by `RotateSession(ctx context.Context, userID, sessionStr, expectedUUID string, entry domain.RefreshSessionEntry) error`.

### `internal/auth/domain/repository.go` (lines 28-38)
- Delete the `SaveSessions` comment and method (lines 32-33). Reword the `RemoveSession` comment to: `RemoveSession deletes one session key from the user's refresh_token_session JSONB in one atomic statement; a missing key or user is a no-op.`

### New test files
See "Test plan".

### Wiring, delivery, routes, model, migrations, domain types
No change (verified).

## Reuse inventory (no duplicate code)

| Capability | Existing code read | Verdict |
|---|---|---|
| Version-based optimistic update | `gormx.OptimisticUpdate` (`internal/shared/gormx/optimistic.go:11-35`), used by `taxonomy/infra/repos.go:521`, `course/infra/repo_helpers.go:153` | Different semantics: needs an integer `row_version` and a column map; here the guard is a JSONB sub-key and `users` has no `row_version` (adding one touches every user write). Not reused; only its "0 rows => sentinel error" idiom is followed (D2). |
| Hand-written copies of that helper | `media/infra/repos.go:283-310`, `course/infra/repo_outline.go:230-242`, `course/infra/repo_instructor.go:209` | Pre-existing duplicates of `OptimisticUpdate`, no relation to sessions. Out of scope: listed as a follow-up in the final response, not touched. |
| Row lock `FOR UPDATE` | `clause.Locking` in `ChangeSlug` (`internal/auth/infra/user_slug.go:85`, GORM model, selects `id, slug`) | Same concept, different shape: `AddSession` scans one JSONB column through raw SQL. Two call sites in one package do not justify a new `gormx` helper; D3 extends the existing raw SQL. |
| JSONB set / delete / compare | none anywhere in Go (only the current `jsonb_set`) | Nothing to reuse; the session repository is the single owner of this column. |
| Savepoint retry | `gormx.RetryInSavepoint` | Unique-key race helper, not applicable. |
| Session types and helpers | `domain.RefreshSessionEntry`, `RefreshTokenSessionMap`, `RefreshTokenSessionMap` Value/Scan (`gormjsonb.go`), `pickOldestSessionKey`, `MaxActiveSessions` | Reused unchanged. |
| Error mapping | `domain.ErrInvalidSession`, handler `RefreshToken` | Reused unchanged. |
| Test helpers | `slugTestUserRepo` pattern (`service_slug_test.go:13-41`), `roleTestPermissionReader` / `roleTestNameReader` (`service_roles_test.go`), `patchMeUseCase` embed pattern (`handler_patch_me_test.go:17-23`), dry-run capture (`gormx/hard_delete_test.go`) | Patterns followed; new test types are added only where no equivalent exists (one session-repo fake, one refresh use-case stub). `make check-dupl` guards against copies. |

## Test plan

| Test | File | Asserts |
|---|---|---|
| `TestRefreshSessionRotatesOnce` | `internal/auth/application/service_session_test.go` | matching UUID returns a token pair; the refresh token's UUID equals the UUID the fake stored; `session_id` unchanged; `remember_me` kept |
| `TestRefreshSessionStaleTokenAfterRotationFails` | same | fake `RotateSession` implements compare-and-swap in memory; a second call with the old token returns `domain.ErrInvalidSession` and an empty `TokenPairResult` |
| `TestRefreshSessionPassesVerifiedUUIDToRotate` | same | fake records `expectedUUID == entry.RefreshTokenUUID` of the stored entry, not the new one |
| `TestRefreshSessionPropagatesLostRace` | same | fake `RotateSession` returns `ErrInvalidSession`: `RefreshSession` returns it unchanged and no tokens |
| `TestRefreshSessionOtherErrorsUnchanged` | same | fake returns a generic error: it is returned as is (maps to 500, not 401) |
| `TestSessionSQLShape` | `internal/auth/infra/user_repo_session_test.go` | `sqlRotateSession` contains `jsonb_set(`, `, false)`, `->> 'refresh_token_uuid' = ?`, `deleted_at IS NULL`; `sqlSelectSessionsForUpdate` ends with `FOR UPDATE`; `sqlStoreSessions` has `deleted_at IS NULL`; `sqlRemoveSession` has `- ?::text` and no `jsonb_set` |
| `TestSaveWithDBOmitsSessionAndSlug` | same | dry-run handle plus `Update` callback: captured `UPDATE` does not contain `refresh_token_session` or `slug`, still contains `display_name`/`updated_at` |
| `TestRefreshTokenLostRaceIs401InvalidSession` | `internal/auth/delivery/handler_refresh_test.go` | stub `AuthUseCase` returning `domain.ErrInvalidSession`: HTTP 401, body code for `InvalidSession`; a generic error gives 500; missing headers give 400 |

Notes: tests set and restore `setting.AppSetting.JWTSecret` (global; no `t.Parallel()`); the user in `FindByID` must pass `checkUserAccessible` (not disabled, not banned, `DeletedAt` nil); permissions need no fake because `permissionSlice` returns `nil, nil` when `permReader` is nil. `Logout` is not tested here: it is unchanged, and `delCachedMe` is safe with a nil Redis client (`service_cache.go:52-56`). Not testable without a real Postgres and therefore covered only by the optional live smoke test: true concurrency of two refreshes and the lock in `AddSession`.

## Documentation change plan (exact, English)

1. **`docs/DATA-MODEL.md` line 280** becomes: `**GORM model:** `internal/auth/infra/user_model.go` (`userRow`). Full-row saves (`GormUserRepository.SaveWithDB`) omit the `slug` and `refresh_token_session` columns: `slug` is written only by user create, `ChangeSlug` and the backfill (`internal/auth/infra/user_slug.go`, `user_slug_backfill.go`); `refresh_token_session` only by user create (default `'{}'`) and by `GormRefreshSessionRepository`.`
2. **`docs/DATA-MODEL.md` lines 296-301** table becomes (keep the first row):
   - `Overflow eviction` -> `Delete the entry with the earliest `refresh_token_expired`, in the same transaction as the add, while the user row is locked`
   - `Add session (may increase count)` -> ``GormRefreshSessionRepository.AddSession` — one transaction: `SELECT ... FOR UPDATE` on the user row, in-memory eviction, whole-map `UPDATE` filtered by `deleted_at IS NULL``
   - `Rotate same session in place` -> ``GormRefreshSessionRepository.RotateSession` — one conditional `UPDATE`: `jsonb_set(..., false)` only when the stored `refresh_token_uuid` equals the verified UUID; 0 rows returns `ErrInvalidSession` (rotated, revoked or user gone)`
   - new row `Remove session` -> ``GormRefreshSessionRepository.RemoveSession` — one `UPDATE` that deletes the key (`refresh_token_session - key`); a missing key is a no-op`
   - new row `Concurrency rule` -> `No writer rewrites the map from an unlocked read, and full-row user saves never write this column`
3. **`docs/FEATURES.md` line 22** becomes: `Max 5 concurrent sessions/user; 6th login evicts the session with the earliest expiry inside one DB transaction that holds a row lock on the user. Session rotation updates in place — does not grow the count — and is a compare-and-swap on the stored refresh UUID: of several concurrent refreshes with the same refresh token exactly one succeeds, the others get 401 `InvalidSession` (`4007`). Logout removes only its own session key in one statement; full-row user saves (register resend, email confirmation, OAuth link) never rewrite sessions.`
4. **`docs/FEATURES.md` line 31** Flow: replace `Refresh (rotates pair, `session_id` unchanged)` by `Refresh (rotates pair with a compare-and-swap on the refresh UUID, `session_id` unchanged; the loser of a race gets 401 `4007`)`.
5. **`docs/FEATURES.md` line 114** (refresh diagram) becomes two lines: `SVC->>DB: RotateSession (UPDATE only if stored refresh_token_uuid = verified uuid; session_id unchanged)` and `SVC->>SVC: 0 rows (already rotated, revoked or user gone) -->|yes| 401 4007`.
6. **`docs/FEATURES.md` line 134** (logout diagram): `SVC->>DB: RemoveSession (atomic jsonb key delete); delCachedMe`.
7. **`docs/ARCHITECTURE.md` line 307**: description becomes `Session ID unknown, UUID mismatch, or refresh rotation lost to a concurrent refresh`.
8. **`docs/SHARED.md` lines 129-137**: `Path` becomes ``internal/auth/application/service_session.go` (`RefreshSession`, `rotateSession`, `issueTokenPair`); session persistence in `internal/auth/infra/user_repo.go` (`GormRefreshSessionRepository`: `LoadSessions`, `AddSession`, `RotateSession`, `RemoveSession`); session entry shape in `internal/auth/domain/user.go` (`RefreshSessionEntry`, `RefreshTokenSessionMap`)` (the current paths `auth_session_tokens.go`, `auth_refresh_rotation.go`, `user_refresh_session.go`, `refresh_session.go` do not exist); `Purpose` becomes `Token issue and rotation, and session persistence. `RotateSession` is the only writer that changes an existing session entry (compare-and-swap on the refresh UUID).` Other fields are kept unless a listed dependency path is also wrong (check each with `test -e`).
9. **`docs/ALL-SPRINTS.md`** In progress table: add `| Atomic refresh-token rotation (compare-and-swap) and no whole-map session overwrites | agent | `fix-refresh-session-rotation-race` |`.
10. **`docs/api_swagger.yaml`** under `/api/v1/auth/refresh` `responses` (after the `"200"` block): add `"400"` (`Missing X-Refresh-Token or X-Session-Id`), `"401"` (`Invalid session (malformed token, unknown session, UUID already rotated or lost concurrent refresh; code InvalidSession) or refresh token expired (code RefreshTokenExpired); clients must clear their session cookies`), `"403"` (`Account disabled (UserDisabled) or banned (UserBanned)`). Then run `ruby scripts/generate-apidog-postman.rb` and confirm `docs/api-dog-import.json` changed only through the script.
11. Re-read the `API:` line of `docs/FEATURES.md` `## auth` (line 29): it already cites `POST /api/v1/auth/{register,login,confirm,refresh,logout}`; keep.
12. `AGENTS.md` / `CLAUDE.md`: only the GitNexus stats sentence may change, through `npx gitnexus analyze`.

## Risks / Trade-offs

- [SQL is not exercised against a real Postgres by automated tests] -> statement-shape tests (D8) plus the live two-request smoke test (needs dev credentials; reported as not run if unavailable).
- [Row lock in `AddSession` serializes concurrent logins of one user] -> the same row is already written by every login; the lock lasts only for the existing short transaction.
- [Strict CAS logs out a client that retries after a lost response] -> accepted (D7); the FE serialises refreshes with a Web Lock, so this is rare.
- [`SaveWithDB` omit changes every full-row save] -> grep shows no caller sets `User.RefreshTokenSession`; covered by `TestSaveWithDBOmitsSessionAndSlug`; `CreateWithDB` untouched.
- [`AddSession` for a missing or deleted user] -> unchanged behavior (0 rows updated, no error), not worsened; the added `deleted_at IS NULL` only makes the filter consistent.
- [Deleting `SaveSessions` from the exported domain interface] -> no other consumer (grep over `internal/`, `cmd/`, `scripts/`); GitNexus d=1 is `RemoveSession` only.
- [`dupl` / `funlen` lint on the new tests] -> restructure shared setup into one helper per file, never suppress.

## Migration Plan

No migration and no data change. Deploy as a normal release; rollback is a plain revert (the persisted JSONB shape is identical for old and new binaries). No flag needed.
