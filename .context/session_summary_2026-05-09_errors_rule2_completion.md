# Session summary — errors-rule-2 remediation (2026-05-09)

## Code

- **Rule 3 / multipart:** Upload part types and helpers live in **`pkg/entities/media_multipart_parts.go`**; handlers and `services/media` use **`entities.*`** (removed duplicate `pkg/media` part files).
- **Rule 6 / jobs:** Orphan and superseded enqueue live in **`internal/jobs/media/media_orphan_enqueue.go`** (`jobmedia`); pending-cleanup batch + metrics in **`internal/jobs/media/media_pending_cleanup_batch.go`**, **`internal/jobs/media/media_cleanup_metrics.go`**; RBAC tickers in **`internal/jobs/rbac/`**; system HTTP adapters in **`internal/jobs/system/system_sync_http.go`**. **`api/system/routes.go`** wires start/stop and keeps `*-sync-now` handlers local.
- **Rule 7 / DTOs:** Replaced **`gin.H`** on auth + system routes with **`dto.*`**; internal RBAC handlers return **`dto.RBACPermissionResponse`** / **`dto.RBACRoleResponse`** via **`pkg/logic/mapping/rbac_internal_response.go`**; **`GET /me/permissions`** and internal **`GET .../users/:id/permissions`** wrap code lists in **`dto.MyPermissionsResponse`** / **`dto.UserRBACPermissionCodesResponse`**. **`api/v1/media/file_handler.go`** reads cleanup metrics from **`jobmedia`**.
- **`services/media/file_service.go`** calls **`jobmedia.EnqueueSupersededPendingCleanup`**.
- **Rule 9 / gormjsonb:** JSONB session map type in **`pkg/gormjsonb/auth/refresh_token_session_map.go`** (`gormjsonbauth.RefreshTokenSessionMap` on **`models/user.go`**).

## Docs

- Synced **`docs/*`**, **`IMPLEMENTATION_PLAN_EXECUTION.md`**, **`README.md`** references where needed; **did not** edit `temporary-docs/loi-quang-ra/errors-rule-2.md`.

## Quality gates

- `golangci-lint` (0 issues), `make check-architecture`, `gofmt`, `go vet ./...`, `go test ./...`, `make build-nocgo`.

## Client note (breaking)

- **`GET /api/v1/me/permissions`**: `data` is now `{ "permissions": [...] }` (was a bare array).
- **`GET /api/internal-v1/rbac/users/:userId/permissions`**: `data` is now `{ "permission_codes": [...] }` (was a bare array).
