# MyCourse Backend — API Reference & cURL Examples


> **Base URL:** `http://localhost:8080` (local) / `https://api.yourdomain.com` (production)  
> Replace `{{BASE_URL}}` with the actual base URL in all examples.  
> **Handlers:** `internal/<domain>/delivery` (routes in `routes.go`, wired from `internal/server/wire.go`).  
> **Last updated:** 2026-06-07

**Audit timestamps:** JSON `created_at` / `updated_at` (and soft-delete `deleted_at` where returned) are **integers** — Unix epoch **seconds**, not ISO strings. See **`docs/database.md`** and migration `000011`.

---

## Table of Contents

1. [Global Conventions](#1-global-conventions)
2. [Authentication (`/api/v1/auth`)](#2-authentication-apiv1auth)
   - [Register](#21-register)
   - [Login](#22-login)
   - [Confirm Email](#23-confirm-email)
   - [Refresh Token](#24-refresh-token)
3. [User Profile (`/api/v1/me`)](#3-user-profile-apiv1me)
   - [Get My Profile](#31-get-my-profile)
   - [Get My Permissions](#32-get-my-permissions)
4. [Health Check](#4-health-check)
5. [System Admin (`/api/system`)](#5-system-admin-apisystem)
   - [System Login](#51-system-login)
   - [Permission Sync Now](#52-permission-sync-now)
   - [Role-Permission Sync Now](#53-role-permission-sync-now)
   - [Create Permission Sync Job](#54-create-permission-sync-job)
   - [Create Role-Permission Sync Job](#55-create-role-permission-sync-job)
   - [Delete Permission Sync Job](#56-delete-permission-sync-job)
   - [Delete Role-Permission Sync Job](#57-delete-role-permission-sync-job)
6. [Internal RBAC — Permissions (`/api/internal-v1/rbac/permissions`)](#6-internal-rbac--permissions)
   - [List Permissions](#61-list-permissions)
   - [Create Permission](#62-create-permission)
   - [Update Permission](#63-update-permission)
   - [Delete Permission](#64-delete-permission)
7. [Internal RBAC — Roles (`/api/internal-v1/rbac/roles`)](#7-internal-rbac--roles)
   - [List Roles](#71-list-roles)
   - [Create Role](#72-create-role)
   - [Get Role by ID](#73-get-role-by-id)
   - [Update Role](#74-update-role)
   - [Set Role Permissions](#75-set-role-permissions)
   - [Delete Role](#76-delete-role)
8. [Internal RBAC — User Roles](#8-internal-rbac--user-roles)
   - [List User Roles](#81-list-user-roles)
   - [Assign Role to User](#82-assign-role-to-user)
   - [Remove Role from User](#83-remove-role-from-user)
9. [Internal RBAC — User Permissions](#9-internal-rbac--user-permissions)
   - [List User Effective Permissions](#91-list-user-effective-permissions)
   - [List User Direct Permissions](#92-list-user-direct-permissions)
   - [Assign Direct Permission to User](#93-assign-direct-permission-to-user)
   - [Remove Direct Permission from User](#94-remove-direct-permission-from-user)
10. [Deprecated / Planned APIs](#10-deprecated--planned-apis)
11. [Media API (`/api/v1/media/*`)](#11-media-api-apiv1media)
12. [Taxonomy (`/api/v1/taxonomy/*`)](#12-taxonomy-apiv1taxonomy)
13. [Instructor management (`/api/v1/instructors`, …)](#13-instructor-management)
14. [Course management (`/api/v1/courses`, …)](#14-course-management)
15. [Webhooks (`/api/v1/webhook/*`)](#15-webhooks-apiv1webhook)
16. [Error Code Reference](#16-error-code-reference)
17. [Postman / API Dog](#17-postman--api-dog)
18. [Local smoke test (migrations `000011` + `000013`)](#18-local-smoke-test-migrations-000011--000013)

**Test code layout:** module-level / integration Go tests belong under repository root **`tests/`** — see `tests/README.md` and root `README.md` (**Testing**).

---

## 1. Global Conventions

### Base URL

```
http://localhost:8080       # local dev
https://api.yourdomain.com    # production
```

### Response Envelope

Every response is JSON in one of two shapes:

**Standard:**
```json
{ "code": 0, "message": "ok", "data": <value> }
```

**Health:**
```json
{ "code": 0, "message": "ok", "status": "ok" }
```

`code: 0` = success. Any non-zero `code` = application error (see [section 14](#14-error-code-reference)).

**Audit timestamps:** `created_at`, `updated_at`, and `deleted_at` (when present) are **Unix epoch integers** (seconds) in JSON — not RFC3339/ISO strings. This applies to auth `/me`, RBAC, taxonomy, and media list responses.

### Authentication Headers

| Header | Used for |
|--------|----------|
| `Authorization: Bearer <access_token>` | All JWT-protected endpoints under `/api/v1` |
| `X-Refresh-Token: <refresh_jwt>` | `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout` |
| `X-Session-Id: <128-hex>` | `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout` |
| `X-API-Key: <key>` | All `/api/internal-v1` endpoints |
| `Authorization: Bearer <system_token>` | All `/api/system` endpoints |

### Rate Limits

| Route Group | Limit |
|-------------|-------|
| `/api/system` | 10 requests / 3 minutes per IP |
| `/api/v1` (unauthenticated) | 60 requests / 1 minute |
| `/api/v1` (authenticated) | 120 requests / 1 minute |
| `/api/internal-v1` | 60 requests / 1 minute |
| APPCLI (`CLI_SYSTEM_LOGIN`, `CLI_REGISTER_NEW_SYSTEM_USER`) | 5 operations / 3 minutes per host (file: `$XDG_CONFIG_HOME/mycourse/cli_rate_limit.json`) |

Rate limit exceeded returns `HTTP 429` with `code: 3006`.  
Circuit breaker open returns `HTTP 503` with `code: 9018`.

### Pagination Query Parameters

Applicable to all `GET` list endpoints:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | 1-based page number |
| `per_page` | int | 20 | Items per page (max 100) |
| `sort_by` | string | — | Field to sort by (endpoint-specific) |
| `sort_order` | string | `asc` | `asc` or `desc` |
| `search_by` | string | — | Field to search in (endpoint-specific) |
| `search_data` | string | — | Search term (used with `search_by`) |

---

## 2. Authentication (`/api/v1/auth`)

### 2.0 CSRF Bootstrap (required before unsafe methods)

**`GET /api/v1/auth/csrf`**

Bootstraps CSRF state using double-submit cookie (`csrf_token`) and FE header (`X-CSRF-Token`).
Current status: CSRF filter is temporarily disabled at router level, so this step is optional for now.

```bash
curl -X GET {{BASE_URL}}/api/v1/auth/csrf -c cookies.txt
```

Use the cookie value for unsafe requests:

```bash
CSRF_TOKEN=$(grep csrf_token cookies.txt | awk '{print $7}')
```

For every unsafe method (`POST`, `PUT`, `PATCH`, `DELETE`), include:

```text
X-CSRF-Token: <csrf_token cookie value>
```

### 2.1 Register

**`POST /api/v1/auth/register`**

Creates a new user account and sends a confirmation email. No token is returned — the user must confirm their email first.

**Request body:**

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `email` | string | ✅ | Valid email format |
| `password` | string | ✅ | Min 8 chars, 1 uppercase, 1 lowercase, 1 special char |
| `display_name` | string | ✅ | 1–255 characters |
| `locale` | string | — | `en` or `vi` (default `vi`); email language + confirmation link path |

```bash
curl -X POST {{BASE_URL}}/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -d '{
    "email":        "alice@example.com",
    "password":     "Str0ng!Pass",
    "display_name": "Alice",
    "locale":       "en"
  }'
```

Confirmation email link: `{APP_CLIENT_BASE_URL}/{locale}/confirm-email?token={uuid}`. Subject/body from `template/languages/confirm_account/{locale}.js`.

**Success (201):**
```json
{ "code": 0, "message": "registration_success", "data": null }
```

**Error examples:**
```json
// 409 — email already registered (confirmed)
{ "code": 4001, "message": "Email address is already registered", "data": null }

// 410 — lifetime confirmation email cap (pending user removed)
{ "code": 4009, "message": "registration was abandoned because confirmation email limits were exceeded", "data": null }

// 429 — Redis window (also Retry-After + X-Mycourse-Register-Retry-After headers)
{ "code": 4010, "message": "too many confirmation emails were sent recently; please try again later", "data": null }

// 502 — Brevo failure after limits
{ "code": 4011, "message": "confirmation email could not be sent; please try again later", "data": null }

// 400 — weak password
{ "code": 4003, "message": "Password must be at least 8 characters and contain uppercase, lowercase, and special characters", "data": null }

// 400 — validation failure (missing field)
{ "code": 2001, "message": "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag", "data": null }
```

---

### 2.2 Login

**`POST /api/v1/auth/login`**

Validates credentials and issues an access token, refresh token, and session ID.  
Returns tokens in the JSON body **and** as cookies (`access_token`, `refresh_token`, `session_id`).

**Request body:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `email` | string | ✅ | — | |
| `password` | string | ✅ | — | |
| `remember_me` | bool | ❌ | `false` | `false` = 3d fixed TTL; `true` = 30d sliding window |

```bash
curl -X POST {{BASE_URL}}/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -c cookies.txt \
  -d '{
    "email":       "alice@example.com",
    "password":    "Str0ng!Pass",
    "remember_me": false
  }'
```

**Success (200):**
```json
{
  "code": 0,
  "message": "login_success",
  "data": {
    "access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "session_id":    "a1b2c3d4...128hexchars..."
  }
}
```

**Response cookies (Set-Cookie)** — example when `remember_me=false` (3 days):
```
access_token=<jwt>; Path=/; Max-Age=900; SameSite=Lax
refresh_token=<jwt>; Path=/; Max-Age=259200; SameSite=Lax
session_id=<128hex>; Path=/; Max-Age=259200; SameSite=Lax
```

When `remember_me=true`, refresh/session cookies use `Max-Age=2592000` (30 days).

**Error examples:**
```json
// 401 — wrong credentials
{ "code": 4002, "message": "Invalid email or password", "data": null }

// 401 — email not confirmed
{ "code": 4004, "message": "Email address has not been confirmed yet", "data": null }

// 403 — account disabled
{ "code": 4005, "message": "Account has been disabled", "data": null }

// 403 — account temporarily banned (banned_until > now)
{ "code": 4012, "message": "Your account is temporarily banned", "data": null }
```

---

### 2.3 Confirm Email

**`POST /api/v1/auth/confirm`**

Confirms the user's email address, assigns the `learner` role, and returns a token pair (user is immediately logged in).

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | ✅ | UUID confirmation token received from registration email |

```bash
curl -X POST "{{BASE_URL}}/api/v1/auth/confirm" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -d '{"token":"550e8400-e29b-41d4-a716-446655440000"}' \
  -c cookies.txt
```

**Success (200):**
```json
{
  "code": 0,
  "message": "email_confirmed",
  "data": {
    "access_token":  "eyJ...",
    "refresh_token": "eyJ...",
    "session_id":    "a1b2c3..."
  }
}
```

**Response cookies:** Same three cookies as login.

**Error examples:**
```json
// 400 — invalid or already-used token
{ "code": 4006, "message": "Invalid or expired confirmation token", "data": null }

// 400 — missing token field
{ "code": 3001, "message": "<validation error>", "data": null }
```

---

### 2.4 Refresh Token

**`POST /api/v1/auth/refresh`**

Rotates the access + refresh token pair for an existing session.  
No request body. Tokens are supplied via custom headers.

**Request headers:**

| Header | Required | Description |
|--------|----------|-------------|
| `X-Refresh-Token` | ✅ | Current refresh JWT |
| `X-Session-Id` | ✅ | Current session ID (128-char hex) |

```bash
curl -X POST {{BASE_URL}}/api/v1/auth/refresh \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -H "X-Refresh-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "X-Session-Id: a1b2c3d4...128hexchars..."
```

**Using cookies from file:**
```bash
curl -X POST {{BASE_URL}}/api/v1/auth/refresh \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -H "X-Refresh-Token: $(grep refresh_token cookies.txt | awk '{print $7}')" \
  -H "X-Session-Id: $(grep session_id cookies.txt | awk '{print $7}')"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "token_refreshed",
  "data": {
    "access_token":  "eyJ...new...",
    "refresh_token": "eyJ...new...",
    "session_id":    "a1b2c3...same...",
  }
}
```

> `session_id` is **unchanged** on rotation.

**Error examples:**
```json
// 400 — missing headers
{ "code": 3001, "message": "missing X-Refresh-Token or X-Session-Id header", "data": null }

// 401 — session not found or UUID mismatch
{ "code": 4007, "message": "Session string unknown, missing, or UUID mismatch", "data": null }

// 401 — refresh expired
{ "code": 4008, "message": "Session has expired — re-login required", "data": null }
```

---

### 2.5 Logout

**`POST /api/v1/auth/logout`**

Revokes the current refresh session in `users.refresh_token_session` and clears auth cookies. Same headers as refresh. No request body.

```bash
curl -X POST {{BASE_URL}}/api/v1/auth/logout \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b cookies.txt \
  -H "X-Refresh-Token: $(grep refresh_token cookies.txt | awk '{print $7}')" \
  -H "X-Session-Id: $(grep session_id cookies.txt | awk '{print $7}')"
```

**Success (200):**
```json
{ "code": 0, "message": "logout_success", "data": null }
```

Response `Set-Cookie` clears `access_token`, `refresh_token`, and `session_id` (`MaxAge: -1`).

**Idempotent:** Session already removed → still `200` + cookie clear.

**Error examples:**
```json
// 400 — missing headers (cookies still cleared)
{ "code": 3001, "message": "missing X-Refresh-Token or X-Session-Id header", "data": null }

// 401 — UUID mismatch (cookies still cleared)
{ "code": 4007, "message": "Session string unknown, missing, or UUID mismatch", "data": null }
```

After logout, `POST /api/v1/auth/refresh` with the same tokens must return **401**.

---

## 3. User Profile (`/api/v1/me`)

> All endpoints require `Authorization: Bearer <access_token>`.

### 3.1 Get My Profile

**`GET /api/v1/me`**

Returns the authenticated user's profile and their current effective permission codes.  
Served from Redis cache (1-minute TTL) on cache hit; Postgres on miss.

```bash
ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X GET {{BASE_URL}}/api/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id":         1,
    "user_code":       "01960000-0000-7000-0000-000000000001",
    "email":           "alice@example.com",
    "display_name":    "Alice",
    "email_confirmed": true,
    "is_disabled":     false,
    "created_at":      1713456789,
    "permissions":     ["course:read", "profile:read", "user:read"]
  }
}
```

> `created_at` is a Unix epoch **integer** (seconds).  
> `permissions` is a sorted array of `permission_name` strings.  
> When an avatar is linked, **`data.avatar`** is a **`dto.MediaFilePublic`** object (see `docs/return_types.md`).

**Access guards:** Returns **404** if the user is soft-deleted. Returns **403** (`4005` disabled, `4012` banned) if `is_disable` is true or `banned_until > now()`. Valid JWT alone is not enough after delete/ban.

### 3.2 Patch My Profile

**`PATCH /api/v1/me`** — body: `{ "avatar_file_id": "<uuid of media_files row>" }`. Use **`""`** to clear. Upload an image via **`POST /api/v1/media/files`** first, then pass the returned **`id`**.

```bash
curl -X PATCH {{BASE_URL}}/api/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"avatar_file_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

**Error examples:**
```json
// 401 — no token
{ "code": 3002, "message": "missing bearer token", "data": null }

// 401 — token expired (also sets header X-Token-Expired: true)
{ "code": 3002, "message": "token expired", "data": null }
```

---

### 3.3 Delete My Account

**`DELETE /api/v1/me`** — soft delete (sets `deleted_at`). Subsequent login, refresh, and `/me` calls fail.

```bash
curl -X DELETE {{BASE_URL}}/api/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**`DELETE /api/v1/me/hard`** — permanent row removal (same permission as soft delete; no extra permission).

```bash
curl -X DELETE {{BASE_URL}}/api/v1/me/hard \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Both return **`account_deleted`** on success.

---

### 3.4 Get My Permissions

**`GET /api/v1/me/permissions`**

Returns the sorted list of permission codes (via roles + direct grants) for the authenticated user, wrapped as **`{ "permissions": [...] }`** in the envelope `data`.  
Requires `user:read` (`P10`) permission.

```bash
curl -X GET {{BASE_URL}}/api/v1/me/permissions \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": { "permissions": ["course:read", "profile:read", "user:read"] }
}
```

**Error examples:**
```json
// 403 — lacks user:read permission
{ "code": 3003, "message": "forbidden", "data": null }
```

---

## 4. Health Check

**`GET /api/v1/health`**

No authentication required. Used by CI/CD health probes and load balancers.

```bash
curl -X GET {{BASE_URL}}/api/v1/health
```

**Success (200):**
```json
{ "code": 0, "message": "ok", "status": "ok" }
```

---

## 5. System Admin (`/api/system`)

> Rate limit: **10 requests / 3 seconds** per IP.  
> All routes require `Authorization: Bearer <system_token>`.

### 5.0 CLI — obtain system token

System login is **CLI-only** (no HTTP endpoint). After DB init, run interactively:

```bash
SYSTEM_TOKEN=$(CLI_SYSTEM_LOGIN=1 go run .)
```

- Prompts for username and password on stderr (hidden input).
- Requires hybrid machine binding on the same host: enrollment file + matching OS fingerprint (see FR-4.4).
- **Success:** stdout prints **only** the raw JWT string (one line). No JSON envelope.
- **Failure:** human-readable message on stderr only.
- Token TTL: **90 seconds** (`SystemAccessTokenTTL()`).

Register a privileged user first (once per host):

```bash
CLI_REGISTER_NEW_SYSTEM_USER=1 go run .
```

---

### 5.1 Permission Sync Now

**`POST /api/system/permission-sync-now`**

Upserts all permissions from `constants.AllPermissions` into the database. Existing rows with unknown `permission_id`s are not deleted.

```bash
curl -X POST {{BASE_URL}}/api/system/permission-sync-now \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "permission_sync_completed", "data": { "synced": 13 } }
```

---

### 5.2 Role-Permission Sync Now

**`POST /api/system/role-permission-sync-now`**

**Destructively** rebuilds all `role_permissions` rows from `constants.RolePermissions`. All existing assignments are deleted first.

```bash
curl -X POST {{BASE_URL}}/api/system/role-permission-sync-now \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "role_permission_sync_completed", "data": { "rows": 32 } }
```

---

### 5.3 Create Permission Sync Job

**`POST /api/system/create-permission-sync-job`**

Starts an in-memory 12-hour ticker that re-runs permission sync automatically.

```bash
curl -X POST {{BASE_URL}}/api/system/create-permission-sync-job \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "permission_sync_job_started", "data": null }
```

---

### 5.4 Create Role-Permission Sync Job

**`POST /api/system/create-role-permission-sync-job`**

Starts an in-memory 12-hour ticker that re-runs role-permission sync automatically.

```bash
curl -X POST {{BASE_URL}}/api/system/create-role-permission-sync-job \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "role_permission_sync_job_started", "data": null }
```

---

### 5.5 Delete Permission Sync Job

**`POST /api/system/delete-permission-sync-job`**

Stops the permission sync ticker.

```bash
curl -X POST {{BASE_URL}}/api/system/delete-permission-sync-job \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "permission_sync_job_stopped", "data": null }
```

---

### 5.6 Delete Role-Permission Sync Job

**`POST /api/system/delete-role-permission-sync-job`**

Stops the role-permission sync ticker.

```bash
curl -X POST {{BASE_URL}}/api/system/delete-role-permission-sync-job \
  -H "Authorization: Bearer $SYSTEM_TOKEN"
```

**Success (200):**
```json
{ "code": 0, "message": "role_permission_sync_job_stopped", "data": null }
```

---

## 6. Internal RBAC — Permissions

> **Auth:** `X-API-Key: <key>`  
> **Rate limit:** 60 requests / 1 minute  
> **Audit timestamps:** `created_at` and `updated_at` are Unix epoch **integers** (seconds), not ISO strings.

```bash
# Store key for convenience
INTERNAL_KEY="your-internal-api-key"
```

### 6.1 List Permissions

**`GET /api/internal-v1/rbac/permissions`**

Paginated, sortable, searchable list of all permissions.

**Query parameters:** Standard `BaseFilter` params.  
**Sortable fields:** `permission_id`, `permission_name`, `description`, `created_at`  
**Searchable fields:** `permission_id`, `permission_name`, `description`

```bash
# Basic list
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/permissions" \
  -H "X-API-Key: $INTERNAL_KEY"

# With pagination + search
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/permissions?page=1&per_page=10&search_by=permission_name&search_data=course&sort_by=permission_id&sort_order=asc" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "result": [
      {
        "permission_id":   "P5",
        "permission_name": "course:read",
        "description":     "",
        "created_at": 1735689600,
        "updated_at": 1735689600
      }
    ],
    "page_info": {
      "page": 1,
      "per_page": 10,
      "total_pages": 1,
      "total_items": 4
    }
  }
}
```

---

### 6.2 Create Permission

**`POST /api/internal-v1/rbac/permissions`**

Creates a new permission entry.

**Request body:**

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `permission_id` | string | ✅ | 1–10 chars (e.g. `P14`) |
| `permission_name` | string | ✅ | 1–50 chars (e.g. `lesson:read`) |
| `description` | string | ❌ | max 512 chars |

```bash
curl -X POST {{BASE_URL}}/api/internal-v1/rbac/permissions \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "permission_id":   "P14",
    "permission_name": "lesson:read",
    "description":     "Read lesson content"
  }'
```

**Success (201):**
```json
{
  "code": 0,
  "message": "created",
  "data": {
    "permission_id":   "P14",
    "permission_name": "lesson:read",
    "description":     "Read lesson content",
    "created_at": 1744970400,
    "updated_at": 1744970400
  }
}
```

**Error examples:**
```json
// 400 — duplicate permission_id
{ "code": 3001, "message": "ERROR: duplicate key value violates unique constraint...", "data": null }
```

---

### 6.3 Update Permission

**`PATCH /api/internal-v1/rbac/permissions/:permissionId`**

Updates a permission. All fields are optional (partial update). If `permission_id` is changed, FK `ON UPDATE CASCADE` propagates automatically.

**Path param:** `permissionId` — the current `permission_id` (e.g. `P14`)

**Request body:**

| Field | Type | Required |
|-------|------|----------|
| `permission_id` | string | ❌ (new ID if renaming) |
| `permission_name` | string | ❌ |
| `description` | string | ❌ |

```bash
curl -X PATCH {{BASE_URL}}/api/internal-v1/rbac/permissions/P14 \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "permission_name": "lesson:view",
    "description":     "View lesson content"
  }'
```

**Success (200):**
```json
{
  "code": 0,
  "message": "updated",
  "data": {
    "permission_id":   "P14",
    "permission_name": "lesson:view",
    "description":     "View lesson content",
    "created_at": 1735689600,
    "updated_at": 1744974000
  }
}
```

**Error examples:**
```json
// 404 — permission not found
{ "code": 3004, "message": "not found", "data": null }
```

---

### 6.4 Delete Permission

**`DELETE /api/internal-v1/rbac/permissions/:permissionId`**

Deletes a permission and cascades to `role_permissions` and `user_permissions`.

**Path param:** `permissionId` — the `permission_id` to delete (e.g. `P14`)

```bash
curl -X DELETE {{BASE_URL}}/api/internal-v1/rbac/permissions/P14 \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{ "code": 0, "message": "deleted", "data": null }
```

---

## 7. Internal RBAC — Roles

### 7.1 List Roles

**`GET /api/internal-v1/rbac/roles`**

**Query params:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `with_permissions` | `0` or `1` | `0` | Include `permissions` array in each role |

```bash
# Without permissions
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/roles" \
  -H "X-API-Key: $INTERNAL_KEY"

# With permissions expanded
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/roles?with_permissions=1" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 1,
      "name": "admin",
      "description": "Business administration",
      "permissions": [
        { "permission_id": "P1", "permission_name": "profile:read", "description": "", "created_at": 1735689600, "updated_at": 1735689600 }
      ],
      "created_at": 1735689600,
      "updated_at": 1735689600
    }
  ]
}
```

> Without `with_permissions=1`, the `"permissions"` key is omitted.

---

### 7.2 Create Role

**`POST /api/internal-v1/rbac/roles`**

**Request body:**

| Field | Type | Required |
|-------|------|----------|
| `name` | string | ✅ |
| `description` | string | ❌ |

```bash
curl -X POST {{BASE_URL}}/api/internal-v1/rbac/roles \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name":        "moderator",
    "description": "Content moderation role"
  }'
```

**Success (201):**
```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": 5,
    "name": "moderator",
    "description": "Content moderation role",
    "created_at": 1744970400,
    "updated_at": 1744970400
  }
}
```

---

### 7.3 Get Role by ID

**`GET /api/internal-v1/rbac/roles/:id`**

```bash
# Without permissions
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/roles/1" \
  -H "X-API-Key: $INTERNAL_KEY"

# With permissions
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/roles/1?with_permissions=1" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):** Same shape as a single role object in list response.

---

### 7.4 Update Role

**`PATCH /api/internal-v1/rbac/roles/:id`**

**Request body:**

| Field | Type | Required |
|-------|------|----------|
| `name` | string | ❌ |
| `description` | string | ❌ |

```bash
curl -X PATCH {{BASE_URL}}/api/internal-v1/rbac/roles/5 \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name":        "content-moderator",
    "description": "Moderates course content"
  }'
```

**Success (200):** Returns the updated role **with** `permissions` preloaded.

---

### 7.5 Set Role Permissions

**`PUT /api/internal-v1/rbac/roles/:id/permissions`**

**Full replace** — deletes all existing role permissions and inserts the provided list.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permission_ids` | string[] | ✅ | Array of `permission_id` strings (e.g. `["P1","P5","P10"]`) |

```bash
curl -X PUT {{BASE_URL}}/api/internal-v1/rbac/roles/5/permissions \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "permission_ids": ["P1", "P5", "P6", "P10"]
  }'
```

**Success (200):** Returns the role with the updated `permissions` array.

**Error examples:**
```json
// 400 — unknown permission_id
{ "code": 3001, "message": "unknown permission_id \"P99\"", "data": null }
```

---

### 7.6 Delete Role

**`DELETE /api/internal-v1/rbac/roles/:id`**

Deletes the role and all its `user_roles` and `role_permissions` assignments.

```bash
curl -X DELETE {{BASE_URL}}/api/internal-v1/rbac/roles/5 \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{ "code": 0, "message": "deleted", "data": null }
```

---

## 8. Internal RBAC — User Roles

### 8.1 List User Roles

**`GET /api/internal-v1/rbac/users/:userId/roles`**

Returns all roles assigned to the user (with permissions preloaded).

```bash
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/users/42/roles" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "id": 4,
      "name": "learner",
      "description": "Consume learning content",
      "permissions": [
        { "permission_id": "P1", "permission_name": "profile:read", ... },
        { "permission_id": "P5", "permission_name": "course:read", ... },
        { "permission_id": "P10", "permission_name": "user:read", ... }
      ],
      "created_at": 1735689600,
      "updated_at": 1735689600
    }
  ]
}
```

---

### 8.2 Assign Role to User

**`POST /api/internal-v1/rbac/users/:userId/roles`**

Idempotent — no error if the role is already assigned.

**Request body:**

| Field | Type | Required |
|-------|------|----------|
| `role_id` | uint | ✅ |

```bash
curl -X POST {{BASE_URL}}/api/internal-v1/rbac/users/42/roles \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "role_id": 3 }'
```

**Success (200):**
```json
{ "code": 0, "message": "assigned", "data": null }
```

**Error examples:**
```json
// 404 — role not found
{ "code": 3004, "message": "role not found", "data": null }
```

---

### 8.3 Remove Role from User

**`DELETE /api/internal-v1/rbac/users/:userId/roles/:roleId`**

```bash
curl -X DELETE {{BASE_URL}}/api/internal-v1/rbac/users/42/roles/3 \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{ "code": 0, "message": "removed", "data": null }
```

---

## 9. Internal RBAC — User Permissions

### 9.1 List User Effective Permissions

**`GET /api/internal-v1/rbac/users/:userId/permissions`**

Returns the **union** of permissions from all roles + direct grants (same as embedded in the JWT), wrapped as **`{ "permission_codes": [...] }`** in the envelope `data`.

```bash
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/users/42/permissions" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": { "permission_codes": ["course:read", "profile:read", "user:read"] }
}
```

---

### 9.2 List User Direct Permissions

**`GET /api/internal-v1/rbac/users/:userId/direct-permissions`**

Returns only permissions granted **directly** to the user (not via roles).

```bash
curl -X GET "{{BASE_URL}}/api/internal-v1/rbac/users/42/direct-permissions" \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{
  "code": 0,
  "message": "ok",
  "data": [
    {
      "permission_id":   "P8",
      "permission_name": "course:create",
      "description":     "Create courses",
      "created_at": 1735689600,
      "updated_at": 1735689600
    }
  ]
}
```

---

### 9.3 Assign Direct Permission to User

**`POST /api/internal-v1/rbac/users/:userId/direct-permissions`**

Idempotent — no error if the permission is already assigned. Provide **either** `permission_id` **or** `permission_name`.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permission_id` | string | ❌ | e.g. `"P8"` |
| `permission_name` | string | ❌ | e.g. `"course:create"` |

At least one of `permission_id` or `permission_name` must be provided.

```bash
# Assign by permission_id
curl -X POST {{BASE_URL}}/api/internal-v1/rbac/users/42/direct-permissions \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "permission_id": "P8" }'

# Assign by permission_name
curl -X POST {{BASE_URL}}/api/internal-v1/rbac/users/42/direct-permissions \
  -H "X-API-Key: $INTERNAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{ "permission_name": "course:create" }'
```

**Success (200):**
```json
{ "code": 0, "message": "assigned", "data": null }
```

**Error examples:**
```json
// 400 — neither field provided
{ "code": 3001, "message": "permission_id or permission_name required", "data": null }

// 404 — permission not found
{ "code": 3004, "message": "permission not found", "data": null }
```

---

### 9.4 Remove Direct Permission from User

**`DELETE /api/internal-v1/rbac/users/:userId/direct-permissions/:permissionId`**

**Path param:** `permissionId` — the `permission_id` string (e.g. `P8`)

```bash
curl -X DELETE {{BASE_URL}}/api/internal-v1/rbac/users/42/direct-permissions/P8 \
  -H "X-API-Key: $INTERNAL_KEY"
```

**Success (200):**
```json
{ "code": 0, "message": "removed", "data": null }
```

---

## 10. Deprecated APIs

No endpoints are currently marked as **deprecated**. When an endpoint is deprecated it will be listed here with the deprecation date, reason, and replacement.

---

## 11. Media API (`/api/v1/media/*`)

> **Auth:** `Authorization: Bearer <access_token>` + permissions `media_file:read` / `media_file:create` / `media_file:update` / `media_file:delete` (see `constants/permissions.go`).  
> **Module details:** `docs/modules/media.md`  
> **Size limit:** one file part per request, max **2 GiB** (`2×1024³` bytes); cap **`constants.MaxMediaUploadFileBytes`** and the default oversize **`message`** string **`constants.MsgFileTooLargeUpload`** both live in **`constants/error_msg.go`** (same literal as `pkg/errcode` `FileTooLarge` and media sentinel — see `docs/architecture.md`). Over the limit → HTTP **413**, JSON `code` **2003** (`FileTooLarge`). Nginx / edge `client_max_body_size` (or equivalent) must be **≥ 2G** on the API host. See `docs/deploy.md`.

Path param `:id` on file routes is the **object key** (Gin matches one URL segment; if your key contains `/`, encode per segment or adjust routing).

### 11.1 List files (paginated)

**`GET /api/v1/media/files`**

Query: `page`, `per_page`, `sort_by`, `sort_order`, optional `search` (filename contains match), `provider` (`S3|GCS|B2|R2|Bunny|Local`), `kind` (`FILE|VIDEO`), `category` (`image|document|video`).  
Sort whitelist: `created_at`, `updated_at`, `filename`, `size_bytes` (default `created_at`, `sort_order` default `desc`).  
`search` applies only to `filename` (`ILIKE %term%`); no ID/object-key search path is provided.  
`category` forces `kind` and applies MIME/extension filters for tabbed UIs (see `docs/modules/media.md`).

```bash
curl -X GET "{{BASE_URL}}/api/v1/media/files?page=1&per_page=20&category=image&sort_by=filename&sort_order=asc" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 11.2 Cleanup metrics

**`GET /api/v1/media/files/cleanup-metrics`**

Returns JSON `data` with counters `cleanup_cloud_deleted`, `cleanup_cloud_failed`, `cleanup_cloud_retried` (orphan cleanup job).

```bash
curl -X GET "{{BASE_URL}}/api/v1/media/files/cleanup-metrics" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 11.3 Upload file/video

**`POST /api/v1/media/files`** (multipart form-data)

Form fields: **`files`** (repeat **1–5** parts per request; legacy single **`file`** still works), optional `kind` (`FILE`/`VIDEO`), `object_key`, `metadata` (JSON string). Per-part max **2 GiB**, combined parts max **2 GiB** total — see `constants.MaxMediaUploadFileBytes`, `MaxMediaMultipartTotalBytes` (`docs/modules/media.md`).

Success envelope `data` = **array** of **`dto.UploadFileResponse`** (one per part; no **`origin_url`** — Sub 12). Bunny Stream uploads may include **`video_id`**, **`thumbnail_url`**, **`embeded_html`**, **`direct_play_url`**, **`hls_playlist_url`**, **`preview_animation_url`** when the backend populated them (`docs/modules/media.md`, `docs/return_types.md`).

```bash
curl -X POST {{BASE_URL}}/api/v1/media/files \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "files=@./sample.mp4" \
  -F "kind=VIDEO" \
  -F 'metadata={"duration":120.5}'
```

Optional second part in the same request:

```bash
curl -X POST {{BASE_URL}}/api/v1/media/files \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "files=@./a.png" \
  -F "files=@./b.png" \
  -F "kind=FILE"
```

### 11.4 Get file descriptor by object key

**`GET /api/v1/media/files/{objectKey}?kind=FILE`**

```bash
curl -G "{{BASE_URL}}/api/v1/media/files/my-key.png" \
  --data-urlencode "kind=FILE" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 11.5 Replace media object

**`PUT /api/v1/media/files/{objectKey}`** (multipart)

Optional form fields: `kind`, `metadata` (JSON string), `reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged`.

Multipart **bundle**: **1–5** `files` parts — the **first** part updates the row `{objectKey}`; **additional** parts create **new** rows. Same aggregate size limits as POST (`docs/modules/media.md`).

```bash
curl -X PUT "{{BASE_URL}}/api/v1/media/files/my-key.png" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "files=@./new-file.png" \
  -F "kind=FILE"
```

### 11.6 Batch delete media objects

**`POST /api/v1/media/files/batch-delete`** (JSON)

Body: `{ "object_keys": ["<key1>", "..."] }` — **1–10** distinct keys, no duplicates. Same permission as single delete (`media_file:delete`). Response `data` includes **`deleted_count`**.

```bash
curl -X POST "{{BASE_URL}}/api/v1/media/files/batch-delete" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"object_keys":["key-a.png","key-b.png"]}'
```

### 11.7 Delete media object

**`DELETE /api/v1/media/files/{objectKey}`**

Optional query `metadata` (JSON) for delete flow.

```bash
curl -X DELETE "{{BASE_URL}}/api/v1/media/files/my-key.png" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 11.8 Decode local token

**`GET /api/v1/media/files/local/{token}`**

```bash
curl -X GET "{{BASE_URL}}/api/v1/media/files/local/<token>" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 11.9 Video processing status

**`GET /api/v1/media/videos/{videoGuid}/status`**

`videoGuid` is the provider video id (e.g. Bunny `video_guid`). Response `data` follows provider (e.g. `status` string).

```bash
curl -X GET "{{BASE_URL}}/api/v1/media/videos/550e8400-e29b-41d4-a716-446655440000/status" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Provider is chosen by server config (`setting.MediaSetting.AppMediaProvider`), not by client payload.

---

## 12. Taxonomy (`/api/v1/taxonomy/*`)

> **Auth:** Bearer JWT. Permissions: levels `course_level:*` (P14–P17), topics `topic:*` (P18–P21), outcomes `course_outcome:*` (P30–P33), skills `course_skill:*` (P34–P37), tags `tag:*` (P22–P25).

All list endpoints support pagination query params as [Global Conventions](#1-global-conventions), plus taxonomy-specific:
- `sort_by`, `sort_desc` (boolean)
- optional `status` = `ACTIVE` | `INACTIVE`
- typed search: `search_by` + `search_value`

Allowed `search_by` values:
- Levels / Topics / Skills / Tags: `name`, `slug`
- Outcomes: `short_description`

**Soft delete (migration `000012`):** Default `GET /taxonomy/{resource}` returns only active rows (`deleted_at IS NULL`). `GET /taxonomy/{resource}/full` includes soft-deleted rows. `DELETE /taxonomy/{resource}/:id` soft-deletes; `DELETE /taxonomy/{resource}/:id/hard` permanently removes the row. Topics/outcomes hard-delete enqueues orphan image cleanup. See **`docs/patterns.md`** (CRUD soft delete).

### 12.1 Course levels

| Method | Path |
|--------|------|
| GET | `/api/v1/taxonomy/levels` |
| GET | `/api/v1/taxonomy/levels/full` |
| POST | `/api/v1/taxonomy/levels` |
| PATCH | `/api/v1/taxonomy/levels/:id` |
| DELETE | `/api/v1/taxonomy/levels/:id` |
| DELETE | `/api/v1/taxonomy/levels/:id/hard` |

**Create body:** `{ "name", "status"? }` — `status` optional (`ACTIVE` / `INACTIVE`). Slug is computed server-side from `name`.  
**Update body:** partial `{ "name"?, "status"? }` — when `name` changes, slug is recomputed server-side.

```bash
# List (optional include_images=false for picker UIs — skips image URL hydration on topics/outcomes)
curl -X GET "{{BASE_URL}}/api/v1/taxonomy/levels?page=1&per_page=20&status=ACTIVE" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -X GET "{{BASE_URL}}/api/v1/taxonomy/topics?per_page=100&include_images=false" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Create
curl -X POST {{BASE_URL}}/api/v1/taxonomy/levels \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Beginner","status":"ACTIVE"}'

# Update
curl -X PATCH {{BASE_URL}}/api/v1/taxonomy/levels/1 \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Beginner (updated)"}'

# Delete (soft)
curl -X DELETE {{BASE_URL}}/api/v1/taxonomy/levels/1 \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Hard delete
curl -X DELETE {{BASE_URL}}/api/v1/taxonomy/levels/1/hard \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 12.2 Course topics (formerly categories)

| Method | Path |
|--------|------|
| GET | `/api/v1/taxonomy/topics` |
| GET | `/api/v1/taxonomy/topics/full` |
| POST | `/api/v1/taxonomy/topics` |
| PATCH | `/api/v1/taxonomy/topics/:id` |
| DELETE | `/api/v1/taxonomy/topics/:id` |
| DELETE | `/api/v1/taxonomy/topics/:id/hard` |

**Create body:** `{ "name", "image_file_id"?, "child_topics"?, "status"? }` — slug is server-computed from `name`. Optional **`image_file_id`** (UUID from **`POST /api/v1/media/files`**). **`child_topics`** input nodes: `{ "id", "name", "children" }` (UUID ids, max depth 12, max 100 nodes; slug derived per node on server).  
**Update body:** partial fields including optional **`child_topics`** array.

```bash
curl -X POST {{BASE_URL}}/api/v1/taxonomy/topics \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Math","image_file_id":"550e8400-e29b-41d4-a716-446655440000","child_topics":[],"status":"ACTIVE"}'
```

### 12.3 Course outcomes

| Method | Path |
|--------|------|
| GET | `/api/v1/taxonomy/outcomes` |
| GET | `/api/v1/taxonomy/outcomes/full` |
| POST | `/api/v1/taxonomy/outcomes` |
| PATCH | `/api/v1/taxonomy/outcomes/:id` |
| DELETE | `/api/v1/taxonomy/outcomes/:id` |
| DELETE | `/api/v1/taxonomy/outcomes/:id/hard` |

**Create body:** `{ "short_description", "description"?, "image_file_id"?, "status"? }` — `description` is a string array (max 8 items, each ≤120 chars).

```bash
curl -X POST {{BASE_URL}}/api/v1/taxonomy/outcomes \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"short_description":"Solve linear equations","description":["Use substitution","Check solutions"],"status":"ACTIVE"}'
```

### 12.4 Course skills

| Method | Path |
|--------|------|
| GET | `/api/v1/taxonomy/skills` |
| GET | `/api/v1/taxonomy/skills/full` |
| POST | `/api/v1/taxonomy/skills` |
| PATCH | `/api/v1/taxonomy/skills/:id` |
| DELETE | `/api/v1/taxonomy/skills/:id` |
| DELETE | `/api/v1/taxonomy/skills/:id/hard` |

**Create body:** `{ "name", "children"?, "status"? }` — slug server-computed; `children` uses the same input tree shape as `child_topics`.

```bash
curl -X POST {{BASE_URL}}/api/v1/taxonomy/skills \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Problem solving","children":[],"status":"ACTIVE"}'
```

### 12.5 Tags

| Method | Path |
|--------|------|
| GET | `/api/v1/taxonomy/tags` |
| GET | `/api/v1/taxonomy/tags/full` |
| POST | `/api/v1/taxonomy/tags` |
| PATCH | `/api/v1/taxonomy/tags/:id` |
| DELETE | `/api/v1/taxonomy/tags/:id` |
| DELETE | `/api/v1/taxonomy/tags/:id/hard` |

**Create body:** `{ "name", "status"? }` — slug server-computed from `name`.

```bash
curl -X GET "{{BASE_URL}}/api/v1/taxonomy/tags?search_by=slug&search_value=react&status=ACTIVE" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

---

## 13. Instructor management

> **Auth:** Bearer token. Permissions **P41–P58** (migration **`000013`**). Full matrix: **`docs/modules/instructor.md`**, **`docs/database.md`**.

Requires `MIGRATE=1` (or applied `000013_instructor_management.up.sql`) and sync: `go run ./cmd/syncpermissions`, `go run ./cmd/syncrolepermissions`.

| Area | Examples |
|------|----------|
| Roster | `GET/POST /api/v1/instructors`, `DELETE /api/v1/instructors/:userId` |
| Applications | `GET/POST /api/v1/instructor-applications`, `POST …/:id/approve`, `POST …/:id/reject` |
| Profiles | `GET/POST /api/v1/instructor-profiles`, `GET …/me` |
| Expertise | `GET/POST /api/v1/instructors/:id/expertise/topics`, `…/skills` |
| Tickets | `GET/POST /api/v1/instructor-tickets`, `POST …/:id/close`, `GET/POST …/:id/messages` |

Identity fields for application/profile responses:
- `full_name` (user display name)
- `avatar` (hydrated URL; empty when no avatar is linked)

```bash
# List roster (admin/sysadmin — instructor_roster:read)
curl -sS "{{BASE_URL}}/api/v1/instructors?page=1&per_page=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Reject application (rejection_reason required)
curl -sS -X POST "{{BASE_URL}}/api/v1/instructor-applications/1/reject" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"rejection_reason":"Incomplete CV"}'
```

---

## 14. Course management

> **Auth:** Bearer JWT. Permissions: `course:create`, `course:update`, `course:delete`, `course_instructor:read`, `course_collaborator_candidate:read` (P67 — instructor-candidates picker), `course_review:*` (P59–P61), `course_catalog:*` / `course_trash:*` (P62–P66) for admin catalog. Full route matrix: **`docs/router.md`**, module notes: **`docs/modules/course.md`**, permission catalog: **`docs/database.md`**.

Requires migration **`000016_course_management`** (or `MIGRATE=1`) and permission sync.

### 14.1 Create course

**`POST /api/v1/courses`** — permission `course:create`

Request body: `{ "title": string }` only. Slug is computed server-side from `title` via `utils.SlugifyName` (not accepted from clients).

```bash
curl -sS -X POST "{{BASE_URL}}/api/v1/courses" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Introduction to Go"}'
```

| HTTP | `code` | Notes |
|------|--------|-------|
| 201 | 0 | `data` = `CourseDetail` (draft v1, owner collaborator, empty outline) |
| 400 | 3001 | Empty slug after slugify (`invalid slug`) |
| 401 | 3002 | Missing/invalid JWT |
| 403 | 3003 | Missing `course:create` |
| 409 | 3005 | Duplicate slug (unique constraint) |
| 500 | 9001 | Internal error |

Example success (truncated):

```json
{
  "code": 0,
  "message": "created",
  "data": {
    "course": {
      "id": 1,
      "owner_user_id": 2,
      "slug": "introduction-to-go",
      "current_draft_version_id": 1,
      "created_at": 1780823439,
      "updated_at": 1780823439
    },
    "collaborator_role": "OWNER",
    "draft_version": {
      "id": 1,
      "course_id": 1,
      "version_no": 1,
      "status": "DRAFT",
      "title": "Introduction to Go",
      "row_version": 1
    },
    "collaborators": [
      {
        "user_id": 2,
        "role": "OWNER",
        "display_name": "Alice",
        "email": "alice@example.com"
      }
    ],
    "outline": []
  }
}
```

### 14.2 List editable courses

**`GET /api/v1/courses/my`** — permission `course_instructor:read`

```bash
curl -sS "{{BASE_URL}}/api/v1/courses/my" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Success `data`: `[]CourseListItem` (courses where caller is owner or collaborator).

### 14.3 Get course detail

**`GET /api/v1/courses/:courseId`** — permission `course_instructor:read`

Optional query: `include_outline=false` — skip outline tree (faster for info/collaborators tabs; `outline` is `[]`).

```bash
curl -sS "{{BASE_URL}}/api/v1/courses/1" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Info tab (no outline)
curl -sS "{{BASE_URL}}/api/v1/courses/1?include_outline=false" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Success `data`: `CourseDetail` with `live_version`, `draft_version`, optional `last_rejection_reason` (when new draft was forked from a rejected version), collaborators, and outline for the active draft (or published when no draft) unless `include_outline=false`.

### 14.4 Update draft basic info

**`PATCH /api/v1/courses/:courseId/basic-info`** — permission `course:update`

Request body: `expected_row_version` (required, `>= 1`) plus required metadata fields (`title` ≥5 non-whitespace, server slugify; `short_description`, `about_course`, `thumbnail_file_id`, `course_level_id`, `course_topic_id`, `tag_ids`, `skill_ids`, `outcome_ids`). `preview_video_file_id` is optional.

```bash
curl -sS -X PATCH "{{BASE_URL}}/api/v1/courses/{{courseId}}/basic-info" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expected_row_version":1,"title":"Introduction to Go","short_description":"Short blurb"}'
```

Other instructor routes (outline CRUD/reorder, leases, review submit/reopen/prepare) follow the same Bearer + permission pattern — see **`docs/router.md`**. **Owner-only** in repo (not route permission): `POST …/draft/prepare`, `POST …/submit-review`, `POST …/reopen-draft` (`EDITOR` → HTTP 403 / code `3003`).

### 14.4a List collaborators (paginated)

**`GET /api/v1/courses/:courseId/collaborators`** — permission `course_instructor:read`

Query: `page`, `per_page`, optional `search` (ILIKE `display_name` / `email`). Course detail (`GET …/:courseId`) still embeds the full collaborator array via `loadCollaborators`.

```bash
curl -sS "{{BASE_URL}}/api/v1/courses/{{courseId}}/collaborators?page=1&per_page=20&search=alice" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 14.4b List instructor candidates (picker)

**`GET /api/v1/courses/:courseId/instructor-candidates`** — permission `course_collaborator_candidate:read` (P67). **Owner-only** in repository (`requireOwnerAccess`). Requires migrations **`000027`** / **`000028`** (or `MIGRATE=1`).

```bash
curl -sS "{{BASE_URL}}/api/v1/courses/{{courseId}}/instructor-candidates?page=1&per_page=20&search=go" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Add/remove collaborators: `POST` / `DELETE` under `…/collaborators` — permission `course:update` (see **`docs/router.md`**).

Requires migration **`000022_course_sub_lesson_estimated_duration`** (or `MIGRATE=1`) for `estimated_duration_ms` on sub-lessons.

### 14.5 Create / update sub-lesson (estimated duration)

**`POST /api/v1/courses/:courseId/sub-lessons`** and **`PATCH /api/v1/courses/:courseId/sub-lessons/:subLessonId`** — permission `course:update`

Outline nodes in `CourseDetail.outline` (and single-item create/update responses) include **`estimated_duration_ms`** (int64 milliseconds):

| Kind | Request `estimated_duration_ms` | Response value |
|------|----------------------------------|----------------|
| `TEXT` / `QUIZ` | Optional; persisted when `0 <= ms <= 999h` | Stored column value |
| `VIDEO` | Ignored (forced to `0` in DB) | `media_files.duration` (seconds) × 1000 |
| Lesson / section | N/A (not writable) | Sum of child resolved durations |

**TEXT example** — 1h 25m 30s = `5130000` ms:

```bash
curl -sS -X POST "{{BASE_URL}}/api/v1/courses/{{courseId}}/sub-lessons" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lesson_id": "{{lessonUuid}}",
    "title": "Reading assignment",
    "kind": "TEXT",
    "is_preview": false,
    "estimated_duration_ms": 5130000,
    "text": { "content_delta": "[{\"insert\":\"Read chapter 1\\n\"}]" }
  }'
```

**VIDEO example** — omit `estimated_duration_ms`; response resolves from linked media:

```bash
curl -sS -X POST "{{BASE_URL}}/api/v1/courses/{{courseId}}/sub-lessons" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lesson_id": "{{lessonUuid}}",
    "title": "Intro video",
    "kind": "VIDEO",
    "is_preview": true,
    "video": { "media_file_id": "{{videoMediaUuid}}" }
  }'
```

Example outline fragment in `GET /api/v1/courses/:courseId` (truncated):

```json
{
  "outline": [
    {
      "id": "019e…",
      "title": "Module 1",
      "estimated_duration_ms": 5130000,
      "lessons": [
        {
          "id": "019e…",
          "title": "Lesson 1",
          "estimated_duration_ms": 5130000,
          "sub_lessons": [
            {
              "id": "019e…",
              "kind": "TEXT",
              "estimated_duration_ms": 5130000
            }
          ]
        }
      ]
    }
  ]
}
```

Full semantics: **`docs/modules/course.md`** → Estimated duration.

### 14.6 Sysadmin course catalog + trash (`/api/v1/course-admin`)

Requires migrations **`000023_course_trash`** and **`000024_course_admin_permissions`** (or `MIGRATE=1`). Granular permissions **P59–P66** (not shell `admin:modify`). Use `Authorization: Bearer $ACCESS_TOKEN` from `POST /api/v1/auth/login` with an account granted the required permissions.

**List approved published courses** (excludes trash and draft-only courses):

```bash
curl -sS '{{BASE_URL}}/api/v1/course-admin/courses' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Accept: application/json'
```

**List trashed courses:**

```bash
curl -sS '{{BASE_URL}}/api/v1/course-admin/courses/trash' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Accept: application/json'
```

**Move to trash** (eligible: published `APPROVED`, draft not `REJECTED`):

```bash
curl -sS -X POST '{{BASE_URL}}/api/v1/course-admin/courses/{{courseId}}/trash' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Accept: application/json'
```

**Restore from trash:**

```bash
curl -sS -X POST '{{BASE_URL}}/api/v1/course-admin/courses/{{courseId}}/restore' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Accept: application/json'
```

**Permanent delete** (course must already be in trash):

```bash
curl -sS -X DELETE '{{BASE_URL}}/api/v1/course-admin/courses/{{courseId}}/permanent' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Accept: application/json'
```

Pass: HTTP **200**, `code: 0` on success; **404** when course not found; **409** / business code when trash eligibility fails. Trashed courses return **404** on `GET /api/v1/courses/:courseId` (edit blocked).

Learner catalog/progress routes live under **`/api/v1/learner-courses/*`** (`course:read`).

---

## 15. Webhooks (`/api/v1/webhook/*`)

> **Auth:** none (public URL — protect at edge / signature if you add it). Mounted on the **no-filter** v1 group (same CORS as app).

### 15.1 Bunny Stream video webhook

**`POST /api/v1/webhook/bunny`**

JSON body:

| Field | Type | Required |
|-------|------|----------|
| `video_library_id` | string | yes |
| `video_guid` | string | yes |
| `status` | int | yes |

On **finished** status the service refreshes persisted **`media_files`** duration/metadata and Bunny delivery fields **`video_id`**, **`thumbnail_url`**, **`embeded_html`**, **`direct_play_url`**, **`hls_playlist_url`**, **`preview_animation_url`** via `GetBunnyVideoByID` + `ApplyBunnyStreamFileColumns` (see `docs/modules/media.md`).

```bash
curl -X POST {{BASE_URL}}/api/v1/webhook/bunny \
  -H "Content-Type: application/json" \
  -d '{"video_library_id":"lib-1","video_guid":"vid-uuid","status":3}'
```

---

## 16. Error Code Reference

| HTTP | `code` | Constant | Meaning |
|------|--------|----------|---------|
| 200 | 0 | `Success` | Operation completed successfully |
| 400 | 1001 | `InvalidJSON` | Request body is not valid JSON |
| 400 | 2001 | `ValidationFailed` | Request validation failed |
| 400 | 2002 | `ValidationField` | Per-field validation error |
| 400 | 3001 | `BadRequest` | Bad request (generic) |
| 401 | 3002 | `Unauthorized` | Missing or invalid JWT / system token |
| 403 | 3003 | `Forbidden` | Authenticated but lacking required permission |
| 404 | 3004 | `NotFound` | Resource not found |
| 409 | 3005 | `Conflict` | Duplicate resource (e.g. email already exists) |
| 429 | 3006 | `TooManyRequests` | Rate limit exceeded |
| 503 | 9018 | `ServiceUnavailable` | Circuit breaker open / service degraded |
| 400 | 4001 | `EmailAlreadyExists` | Email address already registered |
| 401 | 4002 | `InvalidCredentials` | Wrong email/password or system credentials |
| 400 | 4003 | `WeakPassword` | Password does not meet strength requirements |
| 401 | 4004 | `EmailNotConfirmed` | Attempted login before email confirmation |
| 403 | 4005 | `UserDisabled` | Account has been disabled |
| 403 | 4012 | `UserBanned` | Account temporarily banned (`banned_until > now()`) |
| 400 | 4006 | `InvalidConfirmToken` | Invalid or already-used confirmation token |
| 401 | 4007 | `InvalidSession` | Session ID not found or refresh token UUID mismatch |
| 401 | 4008 | `RefreshTokenExpired` | DB-stored `refresh_token_expired` has passed |
| 500 | 9001 | `InternalError` | Internal server error |
| 500 | 9998 | `Panic` | Unhandled panic caught by recovery middleware |
| 500 | 9999 | `Unknown` | Unknown error |

### Special Response Headers

| Header | Direction | Set when |
|--------|-----------|----------|
| `X-Token-Expired: true` | Response | 401 caused by a **cryptographically expired** access JWT only (NOT missing/invalid token) |

### Token Expired Client Flow

```
1. Client sends request with expired access token
2. Server responds: 401 + X-Token-Expired: true
3. Client calls POST /api/v1/auth/refresh (X-Refresh-Token + X-Session-Id headers)
4a. Success → store new tokens, retry original request
4b. 401/403 → force re-login (clear all tokens)
```

---

## 17. Postman / API Dog

> **Import vào Apidog:** (1) [`docs/api-dog-import.json`](./api-dog-import.json) — **Import → Postman**; hoặc (2) [`docs/api_swagger.yaml`](./api_swagger.yaml) — **Import → OpenAPI** (script lưu token nằm trong `x-postman-event` trên Login / Confirm / Refresh / System login). Sau khi gọi **Login** / **Confirm** / **Refresh**, post-processor tự ghi `ACCESS_TOKEN`, `REFRESH_TOKEN`, `SESSION_ID` (chọn **Environment** trước). Sinh lại collection Postman: `ruby scripts/generate-apidog-postman.rb` (đọc script từ swagger, không hardcode trong Ruby).

### Environment Variables

Create an environment with the following variables:

```json
{
  "BASE_URL":       "http://localhost:8080",
  "ACCESS_TOKEN":   "",
  "REFRESH_TOKEN":  "",
  "SESSION_ID":     "",
  "SYSTEM_TOKEN":   "",
  "INTERNAL_KEY":   "your-internal-api-key"
}
```

### Collection Variables Script (Post-Login)

Add this as a **Test script** on the login and confirm endpoints to automatically capture tokens:

```javascript
// Postman Test script — runs after login / confirm / refresh
const data = pm.response.json().data;
if (data && data.access_token) {
    pm.environment.set("ACCESS_TOKEN",  data.access_token);
    pm.environment.set("REFRESH_TOKEN", data.refresh_token);
    pm.environment.set("SESSION_ID",    data.session_id);
}
```

### Token Refresh Pre-request Script

Add to any authenticated request to auto-refresh when needed:

## 18. Local smoke test (migrations `000011` + `000013` + expertise `000017`)

Run on **`http://localhost:8080`** only after `MIGRATE=1` (or manual SQL) and `go run .`. Expect `users.created_at` = `bigint` in Postgres (**`000011`**) — see **`docs/deploy.md`** (Troubleshooting). For instructor APIs, **`000013`** must be applied and permissions synced.

**1. Login** — use a local account that exists in your dev database:

```bash
curl -sS -X POST 'http://localhost:8080/api/v1/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"Str0ng!pw","remember_me":false}'
```

Pass: HTTP **200**, `code: 0`, `data.access_token` present (no **500** scan error).

```bash
export ACCESS_TOKEN="<paste access_token from response>"
```

**2. Taxonomy outcomes**

```bash
curl -sS 'http://localhost:8080/api/v1/taxonomy/outcomes?page=1&per_page=20&sort_desc=false&status=ACTIVE' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H 'Accept: application/json'
```

Pass: HTTP **200**, `code: 0`; each item’s `created_at` / `updated_at` are JSON **numbers** (Unix seconds).

**3. Instructor roster** (admin/sysadmin token with P41)

```bash
curl -sS 'http://localhost:8080/api/v1/instructors?page=1&per_page=20' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H 'Accept: application/json'
```

Pass: HTTP **200**, `code: 0`, paginated `data` array (or empty list).

**4. Instructor expertise topics** (same token; replace `:id` with instructor `user_id`, e.g. `14`)

```bash
curl -sS 'http://localhost:8080/api/v1/instructors/14/expertise/topics' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H 'Accept: application/json'
```

Pass: HTTP **200**, `code: 0`; each item has snake_case `topic_id`, joined `name`, `slug` (not PascalCase `TopicID`). POST add:

```bash
curl -sS -X POST 'http://localhost:8080/api/v1/instructors/14/expertise/topics' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"topic_id":7}'
```

Pass: HTTP **200**, `code: 0`, `data.topic_id` present. If **500** with `course_topic_id` NOT NULL, apply migration **`000017`** (`MIGRATE=1`).

**5. Instructor profile by user_id** (replace `14` with target instructor)

```bash
curl -sS 'http://localhost:8080/api/v1/instructor-profiles/14' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H 'Accept: application/json'
```

Pass: HTTP **404** `code: 3004` when no profile yet (not **500**). If **500** with `column ip.id does not exist` or ambiguous `deleted_at`, apply migration **`000019`**.

---

## Postman — auto-refresh script

```javascript
// Postman Pre-request Script — auto-refresh on expired token
const expiresAt = pm.environment.get("TOKEN_EXPIRES_AT");
const now = Date.now() / 1000;

if (!expiresAt || now > parseInt(expiresAt) - 60) {
    const refreshToken = pm.environment.get("REFRESH_TOKEN");
    const sessionId    = pm.environment.get("SESSION_ID");
    const baseUrl      = pm.environment.get("BASE_URL");

    pm.sendRequest({
        url:    baseUrl + "/api/v1/auth/refresh",
        method: "POST",
        header: {
            "X-Refresh-Token": refreshToken,
            "X-Session-Id":    sessionId
        }
    }, (err, res) => {
        if (!err && res.code === 200) {
            const d = res.json().data;
            pm.environment.set("ACCESS_TOKEN",  d.access_token);
            pm.environment.set("REFRESH_TOKEN", d.refresh_token);
            pm.environment.set("TOKEN_EXPIRES_AT", String(Math.floor(Date.now()/1000) + 900));
        }
    });
}
```
