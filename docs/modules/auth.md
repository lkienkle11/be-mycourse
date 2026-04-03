# Auth Module

## Overview

Authentication uses **stateful JWT sessions** backed by Postgres.  
Three HttpOnly cookies are issued on every successful login or email confirmation and rotated transparently on each token refresh.

| Cookie | Contents | TTL |
|---|---|---|
| `access_token` | Signed HS256 JWT with user identity + permissions | 15 minutes |
| `refresh_token` | Signed HS256 JWT with `user_id` + `uuid` (session correlator) | 30 days (non-remember-me) / 14 days (remember-me) |
| `session_id` | 128-char hex string — identifies the device session in the DB | same as `refresh_token` |

All three cookies are `HttpOnly`, `SameSite=Strict`, and `Secure` in production (`RunMode=release`).  
Tokens are **never** returned in the JSON response body.

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

**Success:** `200 OK` — `login_success` + three auth cookies  
**Errors:** `4002` InvalidCredentials · `4004` EmailNotConfirmed · `4005` UserDisabled

---

### `GET /api/v1/auth/confirm?token=<token>`

Confirms the user's email and immediately issues a token set (so the user is logged in after confirmation).

`remember_me` is always `false` for email-confirmation sessions.

**Success:** `200 OK` — `email_confirmed` + three auth cookies  
**Errors:** `4006` InvalidConfirmToken

---

## Session Model

Session state lives in `users.refresh_token_session` — a `JSONB` column that maps **session string → session metadata**:

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
| `refresh_token_uuid` | string (UUID) | Embedded in the `refresh_token` JWT. Must match on refresh to prevent token reuse across sessions. |
| `remember_me` | bool | Governs TTL extension logic during token rotation. |
| `refresh_token_expired` | timestamp | Authoritative expiry stored in Postgres. The JWT's own `exp` is used only for signature context; actual validity is always checked here. |

The key (session string) is 128 hex chars generated as `HMAC-SHA512(JWT_secret, 32-random-bytes)` — unique per session and tied to the deployment key.

### Session limit

A user may have **at most 5 concurrent sessions** (5 distinct devices/browsers).  
When a 6th login occurs the session with the earliest `refresh_token_expired` is automatically evicted.  
The limit is enforced inside a **database transaction** to prevent races on concurrent logins.

---

## Token Rotation (Transparent Cookie Refresh)

The middleware `AuthJWT` handles expired access tokens automatically — **no dedicated refresh endpoint is needed**.

**Trigger conditions (both must be true):**

1. The `access_token` cookie is present but has expired (JWT `ErrTokenExpired`).
2. The token came from the cookie, not an `Authorization: Bearer` header.

**Rotation flow:**

```
Browser request (expired access_token cookie)
  │
  ▼
middleware.AuthJWT
  ├── ParseAccess → ErrTokenExpired (from cookie)
  ├── Read session_id cookie  ────────────────────────────┐
  ├── Read refresh_token cookie                           │
  ├── ParseRefreshIgnoreExpiry (verify sig, ignore exp)  │  validate
  ├── Load user from DB                                   │
  ├── Check refresh_token_session[session_id].uuid match ┘
  ├── Check refresh_token_session[session_id].refresh_token_expired > now
  │
  ├── [valid] ──► Generate new access_token + refresh_token + new UUID
  │               Update session entry in DB (same key, new UUID + new expiry)
  │               Set three new cookies on the response
  │               Populate Gin context from new access token
  │               c.Next() ──► handler runs normally
  │
  └── [invalid] ─► 401 / 403 and abort
```

**TTL on rotation:**

| `remember_me` | New refresh TTL |
|---|---|
| `true` | Always `14 days` from now |
| `false` | `time.Until(old refresh_token_expired)` — remaining time only, no extension |

The `session_id` cookie value is **unchanged** across rotations. Only `access_token` and `refresh_token` cookies are replaced.

**Bearer-header tokens are never silently refreshed.** API clients managing their own token lifecycle (mobile apps, server-to-server) receive a plain `401` when their access token expires; they must call a refresh endpoint or re-authenticate explicitly.

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
| Session entry persistence (`AddRefreshSession`, `SaveRefreshSession`) | `models/user.go` |
| Cookie issuance | `api/v1/auth.go` — `setAuthCookies` |
| Transparent refresh middleware | `middleware/auth_jwt.go` — `tryTokenRefreshFromCookie` |
| Session limit constant (`MaxActiveSessions = 5`) | `models/user.go` |
| Token TTL constants | `services/auth.go` — `AccessTokenTTL`, `RefreshTokenTTL`, `RememberMeRefreshTTL` |
| DB migration | `migrations/000010_users_refresh_token_session.up.sql` |

---

## Constraints

- A single account is limited to **5 concurrent sessions**; the oldest-expiring session is evicted when the limit is reached.
- `remember_me` is always stored as `false` for sessions created via email confirmation.
- Transparent refresh only occurs for **cookie-sourced** tokens; `Authorization: Bearer` tokens are never auto-refreshed.
- Session rotation updates the **existing session entry in-place** (same `session_id` cookie key) — the session count does not increase on rotation.
- All session writes that change the session count use a **DB transaction** to prevent concurrent logins from exceeding the limit.
