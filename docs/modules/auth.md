# Auth Module

## Overview

The auth module (`internal/auth/`) manages user authentication using **stateful JWT sessions** backed by PostgreSQL. Three tokens are issued on every successful login or email confirmation and rotated via a dedicated refresh endpoint.

| Token | Contents | TTL |
|-------|----------|-----|
| `access_token` | Signed HS256 JWT with user identity + `permissions` claim | 15 minutes |
| `refresh_token` | Signed HS256 JWT with `user_id` + `uuid` (session correlator) | 30 days (standard) / 14 days (remember-me) |
| `session_id` | 128-char hex string — identifies the device session in the DB | Same as `refresh_token` |

All three are issued as **non-HttpOnly** cookies (`SameSite=Lax`, `Secure` in production) and also returned in the JSON response body, so server-side callers can relay them without parsing `Set-Cookie`.

---

## Directory Layout

```
internal/auth/
├── domain/
│   ├── user.go                  # User entity
│   ├── repository.go            # UserRepository + RefreshSessionRepository interfaces
│   ├── errors.go                # Domain errors (ErrEmailAlreadyExists, etc.)
│   └── token_ttl.go             # AccessTokenTTL, RefreshTokenTTL, RememberMeRefreshTTL
├── application/
│   ├── auth_service.go          # AuthService: Register, Login, ConfirmEmail, RefreshSession, GetMe, UpdateMe, DeleteMe
│   ├── cache_keys.go            # Redis key patterns
│   └── email_limits.go          # Confirmation email limits
├── infra/
│   ├── gorm_user_repo.go        # GormUserRepository
│   ├── gorm_session_repo.go     # GormRefreshSessionRepository
│   ├── crypto.go                # Password hashing (bcrypt)
│   └── session_limits.go        # MaxActiveSessions constant
└── delivery/
    ├── handler.go               # HTTP handlers: Register, Login, ConfirmEmail, RefreshToken, GetMe, PatchMe, DeleteMe, GetMyPermissions
    ├── routes.go                # Route registration
    ├── dto.go                   # Request/response DTOs
    └── mapping.go               # Domain → DTO mapping
```

---

## Auth Header Convention

**All protected endpoints require the access token via the `Authorization` header only.** Cookies are NOT read by the auth middleware.

```
Authorization: Bearer <access_token>
```

When the access token is expired:

```
HTTP/1.1 401 Unauthorized
X-Token-Expired: true

{ "code": 3002, "message": "token expired", "data": null }
```

| Situation | HTTP | `X-Token-Expired` | JSON `message` |
|-----------|------|-------------------|----------------|
| No `Authorization: Bearer …` or empty token | `401` | not set | `missing bearer token` |
| Expired access JWT | `401` | `true` | `token expired` |
| Invalid/malformed JWT | `401` | not set | `invalid token` |

---

## Endpoints

### `POST /api/v1/auth/register`

Creates a **pending** user (or re-sends confirmation to an existing unconfirmed user) and sends a Brevo confirmation email.

**Request:**
```json
{ "email": "user@example.com", "password": "Str0ng!pw", "display_name": "Alice" }
```

**Password rules:** minimum 8 characters, at least one uppercase, one lowercase, one special character.

**Confirmation email limits:**

| Layer | Rule |
|-------|------|
| Postgres `users.registration_email_send_total` | At most **15** successful sends while `email_confirmed = false`. Next send deletes the pending row → **`410`** / `4009` RegistrationAbandoned |
| Redis sliding window `mycourse:auth:register:confirm_email_window:{user_id}` | At most **5** sends per **4-hour** window. Exceeded → **`429`** / `4010` + `Retry-After` / `X-Mycourse-Register-Retry-After` headers |
| Brevo failure | If Brevo fails after limits pass, the Redis slot is released → **`502`** / `4011` ConfirmationEmailSendFailed |

**Success:** `201 Created`

**Errors:** `2001` ValidationFailed · `4003` WeakPassword · `4001` EmailAlreadyExists (confirmed) · `4009` RegistrationAbandoned · `4010` Rate limited · `4011` Email send failed

---

### `POST /api/v1/auth/login`

Validates credentials and issues a full token set.

**Request:**
```json
{ "email": "user@example.com", "password": "Str0ng!pw", "remember_me": false }
```

`remember_me` controls TTL rotation behavior:
- `false` → remaining lifetime of session carries forward on each rotation (fixed horizon)
- `true` → refresh TTL always renewed to 14 days from each rotation (sliding window)

**Success:** `200 OK` + three auth cookies + JSON body:
```json
{
  "code": 0, "message": "login_success",
  "data": { "access_token": "<jwt>", "refresh_token": "<jwt>", "session_id": "<128-char-hex>" }
}
```

**Errors:** `4002` InvalidCredentials · `4004` EmailNotConfirmed · `4005` UserDisabled

**Redis keys used:**
- `mycourse:user:me:{user_id}` — cached `MeResponse`, TTL 1 minute
- `mycourse:auth:login:invalid:{normalized_email}` — negative login cache, TTL 1 minute
- `mycourse:auth:login:user_by_email:{normalized_email}` — email → user_id, TTL 30 seconds

---

### `GET /api/v1/auth/confirm?token=<token>`

Confirms email, assigns the `learner` role, and immediately issues a token set. Resets `registration_email_send_total` to 0 and clears Redis window + email→user cache.

**Success:** `200 OK` + auth cookies + same body shape as login

**Errors:** `4006` InvalidConfirmToken

---

### `POST /api/v1/auth/refresh`

Rotates the token pair for an existing session. The client sends tokens via custom headers (not cookies):

```
X-Refresh-Token: <refresh_jwt>
X-Session-Id:    <128-char-hex>
```

**Success:** `200 OK`
```json
{
  "code": 0, "message": "token_refreshed",
  "data": { "access_token": "<new_jwt>", "refresh_token": "<new_jwt>", "session_id": "<same-hex>" }
}
```

**Errors:** `4007` InvalidSession · `4008` RefreshTokenExpired · `4005` UserDisabled · `3001` BadRequest (missing headers)

---

### `GET /api/v1/me` (JWT required)

Returns the current user's profile and effective permission names. Redis cache-first; DB fallback with up to **1 minute** staleness.

---

### `PATCH /api/v1/me` (JWT required)

Updates profile fields (display name, avatar `image_file_id`). Validates the avatar against `media_files` via a Media domain interface.

---

### `DELETE /api/v1/me` (JWT required)

Deletes the current user account.

---

### `GET /api/v1/me/permissions` (JWT + `user:read` permission)

Returns the full list of effective permission names for the current user.

---

## Token Refresh Flow (Client-Driven)

```
Client → BE (valid access token) → 200 OK
Client → BE (expired access token)
  └─ BE: HTTP 401, X-Token-Expired: true

Client → POST /api/v1/auth/refresh
             X-Refresh-Token: <refresh_jwt>
             X-Session-Id:    <session_id>
  ├─ [valid] → 200: new access_token + refresh_token → client retries
  └─ [invalid/expired] → 401/403 → force re-login
```

---

## Session Model

Session state lives in `users.refresh_token_session` — a JSONB column mapping session string → metadata:

```json
{
  "<128-char hex session_id>": {
    "refresh_token_uuid":    "xxxxxxxx-...",
    "remember_me":           false,
    "refresh_token_expired": "2026-05-03T12:00:00Z"
  }
}
```

- Session key: `HMAC-SHA512(JWT_secret, 32-random-bytes)` → hex encoded
- Max **5 concurrent sessions** per user. On the 6th login the session with the earliest `refresh_token_expired` is evicted inside a DB transaction.
- Session rotation updates the entry **in-place** — the count does not increase on rotation.

---

## Error Codes

| Code | Constant | When |
|------|----------|------|
| `4001` | `EmailAlreadyExists` | Register with a confirmed taken email |
| `4002` | `InvalidCredentials` | Wrong email or password |
| `4003` | `WeakPassword` | Password fails strength check |
| `4004` | `EmailNotConfirmed` | Login before confirming email |
| `4005` | `UserDisabled` | Account has been disabled |
| `4006` | `InvalidConfirmToken` | Confirm with stale or unknown token |
| `4007` | `InvalidSession` | session_id or UUID mismatch |
| `4008` | `RefreshTokenExpired` | DB-stored expiry has passed |
| `4009` | `RegistrationAbandoned` | Lifetime email cap reached — pending user deleted |
| `4010` | `RegistrationEmailRateLimited` | Redis sliding window exceeded |
| `4011` | `ConfirmationEmailSendFailed` | Brevo send failure |

---

## Implementation Reference

| Concern | Location |
|---------|----------|
| JWT sign/parse | `internal/shared/token/` |
| AuthService use-cases | `internal/auth/application/auth_service.go` |
| GORM user repository | `internal/auth/infra/gorm_user_repo.go` |
| GORM session repository | `internal/auth/infra/gorm_session_repo.go` |
| HTTP handlers | `internal/auth/delivery/handler.go` |
| Route registration | `internal/auth/delivery/routes.go` |
| Auth middleware (Bearer header) | `internal/shared/middleware/` |
| Redis keys | `internal/auth/application/cache_keys.go` |
| Email limits | `internal/auth/application/email_limits.go` / `internal/auth/domain/token_ttl.go` |
| Brevo email sending | `internal/shared/brevo/` |
| Email templates | `internal/shared/mailtmpl/` |
| CORS / expose headers | `internal/server/router.go` |
