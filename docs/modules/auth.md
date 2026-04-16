# Auth Module

## Overview

Authentication uses **stateful JWT sessions** backed by Postgres.  
Three cookies are issued on every successful login or email confirmation and rotated explicitly via a dedicated refresh endpoint.

| Cookie | Contents | TTL |
|---|---|---|
| `access_token` | Signed HS256 JWT with user identity + `permissions` claim (`permission_name` strings, e.g. `user:read`) | 15 minutes |
| `refresh_token` | Signed HS256 JWT with `user_id` + `uuid` (session correlator) | 30 days (non-remember-me) / 14 days (remember-me) |
| `session_id` | 128-char hex string — identifies the device session in the DB | same as `refresh_token` |

All three cookies are **non-HttpOnly**, `SameSite=Lax`, and `Secure` in production (`RunMode=release`).  
This allows the client-side JavaScript layer to read the tokens and attach them to outgoing requests as custom headers.  
Tokens are also returned in the JSON response body (login, confirm, refresh) so Server-side callers can relay them without parsing `Set-Cookie`.

---

## Auth Header Convention

**All protected endpoints require the access token via the `Authorization` header only.**  
Cookies are NOT read by the auth middleware.

```
Authorization: Bearer <access_token>
```

When the access token is expired the middleware returns:

```
HTTP/1.1 401 Unauthorized
X-Token-Expired: true

{ "code": 3002, "message": "token expired", "data": null }
```

The `X-Token-Expired: true` response header is the signal to the client to call `POST /api/v1/auth/refresh` rather than treating this as a permanent auth failure.

### Missing `Authorization` vs expired token

`middleware.AuthJWT` (`requireJWT` in `middleware/auth_jwt.go`) distinguishes:

| Situation | HTTP | `X-Token-Expired` | Typical JSON `message` |
|-----------|------|-------------------|-------------------------|
| No `Authorization: Bearer …`, or empty token after `Bearer` | `401` | **not set** | `missing bearer token` |
| Access JWT present but cryptographically expired | `401` | **`true`** | `token expired` |
| Access JWT present but invalid / malformed | `401` | not set | `invalid token` |

The official contract for **“rotate tokens, then retry”** is still **`401` + `X-Token-Expired: true`**. The **MyCourse Next.js** client additionally treats **`401` with no non-empty Bearer on the request** as refresh-eligible when `refresh_token` and `session_id` cookies exist (e.g. user cleared only `access_token`). Other integrators may treat `missing bearer token` as a hard logout unless they implement the same heuristic.

---

## Endpoints

### `POST /api/v1/auth/register`

Creates a new user and sends a confirmation email.

**Request body**

```json
{
  "email":        "user@example.com",
  "password":     "Str0ng!pw",
  "display_name": "Alice"
}
```

**Password rules:** minimum 8 characters, at least one uppercase, one lowercase, and one special character.

**Success:** `201 Created` — `registration_success`  
**Errors:** `4001` EmailAlreadyExists · `4003` WeakPassword

---

### `POST /api/v1/auth/login`

Validates credentials and issues a full token set.

**Request body**

```json
{
  "email":       "user@example.com",
  "password":    "Str0ng!pw",
  "remember_me": false
}
```

`remember_me` is optional (defaults to `false`).

- `false` → refresh token TTL = **30 days** (initial). On each rotation the remaining lifetime of the old token is carried forward — the session has a fixed total horizon.
- `true` → refresh token TTL = **14 days**, always renewed from the time of each rotation (sliding window — active users never expire).

**Success:** `200 OK` — `login_success` + three auth cookies + JSON body

```json
{
  "code": 0,
  "message": "login_success",
  "data": {
    "access_token":  "<jwt>",
    "refresh_token": "<jwt>",
    "session_id":    "<128-char-hex>"
  }
}
```

**Errors:** `4002` InvalidCredentials · `4004` EmailNotConfirmed · `4005` UserDisabled

**Redis** — package `mycourse-io-be/services/cache` (`auth_user.go`).

- **`mycourse:user:me:{user_id}`** — JSON `dto.MeResponse`, TTL **1 minute**. Set on successful login and on `GET /me` miss.
- **`mycourse:auth:login:invalid:{normalized_email}`** — negative cache for `4002`. TTL **1 minute**. While present, login returns `4002` without hitting Postgres.
- **`mycourse:auth:login:user_by_email:{normalized_email}`** — maps normalized email → `user_id`. TTL **30 seconds**.

---

### `GET /api/v1/auth/confirm?token=<token>`

Confirms the user's email and immediately issues a token set (user is logged in after confirmation).  
On success the user is assigned the **`learner`** role if not already present.

`remember_me` is always `false` for email-confirmation sessions.

**Success:** `200 OK` — `email_confirmed` + three auth cookies + same JSON body shape as login  
**Errors:** `4006` InvalidConfirmToken

---

### `POST /api/v1/auth/refresh`

Rotates the token pair for an existing session.  
The client sends the current `refresh_token` and `session_id` via custom request headers:

```
X-Refresh-Token: <refresh_jwt>
X-Session-Id:    <128-char-hex>
```

No request body is required.

**Success:** `200 OK` — `token_refreshed`

```json
{
  "code": 0,
  "message": "token_refreshed",
  "data": {
    "access_token":  "<new_jwt>",
    "refresh_token": "<new_jwt>",
    "session_id":    "<same-128-char-hex>"
  }
}
```

The `session_id` value is **unchanged** across rotations. The client must update its stored `access_token` and `refresh_token` and may optionally refresh the `session_id` cookie (it is the same value).

**Errors:**

| Code | When |
|---|---|
| `4007` InvalidSession | `session_id` not found or refresh token UUID mismatch |
| `4008` RefreshTokenExpired | DB-stored `refresh_token_expired` has passed |
| `4005` UserDisabled | Account was disabled after the session was created |
| `3001` BadRequest | Missing `X-Refresh-Token` or `X-Session-Id` header |

**TTL rules on rotation:**

| `remember_me` | New refresh TTL |
|---|---|
| `true` | Always `14 days` from now |
| `false` | `time.Until(old refresh_token_expired)` — remaining time only, no extension |

---

### `GET /api/v1/me`

Returns the current user profile and effective permission names (colon form, same as `permissions.permission_name`), same shape as `dto.MeResponse`.

Requires `Authorization: Bearer <access_token>`.

**Redis:** If `mycourse:user:me:{user_id}` exists and unmarshals correctly, the handler returns it and skips the DB.

Profile and permissions may therefore lag writes by up to **1 minute**.

---

## Token Refresh Flow (Client-Driven)

Token refresh is now the **client's responsibility** — the middleware no longer silently renews expired tokens. The expected client flow is:

```
Client request → BE (valid access token) → 200 OK
Client request → BE (expired access token)
  └─ BE: HTTP 401, X-Token-Expired: true

Client → POST /api/v1/auth/refresh
             X-Refresh-Token: <refresh_jwt>
             X-Session-Id:    <session_id>
  ├─ [valid] → 200: new access_token + refresh_token
  │             Client stores new tokens, retries original request → 200 OK
  └─ [invalid/expired] → 401 / 403 → Client forces re-login
```

The FE Axios interceptor in `src/api/instance.ts` implements this flow with a **mutex** to prevent multiple concurrent requests from each triggering their own refresh simultaneously.

---

## Session Model

Session state lives in `users.refresh_token_session` — a `JSONB` column mapping **session string → metadata**:

```json
{
  "<128-char hex session_id>": {
    "refresh_token_uuid":    "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "remember_me":           false,
    "refresh_token_expired": "2026-05-03T12:00:00Z"
  }
}
```

| Field | Type | Description |
|---|---|---|
| `refresh_token_uuid` | string (UUID) | Embedded in the refresh JWT. Must match on rotation to prevent reuse across sessions. |
| `remember_me` | bool | Governs TTL extension logic during token rotation. |
| `refresh_token_expired` | timestamp | Authoritative expiry in Postgres. JWT `exp` is secondary; this field is always checked. |

The session key is 128 hex chars generated as `HMAC-SHA512(JWT_secret, 32-random-bytes)`.

### Session limit

A user may have **at most 5 concurrent sessions**.  
When a 6th login occurs the session with the earliest `refresh_token_expired` is evicted inside a **database transaction**.

---

## CORS Headers

The following custom headers are whitelisted in the CORS configuration:

| Direction | Header | Purpose |
|---|---|---|
| Request (client → BE) | `X-Refresh-Token` | Carries the refresh JWT to `POST /auth/refresh` |
| Request (client → BE) | `X-Session-Id` | Carries the session ID to `POST /auth/refresh` |
| Response (BE → client) | `X-Token-Expired` | `"true"` when a 401 is due to access token expiry |

---

## Error Codes

| Code | Constant | When |
|---|---|---|
| `4001` | `EmailAlreadyExists` | Register with a taken email |
| `4002` | `InvalidCredentials` | Wrong email or password |
| `4003` | `WeakPassword` | Password fails strength check |
| `4004` | `EmailNotConfirmed` | Login before confirming email |
| `4005` | `UserDisabled` | Account has been disabled |
| `4006` | `InvalidConfirmToken` | Confirm with stale or unknown token |
| `4007` | `InvalidSession` | `session_id` / refresh token missing, unknown, or UUID mismatch |
| `4008` | `RefreshTokenExpired` | DB-stored `refresh_token_expired` has passed |

---

## Implementation Reference

| Concern | Location |
|---|---|
| Token generation (access, refresh, session string) | `pkg/token/jwt.go` |
| Login / ConfirmEmail / RefreshSession business logic | `services/auth.go` |
| Auth middleware (Bearer header only, X-Token-Expired signal) | `middleware/auth_jwt.go` — `requireJWT`, `extractBearerToken` |
| Cookie issuance (non-HttpOnly, SameSite=Lax) | `api/v1/auth.go` — `setAuthCookies` |
| Refresh endpoint handler | `api/v1/auth.go` — `refreshToken` |
| Route registration | `api/v1/routes.go` — `RegisterNotAuthenRoutes` |
| CORS config (AllowHeaders, ExposeHeaders) | `api/router.go` |
| Session entry persistence | `models/user.go` — `AddRefreshSession`, `SaveRefreshSession` |
| Redis keys + TTL | `services/cache/auth_user.go` |
| Session limit constant (`MaxActiveSessions = 5`) | `models/user.go` |
| Token TTL constants | `services/auth.go` — `AccessTokenTTL`, `RefreshTokenTTL`, `RememberMeRefreshTTL` |
| DB schema / sessions column | `migrations/000001_schema.up.sql` |

---

## Constraints

- Cached `/me` data (and embedded permissions) can be stale for up to **1 minute** relative to Postgres / RBAC grants.
- A single account is limited to **5 concurrent sessions**; the oldest-expiring session is evicted when the limit is reached.
- `remember_me` is always stored as `false` for sessions created via email confirmation.
- Session rotation updates the **existing session entry in-place** — the session count does not increase on rotation.
- All session writes that change the session count use a **DB transaction** to prevent concurrent logins from exceeding the limit.
- Cookies are **non-HttpOnly**: protected against CSRF via `SameSite=Lax` and the use of custom headers (`Authorization`, `X-Refresh-Token`, `X-Session-Id`) which cannot be set by cross-site form submissions.
