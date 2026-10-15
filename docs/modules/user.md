# User Module

> **Status: Implemented under `internal/auth/` (not a separate bounded context).**  
> Profile and permission endpoints live on **`/api/v1/me`** and **`/api/v1/me/permissions`**. There is no `internal/user/` package.

---

## Overview

The authenticated user's profile and effective permission set are handled by the **auth** module (`internal/auth/`). All endpoints require `Authorization: Bearer <access_token>`.

| Endpoint | Handler | Service |
|----------|---------|---------|
| `GET /api/v1/me` | `internal/auth/delivery/handler.go` | `AuthService.GetMe` |
| `PATCH /api/v1/me` | same | `AuthService.UpdateMe` (avatar `image_file_id`, `slug`) |
| `DELETE /api/v1/me` | same | soft delete via `gormx` |
| `GET /api/v1/me/permissions` | same | RBAC `PermissionCodesForUser` |

Full API shapes and cURL: **`docs/curl_api.md`** §3, **`docs/modules/auth.md`**.

**Slug:** every user has a public `slug` (returned on `GET /me`). It is generated from `display_name` on registration (OAuth sign-up included), does not follow later `display_name` changes, and is changed only through `PATCH /me` (`slug`; reserved words rejected with 400, collisions auto-suffixed, left slugs reserved in `user_slug_history`). Existing rows are populated by `CLI_BACKFILL_USER_SLUGS=1`. Details: **`docs/modules/auth.md`** "User slug" and **`docs/database.md`**.

**Audit fields:** `created_at` on profile responses is **Unix seconds** (`int64` in JSON). See **`docs/database.md`**.

---

## Implementation reference

| Concern | Location |
|---------|----------|
| HTTP handlers | `internal/auth/delivery/handler.go`, `routes.go` |
| Use cases | `internal/auth/application/service.go`, `service_cache.go` |
| GORM user row | `internal/auth/infra/user_model.go` |
| User slug (create, `ChangeSlug`, backfill) | `internal/auth/infra/user_slug.go`, `user_slug_backfill.go`; shared logic in `internal/shared/slug` |
| Redis `/me` cache | `internal/auth/application/service_cache.go` |
| Permission resolution | `internal/rbac/application/` |
| JWT + active user middleware | `internal/shared/middleware/` |

---

## Testing

Co-located tests under `internal/auth/`, `internal/rbac/`, etc. See root **`README.md`** (**Testing**).
