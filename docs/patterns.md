# Patterns and Conventions


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable **domain** types must be declared in **`pkg/entities`** (no `gorm` / `database/sql` — **`restrict_models_pkg_entity_schema_only`**).
- GORM / JSONB **column** types for `models/*.go` fields belong in **`pkg/sqlmodel`** (e.g. refresh session map, `DeletedAt` alias).
- Do not declare new reusable/domain types inline inside logic implementation files.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Architectural Patterns
- Layered monolith: `api` -> `services` -> `models`.
- Cross-cutting concerns isolated in `middleware` and `pkg/*`.
- RBAC policy-as-code through constants + DB sync.
- Shared **error/sentinel message strings** (and small related numeric caps) belong in **`constants/error_msg.go`** — see file header; numeric JSON codes stay in `pkg/errcode`. If the same sentence is both API default `message` and `errors.New` text, **`pkg/errcode/messages.go` must use the constant** from `error_msg.go` (example: `MsgFileTooLargeUpload` ↔ `FileTooLarge`).
- **Centralized functional errors:** all reusable functional/sentinel errors (`var Err...`) and typed domain errors (`struct ...Error`) must be placed in **`pkg/errors`**. Domain modules (including `services/*`, `repository/*`, `pkg/media/*`) must import from `pkg/errors` instead of declaring duplicate error vars/types.

## Error / ErrorCode Convention (Mandatory)
- Reusable/sentinel/typed errors must be defined in `pkg/errors`.
- Message strings must be centralized in `constants` and referenced by `pkg/errcode/messages.go` (do not duplicate literal text).
- Every business/functional error must have a corresponding numeric code in `pkg/errcode/codes.go`.
- Never set error code/message directly inside `services/*`, `repository/*`, or other feature modules. Those layers only return/propagate errors from `pkg/errors` and map via `pkg/errcode`.

## Constants Placement Convention (Mandatory)
- All constants must be declared under `constants/*` (messages, status, limits, parameters, default values, error-related constants, and any other shared constant).
- Do not declare module-level business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, or other feature folders.
- If a new constant is needed, create/extend the appropriate file in `constants/` and import it where used.
- Keep `pkg/errcode/codes.go` and `pkg/errcode/messages.go` as mapping tables only; source message literals and shared numeric/enum constants must come from `constants/*` whenever applicable.

### Error Implementation Examples (for AI agents)
- **Typed provider error pattern**: follow `pkg/errors/provider_error.go` (`ProviderError`, `AsProviderError`, `HTTPStatusForProviderCode`) for upstream/provider integrations that need stable `code` mapping.
- **Sentinel upload error pattern**: follow `pkg/errors/upload_errors.go` where sentinel `Err...` uses shared message constant from `constants/error_msg.go`, and handler maps it to code in `pkg/errcode`.
- **Auth session sentinels** (`ErrEmailAlreadyExists`, `ErrInvalidCredentials`, …): live in **`pkg/errors/auth.go`** with **`constants.MsgAuth*`**; **`services/auth`** returns those same values; handlers use **`errors.Is(err, pkgerrors.Err…)`** (see `api/v1/auth.go`, `api/v1/me.go`).
- **Profile / taxonomy image FK:** **`pkg/errors/attachment.go`** (`ErrInvalidProfileMediaFile`) uses **`constants.MsgInvalidProfileMediaFile`**; **`services/media/profile_media_validate.go`** returns that sentinel; taxonomy + **`PATCH /me`** handlers map with **`errors.Is`** (see `api/v1/taxonomy/handlers_common.go`, `api/v1/me.go`).
- Minimal checklist when adding new errors:
  1. Add message constant in `constants/error_msg.go`.
  2. Add numeric code in `pkg/errcode/codes.go`.
  3. Wire default message in `pkg/errcode/messages.go` using that constant.
  4. Declare sentinel/typed error in `pkg/errors/*`.
  5. In handler boundary, map error -> `errcode` (never hardcode in service/repository).

## API Patterns
- Standardized response envelope via `pkg/response`.
- DTO-centric bind/validate in handlers (`dto/*`).
- List/filter APIs should rely on `dto.BaseFilter`.

## Security Patterns
- JWT-based auth with permission projection in claims.
- Route-group-level security boundaries:
  - public/authenticated/internal/system.
- Permission middleware checks by `resource:action` names.

## Data Access Patterns
- GORM as primary ORM.
- Raw SQL only when needed (RBAC joins/deletes), with named-param helper (`pkg/sqlnamed`).
- Embedded SQL migrations using `golang-migrate`.
- **PostgreSQL table (relation) names in Go:** keep string literals only in **`constants/dbschema_name.go`**. Call sites use **`dbschema`** accessors — e.g. `dbschema.RBAC.Permissions()`, `dbschema.Media.Files()`, `dbschema.Taxonomy.Tags()`, `dbschema.System.AppConfig()`, `dbschema.AppUser.Table()` — in GORM `TableName()`, dynamic `fmt.Sprintf` SQL, and service templates. Do **not** scatter hardcoded table names in `models/*`, `dbschema/*`, or `services/*`. **`constants` must not import `dbschema`** (prevents import cycles; `dbschema` imports `constants`).

## Domain packages vs `pkg/logic/utils` (Mandatory)

- **`pkg/logic/utils/*`:** generic, cross-feature primitives (parsing, fingerprints, path params, URL joins) without domain coupling.
- **`pkg/media/*`:** media domain — provider HTTP/SDK (`clients.go`, `setup.go`, …) **and** the same-tree policy/helpers previously under `pkg/logic/helper`: resolver/metadata/multipart/upload keys/orphan URL scan/local URL codec/Bunny status enum/runtime guard (`RequireInitialized`). Do **not** duplicate Bunny `video_id` / thumbnail / embed policy outside `pkg/media` (see `media_resolver.go`).
- **`pkg/taxonomy/*`:** small taxonomy-only helpers (e.g. `NormalizeTaxonomyStatus` in `pkg/taxonomy/status.go`).
- **`pkg/requestutil/*`:** transport-adjacent parsing shared by handlers (e.g. `ParsePermissionIDParam` in `params.go`).
- **`pkg/logic/mapping/*`:** model ↔ entity / DTO mapping only.
- Import alias consistency: packages that import `pkg/logic/utils` must use the **`utils.*`** alias (never `util.*`) to avoid undefined-symbol mistakes.

### Documentation-only requests (Mandatory)

If the user asks for **documentation only** (docs, markdown, OpenAPI):

- Update files under **`docs/`** and **`docs/api_swagger.yaml`** as needed.
- **Do not** change production **`*.go`** or **`tests/*.go`** unless the same request explicitly asks for code.

### Full documentation sync when API or schema changes (Mandatory)

When public JSON, DTOs, DB migrations, or persistence columns change for a documented feature, **update every maintained doc that references that feature**, not only `docs/modules/media.md`. Minimum checklist:

1. `docs/modules/<domain>.md` — behaviour, migrations, field tables, code pointers.
2. `docs/return_types.md` — example JSON / tables.
3. `docs/api_swagger.yaml` — `components.schemas` and path response refs if used.
4. `docs/reusable-assets.md` — reusable assets (media, taxonomy, utils, errors) and usage.
5. `docs/data-flow.md` — request/persistence flow bullets.
6. `docs/api-overview.md` — route inventory / one-line contract.
7. `docs/modules.md`, `README.md`, `docs/architecture.md` — module map and directory table.
8. `docs/database.md` — table and/or migration history.
9. `migrations/README.md` — version row for new SQL files.
10. `docs/curl_api.md` — cURL sections and webhook notes where applicable.
11. `docs/requirements.md` — FR bullets if behaviour is normative.
12. `IMPLEMENTATION_PLAN_EXECUTION.md` (repo root) — execution / audit trail when the team uses it.
13. `docs/router.md`, `docs/deploy.md` — only when routing or proxy behaviour for that API changes.

## Services layer file naming (Mandatory)

- Under `services/` (including subpackages such as `services/media/`), **do not** name a `.go` source file with a **filename** prefix `helper_` or suffix `_helper` (examples to avoid: `helper_orphan.go`, `orphan_helper.go`).
- **Why:** those patterns read as misplaced “helper” packages instead of the **service** layer.
- Put reusable non-orchestration logic in **`pkg/media`**, **`pkg/taxonomy`**, **`pkg/logic/utils`**, or **`pkg/requestutil`** as appropriate; keep `services/` filenames aligned with domain or capability (e.g. `orphan_cleanup.go`, `file_service.go`).

## Type Placement Convention
- From now on, for every new code written in any module under `pkg/*` that contains logic handling, all newly introduced reusable types must be created in `pkg/entities` (new file/module or existing entity module), not inline in that logic package file.
- Do not declare new reusable types inside logic-orchestration layers such as `repository/*`, `services/*`, or ad-hoc feature logic files.
- All shared/new types must be defined in **`pkg/entities`** (new module file or existing module file) and referenced from other layers.
- Exception for tests: any type created for test scope can be declared directly inside files under `tests/` when it is only used by those tests.
- Repository exception remains limited: only the repository initialization type may be declared directly in `repository/*`; other reusable/shared types still belong to `pkg/entities`.

## Tests directory (`tests/`)
- **Module-level tests** (integration or black-box packages that import `mycourse-io-be`, shared test harnesses, fixtures used across features, or any test code intentionally kept out of production packages) **belong under repository root `tests/`** (see `tests/README.md`).
- Prefer **`tests/`** when a test suite spans multiple packages or mirrors a user-facing “module” of the API rather than a single `.go` file.
- **All tests** must live under repository root `tests/`; do not add colocated `*_test.go` files next to production implementation files.

## Linting and static analysis (Mandatory baseline)

- **`make check-architecture`** (see root **`Makefile`**) enforces **≤ 3** `*.go` files directly under **`services/`** and **`repository/`**, and requires each business subfolder name to match **`^[a-z0-9_]+$`** with at least one `.go` file — keeps the service layer split by domain (`services/auth/`, `services/rbac/`, `services/media/`, …).
- **`golangci-lint`** is configured at repo root via `.golangci.yml`. **depguard** must declare explicit `rules.main.allow` entries (`$gostd`, `mycourse-io-be`, `github.com`, `golang.org`, `gorm.io`, `google.golang.org`, `gopkg.in`, `go.uber.org`, `go.mongodb.org`, …); without this, depguard’s default “Main” list blocks normal internal and third-party imports.
- **depguard layer rules** use the real Go import path **`mycourse-io-be/repository`** (singular `repository/`, not `repositories/`). The repository-layer rule is named **`restrict_repository`** with file glob **`**/repository/**/*.go`** — keep these aligned if folders are renamed.
- **`restrict_api`** (files `**/api/**/*.go`) blocks **`mycourse-io-be/models`**, **`mycourse-io-be/repository`**, **`gorm.io/gorm`**, and **`database/sql`** so HTTP handlers stay transport-only. **`pkg/errors.ErrNotFound`** is the stable “missing row” sentinel for `errors.Is` in handlers; **`pkg/errors.MapRecordNotFound`** maps `gorm.ErrRecordNotFound` in services and taxonomy repository helpers. **`api/system`** reads the primary DB via **`internal/appdb.Conn()`** (set from `main` right after `models.Setup()`). Taxonomy **category** HTTP handlers use **`dto.CategoryResponse`** only — **`services/taxonomy/category_service.go`** performs **`models.Category`** → DTO mapping (`pkg/logic/mapping`) so **`api/v1/taxonomy/`** never imports **`models`**.
- **gocritic ruleguard** (`rules.go`): package-level **`const (`** blocks under `services/`, `repository/`, `models/`, `dbschema/`, `api/`, `pkg/*` should move to **`constants/`**; the matcher **excludes** **`pkg/errcode`** only (numeric `codes.go`). Example: **`BunnyVideoStatus`** + **`StatusString()`** live in **`constants/bunny_video_status.go`**; **`pkg/media/bunny_video_status.go`** keeps a **type alias** so call sites can still use **`pkg/media.BunnyVideoStatus`** where convenient.
- **`restrict_models_schema_only`** excludes only **`models/setup.go`** (DB bootstrap). **`models/*.go`** (including **`models/user.go`**) stay schema-only: no **`gorm.io/gorm`** or **`database/sql`** imports. Depguard **`restrict_models_pkg_entity_schema_only`** applies the **same** `gorm` / `database/sql` ban to **`pkg/entities/*.go`**, so **`pkg/entities`** holds **pure domain** shapes only. Column-edge types (**`RefreshTokenSessionMap`**, **`RefreshSessionEntry`**, **`DeletedAt`**) live in **`pkg/sqlmodel`** (JSONB `Valuer`/`Scanner` + GORM soft-delete alias). Depguard **`deny_import_pkg_types`** forbids **`mycourse-io-be/pkg/types`** — use **`pkg/entities`** + **`pkg/sqlmodel`** as above. Session JSONB writes (**`SaveRefreshSession`**, **`AddRefreshSession`**) live in **`repository/user_refresh_session.go`**; the concurrent-session cap is **`constants.MaxActiveSessions`**. JWT TTLs live in **`constants/auth_token.go`**. Raw RBAC SQL templates live in **`constants/rbac_sql.go`**. Refresh rotation entrypoints: **`services/auth_refresh_rotation.go`** vs **`services/auth/auth.go`** (**`revive` `file-length-limit`**).
- **revive `file-length-limit`:** max **300** logical lines per `.go` file (see `.golangci.yml`: skips blank lines and comments). When a file grows past the limit, split by **cohesive concern** in the same package (examples: `pkg/media/clients_bunny_get.go` for Bunny Stream GET/parse, `pkg/media/clients_setting_attach.go` for B2/Gcore/Bunny **Storage** wiring from `setting.MediaSetting` (`NewCloudClientsFromSetting` + `attach*` helpers), `pkg/setting/setting_yaml_apply.go` for YAML expand/apply, `services/rbac/rbac_permissions.go` / `services/rbac/rbac_roles.go`, `services/media/file_service_upload.go`, `api/v1/internal/rbac_handler_user_bindings.go`).
- **funlen / gocyclo:** prefer **unexported helpers** in the same package instead of “god” functions — same behaviour and signatures at boundaries, smaller units for review. **funlen** is currently **30 lines / 25 statements**; shrink helpers when you touch a long function. Examples in-tree: Bunny GET split (`bunnyStreamAuthorizedGET`, `parseBunnyVideoGetResponse`, `decodeBunnyVideoDetailBody` in `pkg/media/clients_bunny_get.go`), upload entity composition (`newFileEntityUploadCore`, `attachStreamFieldsToFile` in `pkg/media/media_upload_entity.go`), typed metadata (`buildVideoTypedMetadata` / `buildImageTypedMetadata` in `pkg/media/media_metadata.go`), auth flows (`registerAssertEmailAvailable`, `completeLoginSuccess`, `refreshLoadUserAndEntry` in `services/auth/auth.go`), multipart normalisation (`prepareCreateMultipartBody` in `services/media/file_service.go`; `normalizeUpdateMultipartPayload` + `runUpdateFileMultipartBody` in `services/media/file_service_upload.go`), finished-webhook metadata (`patchBunnyWebhookMetadataJSON`, `applyBunnyFinishedWebhookToRow` in `services/media/video_service.go`), orphan URL branches (`orphanCleanupBunnyMatch`, `orphanCleanupB2Match` in `pkg/media/media_url_orphan.go`), permission row updates (`rbacApplyRenamedPermissionID`, `rbacApplyPermissionNameUpdate` in `services/rbac_permissions.go`).
- **Sentinel / `errors.New` text** must satisfy **staticcheck ST1005** (lowercase first letter, no trailing punctuation quirks). User-visible HTTP default messages still come from `pkg/errcode/messages.go` and **`constants/error_msg.go`** — keep those literals aligned when the same string backs both API `message` and a sentinel `Err…`.
- **`dto.BaseFilter` read-only helpers** (`GetPage`, `GetPerPage`, `GetOffset`, `GetSortOrder`, `HasSearch`, `HasSort`) use **value receivers** so embedded list DTOs (e.g. `CategoryFilter`) implement `dto.TaxonomyListFilter` for shared taxonomy list binding in `api/v1/taxonomy/handlers_common.go`.

## Operational Patterns
- Sync CLI commands for RBAC catalogs.
- Optional in-memory periodic jobs for sync (`internal/jobs/rbac_sync_schedulers.go` + `interval_sync_loop.go`).
- Redis cache-aside for auth/me pathways.
