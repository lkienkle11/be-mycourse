# Logic Flow Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Application Startup
1. Load settings (`setting.Setup()`).
2. Initialize primary DB (`shareddb.Setup()`).
3. Setup Redis (`cache.SetupRedis()`).
4. Configure circuit breaker + start DB probe (`resilience.ConfigureFromSettings`, `resilience.StartDBProbe`).
5. Optional CLI branch (`internal/appcli`) can short-circuit server startup (guarded by `cli_guard.go`; machine enrollment + OS fingerprint via `internal/shared/machineidentity`).
6. Setup Supabase clients.
7. Setup media SDK clients.
8. Optional SQL migration mode: `MIGRATE=1` runs up migration then continues startup; `MIGRATE=2` runs down by `MIGRATE_VERSION_FILE` then exits.
9. Wire dependencies (`server.Wire`) unless startup exited in `MIGRATE=2`.
10. Start background jobs (media pending cleanup).
11. Build router and serve HTTP.

## Auth Flow Logic
- Register: validate input -> check uniqueness -> create user -> send confirmation email.
- Login: validate credentials -> read/cache checks -> issue access/refresh tokens -> persist session map.
- Confirm: verify token -> transactionally confirm email + ensure learner role -> issue token pair.
- Refresh: validate refresh token + session ownership -> rotate token/session and permissions.

## RBAC Sync Logic
- Permission sync: upsert permissions from constants catalog.
- Role-permission sync: delete and rebuild mapping from constants matrix.
- Triggers: CLI commands, system API "sync-now", or scheduler jobs.

## Enforcement Logic
- Authentication gates run before handlers.
- Authorization decisions prefer JWT claim set; fallback to DB if claims absent.
- Error handling is centralized through middleware and errcode mapping.

## Media / Bunny (high level)
- Upload and webhook paths persist **`media_files`** and map public JSON to **`dto.UploadFileResponse`** (no **`origin_url`** key — Sub 12; optional Bunny delivery fields when populated). There is no dedicated sequence diagram in this repo for media; see **`docs/modules/media.md`**, **`docs/data-flow.md`**, and **`docs/router.md`** for the authoritative flow and layering (`internal/media/infra/media_resolver.go`).

## Course / Instructor Logic
- Course authoring: `internal/course` owns draft lifecycle, collaborator roles (`OWNER`, `EDITOR`), review transitions, outline stable IDs, and learner progress. `EDITOR` may edit basic info and outline; **prepare draft**, **submit for review**, and **reopen draft** are **owner-only** (`requireOwnerAccess` → `ErrCourseOwnerOnly`).
- Instructor management: `internal/instructor` owns roster, application review, profile upsert, expertise links, and ticket messaging.
- Both modules are mounted on authenticated `/api/v1` and guarded by RBAC permissions at route level.
