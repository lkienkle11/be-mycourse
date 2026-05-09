# Reusable Assets Inventory (Deep Scan)


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable **domain** types must be declared in **`pkg/entities`** (no `gorm` / `database/sql`).
- GORM / JSONB **column** types for model fields: refresh-session JSONB in **`pkg/gormjsonb/auth`**, soft-delete alias in **`pkg/sqlmodel`**.
- Do not declare new reusable/domain types inline inside logic implementation files.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Function / Method Assets

### Asset: PermissionCodesForUser
- Name: `PermissionCodesForUser`
- Type: Function (service)
- Path: `services/rbac/rbac.go`
- Purpose: Resolve effective permission names from role grants + direct grants.
- Scope: All authorization-sensitive APIs and token issuance.
- Dependencies: `rbacDB`, `pkg/sqlnamed`, `constants` (`RbacSQL*` templates, filled in `services/rbac/rbac.go` `init`).
- Current Usage: `services/auth/auth.go`, `api/v1/me.go`, `api/v1/internal/rbac_handler.go`, `UserHasAllPermissions`.
- Reuse Opportunity:
  - Reuse for all new CRUD permission checks and permission projection in future domains.

### Asset: UserHasAllPermissions
- Name: `UserHasAllPermissions`
- Type: Function (service guard)
- Path: `services/rbac/rbac.go`
- Purpose: Verify required permission set for a user.
- Scope: Authorization middleware fallback and potential service guardrails.
- Dependencies: `PermissionCodesForUser`.
- Current Usage: `middleware/rbac.go`.
- Reuse Opportunity:
  - Direct reuse for `resource:action` checks on new CRUD endpoints.

### Asset: issueTokenPair / RefreshSession
- Name: `issueTokenPair`, `RefreshSession`
- Type: Function (auth service)
- Path: `services/auth/auth_session_tokens.go` (`issueTokenPair`); `services/auth/auth_refresh_rotation.go` (`RefreshSession` + rotation helpers); JSONB writes in `repository/user_refresh_session.go` (`AddRefreshSession`, `SaveRefreshSession`); session entry shape in **`pkg/entities/refresh_session.go`**, JSONB column **`Valuer`/`Scanner`** in **`pkg/gormjsonb/auth/refresh_token_session_map.go`**, soft-delete alias in **`pkg/sqlmodel/deleted_at.go`**
- Purpose: Token issue/rotation and session persistence management.
- Scope: Any auth/session extension features.
- Dependencies: `pkg/token`, `pkg/sqlmodel`, `constants` (TTLs), `models`, `repository`, `services/cache`, RBAC permission resolver.
- Current Usage: auth register/login/confirm/refresh flows.
- Reuse Opportunity:
  - Reuse unchanged for newly protected domain APIs.

### Asset: ListPermissions
- Name: `ListPermissions`
- Type: Function (service list pattern)
- Path: `services/rbac/rbac.go`
- Purpose: Paginated list with safe sort/search whitelist behavior.
- Scope: Reusable blueprint for list endpoints.
- Dependencies: `dto.BaseFilter` semantics, GORM query shaping.
- Current Usage: `api/v1/internal/rbac_handler.go`.
- Reuse Opportunity:
  - Reuse its whitelist + pagination pattern for taxonomy/courses/series lists.

## Type / DTO / Interface Assets

### Asset: BaseFilter
- Name: `BaseFilter`
- Type: Type/DTO
- Path: `dto/filter.go`
- Purpose: Shared pagination/sort/search DTO contract.
- Scope: All list APIs.
- Dependencies: None.
- Current Usage: `dto.PermissionFilter`.
- Reuse Opportunity:
  - Mandatory reuse for all list/read CRUD endpoints in phases 01-12.

### Asset: RBAC DTO Suite
- Name: `CreatePermissionRequest`, `UpdatePermissionRequest`, `CreateRoleRequest`, `SetRolePermissionsRequest`, `AssignUserRoleRequest`, etc.
- Type: Type/DTO
- Path: `dto/rbac.go`
- Purpose: Request schemas and validation tags for CRUD operations.
- Scope: Internal admin API contracts.
- Dependencies: Gin binding/validator tags.
- Current Usage: `api/v1/internal/rbac_handler.go`.
- Reuse Opportunity:
  - Template to design new domain DTO suites with consistent validation style.

### Asset: MeResponse
- Name: `MeResponse`
- Type: Type/DTO
- Path: `dto/auth.go`
- Purpose: Canonical user self payload for auth/me endpoints.
- Scope: User profile/session related flows.
- Dependencies: Auth service builders and cache serializer.
- Current Usage: `services/auth/auth.go`, `services/cache/auth_user.go`, `api/v1/me.go`.
- Reuse Opportunity:
  - Reuse for profile reads and permission-aware user summary payloads.

## Utility / Helper Assets

### Asset: ParseListFilter and parse helpers
- Name: `ParseListFilter`, `buildSortClause`, `buildSearchClause`
- Type: Util/Helper
- Path: `pkg/query/filter_parser.go`
- Purpose: Parse query params into safe pagination/filter/sort metadata with whitelist support.
- Scope: All list endpoints that need reusable sorting/search behavior.
- Dependencies: `strings`, `strconv`.
- Current Usage: taxonomy list endpoints (`course_level`, `category`, `tag`) via `repository/taxonomy`.
- Reuse Opportunity:
  - Reuse for course shell and later listing endpoints to avoid duplicating query parsing logic.

### Asset: Request param helper package
- Name: `CurrentUserID`, `ParseUintParam`
- Type: Util/Helper
- Path: `pkg/requestutil/params.go`
- Purpose: Centralize request-context user extraction and path param integer parsing.
- Scope: All HTTP handlers requiring authenticated user id or `:id` parsing.
- Dependencies: `gin.Context`, `pkg/logic/utils`.
- Current Usage: taxonomy handlers.
- Reuse Opportunity:
  - Reuse in all future CRUD handlers to keep transport-layer parsing behavior consistent.

### Asset: Generic uint path-param parser
- Name: `ParseUintPathParam`
- Type: Util/Helper
- Path: `pkg/logic/utils/params.go`
- Purpose: Parse unsigned integer path params from `gin.Context` with one shared implementation.
- Scope: Internal RBAC and taxonomy handlers (through direct usage or `pkg/requestutil` delegation).
- Dependencies: `gin.Context`, `strconv`.
- Current Usage: `api/v1/internal/rbac_handler.go` (direct calls), `pkg/requestutil/params.go`.
- Reuse Opportunity:
  - Reuse for all future `:id`/`:userId`/`:roleId` style path parsing to eliminate duplicate conversions.

### Asset: Permission id path-param parser
- Name: `ParsePermissionIDParam`
- Type: Util/Helper
- Path: `pkg/requestutil/params.go`
- Purpose: Parse and validate permission id path params (trim + max length check).
- Scope: Internal RBAC permission handlers.
- Dependencies: `gin.Context`, `strings`.
- Current Usage: `api/v1/internal/rbac_handler.go` (direct calls).
- Reuse Opportunity:
  - Reuse for all endpoints handling permission-id route params to keep validation behavior consistent.

### Asset: Shared pagination page builder
- Name: `BuildPage`
- Type: Util/Helper
- Path: `pkg/logic/utils/pagination.go`
- Purpose: Centralize pagination response construction (`Page`, `PerPage`, `TotalPages`, `TotalItems`) and avoid duplicated manual total-page math.
- Scope: All paginated handlers across taxonomy/internal modules and future CRUD modules.
- Dependencies: `pkg/response.PageInfo`.
- Current Usage: `api/v1/taxonomy/*_handler.go`, `api/v1/internal/rbac_handler.go`.
- Reuse Opportunity:
  - Reuse by all list endpoints to keep pagination behavior consistent and reduce duplicated handler logic.

### Asset: Local media URL reversible signer
- Name: `EncodeLocalObjectKey`, `DecodeLocalObjectKey`
- Type: Util/Helper
- Path: `pkg/media/local_url_codec.go`
- Purpose: Build reversible signed URL tokens for local provider objects.
- Scope: Media local-provider read path and future private file links.
- Dependencies: `crypto/hmac`, `crypto/sha256`, `encoding/base64`.
- Current Usage: `pkg/media/clients.go`, `services/media/file_service.go`.
- Reuse Opportunity:
  - Reuse for secure temporary download tokens in other modules.

### Asset: Local media URL token decoder
- Name: `DecodeLocalURLToken`
- Type: Util/Helper
- Path: `pkg/media/local_url_codec.go`
- Purpose: Decode local signed media URL tokens with env-based secret fallback outside service layer.
- Scope: Media local decode endpoint and future local signed-link consumers.
- Dependencies: `os`, `strings`, `DecodeLocalObjectKey`.
- Current Usage: `api/v1/media/file_handler.go`.
- Reuse Opportunity:
  - Reuse for any endpoint needing reversible local object-key token decoding.

### Asset: Media kind/provider resolvers + Bunny metadata policy
- Name: `ResolveMediaKind`, `ResolveMediaProvider`, `ResolveMediaKindFromServer`, `ResolveUploadProvider`, plus `EnrichBunnyVideoDetail`, `EffectiveBunnyThumbnailURL`, `FormatBunnyVideoIDString`, `ResolveBunnyEmbedURL`, `ResolveBunnyEmbedHTML`, `ApplyBunnyDetailToMetadata`
- Type: Util/Helper
- Path: `pkg/media/media_resolver.go`
- Purpose: Normalize upload kind/provider with server-owned inference. Map Bunny GET-video payload into metadata keys **`video_id`**, **`thumbnail_url`**, **`embeded_html`** (see `constants/media_meta_keys.go`).
- Scope: Media upload + finished webhook; HTTP stays in `pkg/media/clients.go`.
- Dependencies: `constants/media.go`, `constants/media_meta_keys.go`, `pkg/entities`, `fmt`, `html`, `path/filepath`, `strconv`, `strings`, `mycourse-io-be/pkg/setting` (resolver config).
- Current Usage: `services/media/file_service.go`, `services/media/video_service.go`, `pkg/media/clients.go`.
- Reuse Opportunity:
  - Reuse for any future media ingestion endpoints to keep provider-kind and Bunny contracts identical.

### Asset: Media metadata JSON key constants
- Name: `MediaMetaKeyVideoGUID`, `MediaMetaKeyBunnyVideoID`, `MediaMetaKeyVideoID`, `MediaMetaKeyThumbnailURL`, `MediaMetaKeyEmbededHTML`
- Type: Constants
- Path: `constants/media_meta_keys.go`
- Purpose: Single source of truth for string keys used in `metadata_json` and merged raw maps (Bunny parity Sub 09).
- Scope: **`pkg/media`**, mapping, services; do not duplicate literal key strings elsewhere.
- Current Usage: `pkg/media/media_resolver.go`, `pkg/media/media_upload_entity.go`, `pkg/logic/mapping/media_model_mapping.go`, `services/media/video_service.go`.
- Reuse Opportunity: Any new writer of `metadata_json` for Bunny should import these constants.

### Asset: Mapping helpers for API DTO contracts
- Name: `ToUploadFileResponse`, `ToCategoryResponse`, `ToCourseLevelResponse`, `ToTagResponse` (+ slice variants)
- Type: Util/Helper
- Path: `pkg/logic/mapping/media_file_mapping.go`, `pkg/logic/mapping/taxonomy_category_mapping.go`, `pkg/logic/mapping/taxonomy_course_level_mapping.go`, `pkg/logic/mapping/taxonomy_tag_mapping.go`
- Purpose: Centralize entity/model -> DTO mapping so handlers do not return raw persistence/entity structs. **`ToUploadFileResponse`** omits canonical origin from the public DTO (Sub 12 — no `origin_url` on `dto.UploadFileResponse`); internal `entities.File.OriginURL` / DB `origin_url` still store it for server use.
- Scope: Media and taxonomy transport responses.
- Dependencies: `dto`, `models`, `pkg/entities`.
- Current Usage: `api/v1/media/file_handler.go`, `api/v1/taxonomy/*_handler.go`.
- Reuse Opportunity:
  - Reuse for all upcoming domain handlers to enforce stable public API contracts.

### Asset: App media provider config
- Name: `MediaSetting.AppMediaProvider`
- Type: Config/Constant source
- Path: `pkg/setting/setting.go`
- Purpose: Server-side source of truth for upload provider selection.
- Scope: Media service provider resolution.
- Dependencies: YAML/env config loading in `pkg/setting`.
- Current Usage: `pkg/media/media_metadata.go`, `services/media/file_service.go`, `pkg/media/clients.go`.
- Reuse Opportunity:
  - Reuse as canonical provider control for all future media upload entry points.

### Asset: Cloud SDK client bootstrap (`MediaSetting`)
- Name: `NewCloudClientsFromSetting`
- Type: Function (`pkg/media`)
- Path: `pkg/media/clients_setting_attach.go`
- Purpose: One-shot construction of `entities.CloudClients` (B2 client + bucket name, Gcore CDN API service, Bunny Storage client) from **`setting.MediaSetting`** fields: `B2KeyID`, `B2AppKey`, `B2Bucket`, `GcoreAPIBaseURL`, `GcoreAPIToken`, `BunnyStorageEndpoint`, `BunnyStoragePassword` (all `strings.TrimSpace`). No `os.Getenv` in this path — values arrive via `setting.Setup()` / YAML `${MEDIA_*}` expansion.
- Scope: App startup only; caller `pkg/media.Setup()` (after `setting.Setup()` in `main.go`).
- Dependencies: `pkg/setting`, `github.com/Backblaze/blazer/b2`, Gcore and Bunny Storage SDKs, `pkg/logic/utils.NormalizeBaseURL`.
- Current Usage: `pkg/media/setup.go`.
- Reuse Opportunity: Any new process that needs the same cloud handles should call `media.Setup` or reuse the global `media.Cloud` rather than duplicating env reads.

### Asset: Media metadata parser helpers
- Name: `ParseMetadataJSON`, `ParseMetadataFromRaw`, `NormalizeMetadata`, `BuildTypedMetadata`, `DefaultMediaProvider`
- Type: Util/Helper
- Path: `pkg/media/media_metadata.go`
- Purpose: Parse raw metadata JSON, normalize metadata payload, and infer typed metadata contract `UploadFileMetadata` with explicit fields/default values (including `width_bytes`, `height_bytes`, `has_password`, `archive_entries`).
- Scope: Media handlers/services and any upload endpoint accepting metadata JSON.
- Dependencies: `encoding/json`, `fmt`, `strings`, `pkg/entities`, `pkg/setting`.
- Current Usage: `api/v1/media/file_handler.go`, `services/media/file_service.go`.
- Reuse Opportunity:
  - Reuse for all future endpoints that accept metadata in raw string form and require backend metadata inference.

### Asset: Media upload object keys (B2 / Bunny / local)
- Name: `ResolveMediaUploadObjectKey`, `BuildB2ObjectKey`
- Type: Helper
- Path: `pkg/media/media_upload_keys.go`
- Purpose: Provider-specific default object keys before upload (8-digit B2 prefix; Bunny empty until GUID; local nano key).
- Scope: Media upload service and any future upload entry point.
- Dependencies: `constants`, `pkg/logic/utils` (`GenerateRandomDigits`), `path/filepath`, `strings`, `time`.
- Current Usage: `services/media/file_service.go`.
- Reuse Opportunity: Reuse instead of duplicating filename sanitization or key rules in `pkg/media`.

### Asset: Content fingerprint (SHA-256 hex)
- Name: `ContentFingerprint`
- Type: Util
- Path: `pkg/logic/utils/parsing.go`
- Purpose: Stable digest of upload bytes for skip-upload dedupe on `PUT /media/files/:id`.
- Scope: Media replace/update flows; any binary fingerprint need.
- Dependencies: `crypto/sha256`, `encoding/hex`.
- Current Usage: `services/media/file_service.go`, `tests/sub06_media_orphan_safety_test.go`.
- Reuse Opportunity: Any feature needing cheap binary equality without storing raw bytes.

### Asset: MergeMediaMetadataJSON
- Name: `MergeMediaMetadataJSON`
- Type: Helper
- Path: `pkg/media/media_metadata_merge.go`
- Purpose: Merge JSONB metadata with overlay map for metadata-only updates.
- Scope: Media row updates preserving unspecified keys.
- Dependencies: `encoding/json`, `pkg/entities.RawMetadata`.
- Current Usage: `services/media/file_service.go`.

### Asset: Superseded cloud cleanup guard
- Name: `ShouldEnqueueSupersededCloudCleanup`
- Type: Helper
- Path: `pkg/media/media_replace_policy.go`
- Purpose: Decide whether replace produced a new cloud identity requiring deferred delete of the prior object.
- Scope: Media replace branch after successful DB save.
- Current Usage: `services/media/file_service.go`.

### Asset: Multipart binders (media create/update)
- Name: `BindCreateFileMultipart`, `BindUpdateFileMultipart`
- Type: Helper (transport parsing)
- Path: `pkg/logic/mapping/multipart_gin_bind.go`
- Purpose: Parse multipart text fields for backward-compat validation and bind allowed update controls (`reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged`); client `kind`/`metadata` are intentionally ignored by service flow.
- Scope: `api/v1/media/file_handler.go` only; keeps handlers thin.
- Dependencies: `github.com/gin-gonic/gin`, `dto`, `pkg/logic/utils` (`ParseBoolLoose` for `skip_upload_if_unchanged`).

### Asset: DeleteStoredObject (unified cloud delete)
- Name: `DeleteStoredObject`
- Type: Function (`pkg/media`)
- Path: `pkg/media/stored_object_delete.go`
- Purpose: Route delete by provider (B2 key vs Bunny GUID vs local noop).
- Scope: Delete compensation, pending cleanup worker, avoids duplicating switch in callers.
- Current Usage: `services/media/file_service.go`, `internal/jobs/media/media_pending_cleanup_batch.go`.

### Asset: Media cleanup worker scheduler
- Name: `StartMediaPendingCleanupJob`, `StopMediaPendingCleanupJob`
- Type: Job (`internal/jobs/media`)
- Path: `internal/jobs/media/media_pending_cleanup_scheduler.go`
- Purpose: In-memory ticker for deferred cloud deletes — **same mutex/cancel/waitgroup pattern** as `internal/jobs/rbac/rbac_sync_schedulers.go` (`syncJobBundle`).
- Scope: Process-wide background loop; interval from `MEDIA_CLEANUP_INTERVAL_SEC` or `constants.MediaCleanupDefaultIntervalSec`.

### Asset: Media concurrency / reuse errors
- Name: `ErrMediaOptimisticLock`, `ErrMediaReuseMismatch`
- Type: Sentinel (`pkg/errors`)
- Path: `pkg/errors/media_errors.go`
- Purpose: Map to HTTP **409** + `errcode.Conflict` when row version or `reuse_media_id` mismatches.
- Dependencies: `constants/error_msg.go` (`MsgMediaOptimisticLockConflict`, `MsgMediaReuseMismatch`).

### Asset: URL join helper (generic)
- Name: `JoinURLPathSegments`
- Type: Util
- Path: `pkg/logic/utils/url.go`
- Purpose: Join CDN/base URL with path segments without duplicated slashes.
- Scope: Any module building hierarchical URLs (media B2 public URLs, etc.).
- Dependencies: `strings`.
- Current Usage: `pkg/media/clients.go`, `tests/sub04_media_pipeline_test.go`.
- Reuse Opportunity: Prefer over manual string concatenation for CDN + bucket + key.

### Asset: Cryptographic random decimal string
- Name: `GenerateRandomDigits`
- Type: Util
- Path: `pkg/logic/utils/random.go`
- Purpose: `n` decimal digits from `crypto/rand` (used for B2 object key prefix).
- Scope: Any feature needing short random numeric IDs.
- Dependencies: `crypto/rand`, `io`.
- Current Usage: `pkg/media/media_upload_keys.go`.
- Reuse Opportunity: Reuse for other token prefixes; do not reimplement with `math/rand`.

### Asset: Bunny Stream — constants vs `pkg/media`
- **Constants** (`constants/bunny_video.go`): `FinishedWebhookBunnyStatus`, `SignBunnyIFrameRegex` — literals only (Global Constants Placement).
- **Bunny video status** (`constants/bunny_video_status.go`): Bunny webhook status constants **0..10** live in `constants/bunny_video_status.go`; status-name mapping helper is `pkg/media.BunnyStatusString(status)`.
- **Webhook signature helpers** (`pkg/media/webhook_signature.go`): `BunnyWebhookSigningSecret`, `BunnyWebhookSignatureExpectedHex`, `IsBunnyWebhookSignatureValid` (v1 + hmac-sha256 + constant-time compare on raw body).
- Current Usage: `api/v1/media/webhook_handler.go`, `services/media/video_service.go`, `tests/sub04_media_pipeline_test.go`, `tests/sub16_bunny_webhook_test.go`.

### Asset: Media provider typed error + HTTP mapping
- Name: `ProviderError`, `AsProviderError`, `HTTPStatusForProviderCode`
- Type: Error / helper
- Path: `pkg/errors/provider_error.go`
- Purpose: Carry `errcode` 9010–9018 for B2/Bunny/encode client failures; map to HTTP 500/502/503/504.
- Scope: Media handlers and provider clients.
- Dependencies: `errors`, `net/http`, `pkg/errcode`.
- Current Usage: `pkg/media/clients.go`, `api/v1/media/file_handler.go`, `services/media/file_service.go`.
- Reuse Opportunity: Extend with more provider-specific codes without changing handler shape.

### Asset: Stable “not found” for API boundaries
- Name: `ErrNotFound`, `MapRecordNotFound`
- Type: Sentinel + mapper
- Path: `pkg/errors/not_found.go`
- Purpose: Let `api/**` handlers use `errors.Is(err, ErrNotFound)` for HTTP **404** without importing `gorm.io/gorm`; services/repository map `gorm.ErrRecordNotFound` at the edge.
- Scope: Internal RBAC handlers, taxonomy update handler, any future CRUD that must not leak ORM types through `services`.
- Dependencies: `errors`, `gorm.io/gorm` (mapper only, inside `pkg/errors`).
- Current Usage: `api/v1/internal/rbac_handler.go`, `api/v1/taxonomy/handlers_common.go`, `services/rbac/rbac.go`, `repository/taxonomy/gorm_shared.go`.
- Reuse Opportunity: Prefer this over returning raw GORM errors from `services` when handlers need a not-found branch.

### Asset: Media upstream errcodes (9010–9018)
- Name: `B2BucketNotConfigured`, `BunnyStreamNotConfigured`, `BunnyCreateFailed`, `BunnyUploadFailed`, `BunnyInvalidResponse`, `BunnyVideoNotFound`, `BunnyGetVideoFailed`, **`ImageEncodeBusy`** (9017), **`VideoTranscodeTimeout`** (9018)
- Type: Constant / API contract
- Path: `pkg/errcode/codes.go`, `pkg/errcode/messages.go`, `constants/error_msg.go` (`MsgMedia*`, `MsgImageEncodeBusy`, `MsgVideoTranscodeTimeout`)
- Purpose: Stable API codes + default JSON messages for media upstream failures.
- Scope: Clients of `pkg/response` and observability.
- Dependencies: `constants` for shared message literals.
- Current Usage: `pkg/errors/provider_error.go`, `api/v1/media/file_handler.go`.
- Reuse Opportunity: Reference from tests and dashboards; keep messages only in `constants/error_msg.go`.

### Asset: Generic parsing / metadata primitives (utils)
- Name: `DetectExtension`, `ImageSizeFromPayload`, `StringFromRaw`, `IntFromRaw`, `FloatFromRaw`, `NonEmpty`, `ParseBoolLoose`, `ContentFingerprint`
- Type: Util
- Path: `pkg/logic/utils/parsing.go`
- Purpose: Raw-metadata helpers, loose bool parsing for multipart text fields, SHA-256 hex fingerprint of byte payloads.
- Scope: Any module needing generic conversion/parsing without domain coupling.
- Dependencies: `pkg/entities`, Go stdlib (`image`, `bytes`, `strings`, `fmt`, `crypto/sha256`, `encoding/hex`).
- Current Usage: `pkg/media/media_metadata.go`, `pkg/media/media_multipart.go`, `services/media/file_service.go`.
- Integration Note (2026-04-27): **`pkg/media`** `FileKindFile` branch uses `utils.ImageSizeFromPayload` and `utils.IntFromRaw`; alias mismatch (`util.*`) is a compile-time risk.
- Reuse Opportunity:
  - Reuse directly in future modules to avoid re-implementing generic conversion/parsing primitives.

### Asset: Taxonomy status normalization
- Name: `NormalizeTaxonomyStatus`
- Type: Util/Helper
- Path: `pkg/taxonomy/status.go`
- Purpose: Map request status strings to taxonomy status string constants (`constants.TaxonomyStatusActive` / `constants.TaxonomyStatusInactive`) with a single default-to-active rule.
- Scope: Taxonomy create/update flows and any future domain using the same enum.
- Dependencies: `strings`, `constants/taxonomy.go`.
- Current Usage: `services/taxonomy/category_service.go`, `services/taxonomy/tag_course_level_services.go`, `services/taxonomy/fields.go`.
- Reuse Opportunity:
  - Reuse whenever another module accepts taxonomy status as raw text.

### Asset: sqlnamed.Postgres
- Name: `Postgres`
- Type: Util/Helper
- Path: `pkg/sqlnamed/postgres.go`
- Purpose: Named SQL to PostgreSQL positional parameter conversion.
- Scope: Complex raw SQL operations.
- Dependencies: `github.com/jmoiron/sqlx`.
- Current Usage: `services/rbac/rbac.go`.
- Reuse Opportunity:
  - Reuse for complex CRUD joins/aggregations where GORM is not ideal.

### Asset: setting.Setup
- Name: `Setup`
- Type: Util/Helper
- Path: `pkg/setting/setting.go`
- Purpose: Unified env + stage YAML config loading.
- Scope: Application and CLI startup.
- Dependencies: env parser, YAML loader, envbool.
- Current Usage: `main.go`, `cmd/syncpermissions/main.go`, `cmd/syncrolepermissions/main.go`.
- Reuse Opportunity:
  - Reuse unchanged for any new command/service modules.

### Asset: Cache helpers for auth/me
- Name: `GetCachedUserMe`, `SetCachedUserMe`, `LoginInvalidCached`, etc.
- Type: Util/Helper
- Path: `services/cache/auth_user.go`
- Purpose: Cache-aside support for login and me endpoints.
- Scope: High-frequency identity reads and login flows.
- Dependencies: `pkg/cache_clients`, Redis.
- Current Usage: `services/auth/auth.go`.
- Reuse Opportunity:
  - Reuse pattern for read-heavy course catalog/progress reads later.

### Asset: Core taxonomy entities (pure shared types)
- Name: `CourseLevel`, `Category`, `Tag`
- Type: Type/Entity
- Path: `pkg/entities/course_level.go`, `pkg/entities/category.go`, `pkg/entities/tag.go`
- Purpose: Share taxonomy field definitions as pure data types across layers.
- Scope: Domain modeling and transfer structures where persistence mapping is not required.
- Dependencies: `constants` (no dbschema and no table-name binding).
- Current Usage: `models/taxonomy_course_level.go`, `models/taxonomy_category.go`, `models/taxonomy_tag.go` as model embedding source.
- Reuse Opportunity:
  - Extend with additional pure entities for future domains while keeping DB mapping (`TableName`) only in `models/*`.

### Asset: File domain types
- Name: `File`, `FileMetadata`, `ImageMetadata`, `VideoMetadata`, `DocumentMetadata`
- Type: Type/Entity
- Path: `pkg/entities/file.go`
- Purpose: Shared media response descriptor for persisted media upload API with base metadata + typed metadata inheritance model.
- Scope: Media upload service + transport response payload.
- Dependencies: `constants/media.go`.
- Current Usage: `services/media/*`, `api/v1/media/*`, `models/media_file.go`, `repository/media/file_repository.go`.
- Reuse Opportunity:
  - Reuse for course lesson media, subtitles, and future asset libraries.

### Asset: Runtime dependency guard
- Name: `RequireInitialized`, `ErrDependencyNotConfigured`
- Type: Util/Helper
- Path: `pkg/media/runtime_guard.go` (guard); sentinel **`ErrDependencyNotConfigured`** in **`pkg/errors/media_errors.go`** (`constants.MsgMediaDependencyNotConfigured`).
- Purpose: Centralize runtime dependency checks (non-nil guards); handlers map **`ErrDependencyNotConfigured`** to **`errcode.InternalError`** via `api/v1/media/file_handler_errors.go`.
- Scope: Services that rely on startup-initialized dependencies.
- Dependencies: `pkg/errors`, `constants/error_msg.go`.
- Current Usage: `services/media/file_service.go`, `api/v1/media/*`.
- Reuse Opportunity:
  - Reuse in future modules needing startup dependency guards.

### Asset: Media constants
- Name: `FileProvider`, `FileKind`, `FileStatus`
- Type: Constant
- Path: `constants/media.go`
- Purpose: Centralized media enums (moved out of entity to keep entity pure).
- Scope: Media service + handler + DTO validation contracts.
- Dependencies: none.
- Current Usage: `pkg/entities/file.go`, `services/media/*`, `api/v1/media/*`.
- Reuse Opportunity:
  - Reuse in future media-dependent modules.

## Constant / ErrorCode Assets

### Convention: Error Placement and Mapping (Mandatory)
- Errors must be declared in `pkg/errors` (sentinel/typed), not inside `services/*` or `repository/*`.
- Error messages must come from shared constants in `constants/error_msg.go` and be wired into `pkg/errcode/messages.go`.
- Error numeric codes must be defined in `pkg/errcode/codes.go` before use.
- Do not hardcode error code/message in feature modules; map at boundary using centralized `errcode`.

### Convention: Constants Placement (Mandatory)
- All constants must be centralized in `constants/*` (messages, status, thresholds, default values, and any other shared constant).
- Feature layers (`services/*`, `repository/*`, `api/*`, `pkg/*`) must not define business constants inline.
- When adding new constants, place them in the appropriate file inside `constants/` and import from there.
- For error flows, keep `pkg/errcode/*` as code/message mapping tables; source literals should come from `constants/*`.

### Convention Examples (Reference Implementations)
- `pkg/errors/provider_error.go`: typed error pattern for provider/upstream failures with stable code + HTTP status mapping helpers.
- `pkg/errors/upload_errors.go`: sentinel error pattern with shared message constant and `errors.Is`-friendly flow.
- Reuse these two patterns when creating new errors so all modules stay consistent.

### Asset: Permission Catalog
- Name: `AllPermissions`, `AllPermissionEntries`
- Type: Constant catalog
- Path: `constants/permissions.go`
- Purpose: Canonical permission IDs and names.
- Scope: RBAC sync, middleware policy, route authorization.
- Dependencies: reflection and tags.
- Current Usage: sync services, route permission declarations, RBAC APIs.
- Reuse Opportunity:
  - Extend with `P14+` entries for new domain CRUD permissions.

### Asset: Role-Permission Matrix
- Name: `RolePermissions`, `AllRolePermissionPairs`
- Type: Constant mapping
- Path: `constants/roles_permission.go`
- Purpose: Declarative role-permission assignment source for DB sync.
- Scope: Global RBAC policy.
- Dependencies: role name constants and permission IDs.
- Current Usage: `internal/rbacsync/role_sync.go`, sync commands.
- Reuse Opportunity:
  - Extend only by adding new `perm_id` tags; keep role set stable.

### Asset: Error code taxonomy
- Name: `pkg/errcode` constants/messages
- Type: Constant/ErrorCode
- Path: `pkg/errcode/codes.go`, `pkg/errcode/messages.go`
- Purpose: Stable application error contract.
- Scope: All handlers and middleware.
- Dependencies: `pkg/response`, `pkg/httperr`.
- Current Usage: Auth, RBAC, middleware, error pipeline.
- Reuse Opportunity:
  - Reuse for all new CRUD failure conditions.

### Tests Exception (Type Placement)
- Types created only for test scope can be declared directly inside files under `tests/`.
- This tests-only exception does not apply to production code paths in `services/*` or `repository/*`.

## Middleware / Validator Assets

### Asset: AuthJWT
- Name: `AuthJWT`
- Type: Middleware
- Path: `middleware/auth_jwt.go`
- Purpose: Access token validation and context projection.
- Scope: All authenticated API groups.
- Dependencies: `pkg/token`, `pkg/setting`, `pkg/response`.
- Current Usage: `/api/v1` authenticated subgroup.
- Reuse Opportunity:
  - Reuse without modification for new authenticated domains.

### Asset: RequirePermission
- Name: `RequirePermission`
- Type: Middleware
- Path: `middleware/rbac.go`
- Purpose: Permission enforcement by action names.
- Scope: Protected endpoint authorization.
- Dependencies: JWT context permissions + `services.UserHasAllPermissions`.
- Current Usage: `/api/v1/me/permissions`.
- Reuse Opportunity:
  - Primary reusable gate for taxonomy/course/commerce CRUD actions.

### Asset: httperr middleware
- Name: `Middleware`, `Recovery`, `Abort`, `HTTPError`
- Type: Middleware/Validator
- Path: `pkg/httperr/*`
- Purpose: Centralize parse/validation/runtime error mapping to response envelope.
- Scope: Global API behavior.
- Dependencies: `pkg/errcode`, `pkg/response`, `pkg/validate`.
- Current Usage: router global middleware + selected handlers.
- Reuse Opportunity:
  - Reuse as standard error strategy across new CRUD handlers.

## Reusable Query/Template Assets

### Asset: RBAC SQL templates
- Name: `rbacSQLPermissionCodesForUserTmpl` and delete templates
- Type: Query Template
- Path: `services/rbac/rbac.go`
- Purpose: Efficient permission resolution and FK-safe deletions.
- Scope: RBAC read/write internals.
- Dependencies: `dbschema.RBAC`, `pkg/sqlnamed`.
- Current Usage: multiple RBAC service methods.
- Reuse Opportunity:
  - Reuse pattern for advanced domain queries requiring controlled SQL.

### Asset: DB schema namespace
- Name: `dbschema.RBAC.*`, `dbschema.Media.*`, `dbschema.Taxonomy.*`, `dbschema.System.*`, `dbschema.AppUser.Table()`
- Type: Query helper / GORM `TableName()` indirection
- Path: `dbschema/*.go` (per domain file); literals in **`constants/dbschema_name.go`**
- Purpose: Centralized table names for SQL safety and consistency; **`constants`** holds strings, **`dbschema`** exposes typed accessors (no import cycle: `constants` does not import `dbschema`).
- Scope: RBAC services/sync, media/taxonomy models, system models, user model + raw session SQL.
- Dependencies: `constants` (`TableRBAC*`, `TableMedia*`, `TableTaxonomy*`, `TableSystem*`, `TableAppUsers`).
- Current Usage: `services/rbac/rbac.go`, `models/*.go`, `internal/rbacsync/*`.
- Reuse Opportunity:
  - Add new `dbschema/<domain>.go` files + new `constants` `Table*` entries for future modules (`course`, etc.).

### Asset: MaxMediaUploadFileBytes
- Name: `MaxMediaUploadFileBytes`
- Type: Constant
- Path: `constants/error_msg.go`
- Purpose: Single source of truth for maximum bytes of one multipart `file` on media create/update (2 GiB).
- Scope: Media handler early reject, service stream cap, documentation cross-references.
- Dependencies: none.

### Asset: MsgFileTooLargeUpload
- Name: `MsgFileTooLargeUpload`
- Type: Constant (string)
- Path: `constants/error_msg.go`
- Purpose: **Single** user-facing string for media upload over 2 GiB: `pkg/errcode` `defaultMessages[FileTooLarge]` **and** `pkg/errors.ErrFileExceedsMaxUploadSize` (`errors.New`). Prevents drift between JSON `message` and sentinel `Error()`.
- Scope: Any future code paths that surface the same copy must import this constant, not re-type the sentence.
- Dependencies: none.

### Asset: ErrFileExceedsMaxUploadSize
- Name: `ErrFileExceedsMaxUploadSize`
- Type: Sentinel error (`var`)
- Path: `pkg/errors/upload_errors.go` (message: `constants.MsgFileTooLargeUpload` in `constants/error_msg.go`, shared with `pkg/errcode/messages.go`)
- Purpose: Stable `errors.Is` check in handlers for oversize uploads after service-side `LimitReader` / header validation.
- Scope: Media upload flow; map to `errcode.FileTooLarge` + HTTP 413 in `api/v1/media/file_handler.go`; service returns same sentinel via `pkgmedia.ErrFileExceedsMaxUploadSize`.
- Dependencies: `constants.MaxMediaUploadFileBytes`, `constants.MsgFileTooLargeUpload`.

### Asset: ErrExecutableUploadRejected (Sub 11)
- Name: `ErrExecutableUploadRejected`
- Type: Sentinel error (`var`)
- Path: `pkg/errors/media_errors.go`
- Purpose: Stable `errors.Is` check for executable/script file denylist rejections on `POST /media/files`.
- Scope: `services/media/file_service.go` (returns), `api/v1/media/file_handler.go` (maps to HTTP 400 + code 2004).
- Dependencies: `constants.MsgExecutableUploadRejected`.

### Asset: ErrImageEncodeBusy (Sub 11)
- Name: `ErrImageEncodeBusy`
- Type: Sentinel error (`var`)
- Path: `pkg/errors/media_errors.go`
- Purpose: Returned by `utils.EncodeWebP` stub (`//go:build !cgo`) when CGO is not available.
- Scope: `pkg/logic/utils/webp_encode_stub.go`, wrapping in `ProviderError{Code: 9017}` in service.
- Dependencies: `constants.MsgImageEncodeBusy`.

### Asset: IsExecutableUploadRejected (Sub 11)
- Name: `IsExecutableUploadRejected`
- Type: Util (generic security check)
- Path: `pkg/logic/utils/executable_check.go`
- Purpose: Check file upload against extension denylist and magic-byte signatures. Returns true when file must be blocked.
- Scope: `services/media/file_service.go` (CreateFile non-image non-video branch).
- Dependencies: `path/filepath`, `strings`.
- Reuse Opportunity: Any future upload endpoint needing the same executable filter.

### Asset: EncodeWebP + AcquireEncodeGate / ReleaseEncodeGate (Sub 11)
- Name: `EncodeWebP`, `AcquireEncodeGate`, `ReleaseEncodeGate`
- Type: Util (image transformation)
- Path: `pkg/logic/utils/webp_encode.go` (`//go:build cgo`), `pkg/logic/utils/webp_encode_stub.go` (`//go:build !cgo`), `pkg/logic/utils/image_encode_gate.go`
- Purpose: Convert image bytes to WebP (bimg/libvips). Semaphore gate caps concurrent encode workers to `constants.MaxConcurrentImageEncode`.
- Scope: `services/media/file_service.go` (CreateFile + UpdateFile image branch).
- Dependencies: `github.com/h2non/bimg` (CGO build), `constants.MaxConcurrentImageEncode`, `pkg/errors.ErrImageEncodeBusy` (stub).
- Reuse Opportunity: Any future service needing image conversion to WebP.

### Asset: IsImageMIMEOrExt (Sub 11)
- Name: `IsImageMIMEOrExt`
- Type: Helper (media-domain image detection)
- Path: `pkg/media/media_resolver.go`
- Purpose: Detect whether a file is an image by MIME prefix or extension; controls WebP conversion branch.
- Scope: `services/media/file_service.go`.
- Dependencies: `path/filepath`, `strings`.

### Asset: Is360pReady (Sub 11)
- Name: `Is360pReady`
- Type: Helper (Bunny domain)
- Path: `pkg/media/bunny_video_status.go`
- Purpose: Check `BunnyVideoDetail` for `status >= ResolutionsFinished` or `"360p"` in `availableResolutions`.
- Scope: `pkg/media/bunny_transcode_wait.go`.
- Dependencies: `pkg/entities.BunnyVideoDetail`.

### Asset: WaitForBunny360p (Sub 11)
- Name: `WaitForBunny360p`
- Type: Function (`pkg/media`)
- Path: `pkg/media/bunny_transcode_wait.go`
- Purpose: Synchronous poll of Bunny GET-video API with exponential backoff until 360p is ready or timeout.
- Scope: `pkg/media/clients.go:UploadBunnyVideo` (called after PUT succeeds).
- Dependencies: `pkg/media.GetBunnyVideoByID`, `constants.*BunnyTranscode*` (historical), `pkg/errors.ProviderError`, `pkg/errcode.VideoTranscodeTimeout`. *(Sub 11 `Is360pReady` removed from codebase — doc note only.)*

### Asset: FileTooLarge (errcode)
- Name: `FileTooLarge`
- Type: Application error code constant + default message
- Path: `pkg/errcode/codes.go`, `pkg/errcode/messages.go`
- Purpose: Distinct application `code` (**2003**) for payload too large vs generic `BadRequest` (**3001**) used for missing multipart file.
- Scope: Any future upload endpoints that need the same client contract.
- Dependencies: `response.Fail` callers supply HTTP 413 where appropriate; default **message** text for this code is **`constants.MsgFileTooLargeUpload`** (referenced from `messages.go`, not duplicated).

### Asset: ParseImageURLForOrphanCleanup
- Name: `ParseImageURLForOrphanCleanup`
- Type: Util/Helper
- Path: `pkg/media/media_url_orphan.go`
- Purpose: Parse a stored image URL string back to `(provider, objectKey, bunnyVideoID, ok)` using configured `MediaSetting` (Bunny base + library ID, Gcore CDN + B2 bucket). Pure function — no I/O.
- Scope: Orphan cleanup flows in any domain service that holds image URL strings.
- Dependencies: `constants/media.go`, `pkg/logic/utils` (NormalizeBaseURL, JoinURLPathSegments), `pkg/setting`.
- Current Usage: `internal/jobs/media/media_orphan_enqueue.go`.
- Reuse Opportunity: Call whenever a stored image URL must be mapped back to a cloud object for deletion.

### Asset: ScanJSONBForImageURLs
- Name: `ScanJSONBForImageURLs`
- Type: Util/Helper
- Path: `pkg/media/media_jsonb_scan.go`
- Purpose: Walk a raw JSONB payload and collect all string values stored under image-field-named keys (`_url`, `image`, `thumbnail`, `cover`, `banner`, `avatar`, `poster`, `icon`).
- Scope: Future lesson/quiz/section cascade delete flows where content images are embedded in JSONB.
- Dependencies: `encoding/json`, `strings`.
- Current Usage: Tests only (`tests/sub07_orphan_image_test.go`). TODO hooks in `media_jsonb_scan.go`.
- Reuse Opportunity: Use in Phase 05+ when lesson/section/quiz content JSONB is added.

### Asset: EnqueueOrphanImageCleanup
- Name: `EnqueueOrphanImageCleanup`
- Type: Function (enqueue helper)
- Path: `internal/jobs/media/media_orphan_enqueue.go`
- Purpose: Single entry-point to schedule deferred cloud-object deletion for any **plain image URL** field on a business entity. DB-lookup first, URL-parse fallback. Inserts `media_pending_cloud_cleanup` row.
- Scope: Legacy URL fields, JSONB-harvested URLs (`ScanJSONBForImageURLs`), future course cover strings — **not** used for taxonomy/user after Sub 14 (those use **`EnqueueOrphanCleanupForMediaFileID`**).
- Dependencies: `repository.FileRepository`, **`pkg/media.ParseImageURLForOrphanCleanup`**, `constants/media.go`, `models`.
- Current Usage: Reserved for URL-shaped domains; see `docs/data-flow.md` (Sub 07 path).
- Reuse Opportunity: Wire into every future service that stores a **URL string** (not `media_files` FK) on delete/update.

### Asset: EnqueueOrphanCleanupForMediaFileID
- Name: `EnqueueOrphanCleanupForMediaFileID` / `EnqueueOrphanCleanupForMediaFileRow`
- Type: Function (enqueue helper)
- Path: `internal/jobs/media/media_orphan_enqueue.go`
- Purpose: Schedule **`media_pending_cloud_cleanup`** from a **`media_files.id`** (or in-memory row) after a referencing FK is cleared or the parent entity is deleted.
- Current Usage: `services/taxonomy/category_service.go`, `services/auth/me_update.go`, `services/auth/user_delete.go`.

### Asset: LoadValidatedProfileImageFile
- Name: `LoadValidatedProfileImageFile`
- Type: Function (service)
- Path: `services/media/profile_media_validate.go`
- Purpose: Resolve UUID → **`media_files`** row; enforce **FILE** kind, **READY** status, raster image MIME/extension via **`mapping.ProfileImageFileAcceptable`**.
- Current Usage: `services/taxonomy/category_service.go`, `services/auth/me_update.go`.
- Errors: any validation failure returns **`pkg/errors.ErrInvalidProfileMediaFile`** only (sentinel text **`constants.MsgInvalidProfileMediaFile`**; no `fmt.Errorf` wrappers); DB errors propagate unchanged. HTTP **400** + `ValidationFailed` at **`PATCH /me`** and taxonomy category create/update boundaries.

### Asset: MediaFilePublic mapping
- Name: `ToMediaFilePublicFromModel`, `ToMediaFilePublicFromEntity`, `ProfileImageFileAcceptable`
- Type: Mapper / policy
- Path: `pkg/logic/mapping/media_public_mapping.go`
- Purpose: Build **`dto.MediaFilePublic`** for API responses; shared image-kind acceptance rules.

### Asset: DelCachedUserMe
- Name: `DelCachedUserMe`
- Type: Cache helper
- Path: `services/cache/auth_user.go`
- Purpose: Invalidate Redis **`mycourse:user:me:{id}`** after profile-changing writes.
- Current Usage: `services/auth/me_update.go`, `services/auth/user_delete.go`.

### Asset: GetByURL (FileRepository)
- Name: `GetByURL`
- Type: Repository method
- Path: `repository/media/file_repository.go`
- Purpose: Find a non-deleted `media_files` row by its public URL or origin URL. Used by orphan cleanup to resolve provider/key from a plain URL string.
- Scope: Orphan cleanup and any future feature needing to look up a media row by URL.
- Dependencies: GORM, `models.MediaFile`.
- Current Usage: `internal/jobs/media/media_orphan_enqueue.go`.

## Gap Analysis (What Must Be Created Later)
- Missing reusable domain DTO/model/service packages for:
  - course shell/versioning
  - sections/lessons/content
  - quiz
  - series
  - coupons/orders/enrollments
  - progress/reviews
- Missing shared query helper namespaces beyond RBAC.
- Missing test helper assets: place all test suites (unit/module-level/integration), fixtures, and harnesses under **`tests/`** only (see `docs/patterns.md`).
- Missing generalized ownership-check helper utilities for instructor/learner/admin scopes.

## Immediate Reuse Plan by Phase Core
- Phase 01-04: reuse `BaseFilter`, `RequirePermission`, `response` helpers, `errcode`.
- Phase 05-08: additionally reuse list/sort whitelist pattern and raw SQL helper patterns.
- Phase 09-12: reuse auth/session + permission resolution functions and middleware gates; add domain-specific shared helpers where duplication appears.
