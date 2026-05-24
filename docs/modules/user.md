# User Module

> **Status: Implemented under `internal/auth/` (not a separate bounded context).**  
> Profile and permission endpoints live on **`/api/v1/me`** and **`/api/v1/me/permissions`**. There is no `internal/user/` package.

---

## Overview

The authenticated user's profile and effective permission set are handled by the **auth** module (`internal/auth/`). All endpoints require `Authorization: Bearer <access_token>`.

| Endpoint | Handler | Service |
|----------|---------|---------|
| `GET /api/v1/me` | `internal/auth/delivery/handler.go` | `AuthService.GetMe` |
| `PATCH /api/v1/me` | same | `AuthService.UpdateMe` |
| `DELETE /api/v1/me` | same | soft delete via `gormx` |
| `GET /api/v1/me/permissions` | same | RBAC `PermissionCodesForUser` |

Full API shapes and cURL: **`docs/curl_api.md`** §3, **`docs/modules/auth.md`**.

**Audit fields:** `created_at` on profile responses is **Unix seconds** (`int64` in JSON). See **`docs/database.md`**.

---

## Implementation reference

| Concern | Location |
|---------|----------|
| HTTP handlers | `internal/auth/delivery/handler.go`, `routes.go` |
| Use cases | `internal/auth/application/service.go`, `service_cache.go` |
| GORM user row | `internal/auth/infra/user_model.go` |
| Redis `/me` cache | `internal/auth/application/service_cache.go` |
| Permission resolution | `internal/rbac/application/` |
| JWT + active user middleware | `internal/shared/middleware/` |

---

## Testing

Co-located tests under `internal/auth/`, `internal/rbac/`, etc. See root **`README.md`** (**Testing**).
