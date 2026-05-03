# MyCourse Backend — Return Types


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

> This document catalogues the **Go return types** of every service function and the **JSON response shapes** of every API endpoint.  
> **Last updated:** 2026-04-30

---

## Table of Contents

1. [Response Envelope](#1-response-envelope)
2. [Core Types (Models / DTOs)](#2-core-types-models--dtos)
3. [Service Layer Return Types](#3-service-layer-return-types)
   - [services/auth.go](#servicesauthgo)
   - [services/rbac.go](#servicesrbacgo)
   - [services/system.go](#servicessystemgo)
4. [API Layer Return Types](#4-api-layer-return-types)
   - [Public API `/api/v1`](#public-api-apiv1)
   - [Media API `/api/v1/media/files`](#media-api-apiv1mediafiles)
   - [System API `/api/system`](#system-api-apisystem)
   - [Internal API `/api/internal-v1`](#internal-api-apiinternal-v1)
5. [Error Response Type](#5-error-response-type)

**Test code layout:** module-level / integration tests live under repository root **`tests/`** — see `tests/README.md` and root `README.md` (**Testing**).

---

## 1. Response Envelope

All API endpoints (except `/health`) return the same JSON envelope shape:

```go
// pkg/response/response.go
type Response struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}
```

```json
{
  "code":    0,
  "message": "ok",
  "data":    <value>
}
```

**Health endpoint** (`GET /api/v1/health`):

```go
type HealthResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Status  string `json:"status"`
}
```

```json
{ "code": 0, "message": "ok", "status": "ok" }
```

**Paginated data** — when `data` is a list:

```go
type PaginatedData struct {
    Result   any      `json:"result"`
    PageInfo PageInfo `json:"page_info"`
}

type PageInfo struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    TotalPages int `json:"total_pages"`
    TotalItems int `json:"total_items"`
}
```

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "result": [ ... ],
    "page_info": {
      "page": 1,
      "per_page": 20,
      "total_pages": 5,
      "total_items": 98
    }
  }
}
```

---

## 2. Core Types (Models / DTOs)

### `models.User`

```go
// models/user.go
type User struct {
    ID                  uint                   `json:"id"`
    UserCode            string                 `json:"user_code"`       // UUIDv7
    Email               string                 `json:"email"`
    HashPassword        string                 `json:"-"`               // never serialized
    DisplayName         string                 `json:"display_name"`
    AvatarURL           string                 `json:"avatar_url"`
    IsDisable           bool                   `json:"is_disable"`
    EmailConfirmed      bool                   `json:"email_confirmed"`
    ConfirmationToken   *string                `json:"-"`               // never serialized
    ConfirmationSentAt  *time.Time             `json:"-"`               // never serialized
    RefreshTokenSession RefreshTokenSessionMap `json:"-"`               // never serialized
    CreatedAt           time.Time              `json:"created_at"`
    UpdatedAt           time.Time              `json:"updated_at"`
    DeletedAt           gorm.DeletedAt         `json:"deleted_at,omitempty"`
}
```

### `dto.MeResponse`

Returned by `GET /api/v1/me`:

```go
// dto/auth.go
type MeResponse struct {
    UserID         uint     `json:"user_id"`
    UserCode       string   `json:"user_code"`       // UUIDv7 string
    Email          string   `json:"email"`
    DisplayName    string   `json:"display_name"`
    AvatarURL      string   `json:"avatar_url"`
    EmailConfirmed bool     `json:"email_confirmed"`
    IsDisabled     bool     `json:"is_disabled"`
    CreatedAt      int64    `json:"created_at"`      // Unix epoch seconds
    Permissions    []string `json:"permissions"`     // sorted permission_name strings
}
```

**Example JSON:**

```json
{
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
```

### `services.TokenPairResult`

Returned by `Login`, `ConfirmEmail`, `RefreshSession`:

```go
// services/auth.go
type TokenPairResult struct {
    AccessToken  string        // HS256 JWT, TTL=15min
    RefreshToken string        // HS256 JWT, TTL=30d/14d
    SessionStr   string        // 128-char hex string (session_id)
    RefreshTTL   time.Duration // used to compute cookie MaxAge
}
```

> `RefreshTTL` is **not** serialized in the API response — it is only used for cookie `MaxAge`.

### `models.Permission`

```go
// models/rbac.go
type Permission struct {
    PermissionID   string    `json:"permission_id"`   // e.g. "P1"
    PermissionName string    `json:"permission_name"` // e.g. "profile:read"
    Description    string    `json:"description"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

### `models.Role`

```go
// models/rbac.go
type Role struct {
    ID          uint         `json:"id"`
    Name        string       `json:"name"`
    Description string       `json:"description"`
    Permissions []Permission `json:"permissions,omitempty"` // populated when with_permissions=1
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
}
```

### `models.UserRole`

```go
type UserRole struct {
    UserID uint `json:"user_id"`
    RoleID uint `json:"role_id"`
}
```

### `models.UserPermission`

```go
type UserPermission struct {
    UserID       uint       `json:"user_id"`
    PermissionID string     `json:"permission_id"`
    Permission   Permission `json:"permission,omitempty"`
}
```

### `models.RefreshSessionEntry`

Stored as JSONB in `users.refresh_token_session`:

```go
type RefreshSessionEntry struct {
    RefreshTokenUUID    string    `json:"refresh_token_uuid"`
    RememberMe          bool      `json:"remember_me"`
    RefreshTokenExpired time.Time `json:"refresh_token_expired"`
}
```

### `models.SystemAppConfig`

```go
// models/system_app_config.go
type SystemAppConfig struct {
    ID                    int       // always 1
    AppCLISystemPassword  string
    AppSystemEnv          string
    AppTokenEnv           string
    UpdatedAt             time.Time
}
```

### `dto.RegisterRequest`

```go
type RegisterRequest struct {
    Email       string `json:"email"        binding:"required,email"`
    Password    string `json:"password"     binding:"required"`
    DisplayName string `json:"display_name" binding:"required,min=1,max=255"`
}
```

### `dto.LoginRequest`

```go
type LoginRequest struct {
    Email      string `json:"email"       binding:"required,email"`
    Password   string `json:"password"    binding:"required"`
    RememberMe bool   `json:"remember_me"`
}
```

### `dto.SystemLoginRequest`

```go
type SystemLoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}
```

### `dto.PermissionFilter` (query params)

```go
type PermissionFilter struct {
    BaseFilter // page, per_page, sort_by, sort_order, search_by, search_data
}
```

### `dto.BaseFilter`

```go
type BaseFilter struct {
    Page       int    `form:"page"`
    PerPage    int    `form:"per_page"`
    SortBy     string `form:"sort_by"`
    SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
    SearchBy   string `form:"search_by"`
    SearchData string `form:"search_data"`
}
```

### `dto.CreatePermissionRequest`

```go
type CreatePermissionRequest struct {
    PermissionID   string `json:"permission_id"   binding:"required,min=1,max=10"`
    PermissionName string `json:"permission_name" binding:"required,min=1,max=50"`
    Description    string `json:"description"     binding:"omitempty,max=512"`
}
```

### `dto.UpdatePermissionRequest`

```go
type UpdatePermissionRequest struct {
    PermissionID   *string `json:"permission_id"   binding:"omitempty,min=1,max=10"`
    PermissionName *string `json:"permission_name" binding:"omitempty,min=1,max=50"`
    Description    *string `json:"description"`
}
```

### `dto.CreateRoleRequest`

```go
type CreateRoleRequest struct {
    Name        string `json:"name"        binding:"required"`
    Description string `json:"description"`
}
```

### `dto.UpdateRoleRequest`

```go
type UpdateRoleRequest struct {
    Name        *string `json:"name"`
    Description *string `json:"description"`
}
```

### `dto.SetRolePermissionsRequest`

```go
type SetRolePermissionsRequest struct {
    PermissionIDs []string `json:"permission_ids"`
}
```

### `dto.AssignUserRoleRequest`

```go
type AssignUserRoleRequest struct {
    RoleID uint `json:"role_id" binding:"required"`
}
```

### `dto.AssignUserPermissionRequest`

```go
type AssignUserPermissionRequest struct {
    PermissionID   *string `json:"permission_id"   binding:"omitempty,max=10"`
    PermissionName *string `json:"permission_name" binding:"omitempty,max=50"`
}
```

---

## 3. Service Layer Return Types

### services/auth.go

| Function | Signature | Return Types |
|----------|-----------|--------------|
| `Register` | `Register(email, password, displayName string) error` | `nil` on success; `ErrWeakPassword`, `ErrEmailAlreadyExists`, or a DB/email error |
| `Login` | `Login(email, password string, rememberMe bool) (TokenPairResult, error)` | `TokenPairResult` on success; `ErrInvalidCredentials`, `ErrEmailNotConfirmed`, `ErrUserDisabled`, or DB error |
| `ConfirmEmail` | `ConfirmEmail(confirmToken string) (TokenPairResult, error)` | `TokenPairResult` on success; `ErrInvalidConfirmToken` or DB error |
| `RefreshSession` | `RefreshSession(sessionStr, refreshTokenStr string) (TokenPairResult, error)` | `TokenPairResult` on success; `ErrInvalidSession`, `ErrUserNotFound`, `ErrUserDisabled`, `ErrRefreshTokenExpired`, or DB error |
| `GetMe` | `GetMe(userID uint) (*dto.MeResponse, error)` | `*dto.MeResponse` on success; `ErrUserNotFound` or DB error |
| `issueTokenPair` _(internal)_ | `issueTokenPair(user User, rememberMe bool, refreshTTL time.Duration) (TokenPairResult, error)` | `TokenPairResult` or error |
| `buildMeResponseFromUser` _(internal)_ | `buildMeResponseFromUser(user User) (*dto.MeResponse, error)` | `*dto.MeResponse` or error from `PermissionCodesForUser` |
| `userPermissionSlice` _(internal)_ | `userPermissionSlice(userID uint) ([]string, error)` | Sorted `[]string` of `permission_name` values |

**Sentinel errors:**

```go
var (
    ErrEmailAlreadyExists  = errors.New("email already registered")
    ErrInvalidCredentials  = errors.New("invalid email or password")
    ErrWeakPassword        = errors.New("password does not meet requirements")
    ErrEmailNotConfirmed   = errors.New("email not confirmed")
    ErrUserDisabled        = errors.New("user account is disabled")
    ErrInvalidConfirmToken = errors.New("invalid or expired confirmation token")
    ErrUserNotFound        = errors.New("user not found")
    ErrInvalidSession      = errors.New("invalid session")
    ErrRefreshTokenExpired = errors.New("refresh token expired")
)
```

---

### services/rbac.go

| Function | Signature | Return Types |
|----------|-----------|--------------|
| `PermissionCodesForUser` | `PermissionCodesForUser(userID uint) (map[string]struct{}, error)` | `map[string]struct{}` keyed by `permission_name`; or DB error |
| `UserHasAllPermissions` | `UserHasAllPermissions(userID uint, requiredActions []string) (bool, string, error)` | `(true, "", nil)` if all granted; `(false, missingAction, nil)` if one is missing; `(false, "", err)` on DB error |
| `ListPermissions` | `ListPermissions(p ListPermissionsParams) ([]models.Permission, int64, error)` | `[]Permission` (current page), `int64` (total count), error |
| `CreatePermission` | `CreatePermission(permissionID, permissionName, description string) (*models.Permission, error)` | `*Permission` on success, or validation/DB error |
| `UpdatePermission` | `UpdatePermission(permissionID string, newPermissionID, permissionName, description *string) (*models.Permission, error)` | `*Permission` on success; `gorm.ErrRecordNotFound` if ID not found; validation/DB error |
| `DeletePermission` | `DeletePermission(permissionID string) error` | `nil` on success, DB error |
| `ListRoles` | `ListRoles(withPerms bool) ([]models.Role, error)` | `[]Role`; roles include `Permissions []Permission` when `withPerms=true` |
| `GetRole` | `GetRole(id uint, withPerms bool) (*models.Role, error)` | `*Role`; `gorm.ErrRecordNotFound` if not found |
| `CreateRole` | `CreateRole(name, description string) (*models.Role, error)` | `*Role` on success, validation/DB error |
| `UpdateRole` | `UpdateRole(id uint, name, description *string) (*models.Role, error)` | `*Role` (with permissions preloaded); `gorm.ErrRecordNotFound` if not found |
| `DeleteRole` | `DeleteRole(id uint) error` | `nil` on success, DB error |
| `SetRolePermissions` | `SetRolePermissions(roleID uint, permissionIDs []string) (*models.Role, error)` | `*Role` (with permissions); `gorm.ErrRecordNotFound` if role not found; error on unknown permission ID |
| `ListUserRoles` | `ListUserRoles(userID uint) ([]models.Role, error)` | `[]Role` (with permissions preloaded), or DB error |
| `AssignUserRole` | `AssignUserRole(userID, roleID uint) error` | `nil` (idempotent); `gorm.ErrRecordNotFound` if role not found |
| `RemoveUserRole` | `RemoveUserRole(userID, roleID uint) error` | `nil` on success, DB error |
| `ListUserDirectPermissions` | `ListUserDirectPermissions(userID uint) ([]models.Permission, error)` | `[]Permission` (direct grants only) |
| `AssignUserPermission` | `AssignUserPermission(userID uint, permissionID string) error` | `nil` (idempotent); `gorm.ErrRecordNotFound` if permission not found |
| `AssignUserPermissionByPermissionName` | `AssignUserPermissionByPermissionName(userID uint, permissionName string) error` | `nil`; `gorm.ErrRecordNotFound` if `permission_name` not found |
| `RemoveUserPermission` | `RemoveUserPermission(userID uint, permissionID string) error` | `nil` on success, DB error |

**`ListPermissionsParams`:**

```go
type ListPermissionsParams struct {
    Offset     int
    Limit      int
    SortBy     string
    SortOrder  string // "asc" | "desc"
    SearchBy   string
    SearchData string
}
```

---

### services/system.go

| Function | Signature | Return Types |
|----------|-----------|--------------|
| `GetSystemAppConfig` | `GetSystemAppConfig(db *gorm.DB) (*models.SystemAppConfig, error)` | `*SystemAppConfig`; `ErrSystemAppConfigMissing` if row `id=1` absent |
| `RegisterSystemPrivilegedUser` | `RegisterSystemPrivilegedUser(db *gorm.DB, username, password string) error` | `nil` on success; `ErrSystemSecretsNotReady` if `app_system_env` not set |
| `SystemLogin` | `SystemLogin(db *gorm.DB, username, password string) (accessToken string, err error)` | `string` (JWT) on success; `ErrSystemLoginFailed`, `ErrSystemSecretsNotReady`, or DB error |
| `VerifySystemAccessToken` | `VerifySystemAccessToken(db *gorm.DB, tokenStr string) error` | `nil` if valid; JWT parse error or `ErrSystemSecretsNotReady` |

**Sentinel errors:**

```go
var (
    ErrSystemAppConfigMissing = errors.New("system_app_config row missing")
    ErrSystemSecretsNotReady  = errors.New("system secrets are not configured in database")
    ErrSystemLoginFailed      = errors.New("invalid system credentials")
)
```

---

### services/cache/auth_user.go

All cache functions return nothing on error (graceful no-op):

| Function | Signature | Return Types |
|----------|-----------|--------------|
| `GetCachedUserMe` | `GetCachedUserMe(ctx, userID uint) (*dto.MeResponse, bool)` | `(*MeResponse, true)` on hit; `(nil, false)` on miss/error |
| `SetCachedUserMe` | `SetCachedUserMe(ctx, me *dto.MeResponse)` | none |
| `GetCachedLoginUserID` | `GetCachedLoginUserID(ctx, normEmail string) (uint, bool)` | `(uid, true)` on hit; `(0, false)` on miss |
| `SetCachedLoginUserID` | `SetCachedLoginUserID(ctx, normEmail string, userID uint)` | none |
| `LoginInvalidCached` | `LoginInvalidCached(ctx, normEmail string) bool` | `true` if negative cache hit |
| `SetLoginInvalidCache` | `SetLoginInvalidCache(ctx, normEmail string)` | none |
| `DelLoginInvalidCache` | `DelLoginInvalidCache(ctx, normEmail string)` | none |
| `NormalizeLoginEmail` | `NormalizeLoginEmail(email string) string` | lowercased email string |

---

### models/user.go

| Function | Signature | Return Types |
|----------|-----------|--------------|
| `SaveRefreshSession` | `SaveRefreshSession(userID uint, sessionStr string, entry RefreshSessionEntry) error` | `nil` on success, DB error |
| `AddRefreshSession` | `AddRefreshSession(userID uint, sessionStr string, entry RefreshSessionEntry) error` | `nil` on success; DB error; evicts oldest session if cap (5) is reached, inside a transaction |

---

## 4. API Layer Return Types

All endpoints return `application/json`. The outer envelope is always `Response` or `HealthResponse` (see section 1).

### Public API `/api/v1`

---

#### `GET /api/v1/health`

| Status | `code` | `data` shape |
|--------|--------|-------------|
| 200 | 0 | N/A — uses `HealthResponse` with `"status": "ok"` |

```json
{ "code": 0, "message": "ok", "status": "ok" }
```

---

#### `POST /api/v1/auth/register`

**Request body:** `dto.RegisterRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 201 | 0 | `null` |
| 400 | 2001 | `null` — validation failed |
| 400 | 4003 | `null` — weak password |
| 409 | 4001 | `null` — email exists |
| 500 | 9001 | `null` |

**Success response:**

```json
{ "code": 0, "message": "registration_success", "data": null }
```

**Also sets cookies:** `access_token`, `refresh_token`, `session_id` — **NOT** set on `/register`. Cookies are set only on login, confirm, and refresh.

---

#### `POST /api/v1/auth/login`

**Request body:** `dto.LoginRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ access_token: string, refresh_token: string, session_id: string }` |
| 400 | 2001 | `null` |
| 401 | 4002 | `null` — invalid credentials |
| 401 | 4004 | `null` — email not confirmed |
| 403 | 4005 | `null` — account disabled |
| 500 | 9001 | `null` |

**Success response:**

```json
{
  "code": 0,
  "message": "login_success",
  "data": {
    "access_token":  "<HS256 JWT string>",
    "refresh_token": "<HS256 JWT string>",
    "session_id":    "<128-char hex string>"
  }
}
```

**Also sets cookies:** `access_token` (MaxAge = 900s), `refresh_token` (MaxAge = refresh TTL seconds), `session_id` (MaxAge = refresh TTL seconds). All are non-HttpOnly, SameSite=Lax.

---

#### `GET /api/v1/auth/confirm`

**Query params:** `token=<uuid>`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ access_token, refresh_token, session_id }` |
| 400 | 3001 | `null` — missing token param |
| 400 | 4006 | `null` — invalid/expired token |
| 500 | 9001 | `null` |

**Success response:** same shape as login (`email_confirmed` in message):

```json
{
  "code": 0,
  "message": "email_confirmed",
  "data": {
    "access_token":  "<JWT>",
    "refresh_token": "<JWT>",
    "session_id":    "<128-hex>"
  }
}
```

**Also sets cookies:** same as login.

---

#### `POST /api/v1/auth/refresh`

**Request headers:** `X-Refresh-Token: <jwt>`, `X-Session-Id: <128-hex>`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ access_token, refresh_token, session_id }` |
| 400 | 3001 | `null` — missing headers |
| 401 | 4007 | `null` — invalid session |
| 401 | 4008 | `null` — refresh expired |
| 403 | 4005 | `null` — account disabled |
| 500 | 9001 | `null` |

**Success response:**

```json
{
  "code": 0,
  "message": "token_refreshed",
  "data": {
    "access_token":  "<new JWT>",
    "refresh_token": "<new JWT>",
    "session_id":    "<same 128-hex>"
  }
}
```

> `session_id` is **unchanged** on rotation.

---

#### `GET /api/v1/me`

**Auth:** `Authorization: Bearer <access_token>` (required)

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `dto.MeResponse` object |
| 401 | 3002 | `null` — missing/invalid/expired JWT |
| 404 | 3004 | `null` — user not found |
| 500 | 9001 | `null` |

**Success `data` shape (`dto.MeResponse`):**

```json
{
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
```

> `created_at` is a Unix epoch **integer** (seconds), not an ISO string.

---

#### `GET /api/v1/me/permissions`

**Auth:** Bearer JWT + `user:read` permission

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `string[]` — sorted permission codes |
| 401 | 3002 | `null` |
| 403 | 3003 | `null` — missing `user:read` |
| 500 | 9001 | `null` |

**Success response:**

```json
{
  "code": 0,
  "message": "ok",
  "data": ["course:read", "profile:read", "user:read"]
}
```

---

### Media API `/api/v1/media/files`

Media endpoints are implemented and return the standard envelope. Public payloads are mapped to `dto.UploadFileResponse`.

**Upload errors (create/replace):**

| Condition | HTTP status | `code` | Typical `message` |
|-----------|-------------|--------|-------------------|
| Missing multipart field `file` | 400 | `3001` (`BadRequest`) | `file is required (multipart field: file)` |
| Single file exceeds **2 GiB** (`constants.MaxMediaUploadFileBytes` in `constants/error_msg.go`) | 413 | `2003` (`FileTooLarge`) | `errcode.DefaultMessage(FileTooLarge)` = **`constants.MsgFileTooLargeUpload`** (single literal; also used for `pkg/errors.ErrFileExceedsMaxUploadSize`) |

| Endpoint | Success `data` |
|----------|-----------------|
| `POST /api/v1/media/files` | `dto.UploadFileResponse` |
| `GET /api/v1/media/files/:id` | `dto.UploadFileResponse` |
| `PUT /api/v1/media/files/:id` | `dto.UploadFileResponse` |
| `DELETE /api/v1/media/files/:id` | `null` |
| `GET /api/v1/media/files` | `[]dto.UploadFileResponse` (current placeholder may be empty) |
| `GET /api/v1/media/files/local/:token` | `{ "object_key": string }` |

`dto.UploadFileResponse`:

Bunny Stream responses may include **`video_id`**, **`thumbnail_url`**, **`embeded_html`** (JSON spelling; `omitempty` when empty). Full behaviour: **`docs/modules/media.md`** (Sub 09).

**Sub 12:** Public media responses **do not include** `origin_url` (`dto.UploadFileResponse` has no such field). Canonical storage URL exists only server-side (`media_files.origin_url`, `entities.File.OriginURL` with `json:"-"`).

```json
{
  "url": "https://...",
  "object_key": "path/to/object",
  "bunny_video_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
  "bunny_library_id": "123456",
  "video_id": "987654321",
  "thumbnail_url": "https://...",
  "embeded_html": "<iframe src=\"https://iframe.mediadelivery.net/embed/123456/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\" ...></iframe>",
  "metadata": {
    "size_bytes": 12345,
    "width_bytes": 1920,
    "height_bytes": 1080,
    "mime_type": "video/mp4",
    "extension": ".mp4",
    "duration_seconds": 157.8,
    "bitrate": 0,
    "fps": 29.97,
    "video_codec": "",
    "audio_codec": "",
    "has_audio": false,
    "is_hdr": false,
    "page_count": 0,
    "has_password": false,
    "archive_entries": 0
  }
}
```

Media metadata is inferred in backend and returned with typed contract `UploadFileMetadata` (not `any`), with zero-value defaults when a field cannot be extracted.

Provider is selected by server config (`setting.MediaSetting.AppMediaProvider`) and is not exposed as client-controlled request field.
For multipart create/update, client-sent `kind` and `metadata` text fields are parsed only for backward-compat validation and ignored by business flow (server-owned policy).

---

### System API `/api/system`

**Rate limit:** 10 requests / 3 seconds per IP.  
All routes except `/login` require `Authorization: Bearer <system_token>`.

---

#### `POST /api/system/login`

**Request body:** `dto.SystemLoginRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ access_token: string, expires_in: number }` |
| 400 | 2001 | `null` — validation |
| 401 | 4002 | `null` — invalid credentials |
| 503 | 9001 | `null` — secrets not configured |
| 500 | 9001 | `null` |

**Success response:**

```json
{
  "code": 0,
  "message": "system_login_ok",
  "data": {
    "access_token": "<system JWT>",
    "expires_in":   3600
  }
}
```

> `expires_in` is in **seconds** (`systemauth.SystemAccessTokenTTL`).

---

#### `POST /api/system/permission-sync-now`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ synced: number }` |
| 401/403 | 3002/3003 | `null` — no system token |
| 500 | 9001 | `null` |

```json
{ "code": 0, "message": "permission_sync_completed", "data": { "synced": 13 } }
```

---

#### `POST /api/system/role-permission-sync-now`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `{ rows: number }` |
| 500 | 9001 | `null` |

```json
{ "code": 0, "message": "role_permission_sync_completed", "data": { "rows": 32 } }
```

---

#### `POST /api/system/create-permission-sync-job`

```json
{ "code": 0, "message": "permission_sync_job_started", "data": null }
```

---

#### `POST /api/system/create-role-permission-sync-job`

```json
{ "code": 0, "message": "role_permission_sync_job_started", "data": null }
```

---

#### `POST /api/system/delete-permission-sync-job`

```json
{ "code": 0, "message": "permission_sync_job_stopped", "data": null }
```

---

#### `POST /api/system/delete-role-permission-sync-job`

```json
{ "code": 0, "message": "role_permission_sync_job_stopped", "data": null }
```

---

### Internal API `/api/internal-v1`

**Auth:** `X-API-Key: <key>` (required on all routes).  
**Rate limit:** 60 requests / 1 minute.

---

#### `GET /api/internal-v1/rbac/permissions`

**Query params:** `dto.PermissionFilter` (page, per_page, sort_by, sort_order, search_by, search_data)

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `PaginatedData{ result: Permission[], page_info: PageInfo }` |
| 400 | 2001 | `null` — invalid query params |
| 500 | 9001 | `null` |

**Success `data`:**

```json
{
  "result": [
    {
      "permission_id":   "P1",
      "permission_name": "profile:read",
      "description":     "",
      "created_at":      "2025-01-01T00:00:00Z",
      "updated_at":      "2025-01-01T00:00:00Z"
    }
  ],
  "page_info": { "page": 1, "per_page": 20, "total_pages": 1, "total_items": 13 }
}
```

---

#### `POST /api/internal-v1/rbac/permissions`

**Request body:** `dto.CreatePermissionRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 201 | 0 | `models.Permission` object |
| 400 | 3001 | `null` — validation / duplicate |
| 500 | 9001 | `null` |

**Success `data`:**

```json
{
  "permission_id":   "P14",
  "permission_name": "lesson:read",
  "description":     "Read lessons",
  "created_at":      "2026-04-18T10:00:00Z",
  "updated_at":      "2026-04-18T10:00:00Z"
}
```

---

#### `PATCH /api/internal-v1/rbac/permissions/:permissionId`

**Request body:** `dto.UpdatePermissionRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `models.Permission` object |
| 400 | 3001 | `null` — invalid ID or body |
| 404 | 3004 | `null` — not found |
| 500 | 9001 | `null` |

---

#### `DELETE /api/internal-v1/rbac/permissions/:permissionId`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` — invalid ID param |
| 500 | 9001 | `null` |

---

#### `GET /api/internal-v1/rbac/roles`

**Query params:** `with_permissions=1` (optional)

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `Role[]` |
| 500 | 9001 | `null` |

**Success `data` (with `with_permissions=1`):**

```json
[
  {
    "id": 1,
    "name": "sysadmin",
    "description": "System-wide administration",
    "permissions": [
      { "permission_id": "P1", "permission_name": "profile:read", "description": "", "created_at": "...", "updated_at": "..." }
    ],
    "created_at": "...",
    "updated_at": "..."
  }
]
```

> Without `with_permissions=1`, the `"permissions"` key is omitted (Go `omitempty`).

---

#### `POST /api/internal-v1/rbac/roles`

**Request body:** `dto.CreateRoleRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 201 | 0 | `models.Role` object (without permissions) |
| 400 | 3001 | `null` |
| 500 | 9001 | `null` |

---

#### `GET /api/internal-v1/rbac/roles/:id`

**Query params:** `with_permissions=1` (optional)

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `models.Role` object |
| 400 | 3001 | `null` — invalid ID |
| 404 | 3004 | `null` — not found |
| 500 | 9001 | `null` |

---

#### `PATCH /api/internal-v1/rbac/roles/:id`

**Request body:** `dto.UpdateRoleRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `models.Role` object (with `permissions` preloaded) |
| 400 | 3001 | `null` |
| 404 | 3004 | `null` |
| 500 | 9001 | `null` |

---

#### `PUT /api/internal-v1/rbac/roles/:id/permissions`

**Request body:** `dto.SetRolePermissionsRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `models.Role` object with `permissions` array |
| 400 | 3001 | `null` — unknown permission ID or body error |
| 404 | 3004 | `null` — role not found |
| 500 | 9001 | `null` |

---

#### `DELETE /api/internal-v1/rbac/roles/:id`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` — invalid ID |
| 500 | 9001 | `null` |

---

#### `GET /api/internal-v1/rbac/users/:userId/roles`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `Role[]` (with `permissions` preloaded) |
| 400 | 3001 | `null` — invalid userID |
| 500 | 9001 | `null` |

---

#### `GET /api/internal-v1/rbac/users/:userId/permissions`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `string[]` — sorted effective permission codes |
| 400 | 3001 | `null` |
| 500 | 9001 | `null` |

```json
{ "code": 0, "message": "ok", "data": ["course:read", "profile:read", "user:read"] }
```

---

#### `GET /api/internal-v1/rbac/users/:userId/direct-permissions`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `Permission[]` — direct grants only |
| 400 | 3001 | `null` |
| 500 | 9001 | `null` |

---

#### `POST /api/internal-v1/rbac/users/:userId/roles`

**Request body:** `dto.AssignUserRoleRequest`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` — invalid input |
| 404 | 3004 | `null` — role not found |
| 500 | 9001 | `null` |

---

#### `DELETE /api/internal-v1/rbac/users/:userId/roles/:roleId`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` |
| 500 | 9001 | `null` |

---

#### `POST /api/internal-v1/rbac/users/:userId/direct-permissions`

**Request body:** `dto.AssignUserPermissionRequest` — provide either `permission_id` or `permission_name`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` — both fields missing, or validation error |
| 404 | 3004 | `null` — permission not found |
| 500 | 9001 | `null` |

---

#### `DELETE /api/internal-v1/rbac/users/:userId/direct-permissions/:permissionId`

| Status | `code` | `data` |
|--------|--------|--------|
| 200 | 0 | `null` |
| 400 | 3001 | `null` — invalid ID |
| 500 | 9001 | `null` |

---

## 5. Error Response Type

All error responses use the standard `Response` envelope with a non-zero `code` and `data: null`:

```json
{
  "code":    <errcode constant>,
  "message": "<human-readable description>",
  "data":    null
}
```

**Full error code table:**

| HTTP | `code` | Constant | Meaning |
|------|--------|----------|---------|
| — | 0 | `Success` | Operation completed successfully |
| 400 | 1001 | `InvalidJSON` | Request body is not valid JSON |
| 400 | 2001 | `ValidationFailed` | Request validation failed |
| 400 | 2002 | `ValidationField` | Per-field validation error |
| 400 | 3001 | `BadRequest` | Bad request (generic) |
| 401 | 3002 | `Unauthorized` | Missing or invalid JWT / system token |
| 403 | 3003 | `Forbidden` | Authenticated but lacking permission |
| 404 | 3004 | `NotFound` | Resource not found |
| 409 | 3005 | `Conflict` | Duplicate resource |
| 429 | 3006 | `TooManyRequests` | Rate limit exceeded |
| 400 | 4001 | `EmailAlreadyExists` | Email already registered |
| 401 | 4002 | `InvalidCredentials` | Wrong email/password or system creds |
| 400 | 4003 | `WeakPassword` | Password fails strength requirements |
| 401 | 4004 | `EmailNotConfirmed` | Login before email confirmation |
| 403 | 4005 | `UserDisabled` | Account disabled |
| 400 | 4006 | `InvalidConfirmToken` | Stale or unknown confirmation token |
| 401 | 4007 | `InvalidSession` | Session ID unknown or UUID mismatch |
| 401 | 4008 | `RefreshTokenExpired` | DB-stored refresh expiry has passed |
| 500 | 9001 | `InternalError` | Internal server error |
| 500 | 9998 | `Panic` | Unhandled panic (caught by recovery middleware) |
| 500 | 9999 | `Unknown` | Unknown error |

**Special response header on token expiry:**

```
X-Token-Expired: true
```

This header is set **only** when a `401` is caused by an expired access JWT (`jwt.ErrTokenExpired`). It is **not** set for missing or malformed tokens.
