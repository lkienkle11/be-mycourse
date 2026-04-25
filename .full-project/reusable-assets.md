# Reusable Assets Inventory (Deep Scan)

## Function / Method Assets

### Asset: PermissionCodesForUser
- Name: `PermissionCodesForUser`
- Type: Function (service)
- Path: `services/rbac.go`
- Purpose: Resolve effective permission names from role grants + direct grants.
- Scope: All authorization-sensitive APIs and token issuance.
- Dependencies: `rbacDB`, `pkg/sqlnamed`, RBAC SQL templates.
- Current Usage: `services/auth.go`, `api/v1/me.go`, `api/v1/internal/rbac_handler.go`, `UserHasAllPermissions`.
- Reuse Opportunity:
  - Reuse for all new CRUD permission checks and permission projection in future domains.

### Asset: UserHasAllPermissions
- Name: `UserHasAllPermissions`
- Type: Function (service guard)
- Path: `services/rbac.go`
- Purpose: Verify required permission set for a user.
- Scope: Authorization middleware fallback and potential service guardrails.
- Dependencies: `PermissionCodesForUser`.
- Current Usage: `middleware/rbac.go`.
- Reuse Opportunity:
  - Direct reuse for `resource:action` checks on new CRUD endpoints.

### Asset: issueTokenPair / RefreshSession
- Name: `issueTokenPair`, `RefreshSession`
- Type: Function (auth service)
- Path: `services/auth.go`
- Purpose: Token issue/rotation and session persistence management.
- Scope: Any auth/session extension features.
- Dependencies: `pkg/token`, `models`, `services/cache`, RBAC permission resolver.
- Current Usage: auth register/login/confirm/refresh flows.
- Reuse Opportunity:
  - Reuse unchanged for newly protected domain APIs.

### Asset: ListPermissions
- Name: `ListPermissions`
- Type: Function (service list pattern)
- Path: `services/rbac.go`
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
- Current Usage: `services/auth.go`, `services/cache/auth_user.go`, `api/v1/me.go`.
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
- Current Usage: taxonomy repositories (`course_level`, `category`, `tag`).
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
- Path: `pkg/logic/helper/permission.go`
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
- Path: `pkg/logic/helper/local_url_codec.go`
- Purpose: Build reversible signed URL tokens for local provider objects.
- Scope: Media local-provider read path and future private file links.
- Dependencies: `crypto/hmac`, `crypto/sha256`, `encoding/base64`.
- Current Usage: `pkg/media/clients.go`, `services/media/file_service.go`.
- Reuse Opportunity:
  - Reuse for secure temporary download tokens in other modules.

### Asset: Media kind/provider resolvers
- Name: `ResolveMediaKind`, `ResolveMediaProvider`
- Type: Util/Helper
- Path: `pkg/logic/helper/media_resolver.go`
- Purpose: Normalize upload kind/provider with consistent fallback rules (video -> Bunny, file -> B2).
- Scope: Media service orchestration and any future upload entrypoint requiring identical fallback behavior.
- Dependencies: `constants/media.go`, `path/filepath`, `strings`.
- Current Usage: `services/media/file_service.go`.
- Reuse Opportunity:
  - Reuse for any future media ingestion endpoints to keep provider-kind resolution behavior identical.

### Asset: Media metadata parser helpers
- Name: `ParseMetadataJSON`, `ParseMetadataFromRaw`, `NormalizeMetadata`
- Type: Util/Helper
- Path: `pkg/logic/helper/media_metadata.go`
- Purpose: Parse raw metadata JSON and normalize metadata payload consistently with shared error formatting.
- Scope: Media handlers/services and any upload endpoint accepting metadata JSON.
- Dependencies: `encoding/json`, `fmt`, `strings`, `pkg/entities`.
- Current Usage: `api/v1/media/file_handler.go`.
- Reuse Opportunity:
  - Reuse for all future endpoints that accept metadata in raw string form to avoid duplicate parsing logic in services.

### Asset: Taxonomy status normalization
- Name: `NormalizeTaxonomyStatus`
- Type: Util/Helper
- Path: `pkg/logic/helper/taxonomy_status.go`
- Purpose: Map request status strings to `constants.TaxonomyStatus` with a single default-to-active rule.
- Scope: Taxonomy create/update flows and any future domain using the same enum.
- Dependencies: `strings`, `constants/taxonomy.go`.
- Current Usage: `services/taxonomy/category_service.go`, `services/taxonomy/course_level_service.go`, `services/taxonomy/tag_service.go`.
- Reuse Opportunity:
  - Reuse whenever another module accepts taxonomy status as raw text.

### Asset: sqlnamed.Postgres
- Name: `Postgres`
- Type: Util/Helper
- Path: `pkg/sqlnamed/postgres.go`
- Purpose: Named SQL to PostgreSQL positional parameter conversion.
- Scope: Complex raw SQL operations.
- Dependencies: `github.com/jmoiron/sqlx`.
- Current Usage: `services/rbac.go`.
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
- Current Usage: `services/auth.go`.
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
- Name: `File`, `FileMetadata`
- Type: Type/Entity
- Path: `pkg/entities/file.go`
- Purpose: Shared media response descriptor for stateless cloud upload API.
- Scope: Media upload service + transport response payload.
- Dependencies: `constants/media.go`.
- Current Usage: `services/media/*`, `api/v1/media/*`.
- Reuse Opportunity:
  - Reuse for course lesson media, subtitles, and future asset libraries.

### Asset: Runtime dependency guard
- Name: `RequireInitialized`, `ErrDependencyNotConfigured`
- Type: Util/Helper
- Path: `pkg/logic/helper/runtime_guard.go`
- Purpose: Centralize runtime dependency checks (non-nil guards) with standardized error message mapped from `pkg/errcode`.
- Scope: Services that rely on startup-initialized dependencies.
- Dependencies: `pkg/errcode`.
- Current Usage: `services/media/file_service.go`.
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
- Path: `services/rbac.go`
- Purpose: Efficient permission resolution and FK-safe deletions.
- Scope: RBAC read/write internals.
- Dependencies: `dbschema.RBAC`, `pkg/sqlnamed`.
- Current Usage: multiple RBAC service methods.
- Reuse Opportunity:
  - Reuse pattern for advanced domain queries requiring controlled SQL.

### Asset: DB schema namespace
- Name: `dbschema.RBAC.*`
- Type: Query helper
- Path: `dbschema/rbac.go`
- Purpose: Centralized table names for SQL safety and consistency.
- Scope: RBAC services and sync internals.
- Dependencies: none.
- Current Usage: `services/rbac.go`, `models/rbac.go`, `internal/rbacsync/*`.
- Reuse Opportunity:
  - Create equivalent namespaces per new module (`dbschema/course`, etc.) as new reusable assets.

## Gap Analysis (What Must Be Created Later)
- Missing reusable domain DTO/model/service packages for:
  - course shell/versioning
  - sections/lessons/content
  - quiz
  - series
  - coupons/orders/enrollments
  - progress/reviews
- Missing shared query helper namespaces beyond RBAC.
- Missing test helper assets (`*_test.go`, fixtures, integration test harness).
- Missing generalized ownership-check helper utilities for instructor/learner/admin scopes.

## Immediate Reuse Plan by Phase Core
- Phase 01-04: reuse `BaseFilter`, `RequirePermission`, `response` helpers, `errcode`.
- Phase 05-08: additionally reuse list/sort whitelist pattern and raw SQL helper patterns.
- Phase 09-12: reuse auth/session + permission resolution functions and middleware gates; add domain-specific shared helpers where duplication appears.
