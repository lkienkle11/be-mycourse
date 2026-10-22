# Shared / Reusable Assets

All reusable utilities, functions, types, constants, middleware, and shared logic across `be-mycourse`'s `internal/shared/` and `pkg/` packages. **Check this file before creating any new utility, middleware, service helper, or shared type** to prevent duplication — this is what AGENTS.md's "Critical Reuse Rule" points agents to check first.

**Shared packages:** `internal/shared/{db,gormx,timex,response,middleware,ratelimit,resilience,machineidentity,constants,setting,cache,mq,logger,httperr,parsebool,taxonomy,i18n,validate,utils,httpx,userpicker,slug,token,errors,uuidx,xdgx,cryptox,useraccess,brevo,mailtmpl,mediaquery,requestprincipal}`. Public `pkg/` currently holds `pkg/supabase`, `pkg/media`, `pkg/logic`, `pkg/errors`, `pkg/errors_func`, `pkg/errcode`, `pkg/sqlnamed`, `pkg/query`, `pkg/token`, `pkg/response`, `pkg/setting`. Domain types live in `internal/<domain>/domain/` (no GORM tags); HTTP DTOs in `internal/<domain>/delivery/`; GORM rows in `internal/<domain>/infra/`.

## Conventions (Mandatory)

- **Constants placement**: all constants must be centralized in `internal/shared/constants/*` (messages, status, thresholds, default values, and any other shared constant). Feature layers (`services/*`, `internal/*/infra/*`, `api/*`, `pkg/*`) must not define business constants inline.
- **Error placement and mapping**: errors must be declared in `pkg/errors` (sentinel/typed), not inside `services/*` or `internal/*/infra/*`. Error messages come from shared constants in `internal/shared/constants/error_msg.go` and are wired into `pkg/errcode/messages.go`. Numeric codes are defined in `pkg/errcode/codes.go` before use. Never hardcode an error code/message in a feature module.
- **Reference implementations** for new error types: `pkg/errors/provider_error.go` + `pkg/errors_func/media/provider_error_funcs.go` (typed error + unwrap/HTTP helper functions) and `pkg/errors/upload_errors.go` (sentinel pattern with shared message constant and `errors.Is`-friendly flow).
- **Tests exception**: types created only for test scope may be declared directly inside files under `tests/`. This exception does not apply to production code in `services/*` or `internal/*/infra/*`.

---

## Function / Method Assets

### Asset: internal/shared/mq (LavinMQ topic messaging)
- **Name:** `SetupLavinMQ`, `Available`, `PublishTopic`, `StartConsumers`, `StartDefaultConsumers`, `Close`
- **Type:** Package (`internal/shared/mq`)
- **Path:** `internal/shared/mq/`
- **Purpose:** CloudAMQP LavinMQ connection (`amqp091-go`), publish to topic exchange routing keys, durable queue bind + consume with manual ack.
- **Scope:** Cross-domain async messaging; routing keys in `internal/shared/constants/mq_topics.go`.
- **Dependencies:** `github.com/rabbitmq/amqp091-go`, `internal/shared/setting` (`LavinMQSetting`), env `CLOUDAMQP_URL`, `LAVINMQ_EXCHANGE`, `LAVINMQ_ENABLED`.
- **Current usage:** `main.go` (`SetupLavinMQ`, `StartDefaultConsumers`); domain modules should call `PublishTopic` / register `Subscription` handlers — no parallel AMQP clients.
- **Reuse:** Add new topics to `mq_topics.go`; register consumers via `StartConsumers` or extend `StartDefaultConsumers`.

### Asset: internal/shared/logger (Zap bootstrap + context)
- **Name:** `InitFromSettings`, `Init`, `Sync`, `WithRequestID`, `FromContext`, `Access`, `ResolveLogDir`
- **Type:** Package (`internal/shared/logger`)
- **Path:** `internal/shared/logger/`
- **Purpose:** Single process-wide logger (via `zap.ReplaceGlobals`) with legacy file tee compatibility and dual rotated sinks (`app.log`, `access.log`) for Loki/Alloy shipping, request correlation fields, optional `zap.RedirectStdLog`.
- **Scope:** All layers after `main` bootstrap; HTTP handlers use `FromContext` for `request_id`. Context correlation uses an **unexported sentinel + `var` in `internal/shared/logger`** (staticcheck SA1029 — not a `const` in `pkg/`, not a string `context` key).
- **Dependencies:** `go.uber.org/zap`, `internal/shared/setting` (`LogSetting`), `gopkg.in/natefinch/lumberjack.v2` (rotation), `internal/shared/xdgx` (path helpers).
- **Current usage:** `main.go`, `middleware/request_logger.go`, `internal/shared/httperr`, `internal/media/jobs/*`, `internal/shared/cache`, media webhook handlers.
- **Reuse:** Prefer `logger.FromContext(ctx)` in new handlers/services; use `logger.Access()` for HTTP access lines; use `zap.L()` only where no context exists.

### Asset: internal/shared/ratelimit (fixed-window counters)
- **Name:** `AllowFixedWindow`, `InMemoryStore`, `FileStore`, `AllowCLI`
- **Type:** Package (`internal/shared/ratelimit`)
- **Path:** `internal/shared/ratelimit/`
- **Purpose:** Single fixed-window implementation for HTTP middleware (`InMemoryStore`) and APPCLI (`FileStore` at `$XDG_CONFIG_HOME/mycourse/cli_rate_limit.json`).
- **Scope:** `middleware.RateLimitLocal`, `middleware.RateLimitSystemIP`, `appcli.guardCLIOperation`.
- **Dependencies:** `internal/shared/constants` (`CLIAttempts`, `CLIMinutes`).
- **Reuse:** Do not add parallel bucket types in middleware or appcli.

### Asset: internal/shared/resilience (circuit breaker)
- **Name:** `Global`, `ConfigureFromSettings`, `StartDBProbe`, `CircuitBreaker.Allow`, `RecordRequestStart`
- **Type:** Package (`internal/shared/resilience`)
- **Path:** `internal/shared/resilience/`
- **Purpose:** DB probe + HTTP load tracking; optional Redis key `mycourse:resilience:circuit`.
- **Scope:** `middleware.CircuitBreakerMiddleware`, `appcli.guardCLIOperation`, `main` bootstrap.
- **Dependencies:** `shareddb.StdDB`, `cache.Redis`, `setting.ResilienceSetting`.
- **Reuse:** Single global instance; HTTP 503 + `9018 ServiceUnavailable`, APPCLI stderr guard message.

### Asset: internal/shared/machineidentity (hybrid machine binding)
- **Name:** `LoadOrCreateMachineIdentityMaterial`, `LoadMachineIdentityMaterial`, `BuildHybridMachineBindingMaterial`, `IdentityFilePath`
- **Type:** Package (`internal/shared/machineidentity`)
- **Path:** `internal/shared/machineidentity/`
- **Purpose:** Enrollment file (`$XDG_CONFIG_HOME/mycourse/machine_identity`) + live OS fingerprint (machine-id / hardware UUID, hostname, `GOOS/GOARCH`) → canonical hybrid binding material (`v1|file:…|mid:…|hw:…|host:…|plat:…`).
- **Scope:** `appcli` register/login (`DeriveMachineSecret` in `machine_secret.go`), `appcli.cli_guard` rate-limit key derivation.
- **Dependencies:** stdlib only (`crypto/rand`, `os`, `os/exec` per platform build tags).
- **Reuse:** Do not duplicate enrollment path or fingerprint readers in `appcli` or domain layers.

### Asset: timex (audit epoch seconds)
- **Name:** `NowUnix`, `PtrUnix`, `UnixOrZero`
- **Type:** Package (`internal/shared/timex`)
- **Path:** `internal/shared/timex/timex.go`
- **Purpose:** Single source for **Unix epoch seconds** on `created_at` / `updated_at` / `deleted_at` (DB `BIGINT`, Go `int64` / `*int64`, JSON numbers).
- **Scope:** All modules that write audit columns; auth ban checks compare `banned_until` to `timex.NowUnix()`.
- **Dependencies:** Go `time` only.
- **Current usage:** `internal/shared/gormx`, auth/media/taxonomy/system infra and application layers.
- **Reuse:** Never use `time.Now()` directly for persisted audit fields — use `NowUnix()`.

### Asset: gormx audit + soft-delete helpers
- **Name:** `TouchCreatedUpdated`, `TouchUpdated`, `SoftDeleteWithAudit`, `ScopeActiveOnly`, `FirstWhere`, `FindActiveByIDs`, `IndexByID`, `UserIDSetByRoleNames`, `CreateAndThen`, `EnsureStringID`, `DefaultConfig`, `NewSQLLogger`
- **Type:** Package (`internal/shared/gormx`)
- **Path:** `internal/shared/gormx/`
- **Purpose:** Shared GORM patterns — audit timestamp writes, active-row scope, generic `First`/`Create` helpers, UUID v7 assignment for empty string primary keys before `Create`, shared `gorm.Config` (`DefaultConfig`), and latency-colored SQL console logging (`NewSQLLogger`).
- **Scope:** Taxonomy/instructor/course inserts with string `id` columns; auth `/me` delete; media upload entity + `UpsertByObjectKey` create; system config sync rows.
- **Dependencies:** `gorm.io/gorm`, `internal/shared/timex`, `internal/shared/uuidx`.
- **Current usage:** `internal/taxonomy/infra`, `internal/instructor/infra`, `internal/course/infra` (`touchCreateCourseEntity`), `internal/auth/infra`, `internal/media/infra`, `internal/system/infra`.
- **Reuse:** Prefer these over ad-hoc `Updates(map…)` for `deleted_at` / `updated_at`. Call `EnsureStringID` (or module `touchCreateCourseEntity`) before any GORM `Create` on UUID string PK rows — zero-value `""` is rejected by PostgreSQL. Open every GORM pool via `gormx.DefaultConfig()` — do not duplicate `&gorm.Config{}` in `db` or `pkg/supabase`.

### Asset: GORM SQL console logger
- **Name:** `NewSQLLogger`, `DefaultConfig`, `SQLLogThresholds`, `SQLLogThresholdSummary`
- **Type:** Package (`internal/shared/gormx`)
- **Path:** `internal/shared/gormx/sql_logger.go`, `gorm_config.go`
- **Purpose:** Log **every** GORM SQL statement to stdout with elapsed time in milliseconds and ANSI-colored SQL text: green `<200ms`, yellow `200ms–2s`, red `>=2s`. Reuses `gorm.io/gorm/logger` color constants and `gorm.io/gorm/utils.FileWithLineNum`. `gorm.ErrRecordNotFound` is logged as a normal query (not an error line).
- **Scope:** Process startup for primary PostgreSQL (`internal/shared/db`) and optional Supabase Postgres (`pkg/supabase/db.go`).
- **Dependencies:** `gorm.io/gorm`, `gorm.io/gorm/logger`, `gorm.io/gorm/utils`.
- **Current usage:** `shareddb.Setup()`, `supabase.SetupDatabase()`.
- **Reuse:** Do not configure a separate GORM logger in domain repos — attach `gormx.DefaultConfig()` only at pool open time.

### Asset: PendingCloudCleanupCounters (media job metrics boundary)
- **Name:** `PendingCloudCleanupCounters`
- **Type:** Function (service)
- **Path:** `internal/media/application/cleanup_job_metrics.go`
- **Purpose:** Expose pending-cloud cleanup atomic counters to HTTP without `api/` importing `internal/jobs/media`.
- **Current usage:** `internal/*/delivery/media/file_handler.go` (`getMediaCleanupMetrics`).

### Asset: System HTTP job adapters (system lane)
- **Name:** `SystemHTTPCreatePermissionSyncJob`, `SystemHTTPCreateRolePermissionSyncJob`, `SystemHTTPStopPermissionSyncJob`, `SystemHTTPStopRolePermissionSyncJob`
- **Type:** `gin.HandlerFunc` wrappers
- **Path:** `internal/system/application_job_http.go`
- **Purpose:** Delegate to `internal/jobs/system` so `api/system` stays free of `internal/jobs` imports.
- **Current usage:** `internal/system/delivery/routes.go`.

### Asset: PermissionCodesForUser
- **Name:** `PermissionCodesForUser`
- **Type:** Function (service)
- **Path:** `internal/rbac/application/rbac.go`
- **Purpose:** Resolve effective permission names from role grants + direct grants.
- **Scope:** All authorization-sensitive APIs and token issuance.
- **Dependencies:** `rbacDB`, `pkg/sqlnamed`, `constants` (`RbacSQL*` templates).
- **Current usage:** `internal/auth/application/auth.go`, `internal/*/delivery/me.go`, `internal/*/delivery/internal/rbac_handler.go`, `UserHasAllPermissions`.
- **Reuse:** Reuse for all new CRUD permission checks and permission projection in future domains.

### Asset: UserHasAllPermissions
- **Name:** `UserHasAllPermissions`
- **Type:** Function (service guard)
- **Path:** `internal/rbac/application/rbac.go`
- **Purpose:** Verify required permission set for a user.
- **Scope:** Authorization middleware fallback and potential service guardrails.
- **Dependencies:** `PermissionCodesForUser`.
- **Current usage:** `middleware/rbac.go`.
- **Reuse:** Direct reuse for `resource:action` checks on new CRUD endpoints.

### Asset: issueTokenPair / RefreshSession
- **Name:** `issueTokenPair`, `RefreshSession`
- **Type:** Function (auth service)
- **Path:** `internal/auth/application/auth_session_tokens.go` (`issueTokenPair`); `internal/auth/application/auth_refresh_rotation.go` (`RefreshSession` + rotation helpers); JSONB writes in `internal/*/infra/user_refresh_session.go`; session entry shape in `internal/<domain>/domain/refresh_session.go`.
- **Purpose:** Token issue/rotation and session persistence management.
- **Scope:** Any auth/session extension features.
- **Dependencies:** `pkg/token`, `constants` (TTLs), `internal/*/infra` GORM rows, `internal/shared/cache`, RBAC permission resolver.
- **Current usage:** auth register/login/confirm/refresh flows.
- **Reuse:** Reuse unchanged for newly protected domain APIs.

### Asset: Register (pending signup / resend)
- **Name:** `Register`
- **Type:** Function (auth service)
- **Path:** `internal/auth/application/register_flow.go`, `internal/auth/application/register_email_send.go`
- **Purpose:** Create pending user or update unconfirmed same email; enforce lifetime + Redis sliding window; send Brevo confirmation; increment `registration_email_send_total` only after successful send.
- **Scope:** `POST /auth/register` only.
- **Dependencies:** `internal/*/infra` GORM rows, `pkg/brevo`, `pkg/errors`, `internal/shared/constants/register_email_limits.go`, `internal/shared/cache/register_email_window.go`, `internal/shared/cache/auth_user.go`.
- **Reuse:** Extend only with explicit product requirements (never bypass `pkg/errors` + `pkg/errcode` mapping in handlers).

### Asset: TryReserveRegisterConfirmationSend / Release / Delete window
- **Name:** `TryReserveRegisterConfirmationSend`, `ReleaseRegisterConfirmationSend`, `DeleteRegisterConfirmationEmailWindow`
- **Type:** Functions (cache)
- **Path:** `internal/shared/cache/register_email_window.go`
- **Purpose:** Atomic Redis ZSET sliding window for registration confirmation sends per `users.id`.
- **Scope:** Registration email rate limit (pairs with `constants.RedisKeyRegisterConfirmEmailWindowPrefix`).
- **Dependencies:** `github.com/redis/go-redis/v9` Lua script.
- **Reuse:** Pattern reference for other Redis sliding-window caps (keep scripts colocated with the feature cache file).

### Asset: RegistrationEmailRateLimitedError
- **Name:** `RegistrationEmailRateLimitedError`
- **Type:** Typed error (`pkg/errors`)
- **Path:** `pkg/errors/register_limits.go`
- **Purpose:** Carry `RetryAfterSeconds` for HTTP **429** + `4010` mapping.
- **Scope:** Register handler only.
- **Reuse:** Same pattern as `ProviderError` for other bounded-retry upstream flows (always `pkg/errors`, never entities).

### Asset: Mapping helpers for API DTO contracts
- **Name:** `toTokensResponse`, `ToMyPermissionsResponse`, `ToUserRBACPermissionCodesResponse`, `ToBatchDeleteMediaFilesResponse`, `ToLocalURLDecodeResponse`, `ToVideoStatusResponse`, `ToMediaCleanupMetricsResponse`, `ToPermissionSyncNowResponse`, `ToRolePermissionSyncNowResponse`, taxonomy `*HTTPPayload` mappers, `ToUploadFileResponse`
- **Type:** Function (mapping)
- **Path:** `internal/auth/delivery/handler.go`, `pkg/logic/mapping/{auth_me_mapping,rbac_internal_response,media_file_mapping,system_response_mapping,taxonomy_category_mapping,taxonomy_model_mapping,taxonomy_course_level_mapping,taxonomy_tag_mapping}.go`
- **Purpose:** Centralize entity/model → DTO mapping so handlers never return raw persistence/entity structs. Taxonomy `HTTPPayload` helpers keep taxonomy handlers free of `internal/*/infra` GORM-row imports.
- **Scope:** Auth, media, system, taxonomy transport responses. Auth `/me` uses bounded-context-private `buildMeProfile`/`toMeResponse` instead of this shared set.
- **Reuse:** Reuse for all upcoming domain handlers to enforce stable public API contracts.

### Asset: BindUpdateAndCreateMultipart
- **Name:** `BindUpdateAndCreateMultipart`, `BindCreateFileMultipart`, `BindUpdateFileMultipart`
- **Type:** Function (mapping / transport parsing)
- **Path:** `internal/media/delivery/mapping.go`
- **Purpose:** Single entry that binds update + create multipart text fields (`reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged`); client `kind`/`metadata` are intentionally ignored by service flow.
- **Current usage:** `internal/media/delivery/handler.go` (`updateFile`).

### Asset: ListPermissions
- **Name:** `ListPermissions`
- **Type:** Function (service list pattern)
- **Path:** `internal/rbac/application/rbac.go`
- **Purpose:** Paginated list with safe sort/search whitelist behavior.
- **Scope:** Reusable blueprint for list endpoints.
- **Reuse:** Reuse its whitelist + pagination pattern for taxonomy/courses/series lists.

---

## Type / DTO / Interface Assets

### Asset: BaseFilter
- **Name:** `BaseFilter`
- **Type:** Type/DTO
- **Path:** `dto/filter.go`
- **Purpose:** Shared pagination/sort/search DTO contract.
- **Scope:** All list APIs.
- **Reuse:** Mandatory reuse for all list/read CRUD endpoints.

### Asset: RBAC DTO Suite
- **Name:** `CreatePermissionRequest`, `UpdatePermissionRequest`, `CreateRoleRequest`, `SetRolePermissionsRequest`, `AssignUserRoleRequest`, etc.
- **Type:** Type/DTO
- **Path:** `dto/rbac.go`
- **Purpose:** Request schemas and validation tags for CRUD operations.
- **Scope:** Internal admin API contracts.
- **Reuse:** Template to design new domain DTO suites with consistent validation style.

### Asset: MeResponse
- **Name:** `MeResponse`
- **Type:** Type/DTO
- **Path:** `internal/auth/delivery/dto.go`
- **Purpose:** HTTP JSON contract for `GET/PATCH /api/v1/me` (handler boundary only), including non-null display-only `roles`.
- **Dependencies:** Built from `auth/domain.MeProfile` via the private `toMeResponse` mapper.
- **Reuse:** Keep transport-only; auth application/cache use `domain.MeProfile`.

### Asset: MeProfile (service + cache shape)
- **Name:** `MeProfile`
- **Type:** Type/Entity
- **Path:** `internal/auth/domain/user.go`
- **Purpose:** Non-DTO user self projection for `internal/auth/application` and Redis `/me` cache; carries raw avatar identifiers plus sorted permissions and ordered display-only role names.
- **Current usage:** `internal/auth/application/service.go`, `service_cache.go`; built by the private `buildMeProfile` helper.

### Asset: Core taxonomy entities (pure shared types)
- **Name:** `CourseLevel`, `Category`, `Tag`
- **Type:** Type/Entity
- **Path:** `internal/<domain>/domain/{course_level,category,tag}.go`
- **Purpose:** Share taxonomy field definitions as pure data types across layers.
- **Reuse:** Extend with additional pure entities for future domains while keeping DB mapping (`TableName`) only in `models/*`.

### Asset: File domain types
- **Name:** `File`, `FileMetadata`, `ImageMetadata`, `VideoMetadata`, `DocumentMetadata`
- **Type:** Type/Entity
- **Path:** `internal/<domain>/domain/file.go`
- **Purpose:** Shared media response descriptor with base metadata + typed metadata inheritance model.
- **Reuse:** Reuse for course lesson media, subtitles, and future asset libraries.

---

## Utility / Helper Assets

### Asset: ParseListFilter and parse helpers
- **Name:** `ParseListFilter`, `buildSortClause`, `buildSearchClause`
- **Path:** `pkg/query/filter_parser.go`
- **Purpose:** Parse query params into safe pagination/filter/sort metadata with whitelist support.
- **Reuse:** Reuse for course shell and later listing endpoints to avoid duplicating query parsing logic.

### Asset: Request param helper package
- **Name:** `CurrentUserID`, `ParseUUIDParam`
- **Path:** `internal/shared/utils/requestutil.go`
- **Purpose:** Centralize request-context user extraction and UUID path param parsing.
- **Current usage:** auth, course, instructor, taxonomy, and RBAC delivery handlers.
- **Reuse:** Reuse in all future CRUD handlers to keep transport-layer parsing behavior consistent.

### Asset: Route-level permission adapter
- **Name:** `RoutePermission`
- **Path:** `internal/shared/utils/route_permission.go`
- **Purpose:** Shared delivery-layer adapter that turns `middleware.RequirePermission` into a reusable route-registration helper.
- **Scope:** Permission-gated route registration in `internal/auth`, `internal/course`, `internal/media`, `internal/instructor`, `internal/taxonomy`.
- **Reuse:** Reuse this instead of defining local `rp := func(...){...}` wrappers in new delivery route files.

### Asset: Generic path-param parsers
- **Name:** `ParseUUIDPathParam`, `ParseUintPathParam`
- **Path:** `internal/shared/utils/params.go`
- **Purpose:** Parse UUID or unsigned integer path params from `gin.Context` with shared implementations.
- **Scope:** UUID entity routes use `ParseUUIDPathParam`; numeric catalog routes (e.g. RBAC `roleId`) use `ParseUintPathParam`.
- **Reuse:** Reuse for all future `:id`/`:userId`/`:roleId` style path parsing.

### Asset: Permission id path-param parser
- **Name:** `ParsePermissionIDParam`
- **Path:** `internal/shared/utils/requestutil.go`
- **Purpose:** Parse and validate permission id path params (trim + max length check).
- **Current usage:** internal RBAC permission handlers.

### Asset: Shared pagination page builder
- **Name:** `BuildPage`
- **Path:** `pkg/logic/utils/pagination.go`
- **Purpose:** Centralize pagination response construction (`Page`, `PerPage`, `TotalPages`, `TotalItems`).
- **Reuse:** Reuse by all list endpoints to keep pagination behavior consistent.

### Asset: Local media URL reversible signer / decoder
- **Name:** `EncodeLocalObjectKey`, `DecodeLocalObjectKey`, `DecodeLocalURLToken`
- **Path:** `pkg/media/local_url_codec.go`
- **Purpose:** Build reversible signed URL tokens for local-provider media objects; decode with env-based secret fallback.
- **Reuse:** Reuse for secure temporary download tokens in other modules.

### Asset: Media kind/provider resolvers + Bunny metadata policy
- **Name:** `ResolveMediaKind`, `ResolveMediaProvider`, `ResolveMediaKindFromServer`, `ResolveUploadProvider`, `EnrichBunnyVideoDetail`, `EffectiveBunnyThumbnailURL`, `FormatBunnyVideoIDString`, `ResolveBunnyEmbedURL`, `ResolveBunnyEmbedHTML`, `ApplyBunnyDetailToMetadata`
- **Path:** `pkg/media/media_resolver.go`
- **Purpose:** Normalize upload kind/provider with server-owned inference. Map Bunny GET-video payload into metadata keys `video_id`, `thumbnail_url`, `embeded_html`, `direct_play_url`, `hls_playlist_url`, `preview_animation_url`.
- **Reuse:** Reuse for any future media ingestion endpoints to keep provider-kind and Bunny contracts identical.

### Asset: Media metadata JSON key constants
- **Name:** `MediaMetaKeyVideoGUID`, `MediaMetaKeyBunnyVideoID`, `MediaMetaKeyVideoID`, `MediaMetaKeyThumbnailURL`, `MediaMetaKeyEmbededHTML`, `MediaMetaKeyDirectPlayURL`, `MediaMetaKeyHLSPlaylistURL`, `MediaMetaKeyPreviewAnimationURL`
- **Path:** `internal/media/domain/meta_keys.go`
- **Purpose:** Single source of truth for string keys used in `metadata_json`. Do not duplicate literal key strings elsewhere.

### Asset: App media provider config
- **Name:** `MediaSetting.AppMediaProvider`
- **Path:** `pkg/setting/setting.go`
- **Purpose:** Server-side source of truth for upload provider selection.
- **Reuse:** Reuse as canonical provider control for all future media upload entry points.

### Asset: Cloud SDK client bootstrap (`MediaSetting`)
- **Name:** `NewCloudClientsFromSetting`, `Setup`
- **Path:** `internal/media/infra/provider_runtime.go`
- **Purpose:** One-shot construction of `CloudClients` (R2 S3 client + bucket name, shared HTTP client) from `setting.MediaSetting.R2.*`. Bunny Stream uses the HTTP client directly.
- **Scope:** App startup only; caller `mediainfra.Setup()` (after `setting.Setup()` in `main.go`).
- **Reuse:** Any new process needing the same cloud handles should call `mediainfra.Setup` or reuse `mediainfra.Cloud` rather than duplicating env reads.

### Asset: Media metadata parser helpers
- **Name:** `ParseMetadataJSON`, `ParseMetadataFromRaw`, `NormalizeMetadata`, `BuildTypedMetadata`, `DefaultMediaProvider`
- **Path:** `internal/media/infra/media_entity_metadata.go`
- **Purpose:** Parse raw metadata JSON, normalize metadata payload, infer typed metadata contract `UploadFileMetadata` with explicit fields/default values.
- **Reuse:** Reuse for all future endpoints that accept metadata in raw string form and require backend metadata inference.

### Asset: Media upload object keys (R2 / Bunny / local)
- **Name:** `ResolveMediaUploadObjectKey`, `BuildObjectStorageKey`
- **Path:** `pkg/media/media_upload_keys.go`
- **Purpose:** Provider-specific default object keys before upload (8-digit R2 prefix; Bunny empty until GUID; local nano key).
- **Reuse:** Reuse instead of duplicating filename sanitization or key rules in `pkg/media`.

### Asset: Content fingerprint (SHA-256 hex)
- **Name:** `ContentFingerprint`
- **Path:** `pkg/logic/utils/parsing.go`
- **Purpose:** Stable digest of upload bytes for skip-upload dedupe on `PUT /media/files/:id`.
- **Reuse:** Any feature needing cheap binary equality without storing raw bytes.

### Asset: MergeMediaMetadataJSON
- **Name:** `MergeMediaMetadataJSON`
- **Path:** `pkg/media/media_metadata_merge.go`
- **Purpose:** Merge JSONB metadata with overlay map for metadata-only updates.

### Asset: Superseded cloud cleanup guard
- **Name:** `ShouldEnqueueSupersededCloudCleanup`
- **Path:** `pkg/media/media_replace_policy.go`
- **Purpose:** Decide whether a replace produced a new cloud identity requiring deferred delete of the prior object.

### Asset: DeleteStoredObject (unified cloud delete)
- **Name:** `DeleteStoredObject`
- **Path:** `pkg/media/stored_object_delete.go`
- **Purpose:** Route delete by provider (R2 key vs Bunny GUID vs local noop) — avoids duplicating the switch in callers.

### Asset: Media cleanup worker scheduler
- **Name:** `StartMediaPendingCleanupJob`, `StopMediaPendingCleanupJob`
- **Path:** `internal/jobs/media/media_pending_cleanup_scheduler.go`
- **Purpose:** In-memory ticker for deferred cloud deletes — same mutex/cancel/waitgroup pattern as `internal/jobs/rbac/rbac_sync_schedulers.go` (`syncJobBundle`).

### Asset: Media concurrency / reuse errors
- **Name:** `ErrMediaOptimisticLock`, `ErrMediaReuseMismatch`
- **Type:** Sentinel (`pkg/errors`)
- **Path:** `pkg/errors/media_errors.go`
- **Purpose:** Map to HTTP **409** + `errcode.Conflict` when row version or `reuse_media_id` mismatches.

### Asset: URL join helper (generic)
- **Name:** `JoinURLPathSegments`
- **Path:** `pkg/logic/utils/url.go`
- **Purpose:** Join CDN/base URL with path segments without duplicated slashes.
- **Reuse:** Prefer over manual string concatenation for CDN + bucket + key.

### Asset: Cryptographic random decimal string
- **Name:** `GenerateRandomDigits`
- **Path:** `pkg/logic/utils/random.go`
- **Purpose:** `n` decimal digits from `crypto/rand` (used for B2 object key prefix).
- **Reuse:** Reuse for other token prefixes; do not reimplement with `math/rand`.

### Asset: Bunny Stream — constants vs `pkg/media`
- **Constants** (`internal/shared/constants/bunny_video.go`, `bunny_video_status.go`): webhook status + signing literals; status-name mapping helper `pkg/media.BunnyStatusString(status)`.
- **Webhook signature helpers** (`pkg/media/webhook_signature.go`): `BunnyWebhookSigningSecret`, `BunnyWebhookSignatureExpectedHex`, `IsBunnyWebhookSignatureValid` (v1 + hmac-sha256 + constant-time compare on raw body).

### Asset: Media provider typed error + HTTP mapping
- **Name:** `ProviderError` (type in `pkg/errors`); `AsProviderError`, `HTTPStatusForProviderCode` (`pkg/errors_func/media`)
- **Path:** `pkg/errors/provider_error.go`; `pkg/errors_func/media/provider_error_funcs.go`
- **Purpose:** Carry `errcode` 9010–9019 for R2/Bunny/encode client failures; map to HTTP 500/502/503/504.
- **Reuse:** Extend with more provider-specific codes without changing handler shape.

### Asset: Stable "not found" for API boundaries
- **Name:** `ErrNotFound` (sentinel in `pkg/errors`); `MapRecordNotFound` (`pkg/errors_func/db`)
- **Path:** `pkg/errors/not_found.go`; `pkg/errors_func/db/map_record_not_found.go`
- **Purpose:** Let delivery handlers use `errors.Is(err, ErrNotFound)` for HTTP **404** without importing `gorm.io/gorm`; services/repository map `gorm.ErrRecordNotFound` at the edge.
- **Reuse:** Prefer this over returning raw GORM errors from services when handlers need a not-found branch.

### Asset: Generic parsing / metadata primitives (utils)
- **Name:** `DetectExtension`, `ImageSizeFromPayload`, `StringFromRaw`, `IntFromRaw`, `FloatFromRaw`, `NonEmpty`, `ParseBoolLoose`, `ContentFingerprint`
- **Path:** `internal/shared/utils/parsing.go` (bool: `ParseBoolLoose` → `internal/shared/parsebool.Loose`)
- **Purpose:** Raw-metadata helpers, loose bool parsing for multipart text fields, SHA-256 hex fingerprint of byte payloads.
- **Reuse:** Reuse directly in future modules to avoid re-implementing generic conversion/parsing primitives.

### Asset: Text rules validators (utils + validate)
- **Name:** `CountNonWhitespace`, `CountDeltaNonWhitespace`, `CountDeltaRunes`, `CountRunes`, `nonwhitespace_min`, `delta_nonwhitespace_min`
- **Path:** `internal/shared/utils/text_rules.go`, `internal/shared/validate/text_rules.go`
- **Purpose:** Count visible characters (Unicode whitespace excluded) for plain strings and Quill Delta JSON; registers custom `go-playground/validator` tags.
  - `CountDeltaRunes` counts Unicode code points (including whitespace) on Quill Delta JSON string inserts; invalid/non-JSON input and valid JSON without an `ops` array fall back to `CountRunes` for legacy plain text (matches FE's `coerceToDelta` payload-class rule — prevents FE/BE drift). Structural allow-list validation of the Delta itself is a separate function, `validateBioDelta` in `internal/instructor/application/validate_profile.go` — counting alone never validates structure.
  - `CountRunes` counts via `utf8.RuneCountInString` — prefer over `len(string)` (UTF-8 bytes).
- **Scope:** Course field validation (title, descriptions, about-course Delta) uses `CountNonWhitespace`/`CountDeltaNonWhitespace`. Instructor application `bio` (100–2000) uses `CountDeltaRunes`; `teaching_content_ideas` (plain string) uses `CountRunes`. Do not cross-use these — see each entry's scope note.

### Asset: slug (shared)
- **Name:** `ResolveCreate`, `ResolveUpdate`, `AvailableFunc`, `ConflictError`, `ParseManual`, `ParseUpdateInput`, `ErrInvalidFormat`, `ErrEmpty`, `ErrReserved`, `IsReserved`, `ValidateManualFormat`, `GenerateRandomSuffix`, `RetryWithSuffix`, `BuildSuffixedCandidate`, `TruncateToLen`, `TruncateForSuffix`, `GenerateAutoSlugBase`, `Kind`
- **Type:** Package (`internal/shared/slug`)
- **Path:** `internal/shared/slug/{resolve,input,errors,reserved,format,suffix,autobase}.go`
- **Purpose:** Single owner of slug resolution for every slug-bearing entity. `ResolveCreate(manual, name, kind, maxLen, avail)` returns the stored slug or a `*ConflictError` (with a recommended alternative) when a manual value is taken; with no manual value it auto-generates from `name` and suffixes on collision. `ResolveUpdate(newSlug, maxLen, avail)` never conflicts — a taken value is auto-suffixed. `AvailableFunc` is the caller-supplied uniqueness check. `IsReserved` checks the reserved-word list (user slugs only). `Kind` is a typed enum (`KindCourse`, `KindCourseLevel`, `KindCourseTopic`, `KindCourseSkill`, `KindTag`, `KindTreeNode`, `KindUser`, ...).
- **Scope:** Any entity with a slug: courses, the 4 taxonomy root tables, taxonomy tree nodes, `users.slug`. A new slug-bearing entity adds a `Kind` + an `AvailableFunc` — never a new resolution or auto-generation function.
- **Reuse:** Prefer this over any domain-specific slug-collision, validation, or auto-generation algorithm.

### Asset: gormslug (GORM slug helpers)
- **Name:** `AvailableIn[R]`, `IsDuplicate`, `RetryAttempts`
- **Type:** Package (`internal/shared/slug/gormslug`)
- **Path:** `internal/shared/slug/gormslug/gormslug.go`
- **Purpose:** GORM-bound companion of `internal/shared/slug`. `AvailableIn` builds the `slug.AvailableFunc` for any row model with `slug`/`deleted_at` columns, optionally excluding one row id. `IsDuplicate` reports a unique violation on a named slug index (delegates to `gormx.IsUniqueViolation`). `RetryAttempts` (5) is the shared savepoint retry budget.
- **Scope:** Course, taxonomy, and user repositories — each keeps only its own unique-index name list. Do not re-implement availability SELECTs or duplicate-key detection per module.

### Asset: FailIfSlugConflict (response)
- **Name:** `FailIfSlugConflict`
- **Path:** `internal/shared/response/slug_conflict.go`
- **Purpose:** Maps a `slug.ConflictError` to HTTP 409 with app code `SlugConflict` (**3007**) and `data.recommended_slug`.
- **Scope:** Every handler that creates a slug-bearing entity from a manual slug (course create, taxonomy create). User slug changes never use it — `PATCH /me` auto-suffixes and never returns 409.

### Asset: RetryInSavepoint (gormx transactional retry)
- **Name:** `RetryInSavepoint`
- **Path:** `internal/shared/gormx/retry_savepoint.go`
- **Purpose:** Retry a write inside an already-open GORM transaction after a recoverable error (e.g. a unique-key violation), using `tx.SavePoint`/`tx.RollbackTo` instead of retrying on the same `*gorm.DB`. PostgreSQL aborts the *entire* surrounding transaction on any statement error — retrying without a savepoint means every attempt after the first fails with a generic "current transaction is aborted" error, never actually retrying. (Fixed a shipped bug in course's slug-update retry.) Sibling helper `gormx.IsUniqueViolation(err, constraintNames...)` is the shared unique-violation test for the `isRetryable` callback.
- **Reuse:** Any domain adding slug (or other unique-constraint) retry logic inside a transaction must use this instead of re-deriving the savepoint pattern.

### Asset: Content locale helpers (i18n)
- **Name:** `CanonicalizeLocale`, `NegotiateReadLocale`, `ResolveText`
- **Type:** Package (`internal/shared/i18n`)
- **Path:** `internal/shared/i18n/locale.go`
- **Purpose:** **Write** path: `CanonicalizeLocale` (BCP47; keep region; empty/invalid → 4xx). **Read** path: `NegotiateReadLocale` + `ResolveText` (exact → base language → default `en` → canonical field). Separate APIs — do not use one "normalize then fall back to en" function for both.
- **Scope:** Taxonomy translation tables / JSONB `translations`, instructor chip joins, any future content localization.
- **Reuse:** Keep `mailtmpl.NormalizeLanguageCode` for **email** en/vi only — never for taxonomy BCP47.

### Asset: Taxonomy tree + description validators
- **Name:** `TreeNode`, `PrepareTreeForWriteWithSlugPolicy`, `ResolveTreeSlugs`, `BuildNodeIndexByID`, `ValidateTree`, `ValidateDescriptionParagraphs`
- **Type:** Package (`internal/shared/taxonomy`)
- **Purpose:** JSONB tree shape `{ id, name, slug, translations?, children[] }` with per-node `translations` map. `PrepareTreeForWriteWithSlugPolicy` is the sole write-path entry point: syncs each node's canonical↔`en` translation, ensures an `en` translation exists, then resolves slugs via `ResolveTreeSlugs` — preserves an existing node's slug when empty+recognized-by-id, auto-generates for new/unrecognized nodes via the shared `sharedslug.GenerateAutoSlugBase`, and silently auto-resolves any in-tree slug collision.
- **Scope:** `internal/taxonomy` application + infra layers.
- **Reuse:** Do not invent a second tree node type or strip `translations` on write/read paths that need them. Do not reintroduce an unconditional recompute-from-name slug helper.

### Asset: auth/infra.CheckPassword (bcrypt verify)
- **Name:** `CheckPassword`
- **Path:** `internal/auth/infra/crypto.go`
- **Purpose:** Verify a plain-text password against a bcrypt hash.
- **Reuse:** Do not add duplicate bcrypt compare helpers in `appcli` or `system/infra`.

### Asset: parsebool (env / YAML booleans)
- **Name:** `Loose`, `EnvEnabled`
- **Path:** `internal/shared/parsebool/parsebool.go`
- **Purpose:** Single implementation for loose truthy strings (`true`, `1`, `yes`, `y`, `on`).
- **Current usage:** `internal/shared/setting`, `internal/appcli/register_system_user.go`; multipart forms use `utils.ParseBoolLoose` (delegates here).

### Asset: sqlnamed.Postgres
- **Name:** `Postgres`
- **Path:** `pkg/sqlnamed/postgres.go`
- **Purpose:** Named SQL to PostgreSQL positional parameter conversion.
- **Reuse:** Reuse for complex raw SQL operations where GORM is not ideal.

### Asset: setting.Setup
- **Name:** `Setup`
- **Path:** `internal/shared/setting/setting.go`
- **Purpose:** Unified env + stage YAML config loading.
- **Scope:** Application and CLI startup.
- **Reuse:** Reuse unchanged for any new command/service module.

### Asset: Cache helpers for auth/me
- **Name:** `getCachedMe`, `setCachedMe`, `loginInvalidCached`, `getCachedLoginUserID`, `setCachedLoginUserID`, `DelCachedUserMe`
- **Path:** `internal/auth/application/service_cache.go`, `internal/shared/cache/auth_user.go`
- **Purpose:** Cache-aside support for login and `/me`. Login email cache stores plain `user_id` only (no password hash in Redis). Negative cache applies when active-user lookup returns not found; wrong-password attempts never populate the negative cache. `DelCachedUserMe` invalidates `mycourse:user:me:{id}` after profile-changing writes.

### Asset: Generic JSON cache-aside helper
- **Name:** `GetJSON[T any]`, `SetJSON[T any]`
- **Path:** `internal/shared/cache/json_cache.go`
- **Purpose:** Generic, type-safe cache-aside get/set for read-heavy public list endpoints, following the same fail-open (`cache.RedisAvailable()`-guarded) contract as the auth cache, but with no per-request validation logic.
- **Scope:** Public, non-personalized list reads.
- **Current usage:** `internal/course/application/service_catalog.go` (`mycourse:catalog:trending_courses:limit:{n}`, 5 min TTL), `internal/instructor/application/service_catalog.go` (`mycourse:catalog:popular_instructors:limit:{n}`, 5 min TTL).
- **Reuse:** Reuse for any future public, non-personalized list read before adding another bespoke cache-aside implementation. Extends the auth `/me` cache-aside pattern and the existing `internal/shared/ratelimit/` unauthenticated tier rather than inventing new mechanisms — this is also the current implementation of the public SEO cache pattern (course/instructor public catalogue caching), see `docs/FEATURES.md` auth/course entries for the SEO/security notes that used to live in `security-public-seo-notes.md`.

### Asset: Generic normalization / set primitives (utils)
- **Name:** `SameStringSet`, `UniqueUint`, `NilIfBlank`, `NilIfZeroUint`, `NormalizeJSON`
- **Path:** `internal/shared/utils/normalize.go`
- **Purpose:** Shared helpers for set-equivalence checks, dedupe IDs, nullable SQL update values, and fallback JSON normalization.
- **Current usage:** `internal/course/infra/*` (reorder checks, version ref validation, sub-lesson text handling, learner progress payload normalization).
- **Reuse:** Reuse in other bounded contexts instead of redefining local `nullable*`/`normalize*`/set-compare helpers.

### Asset: userpicker (paginated candidate list SQL)
- **Name:** `ListRows`, `UserPickerSelectSQL`, `Row`, `ListFilter`, `EligiblePickerWhereClause`, `ActiveUserWhereClause`, `EligibilitySortTierExpr`
- **Type:** Package (`internal/shared/userpicker`)
- **Purpose:** Shared count + paginated list for user-picker endpoints; callers supply business-specific `baseSQL` + bind args. `EligiblePickerWhereClause`/`ActiveUserWhereClause` are SQL `WHERE` fragments for candidate pickers/lists; `EligibilitySortTierExpr` is a `CASE` expression for audit-list sort ordering.
- **Scope:** `internal/instructor/infra` (`ListRosterCandidates`), `internal/course/infra` (`ListInstructorCandidates`), `ListApplications`/`ListProfiles` sort.
- **Reuse:** Add new picker endpoints by composing `UserPickerSelectSQL` + role/join predicates, not duplicating count/list/search logic.

### Asset: User display name / email search SQL
- **Name:** `UserDisplayNameEmailSearchSQL`
- **Path:** `internal/shared/utils/user_search_sql.go`
- **Purpose:** Shared ILIKE clause for `users.display_name`/`users.email` (alias `u`); trims search term, returns named `@search` args.
- **Current usage:** `ListCollaborators`, `ListInstructorCandidates`.

### Asset: User assignment eligibility (Go)
- **Name:** `CheckEligibleForAssignment`, `AssignmentSnapshot`, `AssignmentFailureMessage`
- **Type:** Functions/types (`internal/shared/useraccess/eligibility.go`)
- **Purpose:** Go-side mutation validator with same semantics as login (`CheckAccessible` + `email_confirmed`). Returns sentinel errors; `AssignmentFailureMessage` maps them for bulk `failed[]` payloads.
- **Current usage:** `planBulkRosterWrites`, `ApproveApplication`, `GormUserRoleRepository.AssignRole`.

### Asset: Batch assignment snapshot load / RBAC user ID set / bulk ID normalization
- **Name:** `LoadAssignmentSnapshotsByIDs`, `LoadAssignmentSnapshotByID` (`internal/shared/gormx/user_assignment.go`); `UserIDSetByRoleNames` (`internal/shared/gormx/user_role_set.go`); `PrepareBulkUserIDs` (`internal/shared/utils/bulk_user_ids.go`)
- **Purpose:** Batch-load assignment snapshots without N+1 queries; batch query `user_roles`+`roles` for a role-name membership set; trim/dedupe bulk user-ID request bodies.
- **Current usage:** Roster bulk, collaborator bulk, RBAC role assign.

### Asset: PostgreSQL pool tuning / schema setup
- **Name:** `tunePool`, `Setup`, `Database.AppSchemaName`, `Database.EnsureAppSchemaName`
- **Path:** `internal/shared/db/db.go`, `internal/shared/setting/setting.go`, `internal/shared/constants/dbschema_name.go`
- **Purpose:** After `gorm.Open`, configure `database/sql` pool (`MaxOpenConns=50`, `MaxIdleConns=25`, `ConnMaxLifetime=5m`, `ConnMaxIdleTime=1m`). Resolve app PostgreSQL schema from `SCHEMA_NAME_APP` or default `public`; run `SET search_path` on every new GORM connection; pass the same name to golang-migrate.
- **Current usage:** `main` bootstrap via `shareddb.Setup()`.

### Asset: Taxonomy list total inference
- **Name:** `taxonomyListTotal`, `listTaxonomyWithImageURLs`, `taxonomyGetByID`
- **Path:** `internal/taxonomy/infra/repos_crud_helper.go`, `repos.go`
- **Purpose:** After paginated `Find`, skip `COUNT(*)` when `len(rows) < pageSize` and derive `total_items` directly. `listTaxonomyWithImageURLs` skips image hydration when `include_images=false`. `taxonomyGetByID` is the shared get-by-id for all five taxonomy roots.
- **Reuse:** Do not add a parallel get-by-id or count-inference helper per taxonomy repository.

### Asset: Runtime dependency guard
- **Name:** `RequireInitialized`, `ErrDependencyNotConfigured`
- **Path:** `pkg/media/runtime_guard.go` (guard); `pkg/errors/media_errors.go` (sentinel)
- **Purpose:** Centralize runtime dependency checks (non-nil guards); handlers map `ErrDependencyNotConfigured` to `errcode.InternalError`.
- **Reuse:** Reuse in future modules needing startup dependency guards.

### Asset: Media constants
- **Name:** `FileProvider`, `FileKind`, `FileStatus`
- **Path:** `internal/shared/constants/media.go`
- **Purpose:** Centralized media enums (kept out of the entity to keep it pure).

### Asset: Executable upload denylist (`IsExecutableUploadRejected`)
- **Name:** `IsExecutableUploadRejected`, `ErrExecutableUploadRejected`
- **Path:** `pkg/logic/utils/executable_check.go` (check); `pkg/errors/media_errors.go` (sentinel)
- **Purpose:** Check a file upload against an extension denylist and magic-byte signatures; returns true when the file must be blocked. Maps to HTTP 400 + app code **2004**.
- **Scope:** `internal/media/application/file_service.go` (`CreateFile` non-image/non-video branch).
- **Reuse:** Any future upload endpoint needing the same executable filter.

### Asset: EncodeWebP + AcquireEncodeGate / ReleaseEncodeGate
- **Name:** `EncodeWebP`, `AcquireEncodeGate`, `ReleaseEncodeGate`, `ErrImageEncodeBusy`, `IsImageMIMEOrExt`
- **Path:** `pkg/logic/utils/webp_encode.go` (`//go:build cgo`), `webp_encode_stub.go` (`//go:build !cgo`), `image_encode_gate.go`; `IsImageMIMEOrExt` in `pkg/media/media_resolver.go`
- **Purpose:** Convert image bytes to WebP (bimg/libvips) with a buffered-channel semaphore capping concurrent encode goroutines to `constants.MaxConcurrentImageEncode`. `IsImageMIMEOrExt` detects whether a file is an image by MIME prefix or extension to control the conversion branch. Without CGO, the stub returns `ErrImageEncodeBusy` → `ProviderError{Code: 9017}`.
- **Scope:** `internal/media/application/file_service.go` (`CreateFile`/`UpdateFile` image branch).
- **Reuse:** Any future service needing image conversion to WebP.

### Asset: Bunny 360p readiness poll
- **Name:** `Is360pReady`, `WaitForBunny360p`
- **Path:** `pkg/media/bunny_video_status.go`, `pkg/media/bunny_transcode_wait.go`
- **Purpose:** Check `BunnyVideoDetail` for `status >= ResolutionsFinished` or `"360p"` in `availableResolutions`; `WaitForBunny360p` is a synchronous poll of Bunny's GET-video API with exponential backoff until 360p is ready or timeout.
- **Current usage:** `pkg/media/clients.go:UploadBunnyVideo` (called after PUT succeeds).

### Asset: EnqueueOrphanCleanup helpers
- **Name:** `EnqueueOrphanCleanupForFileID`, `EnqueueOrphanCleanupByObjectKey` (`internal/media/jobs/enqueue.go`); `EnqueueOrphanCleanupForMediaFileID`/`...Row` (`internal/jobs/media/media_orphan_enqueue.go`)
- **Purpose:** Schedule deferred pending-cloud cleanup after an FK reference is cleared or the parent entity is deleted. Prefer the file-ID helper when the caller holds `media_files.id`; use the object-key variant only when the caller already has a raw `object_key` and no row.
- **Current usage:** `internal/server/wire.go`, `internal/taxonomy/application/category_service.go`, `internal/auth/application/{me_update,user_delete}.go`.
- **Reuse:** Always prefer the file-ID path over the object-key path when a UUID is available.

### Asset: LoadValidatedProfileImageFile
- **Name:** `LoadValidatedProfileImageFile`
- **Path:** `internal/media/application/profile_media_validate.go`
- **Purpose:** Resolve UUID → `media_files` row; enforce FILE kind, READY status, raster image MIME/extension via `mapping.ProfileImageFileAcceptable`. Any validation failure returns `pkg/errors.ErrInvalidProfileMediaFile` only (no wrapped errors); DB errors propagate unchanged.
- **Current usage:** `internal/taxonomy/application/category_service.go`, `internal/auth/application/me_update.go`.

### Asset: MediaFilePublic mapping
- **Name:** `ToMediaFilePublicFromModel`, `ToMediaFilePublicFromEntity`, `ProfileImageFileAcceptable`
- **Path:** `pkg/logic/mapping/media_public_mapping.go`
- **Purpose:** Build the public `MediaFilePublic` API shape; shared image-kind acceptance rules.

### Asset: GetByURL (FileRepository)
- **Name:** `GetByURL`
- **Path:** `internal/*/infra/media/file_repository.go`
- **Purpose:** Find a non-deleted `media_files` row by its public URL or origin URL — used by orphan cleanup to resolve provider/key from a plain URL string.
- **Current usage:** `internal/jobs/media/media_orphan_enqueue.go`.

### Asset: Course outline / versioning helpers
- **Name:** `reorderStableIDRows` (+ `applyStableIDReorderRowMeta`, `tableNameForModel`, `buildStableIDOrderIndexCase`), `createNextDraftVersion`/`createDraftVersion`, `hydrateLessonRowsWithSubLessons`, `loadOutlineSequential`, `parallelReadDB`
- **Path:** `internal/course/infra/{repo_helpers,repo_versioning,repo_access,repo_outline}.go`
- **Purpose:** Reassign `order_index` for active outline rows via a two-phase bulk `CASE` UPDATE (not one UPDATE per row) to avoid unique-index collisions; allocate the next draft version (`version_no = MAX+1`), optionally cloning metadata/refs/outline from a source version; hydrate sub-lessons after reorder; load the full outline sequentially (safe inside GORM transactions, avoids `conn busy`/`bad connection`); give each `errgroup` goroutine its own GORM session (`db.Session(&gorm.Session{NewDB: true})`) for safe concurrent reads scoped to a transaction.
- **Scope:** Course outline/versioning infra only — do not duplicate reorder loops, version-clone logic, or raw `*gorm.DB` sharing across goroutines elsewhere.

### Asset: Curriculum estimated duration / batch read helpers
- **Name:** `resolveSubLessonEstimatedDurationMs`, `normalizeSubLessonEstimatedDurationMs`, `applyOutlineEstimatedDurations`, `applySubLessonListEstimatedDurations`, `batchMediaDurationMs`, `batchMediaURLAndDurationMsMaps`, `batchHydrateSubLessons`, `hydrateSubLessonKinds`, `batchMediaURLMap`, `loadOutlineTreeRows`, `assembleOutlineSections`
- **Path:** `internal/course/infra/{duration,repos,repo_outline_batch,repo_access}.go`
- **Purpose:** Resolve sub-lesson `estimated_duration_ms` (VIDEO from linked `media_files` row; TEXT/QUIZ from DB column); batch-load media durations/URLs in one query; batch-load/assemble the outline tree without per-sub-lesson N+1 queries. Video length is persisted by the media module (Bunny webhook / backfill), not fetched from cloud at course read time.
- **Scope:** Course outline read/write only — single resolver for duration semantics.

### Asset: Course version read assembly
- **Name:** `loadCourseVersionAssets`, `loadCourseVersionAssetsBatch`, `loadVersionRefIDsBatch`, `mapCourseVersionRow`, `loadCourseDetailParts`, `loadPublishedDraftVersionRows`, `requireCourseAccess`
- **Path:** `internal/course/infra/{repos,repo_access}.go`
- **Purpose:** Batch parallel load of version tag/skill/outcome refs and thumbnail/preview media URLs across live + draft versions; parallel assembly of `CourseDetail`. `requireCourseAccess` resolves owner/collaborator role in one JOIN query. `GET` supports `include_outline=false`.
- **Current usage:** `GetCourseDetail`, `loadCourseDetail`, `toCourseVersion`.

### Asset: Instructor roster / collaborator bulk add (batch repositories)
- **Name:** `AddRosterBulk` (`internal/instructor/infra/repo_roster_bulk.go`); `AddCollaboratorsBulk` (`internal/course/infra/repo_collaborators_bulk.go`, `internal/course/application/service_collaborators_bulk.go`)
- **Purpose:** Batch roster/collaborator add in one transaction: batch user load, batch platform-staff/instructor-eligibility check (`gormx.UserIDSetByRoleNames`), batch existing-member check, then a batched write (`CreateInBatches` + `ON CONFLICT DO NOTHING` for roster; `RoleBindingService.Assign` for collaborators, via the resource-scoped role gate rather than a Course-owned table). Per-user business failures land in `failed[]`; infrastructure errors roll back the whole write. Idempotent under concurrent requests.
- **Scope:** `POST /instructors/bulk` and `POST /courses/:courseId/collaborators/bulk` only (legacy single-add collaborator POST removed).

---

## Middleware / Validator Assets

### Asset: AuthJWT
- **Name:** `AuthJWT`
- **Path:** `middleware/auth_jwt.go`
- **Purpose:** Access token validation and context projection.
- **Scope:** All authenticated API groups.
- **Reuse:** Reuse without modification for new authenticated domains.

### Asset: RequirePermission
- **Name:** `RequirePermission`
- **Path:** `middleware/rbac.go`
- **Purpose:** Permission enforcement by action names.
- **Dependencies:** JWT context permissions + `services.UserHasAllPermissions`.
- **Current usage:** all permission-gated API routes, typically via `internal/shared/utils.RoutePermission(...)`.
- **Reuse:** Primary reusable gate for taxonomy/course/commerce CRUD actions.

### Asset: httperr middleware
- **Name:** `Middleware`, `Recovery`, `Abort`, `HTTPError`
- **Path:** `internal/shared/httperr/*`
- **Purpose:** Centralize parse/validation/runtime error mapping to the response envelope after handlers return.
- **Scope:** Global API behavior (`internal/server/router.go`); handlers typically use `response.AbortFail` or `c.Error(err)` + `httperr.Abort`.

---

## Reusable Query/Template Assets

### Asset: RBAC SQL templates
- **Name:** `rbacSQLPermissionCodesForUserTmpl` and delete templates
- **Path:** `internal/rbac/application/rbac.go`
- **Purpose:** Efficient permission resolution and FK-safe deletions.
- **Reuse:** Reuse pattern for advanced domain queries requiring controlled SQL.

### Asset: DB schema namespace
- **Name:** `dbschema.RBAC.*`, `dbschema.Media.*`, `dbschema.Taxonomy.*`, `dbschema.System.*`, `dbschema.AppUser.Table()`
- **Path:** `dbschema/*.go` (per domain file); literals in `internal/shared/constants/dbschema_name.go`
- **Purpose:** Centralized table names for SQL safety and consistency; `constants` holds strings, `dbschema` exposes typed accessors (no import cycle — `constants` does not import `dbschema`).
- **Reuse:** Add new `dbschema/<domain>.go` files + new `constants` `Table*` entries for future modules.

### Asset: Permission Catalog
- **Name:** `AllPermissions`, `AllPermissionEntries`
- **Path:** `internal/shared/constants/permissions.go`
- **Purpose:** Canonical permission IDs and names — see `docs/DATA-MODEL.md`'s Permission Catalog appendix for the full current list and count.
- **Scope:** RBAC sync, middleware policy, route authorization.
- **Reuse:** Extend with new entries for new domain CRUD permissions here only — never hardcode a permission ID elsewhere.

### Asset: Role-Permission Matrix
- **Name:** `RolePermissions`, `AllRolePermissionPairs`
- **Path:** `internal/shared/constants/roles_permission.go`
- **Purpose:** Declarative role-permission assignment source for DB sync.
- **Current usage:** `internal/rbacsync/role_sync.go`, sync commands.
- **Reuse:** Extend only by adding new `perm_id` tags; keep role set stable.

### Asset: Error code taxonomy
- **Name:** `internal/shared/errors` (codes + default messages)
- **Path:** `internal/shared/errors/{errcode_codes,errcode_messages}.go`, sentinels in domain `*.go` files
- **Purpose:** Stable application error contract.
- **Scope:** All handlers and middleware.
- **Reuse:** Reuse for all new CRUD failure conditions.

### Asset: Upload size limit constants
- **Name:** `MaxMediaUploadFileBytes`, `MsgFileTooLargeUpload`, `ErrFileExceedsMaxUploadSize`, `FileTooLarge` (errcode)
- **Path:** `internal/shared/constants/error_msg.go`; sentinel in `pkg/errors/upload_errors.go`; code in `pkg/errcode/{codes,messages}.go`
- **Purpose:** Single source of truth for the maximum bytes of one multipart `file` on media create/update (2 GiB) and the single user-facing string for the over-limit case, shared between the JSON `message` and the sentinel `Error()` to prevent drift. Distinct application code (**2003**) from generic `BadRequest` (**3001**).
- **Reuse:** Any future code path surfacing the same message must import this constant, not re-type the sentence.

---

## Known Gaps (Not Yet Built)

Confirmed by source check (not yet present as of this writing):

- **Series** — no `series` package anywhere under `internal/`.
- **Coupons / Orders** — no `coupon` or `order` package anywhere under `internal/`.
- **Payment / checkout bounded context** — no standalone package anywhere under `internal/`; price/discount/rating storefront fields on the public catalog (`GET /catalog/courses/trending`, `GET /catalog/instructors/popular`) remain planned pending this.
- **Dedicated quiz domain** — sub-lesson kind `QUIZ` exists in the course outline (see duration helpers above), but there is no standalone quiz package with its own question/answer/grading model.

Previously listed here as missing but **confirmed already built** (do not re-add without checking first): course shell/versioning, sections/lessons/content, and progress/reviews (`internal/course/infra/repo_review.go`, `internal/course/delivery/handler_review.go` exist).

- Missing shared query helper namespaces beyond RBAC.
- Missing generalized ownership-check helper utilities for instructor/learner/admin scopes.
- Place all test suites (unit/module-level/integration), fixtures, and harnesses under `tests/` only — see `docs/ARCHITECTURE.md`.
