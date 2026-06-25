# MyCourse Backend — Requirements


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

> **Module:** `mycourse-io-be`  
> **Language:** Go 1.25 · Gin · GORM · PostgreSQL · Redis  
> **Last updated:** 2026-04-18

---

## Table of Contents

1. [Functional Requirements](#functional-requirements)
   - [FR-1 Authentication](#fr-1-authentication)
   - [FR-2 User Profile](#fr-2-user-profile)
   - [FR-3 Role-Based Access Control (RBAC)](#fr-3-role-based-access-control-rbac)
   - [FR-4 System Administration](#fr-4-system-administration)
   - [FR-5 RBAC Synchronization (Internal)](#fr-5-rbac-synchronization-internal)
   - [FR-6 Internal RBAC HTTP API](#fr-6-internal-rbac-http-api)
   - [FR-7 Health Check](#fr-7-health-check)
   - [FR-8 Course Management](#fr-8-course-management)
   - [FR-9 Lesson Management](#fr-9-lesson-management)
   - [FR-10 Enrollment](#fr-10-enrollment)
   - [FR-11 Media Upload Gateway](#fr-11-media-upload-gateway)
2. [Non-Functional Requirements](#non-functional-requirements)
   - [NFR-1 Performance & Availability](#nfr-1-performance--availability)
   - [NFR-2 Security](#nfr-2-security)
   - [NFR-3 Observability & Maintainability](#nfr-3-observability--maintainability)

---

## Functional Requirements

### FR-1 Authentication

> **Source:** `services/auth/auth.go`, `services/auth/register_flow.go`, `api/v1/auth.go`, `api/v1/routes.go`

#### FR-1.1 User Registration

- The system **MUST** allow a new user to register with an email address, a password, and a display name.
- The system **MUST** validate that the password is at least 8 characters long and contains at least one uppercase letter, one lowercase letter, and one special character (not a letter, digit, or space).
- The system **MUST** reject registration with **`409`** / `4001` EmailAlreadyExists when the email is already registered **and** `email_confirmed` is true.
- When the email exists but `email_confirmed` is false, the system **MUST** treat the request as a **resend / update pending** flow: update password hash, display name, and confirmation token, then apply the same confirmation-email policies as for a new user.
- The system **MUST** bcrypt-hash the password before persisting it.
- The system **MUST** generate a UUIDv7 `user_code` for the new user before insert.
- The system **MUST** generate a UUID confirmation token, persist it in `users.confirmation_token`, and send a confirmation email via Brevo to the provided email address.
- The system **MUST** persist **`users.registration_email_send_total`**: increment only after each **successful** Brevo send while pending; cap at **15** — if the next send would exceed the cap, the system **MUST** hard-delete the pending user and return **`410`** with app code **`4009`** RegistrationAbandoned.
- The system **MUST** enforce a Redis-backed sliding window of **5** successful confirmation emails per **`users.id`** per **4 hours**; when exceeded the system **MUST** return **`429`** with app code **`4010`**, **`Retry-After`**, and **`X-Mycourse-Register-Retry-After`** (seconds until retry).
- If Brevo fails after limits checks, the system **MUST** return **`502`** with app code **`4011`** ConfirmationEmailSendFailed and **MUST NOT** increment `registration_email_send_total` for that attempt.
- The registration endpoint **MUST** return HTTP `201 Created` with `code: 0` and `message: "registration_success"` on success. No token is returned.

**Error cases:**

| Condition | HTTP | App Code |
|-----------|------|----------|
| Email already registered and confirmed | 409 | 4001 `EmailAlreadyExists` |
| Password fails strength check | 400 | 4003 `WeakPassword` |
| Request body fails JSON binding | 400 | 2001 `ValidationFailed` |
| Lifetime confirmation-email cap (row deleted) | 410 | 4009 `RegistrationAbandoned` |
| Redis window exceeded | 429 | 4010 `RegistrationEmailRateLimited` |
| Brevo / transport failure after limits | 502 | 4011 `ConfirmationEmailSendFailed` |
| Other DB / unexpected errors | 500 | 9001 `InternalError` |

---

#### FR-1.2 Email Confirmation

- The system **MUST** expose a `POST /api/v1/auth/confirm` endpoint that accepts a token in request body and looks up the user by `confirmation_token`.
- On match, the system **MUST** atomically:
  1. Set `email_confirmed = true` on the user.
  2. Clear `confirmation_token` to `NULL`.
  3. Set `registration_email_send_total` to **0**.
  4. Assign the **`learner`** role to the user if not already present (`user_roles` `FirstOrCreate`).
- After confirmation the system **MUST** issue a full token pair (access + refresh + session) and return it in the JSON body and as cookies (`remember_me` is always `false`, refresh TTL 3 days).
- After confirmation the system **MUST** delete the Redis registration confirmation window key for that user and clear the login email→user cache for that normalized email.
- An invalid or already-consumed token **MUST** return `400` with app code `4006 InvalidConfirmToken`.

---

#### FR-1.2.1 CSRF Protection

- The system **MUST** implement CSRF protection using the double-submit cookie pattern.
- The CSRF cookie name **MUST** be `csrf_token`.
- The request header name **MUST** be `X-CSRF-Token`.
- The system **MUST** expose `GET /api/v1/auth/csrf` for FE bootstrap to obtain/set CSRF token state before unsafe calls.
- For unsafe methods (`POST`, `PUT`, `PATCH`, `DELETE`), the system **MUST** validate that header `X-CSRF-Token` matches cookie `csrf_token`.
- Missing or mismatched CSRF token on unsafe methods **MUST** be rejected with an error response message in English.
- Temporary rollout note: middleware enforcement may be disabled at router level while preserving the full CSRF logic and endpoint for later re-enable.

---

#### FR-1.3 User Login

- The system **MUST** validate email + password against the stored bcrypt hash.
- The system **MUST** reject login if the account's `is_disable` flag is `true` (`ErrUserDisabled`, HTTP **403**, code **4005**).
- The system **MUST** reject login if `banned_until` is set and `banned_until > now()` (`ErrUserBanned`, HTTP **403**, code **4012**). Ban lifts automatically when time passes.
- The system **MUST** reject login for soft-deleted users (`deleted_at IS NOT NULL`) as invalid credentials (`ErrInvalidCredentials`, HTTP **401**, code **4002**).
- Access checks are implemented in **`internal/auth/application/service_access.go`** (`checkUserAccessible`), not in the domain layer.
- Every JWT-protected `/api/v1` route **MUST** pass **`middleware.RequireActiveUser`** after `AuthJWT`, which calls `AuthService.EnsureActiveUser` (Postgres lookup) so ban/disable/delete apply even when the access token has not expired.
- The system **MUST** reject login if `email_confirmed` is `false` (`ErrEmailNotConfirmed`).
- The system **SHOULD** use a short-lived Redis negative cache (`mycourse:auth:login:invalid:{normalized_email}`, TTL 1 min) to avoid Postgres hits on repeat failed attempts.
- The system **SHOULD** cache the email → user ID mapping in Redis (`mycourse:auth:login:user_by_email:{normalized_email}`, TTL 30 s) to skip the unique-email query on subsequent logins.
- On success, the system **MUST** issue a token pair consisting of:
  - An **access token** (HS256 JWT, TTL 15 min) embedding `user_id`, `user_code`, `email`, `display_name`, `created_at`, and the user's `permissions` array.
  - A **refresh token** (HS256 JWT, TTL configurable via `remember_me`).
  - A **session string** (128 hex chars = HMAC-SHA512 over 32 random bytes).
- The system **MUST** persist the session entry in `users.refresh_token_session` (JSONB, max 5 entries; evict the oldest-expiring entry on overflow, inside a DB transaction).
- `remember_me = false` → initial refresh TTL = **3 days**; subsequent rotations carry forward the remaining lifetime.
- `remember_me = true` → initial and all subsequent refresh TTLs = **30 days** from the moment of rotation (sliding window).
- The system **MUST** set three **HttpOnly**, SameSite=Lax cookies (`access_token`, `refresh_token`, `session_id`), and return the three token values in the JSON body (`access_token`, `refresh_token`, `session_id` only).

**Error cases:**

| Condition | HTTP | App Code |
|-----------|------|----------|
| Wrong email or password | 401 | 4002 `InvalidCredentials` |
| Email not confirmed | 401 | 4004 `EmailNotConfirmed` |
| Account disabled | 403 | 4005 `UserDisabled` |
| Account temporarily banned | 403 | 4012 `UserBanned` |
| Request validation failure | 400 | 2001 `ValidationFailed` |
| DB error | 500 | 9001 `InternalError` |

---

#### FR-1.4 Token Refresh (Session Rotation)

- The client **MUST** supply `X-Refresh-Token` and `X-Session-Id` request headers **or** equivalent HttpOnly cookies (`refresh_token`, `session_id`) when using browser credentials (no body required).
- The system **MUST** parse the refresh JWT ignoring expiry (the DB record is authoritative), verify the `session_id` key exists in `users.refresh_token_session`, and confirm the embedded UUID matches the stored `refresh_token_uuid`.
- The system **MUST** check `refresh_token_expired` in Postgres. If the stored timestamp is in the past, return `4008 RefreshTokenExpired`.
- On valid rotation, the system **MUST**:
  1. Generate a new `uuid` (session UUID) and new access + refresh tokens.
  2. Update the **existing session entry in-place** via `jsonb_set` (session count unchanged).
  3. Return new `access_token`, `refresh_token` (with the same `session_id`) in the JSON body.
- The `session_id` value **MUST NOT** change on rotation; only the token pair changes.

**Error cases:**

| Condition | HTTP | App Code |
|-----------|------|----------|
| Missing `X-Refresh-Token` or `X-Session-Id` | 400 | 3001 `BadRequest` |
| Session ID not found or UUID mismatch | 401 | 4007 `InvalidSession` |
| `refresh_token_expired` has passed | 401 | 4008 `RefreshTokenExpired` |
| Account disabled | 403 | 4005 `UserDisabled` |
| Account temporarily banned | 403 | 4012 `UserBanned` |
| DB error | 500 | 9001 `InternalError` |

---

### FR-2 User Profile

> **Source:** `api/v1/me.go`, `services/auth/auth.go`, `services/cache/auth_user.go`

#### FR-2.1 Get My Profile (`GET /api/v1/me`)

- The system **MUST** return the authenticated user's non-sensitive profile: `user_id`, `user_code`, `email`, `display_name`, optional nested **`avatar`** (`dto.MediaFilePublic` when `avatar_file_id` is set), `email_confirmed`, `is_disabled`, `created_at` (Unix epoch), and `permissions` (sorted `permission_name` strings).
- Sensitive fields (`hash_password`, `confirmation_token`, `confirmation_sent_at`, `refresh_token_session`) **MUST NOT** be returned.
- The system **SHOULD** serve the response from Redis cache (`mycourse:user:me:{user_id}`, TTL 1 min) on cache hit.
- On a cache miss the system **MUST** load the user via active-only lookup and run **`checkUserAccessible`** (`internal/auth/application/service_access.go`) before building the response.
- Requires `Authorization: Bearer <access_token>`.

**Error cases:**

| Condition | HTTP | App Code |
|-----------|------|----------|
| Missing or invalid JWT | 401 | 3002 `Unauthorized` |
| User soft-deleted or not found | 404 | 3004 `NotFound` |
| Account disabled | 403 | 4005 `UserDisabled` |
| Account temporarily banned | 403 | 4012 `UserBanned` |
| DB error | 500 | 9001 `InternalError` |

#### FR-2.1b Patch My Profile (`PATCH /api/v1/me`)

- The system **MUST** accept partial updates with optional **`avatar_file_id`** (UUID of an existing **`media_files`** row suitable as a raster image). Empty string **MUST** clear the FK.
- The system **MUST** validate the file (kind **FILE**, status **READY**, image MIME or common raster extension) and **MUST NOT** accept arbitrary URLs from the client to set storage.
- On success the system **MUST** invalidate the Redis `/me` cache entry for that user.
- Invalid file id **MUST** return HTTP **400** with application code **2001** (`ValidationFailed`) and **`ErrInvalidProfileMediaFile`** semantics.
- Same **`checkUserAccessible`** guards as FR-2.1 (`4005` / `4012` / `404`).

---

#### FR-2.1c Delete My Account

- **`DELETE /api/v1/me`** **MUST** soft-delete the authenticated user (`deleted_at` + `updated_at` via `gormx.SoftDeleteWithAudit`).
- **`DELETE /api/v1/me/hard`** **MUST** permanently remove the user row (GORM `Delete`).
- Both endpoints **MUST** schedule avatar orphan cleanup when `avatar_file_id` is set.
- There is **no** `GET /me/full` endpoint.
- Setting `banned_until` via admin API is **out of scope**; the DB column exists for future use.

**Error cases:** same access guards as FR-2.1 (`404` / `403` `4005` / `403` `4012`).

---

#### FR-2.2 Get My Permissions (`GET /api/v1/me/permissions`)

- The system **MUST** resolve and return a **sorted** list of `permission_name` strings for the authenticated user (union of role-based + direct grants).
- The JSON envelope **`data`** **MUST** be an object **`{ "permissions": string[] }`** (not a bare array).
- This endpoint requires `Authorization: Bearer` **and** the caller **MUST** hold the `user:read` (`P10`) permission.

**Error cases:**

| Condition | HTTP | App Code |
|-----------|------|----------|
| Missing or invalid JWT | 401 | 3002 `Unauthorized` |
| Missing `user:read` permission | 403 | 3003 `Forbidden` |
| DB error | 500 | 9001 `InternalError` |

---

### FR-3 Role-Based Access Control (RBAC)

> **Source:** `services/rbac/rbac.go`, `models/rbac.go`, `constants/permissions.go`, `constants/roles.go`, `constants/roles_permission.go`

#### FR-3.1 Permission Catalog

- The system **MUST** maintain a flat, stable catalog of permissions identified by a string PK (`permission_id`, e.g. `P1`) and a colon-form `permission_name` (e.g. `profile:read`).
- The canonical catalog is defined in `constants/permissions.go` (`AllPermissions`).
- Current catalog (13 entries):

| ID | Name | Description |
|----|------|-------------|
| P1 | `profile:read` | Read own profile |
| P2 | `profile:update` | Update own profile |
| P3 | `profile:delete` | Delete own profile |
| P4 | `profile:create` | Create profiles |
| P5 | `course:read` | Read courses |
| P6 | `course:update` | Update courses |
| P7 | `course:delete` | Delete courses |
| P8 | `course:create` | Create courses |
| P9 | `course_instructor:read` | View course instructor details |
| P10 | `user:read` | Read user records |
| P11 | `user:update` | Update user records |
| P12 | `user:delete` | Delete user records |
| P13 | `user:create` | Create user records |

#### FR-3.2 Role Definitions

- The system **MUST** define four roles: `sysadmin`, `admin`, `instructor`, `learner`.
- Default role-permission assignments (from `constants/roles_permission.go`):

| Role | Permissions |
|------|-------------|
| `sysadmin` | P1–P13 (full catalog) |
| `admin` | P1–P8, P10–P13 (all except `course_instructor:read`) |
| `instructor` | P1, P5, P6, P7, P9, P10 |
| `learner` | P1, P5, P10 |

#### FR-3.3 Effective Permissions

- A user's effective permissions = **UNION** of permissions from all assigned roles (via `user_roles` → `role_permissions`) **PLUS** any direct `user_permissions` grants.
- This union is computed at login time and embedded in the JWT `permissions` claim as sorted `permission_name` strings.
- The `middleware.RequirePermission` middleware checks the claim set without a DB round-trip per request.

---

### FR-4 System Administration

> **Source:** `internal/system/delivery/`, `internal/system/application/service.go`, `internal/system/infra/crypto.go`, `internal/shared/cryptox/`

#### FR-4.1 System Login (CLI)

- A privileged operator **MUST** obtain a system access token via the CLI, not HTTP.
- When `CLI_SYSTEM_LOGIN` env var is truthy (`true`, `1`, `yes`, `y`, `on`), the binary **MUST** run a CLI flow after DB init, prompt for `username` and `password`, verify credentials and machine binding, mint a JWT, print **only** the raw token to **stdout** (one line + newline), then **exit**.
- Credentials are HMAC-SHA256 derived using `system_app_config.app_system_env` (never stored in plaintext).
- On success the system **MUST** issue a short-lived system access token (JWT signed with `system_app_config.app_token_env`; TTL **90 seconds**).
- The system access token is required for **all** `/api/system/*` routes via `Authorization: Bearer <token>`.
- Example: `SYSTEM_TOKEN=$(CLI_SYSTEM_LOGIN=1 go run .)`

**CLI error cases (stderr only):**

| Condition | Message |
|-----------|---------|
| Wrong username or password | `Failure: invalid system credentials.` |
| Machine binding mismatch | `Failure: system account is bound to another machine.` |
| Missing machine identity file | `Failure: machine identity file not found: ...` |
| `app_system_env` or `app_token_env` not configured | `Failure: system token secrets are not configured in database.` |
| CLI rate limit exceeded (5 ops / 3 min per host) | `Failure: too many CLI attempts; try again in a few minutes.` |
| Circuit breaker open (DB/load degraded) | `Failure: service temporarily unavailable; try again later.` |

#### FR-4.2 Privileged User Registration (CLI)

- When `CLI_REGISTER_NEW_SYSTEM_USER` env var is truthy (`true`, `1`, `yes`, `y`, `on`), the binary **MUST** run a CLI flow after DB init, prompt for credentials, bind the account to the local machine, write to `system_privileged_users`, print success, then **exit**.
- Before registering, the CLI **MUST** verify the operator-entered app password against `system_app_config.app_cli_system_password` using bcrypt (`auth/infra.CheckPassword`).
- The CLI **MUST** read or create a local enrollment secret file and combine it with a live OS machine fingerprint (machine-id / hardware UUID, hostname, platform) into hybrid binding material, then derive `machine_secret = CredentialHMACHex(app_system_env, hybridMaterial)` stored on the user row.
- Source: `internal/appcli/register_system_user.go`, `internal/shared/machineidentity/` (`LoadOrCreateMachineIdentityMaterial`, platform fingerprint readers).

#### FR-4.3 System Configuration

- `system_app_config` is a **singleton** row (always `id = 1`).
- `app_cli_system_password`, `app_system_env`, and `app_token_env` **MUST** be stored as **bcrypt hashes (cost 14)**, managed out-of-band (direct SQL UPDATE). They are never plaintext in the database.
- `app_system_env` provides HMAC key material for deriving privileged user credentials (`CredentialHMACHex`).
- `app_token_env` provides JWT signing key material for system access tokens (`MintSystemAccessToken` / `ParseSystemAccessToken`).
- Rotating `app_system_env` invalidates existing `system_privileged_users` rows — re-run the CLI registration flow after rotation.

#### FR-4.4 Machine Identity (hybrid binding)

- **Enrollment file** (random 32-byte secret, mode `0600`): `$XDG_CONFIG_HOME/mycourse/machine_identity` (fallback: `~/.config/mycourse/machine_identity`). Created on first register on that host.
- **OS fingerprint** (read on each register/login): platform machine-id / hardware UUID, hostname, `GOOS/GOARCH`.
- **Hybrid HMAC input** (canonical, versioned): `v1|file:<secret>|mid:<id>|hw:<uuid>|host:<name>|plat:<os/arch>` → `CredentialHMACHex(app_system_env, ...)`.
- Copying only the DB row or only the file to another host **MUST** fail login (fingerprint mismatch).
- Login **MUST NOT** create the enrollment file — register on the host first.
- Changing hostname or OS machine-id **requires re-register** on that host.

---

### FR-5 RBAC Synchronization (Internal)

> **Source:** `internal/rbacsync/`, `internal/jobs/`, `api/system/routes.go`

#### FR-5.1 Permission Sync

- The system **MUST** provide a mechanism to **upsert** `permissions` table rows from `rbaccatalog.AllPermissionEntries()` by `permission_id`.
- Extra rows present only in the database are **left unchanged** (non-destructive).
- Trigger options (all require system auth):
  - `POST /api/system/permission-sync-now` — runs synchronously, returns `{"synced": n}`.
  - `POST /api/system/create-permission-sync-job` — starts an in-memory 12-hour ticker.
  - `POST /api/system/delete-permission-sync-job` — stops the ticker.
- Also runnable from CLI: `go run ./cmd/syncpermissions`.

#### FR-5.2 Role-Permission Sync

- The system **MUST** provide a mechanism to **rebuild** `role_permissions` from `rbaccatalog.AllRolePermissionPairs()`.
- This is a **destructive** operation: all existing `role_permissions` rows are deleted, then re-inserted.
- Roles are resolved by name; `permission_id` values are taken directly from struct tags.
- Trigger options (all require system auth):
  - `POST /api/system/role-permission-sync-now` — runs synchronously, returns `{"rows": n}`.
  - `POST /api/system/create-role-permission-sync-job` — starts an in-memory 12-hour ticker.
  - `POST /api/system/delete-role-permission-sync-job` — stops the ticker.
- Also runnable from CLI: `go run ./cmd/syncrolepermissions`.

---

### FR-6 Internal RBAC HTTP API

> **Source:** `api/v1/internal/rbac_handler.go`, `api/v1/internal/routes.go`, `api/v1/routes.go`, `services/rbac/rbac.go`  
> **Auth:** `X-API-Key` header (via `middleware.RequireInternalAPIKey`)

All endpoints under `/api/internal-v1/rbac/` are protected by an internal API key and intended for back-office tooling, not end-user clients.

#### FR-6.1 Permissions CRUD

| Endpoint | Description |
|----------|-------------|
| `GET /api/internal-v1/rbac/permissions` | Paginated, sortable, searchable list of permissions |
| `POST /api/internal-v1/rbac/permissions` | Create a new permission |
| `PATCH /api/internal-v1/rbac/permissions/:permissionId` | Update a permission (`permission_id`, `permission_name`, `description`) |
| `DELETE /api/internal-v1/rbac/permissions/:permissionId` | Delete a permission (cascades to `role_permissions` and `user_permissions`) |

Constraints on `permission_id`: max 10 chars, alphanumeric.  
Constraints on `permission_name`: max 50 chars, must be unique.

#### FR-6.2 Roles CRUD

| Endpoint | Description |
|----------|-------------|
| `GET /api/internal-v1/rbac/roles` | List all roles; optional `?with_permissions=1` |
| `POST /api/internal-v1/rbac/roles` | Create a new role |
| `GET /api/internal-v1/rbac/roles/:id` | Get role by ID; optional `?with_permissions=1` |
| `PATCH /api/internal-v1/rbac/roles/:id` | Update role name/description |
| `PUT /api/internal-v1/rbac/roles/:id/permissions` | Replace all permissions on a role (full replace, not merge) |
| `DELETE /api/internal-v1/rbac/roles/:id` | Delete role and all its assignments |

#### FR-6.3 User–Role Assignments

| Endpoint | Description |
|----------|-------------|
| `GET /api/internal-v1/rbac/users/:userId/roles` | List roles assigned to a user |
| `POST /api/internal-v1/rbac/users/:userId/roles` | Assign a role to a user (idempotent) |
| `DELETE /api/internal-v1/rbac/users/:userId/roles/:roleId` | Remove a role from a user |

#### FR-6.4 User–Direct Permission Assignments

| Endpoint | Description |
|----------|-------------|
| `GET /api/internal-v1/rbac/users/:userId/permissions` | Effective permissions (roles + direct grants) |
| `GET /api/internal-v1/rbac/users/:userId/direct-permissions` | Direct-grant permissions only |
| `POST /api/internal-v1/rbac/users/:userId/direct-permissions` | Assign a direct permission (by `permission_id` or `permission_name`) |
| `DELETE /api/internal-v1/rbac/users/:userId/direct-permissions/:permissionId` | Remove a direct permission |

Response shapes (envelope `data`): effective permission codes use **`{ "permission_codes": string[] }`**; direct-grant rows use **`dto.RBACPermissionResponse[]`**; user role listings use **`dto.RBACRoleResponse[]`** (roles may include nested permissions when preloaded).

---

### FR-7 Health Check

- `GET /api/v1/health` **MUST** return `HTTP 200` with `{ "code": 0, "message": "ok", "status": "ok" }`.
- This endpoint requires no authentication and is used by CI/CD health probes.

---

### FR-8 Course Management

> **Status: Implemented in `internal/course/`.** See `docs/modules/course.md`.

- The system **MUST** provide course authoring endpoints under `/api/v1/courses` for create/read/update/delete and collaborator management.
- Paginated collaborator list (`GET …/collaborators`) **MUST** support `page`, `per_page`, and optional `search` on collaborator `display_name` / `email`.
- Instructor-candidate picker (`GET …/instructor-candidates`) **MUST** require dedicated permission **`course_collaborator_candidate:read` (P67)** at the route layer; repository **MUST** restrict picker access to course owners (`requireOwnerAccess`).
- The system **MUST** enforce role-gated review endpoints under `/api/v1/course-reviews` for pending queue, approve, and reject actions.
- The system **MUST** expose sysadmin catalog endpoints under `/api/v1/course-admin` for listing all non-trashed courses, listing trashed approved courses, moving eligible courses to trash, restoring from trash, and permanently deleting trashed courses (granular permissions **P62–P66**, not shell `admin:modify`).
- Trashed courses (`courses.trashed_at` set) **MUST** be excluded from edit, learn, and normal catalog lists; instructor delete of an approved published course **MUST** move the course to trash when eligible instead of immediate soft-delete.
- The system **MUST** support draft-first editing with one active draft per course, optimistic locking (`row_version`), and resource edit leases.
- Learner course delivery and enrollment/progress APIs **MUST** be exposed under `/api/v1/learner-courses`.

---

### FR-9 Lesson Management

> **Status: Implemented inside the Course bounded context.** See `docs/modules/lesson.md` and `docs/modules/course.md`.

- Lessons **MUST** belong to course sections in a specific course version and maintain sortable order.
- Sub-lessons **MUST** support `VIDEO`, `QUIZ`, and `TEXT` payloads with version-scoped persistence.
- Reorder operations for sections/lessons/sub-lessons **MUST** be validated against stable-id sets and applied atomically.

---

### FR-10 Enrollment

> **Status: Implemented inside the Course bounded context.** See `docs/modules/enrollment.md` and `docs/modules/course.md`.

- Learners **MUST** enroll through `POST /api/v1/learner-courses/:courseId/enroll`.
- Enrollment records **MUST** track the learner's active published version and prevent duplicates.
- Progress tracking **MUST** use stable content identifiers so progress can migrate across approved versions.

---

### FR-11 Media Upload Gateway

> **Status: Implemented.** See `docs/modules/media.md`.

- The system **MUST** provide unified media endpoints under `/api/v1/media/files` for `GET/POST/PUT/DELETE/OPTIONS`.
- The system **MUST** persist media cloud metadata in local DB (`media_files`) after successful create/update/delete sync while keeping provider upload execution in cloud services.
- The system **MUST** select provider from server configuration (`setting.MediaSetting.AppMediaProvider`), not client request fields.
- The system **MUST** infer typed metadata (`ImageMetadata` / `VideoMetadata` / `DocumentMetadata`) in backend.
- The system **MUST** keep **media** feature logic under **`internal/media/{application,infra}`**, **taxonomy** JSONB validators under **`internal/shared/taxonomy`**, Gin/HTTP helpers under **`internal/shared/{httperr,response,validate,utils}`**, and cross-domain primitives under **`internal/shared/*`**. The legacy **`pkg/logic/*`**, **`pkg/media`**, **`pkg/taxonomy`**, and **`pkg/requestutil`** trees **MUST NOT** be reintroduced.
- The system **MUST** restrict direct runtime `os.Getenv` reads in media runtime paths and use `setting.MediaSetting` as source-of-truth after `setting.Setup()`.
- The system **MUST** expose Bunny Stream delivery fields on successful media responses when available: **`video_id`**, **`thumbnail_url`**, **`embeded_html`**, **`direct_play_url`**, **`hls_playlist_url`**, **`preview_animation_url`** on `dto.UploadFileResponse`, persisted on **`media_files`** (migrations **`000005_media_bunny_response_fields`**, **`000021_media_bunny_delivery_urls`**) and in **`metadata_json`** using keys from **`internal/media/domain/meta_keys.go`** — see **`docs/modules/media.md`**. **`MEDIA_BUNNY_STREAM_CDN_HOSTNAME`** **MUST** be configured for CDN-backed URLs (thumbnail, HLS, preview).
- The system **MUST** reject a single uploaded file larger than **2 GiB** (`2×1024×1024×1024` bytes) on media create/update (handler + service), returning HTTP **413** and application code **2003** (`FileTooLarge`). The byte cap and the **single** oversize message constant **`constants.MsgFileTooLargeUpload`** **MUST** live in **`constants/error_msg.go`** and **MUST** be the same string used for the default JSON `message` for `FileTooLarge` in `pkg/errcode/messages.go` and for `pkg/errors.ErrFileExceedsMaxUploadSize` (no duplicate literals). See `docs/architecture.md` directory map. Deployment **MUST** configure reverse proxies / load balancers with a body limit **at least** that large on API routes so requests are not dropped before the application (see `docs/deploy.md`).
- The system **MUST** convert all image uploads (create + update) to **WebP** format using `bimg`/libvips (`CGO_ENABLED=1`) before sending to the storage provider. Concurrency **MUST** be bounded to `constants.MaxConcurrentImageEncode` simultaneous encode workers. Encode failure **MUST** return HTTP **503** + code **9017** (`ImageEncodeBusy`).
- The system **MUST** reject non-image, non-video file uploads (`POST /media/files`) when the file extension or magic bytes match the executable/script denylist. Rejected uploads **MUST** return HTTP **400** + code **2004** (`ExecutableUploadRejected`). Denylist logic **MUST** reside in `pkg/logic/utils/executable_check.go`.

---

## Non-Functional Requirements

### NFR-1 Performance & Availability

#### NFR-1.1 Rate Limiting

The system **MUST** enforce per-IP rate limits at the middleware layer:

| Route Group | Limit | Window |
|-------------|-------|--------|
| `/api/system` (`RateLimitSystemIP`) | 10 requests | 3 minutes |
| `/api/v1` unauthenticated (`RateLimitLocal`) | 60 requests | 1 minute |
| `/api/v1` authenticated (`RateLimitLocal`) | 120 requests | 1 minute |
| `/api/internal-v1` (`RateLimitLocal`) | 60 requests | 1 minute |
| APPCLI (`CLI_SYSTEM_LOGIN` + `CLI_REGISTER_NEW_SYSTEM_USER`, file-backed) | 5 operations | 3 minutes (shared per host) |

Rate-limit overrides can be set per IP via `middleware.SetSystemRateLimitOverride`.  
Exceeded HTTP limits return `HTTP 429` with app code `3006 TooManyRequests`.

#### NFR-1.1b Circuit Breaker

- A global circuit breaker (`internal/shared/resilience`) **MUST** protect HTTP ingress and APPCLI entry.
- **DB probe:** background `PingContext` on the primary pool; after `resilience.db_failures_to_open` consecutive failures (default 3) the circuit opens.
- **Load probe:** in-flight HTTP requests and rolling 5xx count can open the circuit when thresholds are exceeded.
- **Open state:** HTTP returns `503` + app code `9018 ServiceUnavailable`; APPCLI prints `Failure: service temporarily unavailable; try again later.`
- **Half-open:** allows a small probe quota; success closes the circuit; failure re-opens.
- Optional Redis key `mycourse:resilience:circuit` shares state when Redis is available; degrades to in-process only when Redis is down.
- Configuration: YAML `resilience:` block in `config/app.yaml` (see `setting.ResilienceSetting`).

#### NFR-1.2 Response Compression

All responses **MUST** be gzip-compressed by default (via `gin-contrib/gzip` at default compression level).

#### NFR-1.3 Caching

- Redis is used for auth caching with the following TTLs:
  - `mycourse:user:me:{user_id}` — 1 minute (profile + permissions)
  - `mycourse:auth:login:invalid:{normalized_email}` — 1 minute (negative login cache)
  - `mycourse:auth:login:user_by_email:{normalized_email}` — 30 seconds (email → user ID)
- If Redis is unavailable, all cache helpers degrade gracefully to no-ops.

#### NFR-1.4 Session Limits

- A user **MUST NOT** have more than **5 concurrent sessions**.
- When the limit is reached, the session with the earliest `refresh_token_expired` is evicted atomically.

#### NFR-1.5 Startup & Build

- The server **MUST** listen on the port defined by `setting.ServerSetting.Port` (default **8080**).
- Database migrations support env modes:
  - `MIGRATE=1` applies pending up migrations, then continues normal server startup.
  - `MIGRATE=2` + `MIGRATE_VERSION_FILE=<file>.down.sql` rolls back to `version(file)-1` then exits.
- PostgreSQL app schema **MUST** be configurable via optional env `SCHEMA_NAME_APP` (YAML `database.schema_name`); when unset, the backend **MUST** default to `public` and set `search_path` on every GORM connection (see **`docs/database.md`**).
- The binary supports a one-shot CLI flow for privileged user registration (`CLI_REGISTER_NEW_SYSTEM_USER=1`).

#### NFR-1.6 Test code layout

- All new test code (**unit**, integration suites, black-box packages importing `mycourse-io-be`, cross-feature harnesses, and shared fixtures) **MUST** be added under repository root **`tests/`** and **MUST NOT** be placed under `api/`, `services/`, or `pkg/`.
- The canonical description of this split lives in **`docs/patterns.md`** and **`README.md`** (section **Testing**).

---

### NFR-2 Security

#### NFR-2.1 Password Storage

- All passwords **MUST** be hashed with **bcrypt** at the default cost before storage.
- Raw passwords **MUST NOT** be logged or stored.

#### NFR-2.2 JWT Security

- Access tokens **MUST** be HS256-signed JWTs with a 15-minute TTL.
- Refresh tokens **MUST** be HS256-signed JWTs. Their expiry is authoritative in the database, not the JWT `exp` field alone.
- The JWT secret **MUST** be provided via the `JWT_SECRET` environment variable (never hardcoded).
- Expired tokens **MUST** cause a `401` response with the `X-Token-Expired: true` header.

#### NFR-2.3 Cookie Security

- Auth cookies (`access_token`, `refresh_token`, `session_id`) **MUST** be set with `SameSite=Lax` and **`HttpOnly=true`**.
- In production (`RunMode=release`) cookies **MUST** have `Secure=true`.
- The backend **MUST** accept the access JWT from `Authorization: Bearer …` **or** the HttpOnly `access_token` cookie. Refresh/logout **MUST** accept session credentials from custom headers **or** HttpOnly cookies.

#### NFR-2.4 CORS

- The `CORS_ALLOWED_ORIGINS` environment variable **MUST** control the list of allowed origins (comma-separated).
- Credentials **MUST** be allowed (`AllowCredentials: true`).
- Exposed headers: `X-Token-Expired`.
- Allowed custom request headers: `Authorization`, `X-API-Key`, `X-Refresh-Token`, `X-Session-Id`.

#### NFR-2.5 System Credential Security

- System privileged user credentials **MUST** be stored as HMAC-hex derivations using `app_system_env`, never as plaintext.
- `system_app_config.app_cli_system_password`, `app_system_env`, and `app_token_env` **MUST** be bcrypt hashes (cost 14) at rest; the hash strings are used as HMAC/JWT key material.
- System access tokens are short-lived JWTs signed with `app_token_env` (stored in the `system_app_config` singleton row).

#### NFR-2.6 Internal API Protection

- All routes under `/api/internal-v1` **MUST** require a valid `X-API-Key` header.
- The API key is validated by `middleware.RequireInternalAPIKey`.

#### NFR-2.7 Input Validation

- All request bodies **MUST** be validated via Gin's `ShouldBindJSON` / `ShouldBindQuery` with binding tags.
- Validation failures **MUST** return `HTTP 400` with app code `2001 ValidationFailed`.

---

### NFR-3 Observability & Maintainability

#### NFR-3.1 Unified Response Envelope

- Every API response **MUST** use the `pkg/response` helpers. The envelope format is:
  - Standard: `{ "code": int, "message": string, "data": any }`
  - Health: `{ "code": int, "message": string, "status": string }`
  - Paginated: `data` is a `PaginatedData` object with `result` + `page_info`.
- Raw `gin.H` envelopes in handlers are **NOT** permitted.

#### NFR-3.2 Error Codes

- All error scenarios **MUST** use the numeric application error codes defined in `pkg/errcode/codes.go`.
- Default messages for each code are defined in `pkg/errcode/messages.go`.
- Reusable functional/sentinel errors (`Err*`) and typed feature errors **MUST** be defined in `pkg/errors` (not inside `services/*`, `repository/*`, or feature runtime packages).

#### NFR-3.2.1 Shared Type Placement

- For all new code from now on, if a module under `pkg/*` contains logic handling, newly introduced reusable types **MUST** be declared in `pkg/entities` instead of inline declaration in that logic module file.
- New reusable types **MUST NOT** be introduced in logic-orchestration layers (`services/*`, `repository/*`, or other business-flow files).
- New and reused domain types **MUST** be placed under `pkg/entities` (new module file or existing module file), then imported where needed.

#### NFR-3.2.2 Helper vs Util Placement

- Cross-feature generic logic/functions (e.g. parse/url/normalize/general transformers) **MUST** be implemented under `pkg/logic/utils`.
- Feature-specific logic tied to **media** (resolver, metadata, multipart bind, orphan URL policy, …) **MUST** live under **`internal/media/`** layers. Taxonomy tree/description validation **MUST** live in **`internal/shared/taxonomy`** and be consumed by **`internal/taxonomy`**.
- For functions created inside `application/` or `infra/`: if the logic is standalone or expected to be reused across modules, extract to **`internal/shared/utils`** (generic), **`internal/shared/taxonomy`** (taxonomy JSONB), or the owning domain’s `application`/`infra` package, as appropriate.

#### NFR-3.3 Structured Logging

- The application **MUST** use the structured logger in `pkg/logger/logger.go` (based on `zap`).
- Log output must include at minimum: timestamp, level, message, and relevant context fields.

#### NFR-3.4 Panic Recovery

- The `internal/shared/httperr.Recovery()` middleware **MUST** catch unhandled panics, log them, and return `HTTP 500` with app code `9998 Panic` rather than crashing the process.

#### NFR-3.5 Configuration Management

- All environment-specific configuration **MUST** be provided via YAML files (`config/app.yaml`, `config/app-<STAGE>.yaml`) with environment variable substitution via `pkg/setting`.
- No secrets (JWT keys, DB passwords, API keys) may be hardcoded in source code.

#### NFR-3.6 Database Migrations

- Schema changes **MUST** be implemented as numbered, embedded SQL migration files under `migrations/` and applied via `golang-migrate`.
- A companion `000001_schema.down.sql` **MUST** exist to support rollback.

#### NFR-3.7 CI/CD

- Pushing to the `master` branch **MUST** trigger `.github/workflows/deploy-dev.yml`:
  - Install `libvips-dev libhdf5-dev pkg-config` on the runner after **`vegardit/fast-apt-mirror.sh`** (so downloads use the fast mirror). **`libhdf5-dev`** supplies **`hdf5.pc`** for **matio** (**CGO_ENABLED=1** / bimg).
  - Build the `mycourse-io-be-dev` binary with `CGO_ENABLED=1`.
  - On the deploy host, back up **only** the current dev binary to `bin/mycourse-io-be-dev.prev`, then `rsync` the new binary onto `bin/mycourse-io-be-dev` (ecosystem is **not** backed up in CI; the script owns `ecosystem.config.cjs.prev`).
  - Run `scripts/pm2-reload-with-binary-rollback.sh`: snapshot `ecosystem.config.cjs`, pull **only** that file from `origin/master`, reload PM2, health-check `GET /api/v1/health`, treat PM2 autorestart exhaustion as failure; on success, full `git pull`; on failure, restore previous binary **and** ecosystem from `.prev`, reload, and health-check again.
  - `ecosystem.config.cjs` **MUST** cap crash loops with `min_uptime` and `max_restarts: 3` for every PM2 app entry (dev, staging, prod).
