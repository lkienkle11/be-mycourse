# Session summary — int64 audit timestamps (2026-05-23)

## Goal

Convert all audit columns (`created_at`, `updated_at`, `deleted_at`) to **BIGINT Unix epoch seconds** end-to-end: PostgreSQL → Go domain/infra → JSON API.

## Completed

### Shared foundation
- `internal/shared/timex/timex.go` — `NowUnix`, `PtrUnix`, `UnixOrZero`
- `internal/shared/gormx/audit_timestamps.go` — `TouchCreatedUpdated`, `TouchUpdated`, `SoftDeleteWithAudit`
- `internal/shared/token/jwt.go` — `GenerateAccess(..., createdAt int64, ...)`

### Migration
- `migrations/000011_audit_timestamps_bigint.up.sql` / `.down.sql`
- `migrations/README.md` updated

### Modules refactored
- **Auth:** domain `int64`/`*int64`, manual soft delete, raw SQL `NOW()` → unix
- **RBAC:** domain/infra/delivery `int64` (breaking API: was ISO strings)
- **Taxonomy:** removed RFC3339 formatting, DTOs `int64`
- **Media:** domain/infra/application `int64` audit fields, soft delete via `SoftDeleteWithAudit`
- **System:** `app_config.updated_at`, `privileged_users.created_at` as `int64`

### Docs synced
- `docs/database.md`, `docs/return_types.md`, `docs/curl_api.md`

### Verification
- `go test ./...` — pass
- `golangci-lint run` — 0 issues
- `make check-architecture`, `make check-dupl`, `make check-layout`, `make build-nocgo` — pass
- `npx gitnexus analyze --force` — re-indexed

## Breaking changes (FE follow-up)

| API | Before | After |
|-----|--------|-------|
| Taxonomy list/create/update | `"created_at": "2026-05-20T10:00:00Z"` | `"created_at": 1747744800` |
| RBAC permissions/roles | ISO8601 strings | Unix integers |

Auth `/me` and JWT `created_at` were already `int64` — unchanged behavior.

## Out of scope (unchanged)

- `confirmation_sent_at`, `next_run_at`, JWT `exp`/`iat`, refresh session expiry

## Deploy note

Run migration on staging/prod before deploying new binary:

```bash
MIGRATE=1 go run .
```
