# User Module

## Overview

The User module exposes the current authenticated user's profile and effective permission set. All endpoints in this module require a valid JWT access token via `Authorization: Bearer <token>`.

---

## Implemented Endpoints

### `GET /api/v1/me`

Returns the non-sensitive profile fields for the authenticated user together with their current effective permission codes.

**Auth:** Bearer JWT (required)  
**Rate limit:** 120 requests / 1 minute

**Response shape:** `dto.MeResponse`

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id":         1,
    "user_code":       "01960000-0000-7000-0000-000000000001",
    "email":           "user@example.com",
    "display_name":    "Alice",
    "avatar_url":      "",
    "email_confirmed": true,
    "is_disabled":     false,
    "created_at":      1713456789,
    "permissions":     ["course:read", "profile:read", "user:read"]
  }
}
```

**Caching:** The response is cached in Redis at `mycourse:user:me:{user_id}` with a TTL of **1 minute**. Writes to the user profile or RBAC assignments may therefore take up to 1 minute to be reflected here.

**Errors:**
| HTTP | App Code | When |
|------|----------|------|
| 401 | 3002 | Missing or invalid JWT |
| 404 | 3004 | User record deleted from DB |
| 500 | 9001 | DB or Redis error |

---

### `GET /api/v1/me/permissions`

Returns a **sorted** list of permission code strings (`permission_name` values) granted to the authenticated user — via roles and direct grants.

**Auth:** Bearer JWT + `user:read` permission (required)  
**Rate limit:** 120 requests / 1 minute

```json
{
  "code": 0,
  "message": "ok",
  "data": ["course:read", "profile:read", "user:read"]
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
| Service (`GetMe`, `PermissionCodesForUser`) | `services/auth.go`, `services/rbac.go` |
| Redis cache (`GetCachedUserMe`, `SetCachedUserMe`) | `services/cache/auth_user.go` |
| DTO (`MeResponse`) | `dto/auth.go` |
| Permission constant (`UserRead`) | `constants/permissions.go` |
| JWT middleware | `middleware/auth_jwt.go` |
| Permission middleware | `middleware/rbac.go` |

---

## Testing

- **All tests** for this module (unit/module-level/integration): **`tests/`** at repo root (`tests/README.md`, root `README.md` **Testing**).
