# Data Flow

## Primary Request Flow

1. HTTP request enters the Gin router (`internal/server/router.go`).
2. Global middleware executes: `RequestLogger` (structured access log + `X-Request-ID`), `CircuitBreakerMiddleware`, `httperr.Middleware`, `httperr.Recovery`, CORS, gzip.
3. Group middleware executes based on route group (rate limit, JWT auth, active-user DB check, API key, system token).
4. Delivery handler (in `internal/<domain>/delivery/`) binds and validates the request DTO, then calls the application service.
5. Application service (`internal/<domain>/application/`) executes business logic using injected domain interfaces (repos, external clients).
6. Infrastructure implementations (`internal/<domain>/infra/`) perform DB queries, cloud API calls, etc.
7. Handler maps domain result to response DTO and writes the standard JSON envelope via `internal/shared/response`.

---

## End-to-End Flows

### Auth Register

```
POST /api/v1/auth/register
  └─ internal/auth/delivery/handler.go (Register)
       └─ AuthService.Register
            ├─ Check email uniqueness via UserRepository
            ├─ Hash password via crypto
            ├─ Create pending user row (or update unconfirmed)
            ├─ Check Postgres lifetime cap (users.registration_email_send_total ≤ 15)
            │   └─ If exceeded: delete pending user → 410 / 4009 RegistrationAbandoned
            ├─ Check Redis sliding window (5 sends / 4h)
            │   └─ If exceeded: 429 / 4010 + Retry-After headers
            └─ Send confirmation email via Brevo
                 └─ On Brevo failure: release Redis slot → 502 / 4011
```

### Auth Login

```
POST /api/v1/auth/login
  └─ internal/auth/delivery/handler.go (Login)
       └─ AuthService.Login
            ├─ Check Redis negative cache (mycourse:auth:login:invalid:{email}) — skip DB when prior active-user lookup was not found
            ├─ Load user from DB (email→user_id Redis cache skips email query; active rows only; cache hit still FindByID)
            ├─ checkUserAccessible (application/service_access.go)
            ├─ Verify password (bcrypt) — only after checkUserAccessible passes and email is confirmed; wrong password does not set negative cache
            ├─ Issue access token + refresh token + session_id
            ├─ Persist refresh session to users.refresh_token_session JSONB
            │   └─ Evict oldest session if count > MaxActiveSessions (5)
            ├─ Set auth cookies (HttpOnly, SameSite=Lax)
            └─ Return JSON body with tokens (access_token, refresh_token, session_id)
```

### Auth Token Refresh

```
POST /api/v1/auth/refresh
  └─ internal/auth/delivery/handler.go (RefreshToken)
       └─ AuthService.RefreshSession
            ├─ Parse X-Refresh-Token header (refresh JWT)
            ├─ Parse X-Session-Id header
            ├─ Load user (active only) + checkUserAccessible
            ├─ Load session entry from users.refresh_token_session JSONB
            ├─ Validate UUID and expiry
            ├─ Issue new access token + refresh token
            ├─ Update session entry in-place (count unchanged)
            └─ Return new tokens (same JSON shape as login)
```

### Auth Logout

```
POST /api/v1/auth/logout
  └─ internal/auth/delivery/handler.go (Logout)
       └─ AuthService.Logout
            ├─ Parse X-Refresh-Token + X-Session-Id headers
            ├─ Load session entry from users.refresh_token_session JSONB
            ├─ Remove session key (idempotent if already absent)
            ├─ delCachedMe(userID)
            └─ clearAuthCookies on response (always, including 401)
```

### GET /me

```
GET /api/v1/me (JWT required)
  └─ internal/auth/delivery/handler.go (GetMe)
       └─ AuthService.GetMe
            ├─ Check Redis cache (mycourse:user:me:{user_id}) — up to 1 minute stale
            ├─ On miss: loadAccessibleUser → checkUserAccessible → permissions from DB
            ├─ Cache result
            └─ Return MeResponse DTO
```

### DELETE /me (soft) and DELETE /me/hard

```
DELETE /api/v1/me | DELETE /api/v1/me/hard (JWT required)
  └─ internal/auth/delivery/handler.go (DeleteMe | HardDeleteMe)
       └─ AuthService.SoftDeleteUser | HardDeleteUser
            ├─ loadAccessibleUser → checkUserAccessible
            ├─ SoftDelete (gormx.SoftDeleteWithAudit) | HardDelete (GORM Delete)
            ├─ delCachedMe(userID)
            └─ Enqueue avatar orphan cleanup when avatar_file_id set
```

### Permission Check (middleware)

```
middleware.RequirePermission(checker, "resource:action")
  ├─ Check JWT embedded permissions (fast path)
  └─ On miss: RBACService.PermissionCodesForUser → DB lookup
       └─ 403 Forbidden if permission absent
```

### System Sync

```
POST /api/system/permission-sync-now  (system JWT required)
  └─ internal/system/delivery/handler.go (permissionSyncNow)
       └─ SystemService → PermissionSyncer.SyncNow
            └─ Upsert permissions from constants.AllPermissions into DB
```

### Taxonomy CRUD

Default list/get: active rows only (`gormx.ScopeActiveOnly`). Soft delete: `DELETE /:id`. Hard delete: `DELETE /:id/hard`. Include deleted: `GET /full`. See **`docs/patterns.md`**.

```
POST /api/v1/taxonomy/topics  (JWT + topic:create required)
  └─ internal/taxonomy/delivery/handler.go
       └─ TaxonomyService.CreateTopic
            ├─ shared/taxonomy.ValidateTree(child_topics)
            ├─ Validate image_file_id via MediaFileValidator interface
            ├─ Create row via GormCourseTopicRepository
            └─ Return CourseTopicResponse DTO
```

### Media Upload

```
POST /api/v1/media/files  (JWT + media_file:create required)
  └─ internal/media/delivery/handler.go (createFile)
       └─ MediaService.CreateFiles
            ├─ Bind multipart form (1–5 parts)
            ├─ Validate per-part size (≤ 2 GiB) and aggregate size (≤ 2 GiB)
            ├─ For each part (up to 5 concurrent workers):
            │   ├─ Trusted MIME routing (`MIMEForUploadRouting` → kind/provider/WebP decisions)
            │   ├─ Executable denylist check (non-image/non-video)
            │   ├─ WebP encoding (images — bimg/libvips CGO)
            │   ├─ Storage MIME canonicalization (`CanonicalStorageMIME` + `applyStorageMIMEPolicy`)
            │   └─ Provider upload (R2 with S3 ContentType from canonical MIME, or Bunny Stream)
            ├─ Infer typed metadata (image/video/document)
            ├─ Persist rows to media_files
            └─ Return array of UploadFileResponse
```

### Media Bunny Webhook

```
POST /webhook/bunny  (no-auth, no rate limit — no-filter lane)
  └─ internal/media/delivery/handler.go (bunnyWebhook)
       ├─ Verify HMAC-SHA256 signature on raw body
       ├─ Parse + validate JSON (VideoLibraryId, VideoGuid, Status)
       └─ MediaService.HandleBunnyVideoWebhook
            ├─ Status 3/4 (finished): load row, fetch Bunny detail,
            │   merge metadata (video_id, thumbnail_url, embeded_html, direct_play_url,
            │   hls_playlist_url, preview_animation_url, telemetry),
            │   upsert media_files row
            ├─ Status 5/8 (failed): mark row FAILED (idempotent)
            └─ Other statuses: intentionally ignored
```

### Orphan Image Cleanup

```
Background: OrphanEnqueuer.EnqueueOrphanCleanupForFileID(fileID)
  └─ Load File row from FileRepository
  └─ Insert media_pending_cloud_cleanup row (skip Local provider)

Background job (cleanup_scheduler.go — started in main.go):
  └─ Poll media_pending_cloud_cleanup rows in batches
  └─ Delete cloud objects (R2 or Bunny)
  └─ Mark rows done / remove from table
```

### Course Authoring + Review

```
POST/PATCH /api/v1/courses/*  (JWT + course:update)
  └─ internal/course/delivery/*
       └─ CourseService (application)
            ├─ validate collaborator/owner access in repo layer
            ├─ mutate draft version rows with row_version checks
            ├─ maintain stable_id for outline resources
            └─ persist version-scoped sections/lessons/sub-lessons

POST /api/v1/courses/:courseId/draft/prepare
  └─ owner-only (requireOwnerAccess); fork next DRAFT from published when none exists

POST /api/v1/courses/:courseId/submit-review
  └─ owner-only; transition DRAFT -> IN_REVIEW (same version_no; validateDraftForReview inside tx)

POST /api/v1/course-reviews/:courseId/approve|reject  (course_review:approve|reject)
  └─ approve: IN_REVIEW -> APPROVED; courses.current_published_version_id = version; clear draft pointer
  └─ reject: IN_REVIEW -> REJECTED on submitted row; createNextDraftVersion (max+1) -> new DRAFT pointer

POST /api/v1/courses/:courseId/reopen-draft
  └─ owner-only; legacy REJECTED draft pointer -> fork new DRAFT at max+1 (clone from rejected row)
```

### Learner Enrollment + Progress

```
POST /api/v1/learner-courses/:courseId/enroll
  └─ CourseService.Enroll
       ├─ ensure course has published version
       ├─ create/update enrollment row
       └─ set current_version_id for learner

GET/POST /api/v1/learner-courses/:courseId/progress
  └─ CourseService.GetProgress / SaveProgress
       ├─ load enrollment + current version
       ├─ read/write progress keyed by stable_content_id
       └─ keep progress portable across approved version switches
```

---

## Persistence Boundaries

- **PostgreSQL** via GORM (primary datastore for all domains).
- **Redis** optional cache: auth `/me` responses, login negative cache, email confirmation sliding window.
- **LavinMQ** optional topic messaging: publish/consume via `internal/shared/mq` when `LAVINMQ_ENABLED=true` and `CLOUDAMQP_URL` is set (CloudAMQP console AMQP URL). Default exchange `amq.topic`; routing keys in `internal/shared/constants/mq_topics.go`.
- **SQL migrations** via embedded files + `golang-migrate` (`shareddb.MigrateDatabase()`).
- **Cloudflare R2 / BunnyCDN** object storage for media files.

---

## Data-Risk Hotspots

- Permission staleness: JWT embedded permissions can lag up to 1 minute after RBAC changes (Redis cache TTL).
- `role_permissions` full rebuild via sync can immediately revoke access if `constants.RolePermissions` is incomplete.
- Session JSONB map in `users` centralizes all refresh sessions for a user in a single row — concurrent logins use a DB transaction to prevent race conditions.
- Bunny webhook must be idempotent — the same status can be delivered multiple times.
