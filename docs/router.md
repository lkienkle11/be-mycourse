# Router

## Initialization

Router is created in `internal/server/router.go` → `InitRouter(svc *Services, h *Handlers) *gin.Engine`.

Key setup:
- `router.MaxMultipartMemory = constants.MediaMultipartParseMemoryBytes` (64 MiB) — large multipart bodies spill to temp disk during parse; per-part and aggregate caps are enforced in the media handler.
- `main.go` runs `router.Run(":"+setting.ServerSetting.Port)` after wiring.

---

## Global Middleware Stack

Applied to all routes in this order:

1. `middleware.RequestLogger()` — structured access log + `X-Request-ID` propagation
2. `middleware.CircuitBreakerMiddleware()` — global circuit breaker + load tracking
3. `internal/shared/httperr.Middleware()` — centralized error handling
3. `internal/shared/httperr.Recovery()` — panic recovery + stack log
4. `cors.New(ginDefaultCORS())` — CORS with `AllowCredentials: true`
5. `gzip.Gzip(gzip.DefaultCompression)` — response compression
6. CSRF middleware logic exists in codebase, but enforcement is temporarily disabled at router level

---

## Route Groups

### `/api/system` — Privileged system operations

```
Middleware: BeforeInterceptor, RateLimitSystemIP(10 req / 3 min per IP)
```

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/system/permission-sync-now` | System token | Immediate permission sync |
| POST | `/api/system/role-permission-sync-now` | System token | Immediate role-permission sync |
| POST | `/api/system/create-permission-sync-job` | System token | Start 12h periodic permission sync job |
| POST | `/api/system/create-role-permission-sync-job` | System token | Start 12h periodic role-permission sync job |
| POST | `/api/system/delete-permission-sync-job` | System token | Stop permission sync job |
| POST | `/api/system/delete-role-permission-sync-job` | System token | Stop role-permission sync job |

---

### `/api/v1` no-filter lane — Webhook callbacks

```
Middleware: none (mounted before BeforeInterceptor)
```

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/webhook/bunny` | None | Bunny Stream video status webhook |

---

### `/api/v1` unauthenticated — Public endpoints

```
Middleware: BeforeInterceptor, RateLimitLocal(60 req / 1 min)
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/auth/csrf` | Bootstrap CSRF token endpoint (currently optional while CSRF filter is disabled) |
| POST | `/api/v1/auth/register` | Register (pending user + confirmation email) |
| POST | `/api/v1/auth/login` | Login — issues access/refresh tokens |
| POST | `/api/v1/auth/confirm` | Confirm email from FE-submitted token (issues tokens on success) |
| POST | `/api/v1/auth/refresh` | Rotate token pair via `X-Refresh-Token` / `X-Session-Id` headers |
| POST | `/api/v1/auth/logout` | Revoke session + clear auth cookies (`X-Refresh-Token` / `X-Session-Id`) |

---

### `/api/v1` authenticated — Protected user endpoints

```
Middleware: BeforeInterceptor, RateLimitLocal(120 req / 1 min), AuthJWT
```

#### Auth / Me

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/me` | None (JWT only) | Get current user profile |
| PATCH | `/api/v1/me` | None (JWT only) | Update current user profile |
| DELETE | `/api/v1/me` | None (JWT only) | Soft-delete current user account |
| DELETE | `/api/v1/me/hard` | None (JWT only) | Permanently delete current user account |
| GET | `/api/v1/me/permissions` | `user:read` | Get current user permission list |

#### Taxonomy

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/taxonomy/levels` | `course_level:read` | List active course levels |
| GET | `/api/v1/taxonomy/levels/full` | `course_level:read` | List course levels including soft-deleted |
| POST | `/api/v1/taxonomy/levels` | `course_level:create` | Create course level |
| PATCH | `/api/v1/taxonomy/levels/:id` | `course_level:update` | Update course level |
| DELETE | `/api/v1/taxonomy/levels/:id` | `course_level:delete` | Soft-delete course level |
| DELETE | `/api/v1/taxonomy/levels/:id/hard` | `course_level:delete` | Hard-delete course level |
| GET | `/api/v1/taxonomy/topics` | `topic:read` | List active course topics |
| GET | `/api/v1/taxonomy/topics/full` | `topic:read` | List topics including soft-deleted |
| POST | `/api/v1/taxonomy/topics` | `topic:create` | Create course topic |
| PATCH | `/api/v1/taxonomy/topics/:id` | `topic:update` | Update course topic |
| DELETE | `/api/v1/taxonomy/topics/:id` | `topic:delete` | Soft-delete course topic |
| DELETE | `/api/v1/taxonomy/topics/:id/hard` | `topic:delete` | Hard-delete course topic (orphan image cleanup) |
| GET | `/api/v1/taxonomy/outcomes` | `course_outcome:read` | List active course outcomes |
| GET | `/api/v1/taxonomy/outcomes/full` | `course_outcome:read` | List outcomes including soft-deleted |
| POST | `/api/v1/taxonomy/outcomes` | `course_outcome:create` | Create course outcome |
| PATCH | `/api/v1/taxonomy/outcomes/:id` | `course_outcome:update` | Update course outcome |
| DELETE | `/api/v1/taxonomy/outcomes/:id` | `course_outcome:delete` | Soft-delete course outcome |
| DELETE | `/api/v1/taxonomy/outcomes/:id/hard` | `course_outcome:delete` | Hard-delete course outcome (orphan image cleanup) |
| GET | `/api/v1/taxonomy/skills` | `course_skill:read` | List active course skills |
| GET | `/api/v1/taxonomy/skills/full` | `course_skill:read` | List skills including soft-deleted |
| POST | `/api/v1/taxonomy/skills` | `course_skill:create` | Create course skill |
| PATCH | `/api/v1/taxonomy/skills/:id` | `course_skill:update` | Update course skill |
| DELETE | `/api/v1/taxonomy/skills/:id` | `course_skill:delete` | Soft-delete course skill |
| DELETE | `/api/v1/taxonomy/skills/:id/hard` | `course_skill:delete` | Hard-delete course skill |
| GET | `/api/v1/taxonomy/tags` | `tag:read` | List active tags |
| GET | `/api/v1/taxonomy/tags/full` | `tag:read` | List tags including soft-deleted |
| POST | `/api/v1/taxonomy/tags` | `tag:create` | Create tag |
| PATCH | `/api/v1/taxonomy/tags/:id` | `tag:update` | Update tag |
| DELETE | `/api/v1/taxonomy/tags/:id` | `tag:delete` | Soft-delete tag |
| DELETE | `/api/v1/taxonomy/tags/:id/hard` | `tag:delete` | Hard-delete tag |

#### Media Files

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| OPTIONS | `/api/v1/media/files` | — | CORS preflight |
| GET | `/api/v1/media/files` | `media_file:read` | List media files (paginated) |
| GET | `/api/v1/media/files/cleanup-metrics` | `media_file:read` | Orphan cleanup counters |
| POST | `/api/v1/media/files` | `media_file:create` | Upload 1–5 file parts |
| OPTIONS | `/api/v1/media/files/batch-delete` | — | CORS preflight |
| POST | `/api/v1/media/files/batch-delete` | `media_file:delete` | Batch delete up to 10 files |
| OPTIONS | `/api/v1/media/files/:id` | — | CORS preflight |
| GET | `/api/v1/media/files/:id` | `media_file:read` | Get single file |
| PUT | `/api/v1/media/files/:id` | `media_file:update` | Bundle update (1–5 parts) |
| DELETE | `/api/v1/media/files/:id` | `media_file:delete` | Delete single file |
| OPTIONS | `/api/v1/media/files/local/:token` | — | CORS preflight |
| GET | `/api/v1/media/files/local/:token` | `media_file:read` | Decode local signed URL token |
| GET | `/api/v1/media/videos/:id/status` | `media_file:read` | Bunny video processing status |

#### Course

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/courses/my` | `course_instructor:read` | List editable courses |
| POST | `/api/v1/courses` | `course:create` | Create course root |
| GET | `/api/v1/courses/:courseId` | `course_instructor:read` | Get course detail |
| POST | `/api/v1/courses/:courseId/draft/prepare` | `course:update` | Ensure one active draft |
| PATCH | `/api/v1/courses/:courseId/basic-info` | `course:update` | Update draft basic info (`title` → server slugify updates `courses.slug`) |
| DELETE | `/api/v1/courses/:courseId` | `course:delete` | Delete course (owner-only in service) |
| GET | `/api/v1/courses/:courseId/collaborators` | `course_instructor:read` | List collaborators |
| POST | `/api/v1/courses/:courseId/collaborators` | `course:update` | Add collaborator |
| DELETE | `/api/v1/courses/:courseId/collaborators/:userId` | `course:update` | Remove collaborator |
| POST | `/api/v1/courses/:courseId/sections` | `course:update` | Create section |
| PATCH | `/api/v1/courses/:courseId/sections/:sectionId` | `course:update` | Update section |
| DELETE | `/api/v1/courses/:courseId/sections/:sectionId` | `course:update` | Delete section |
| POST | `/api/v1/courses/:courseId/sections/reorder` | `course:update` | Reorder sections |
| POST | `/api/v1/courses/:courseId/lessons` | `course:update` | Create lesson |
| PATCH | `/api/v1/courses/:courseId/lessons/:lessonId` | `course:update` | Update lesson |
| DELETE | `/api/v1/courses/:courseId/lessons/:lessonId` | `course:update` | Delete lesson |
| POST | `/api/v1/courses/:courseId/sections/:sectionId/lessons/reorder` | `course:update` | Reorder lessons |
| POST | `/api/v1/courses/:courseId/sub-lessons` | `course:update` | Create sub-lesson (`VIDEO`/`QUIZ`/`TEXT`) |
| PATCH | `/api/v1/courses/:courseId/sub-lessons/:subLessonId` | `course:update` | Update sub-lesson |
| DELETE | `/api/v1/courses/:courseId/sub-lessons/:subLessonId` | `course:update` | Delete sub-lesson |
| POST | `/api/v1/courses/:courseId/lessons/:lessonId/sub-lessons/reorder` | `course:update` | Reorder sub-lessons |
| POST | `/api/v1/courses/:courseId/leases/acquire` | `course:update` | Acquire edit lease |
| POST | `/api/v1/courses/:courseId/leases/heartbeat` | `course:update` | Refresh edit lease |
| POST | `/api/v1/courses/:courseId/leases/release` | `course:update` | Release edit lease |
| POST | `/api/v1/courses/:courseId/submit-review` | `course:update` | Submit draft for review |
| POST | `/api/v1/courses/:courseId/reopen-draft` | `course:update` | Reopen rejected draft |
| GET | `/api/v1/course-reviews/pending` | `admin:modify` | List pending drafts |
| POST | `/api/v1/course-reviews/:courseId/approve` | `admin:modify` | Approve draft/publish |
| POST | `/api/v1/course-reviews/:courseId/reject` | `admin:modify` | Reject draft with reason |
| GET | `/api/v1/learner-courses` | `course:read` | List published learner catalog |
| GET | `/api/v1/learner-courses/:courseId` | `course:read` | Get learning course detail |
| POST | `/api/v1/learner-courses/:courseId/enroll` | `course:read` | Enroll learner |
| GET | `/api/v1/learner-courses/:courseId/progress` | `course:read` | Get learner progress |
| POST | `/api/v1/learner-courses/:courseId/progress` | `course:read` | Save learner progress item |

#### Instructor Management

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/api/v1/instructors` | `instructor_roster:read` | List roster |
| POST | `/api/v1/instructors` | `instructor_roster:create` | Add roster by email |
| DELETE | `/api/v1/instructors/:id` | `instructor_roster:delete` | Remove instructor role |
| GET | `/api/v1/instructors/:id/expertise/topics` | `instructor_expertise:read` | List expertise topics |
| POST | `/api/v1/instructors/:id/expertise/topics` | `instructor_expertise:create` | Add expertise topic |
| DELETE | `/api/v1/instructors/:id/expertise/topics/:topicRowId` | `instructor_expertise:delete` | Delete expertise topic row |
| GET | `/api/v1/instructors/:id/expertise/skills` | `instructor_expertise:read` | List expertise skills |
| POST | `/api/v1/instructors/:id/expertise/skills` | `instructor_expertise:create` | Add expertise skill |
| DELETE | `/api/v1/instructors/:id/expertise/skills/:skillRowId` | `instructor_expertise:delete` | Delete expertise skill row |
| GET | `/api/v1/instructor-applications` | `instructor_application:read` | List applications |
| POST | `/api/v1/instructor-applications` | `instructor_application:create` | Submit application |
| GET | `/api/v1/instructor-applications/:id` | `instructor_application:read` | Get application |
| POST | `/api/v1/instructor-applications/:id/approve` | `instructor_application:approve` | Approve application |
| POST | `/api/v1/instructor-applications/:id/reject` | `instructor_application:reject` | Reject application |
| DELETE | `/api/v1/instructor-applications/:id` | `instructor_application:delete` | Delete application |
| GET | `/api/v1/instructor-profiles` | `instructor_profile:read` | List profiles |
| GET | `/api/v1/instructor-profiles/me` | `instructor_profile:read` | Get own profile |
| GET | `/api/v1/instructor-profiles/:id` | `instructor_profile:read` | Get profile by user |
| POST | `/api/v1/instructor-profiles` | `instructor_profile:create` | Upsert own profile |
| PATCH | `/api/v1/instructor-profiles/:id` | `instructor_profile:update` | Upsert profile by user |
| DELETE | `/api/v1/instructor-profiles/:id` | `instructor_profile:delete` | Delete profile |
| GET | `/api/v1/instructor-tickets` | `instructor_application:read` | List tickets |
| POST | `/api/v1/instructor-tickets` | `instructor_application:create` | Create ticket |
| POST | `/api/v1/instructor-tickets/:id/close` | `instructor_ticket:close` | Close ticket |
| GET | `/api/v1/instructor-tickets/:id/messages` | `instructor_application:read` | List ticket messages |
| POST | `/api/v1/instructor-tickets/:id/messages` | `instructor_application:create` | Add ticket message |
| GET | `/api/v1/instructor-stubs/assignments` | `instructor_profile:read` | Placeholder route |
| GET | `/api/v1/instructor-stubs/activity-log` | `instructor_profile:read` | Placeholder route |

---

### `/api/internal-v1` — Internal RBAC administration

```
Middleware: RateLimitLocal(60 req / 1 min), BeforeInterceptor, RequireInternalAPIKey
```

#### RBAC Permissions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/internal-v1/rbac/permissions` | List all permissions |
| POST | `/api/internal-v1/rbac/permissions` | Create a permission |
| PATCH | `/api/internal-v1/rbac/permissions/:permissionId` | Update a permission |
| DELETE | `/api/internal-v1/rbac/permissions/:permissionId` | Delete a permission |

#### RBAC Roles

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/internal-v1/rbac/roles` | List all roles |
| POST | `/api/internal-v1/rbac/roles` | Create a role |
| GET | `/api/internal-v1/rbac/roles/:id` | Get role by ID |
| PATCH | `/api/internal-v1/rbac/roles/:id` | Update a role |
| PUT | `/api/internal-v1/rbac/roles/:id/permissions` | Set (replace) role permissions |
| DELETE | `/api/internal-v1/rbac/roles/:id` | Delete a role |

#### RBAC User Bindings

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/internal-v1/rbac/users/:userId/roles` | List user roles |
| GET | `/api/internal-v1/rbac/users/:userId/permissions` | List user effective permissions |
| GET | `/api/internal-v1/rbac/users/:userId/direct-permissions` | List user direct permissions |
| POST | `/api/internal-v1/rbac/users/:userId/roles` | Assign role to user |
| DELETE | `/api/internal-v1/rbac/users/:userId/roles/:roleId` | Remove role from user |
| POST | `/api/internal-v1/rbac/users/:userId/direct-permissions` | Assign direct permission to user |
| DELETE | `/api/internal-v1/rbac/users/:userId/direct-permissions/:permissionId` | Remove direct permission from user |

---

## Authorization Summary

| Group | Auth mechanism |
|-------|---------------|
| `/api/system` | System JWT (`RequireSystemAccessToken`) |
| `/api/v1` no-filter | None |
| `/api/v1` unauthenticated | None (rate limited); CSRF validation is temporarily disabled |
| `/api/v1` authenticated | `AuthJWT` (Bearer token) + `RequirePermission` per endpoint; CSRF validation is temporarily disabled |
| `/api/internal-v1` | `RequireInternalAPIKey` (`X-API-Key` header) |

---

## Route Source Files

| Route group | Source |
|-------------|--------|
| System routes | `internal/system/delivery/routes.go` |
| Auth / me routes | `internal/auth/delivery/routes.go` |
| Taxonomy routes | `internal/taxonomy/delivery/routes.go` |
| Media routes | `internal/media/delivery/routes.go` |
| Media webhook routes | `internal/media/delivery/routes.go` (`RegisterWebhookRoutes`) |
| RBAC routes | `internal/rbac/delivery/routes.go` |
| Router mounting | `internal/server/router.go` |
