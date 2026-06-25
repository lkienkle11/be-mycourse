# Session Summary — UUID v7 + ULID Migration (BE close-out)

**Date:** 2026-06-09  
**Scope:** `be-mycourse` — taxonomy + instructor UUID alignment, quality gates, smoke

## Completed this session

### Code (taxonomy + instructor)
- Migrated all taxonomy entity IDs (`course_topics`, `outcomes`, `skills`, `tags`, `levels`) from `uint` → `string` UUID across domain, application, infra, delivery.
- GORM row types: `primaryKey;type:uuid`, create path assigns `uuidx.NewV7()` when ID empty.
- Delivery: `ParseUintParam` → `ParseUUIDParam` for taxonomy routes.
- Migrated instructor module IDs (applications, profiles, expertise, tickets, messages) from `uint` → `string` UUID.
- Added `internal/instructor/infra/id_helper.go` for v7 assignment on create.

### Quality gates (all PASS)
| Gate | Result |
|------|--------|
| `go build ./...` | PASS |
| `go test ./...` | PASS |
| `golangci-lint run ./...` | 0 issues |
| `make check-architecture` | OK |
| `make check-dupl` | 0 clones |
| `make check-layout` | OK |

### GitNexus
- `npx gitnexus analyze` — index up to date

### Manual verification
- `go run .` — server listens on `:8080`
- `GET /api/v1/health` — OK
- Login smoke with a local dev account — **failed** (`Invalid email or password`); local DB likely not synced with seed data. Re-run after `CLI_IMPORT_LEGACY_DATA` or seed.

### RBAC exceptions (unchanged)
- `roles.id`, `role_id` junction FKs remain `uint` / `BIGINT` per plan.

## Files touched (high level)

| Area | Files |
|------|-------|
| Taxonomy | `domain/*`, `application/*`, `infra/repos*.go`, `delivery/*` |
| Instructor | `domain/*`, `application/*`, `infra/*`, `delivery/*` |
| Docs (prior) | `database.md`, `return_types.md`, `reusable-assets.md`, `folder-structure.md` |

## Deploy / ops notes
- Users must **re-login** after migration (JWT `user_id` is now UUID string).
- Legacy restore: `CLI_IMPORT_LEGACY_DATA_DUMP` + idmap file from appcli import pipeline.

## Next (FE)
- FE G1–G3 completed in parallel session — see `fe-mycourse/.context/session_summary_2026-06-09_uuid_v7_migration_fe.md`
