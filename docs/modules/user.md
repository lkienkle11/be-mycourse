# User Module


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Overview

The User module exposes the current authenticated user's profile and effective permission set. All endpoints in this module require a valid JWT access token via `Authorization: Bearer <token>`.

---

## Implemented Endpoints

### `GET /api/v1/me`

Returns the non-sensitive profile fields for the authenticated user together with their current effective permission codes.

**Auth:** Bearer JWT (required)  
**Rate limit:** 120 requests / 1 minute

**Response shape:** `dto.MeResponse` — when an avatar file is linked, **`data.avatar`** is a **`dto.MediaFilePublic`** value (same underlying type as **`entities.MediaFilePublic`**: `id`, `kind`, `provider`, `filename`, `mime_type`, `size_bytes`, `width`, `height`, `url`, `duration`, `content_fingerprint`, `status`). Omitted when unset.

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id":         1,
    "user_code":       "01960000-0000-7000-0000-000000000001",
    "email":           "user@example.com",
    "display_name":    "Alice",
    "email_confirmed": true,
    "is_disabled":     false,
    "created_at":      1713456789,
    "permissions":     ["course:read", "profile:read", "user:read"]
  }
}
```

### `PATCH /api/v1/me`

Updates the authenticated user’s profile fields. Currently supports **`avatar_file_id`** (optional UUID string referencing **`media_files.id`**). Empty string clears the avatar FK. Invalid / non-image / non-ready files return **400** + `errcode.ValidationFailed` (`pkg/errors.ErrInvalidProfileMediaFile`, text from **`constants.MsgInvalidProfileMediaFile`**).

**Caching:** `DelCachedUserMe` runs after a successful PATCH so the next `GET /me` reloads from Postgres.

---

**Caching (GET):** The response is cached in Redis at `mycourse:user:me:{user_id}` with a TTL of **1 minute**. RBAC-only changes may therefore take up to 1 minute to be reflected here; **`PATCH /me`** invalidates the cache immediately.

**Errors (`GET /me`):**
| HTTP | App Code | When |
|------|----------|------|
| 401 | 3002 | Missing or invalid JWT |
| 404 | 3004 | User record deleted from DB |
| 500 | 9001 | DB or Redis error |

**Errors (`PATCH /me`):** same as above, plus **400** / **2001** (`ValidationFailed`) when **`avatar_file_id`** is not a usable image **`media_files`** row.

---

### `GET /api/v1/me/permissions`

Returns a **sorted** list of permission code strings (`permission_name` values) granted to the authenticated user — via roles and direct grants — wrapped as **`dto.MyPermissionsResponse`** (`{ "permissions": [...] }`).

**Auth:** Bearer JWT + `user:read` permission (required)  
**Rate limit:** 120 requests / 1 minute

```json
{
  "code": 0,
  "message": "ok",
  "data": { "permissions": ["course:read", "profile:read", "user:read"] }
}
```

**Errors:**
| HTTP | App Code | When |
|------|----------|------|
| 401 | 3002 | Missing or invalid JWT |
| 403 | 3003 | User lacks `user:read` permission |
| 500 | 9001 | DB or permission resolution error |

---

## Business Logic

- `GET /me` returns only non-sensitive fields: sensitive columns (`hash_password`, `confirmation_token`, `confirmation_sent_at`, `refresh_token_session`) are never serialized.
- Permissions are resolved from the union of the user's roles (via `user_roles` → `role_permissions`) and direct `user_permissions` grants, then sorted alphabetically.
- The Redis cache (`authcache.GetCachedUserMe`) is checked before hitting Postgres. A cache miss triggers a full DB read and re-population of the cache.

## Constraints

- Both endpoints require a valid JWT access token (`Authorization: Bearer`). Cookie-based auth is not supported.
- The `/me/permissions` endpoint additionally requires the caller to hold the `user:read` (`P10`) permission.
- Cached profile data may lag Postgres by up to **1 minute**.

## Implementation Reference

| Concern | Location |
|---------|----------|
| Handler (`getMe`, `getMyPermissions`) | `api/v1/me.go` |
| Route registration | `api/v1/routes.go` — `RegisterAuthenRoutes` |
| Service (`GetMe`, `PermissionCodesForUser`) | `services/auth/auth.go`, `services/rbac/rbac.go` |
| Redis cache (`GetCachedUserMe`, `SetCachedUserMe`) | `services/cache/auth_user.go` |
| DTO (`MeResponse`) | `dto/auth.go` |
| Permission constant (`UserRead`) | `constants/permissions.go` |
| JWT middleware | `middleware/auth_jwt.go` |
| Permission middleware | `middleware/rbac.go` |

---

## Testing

- **All tests** for this module (unit/module-level/integration): **`tests/`** at repo root (`tests/README.md`, root `README.md` **Testing**).
