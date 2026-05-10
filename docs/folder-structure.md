# Folder Structure (Root -> Deepest)


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable **domain** types must be declared in **`pkg/entities`** (no `gorm` / `database/sql`).
- GORM / JSONB **column** types for model fields: refresh-session JSONB in **`pkg/gormjsonb/auth`**, soft-delete **`DeletedAt`** alias in **`models/deleted_at.go`**.
- Do not declare new reusable/domain types inline inside logic implementation files.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Full Folder Tree
```text
be-mycourse/
├── .claude/
│   └── skills/
│       └── gitnexus/
├── .context/
├── .cursor/
│   ├── rules/
│   └── skills/
│       └── session-context-handoff/
├── docs/
├── .github/
│   └── workflows/
├── .gitnexus/
├── api/
│   ├── system/
│   └── v1/
├── cmd/
│   ├── syncpermissions/
│   └── syncrolepermissions/
├── config/
├── constants/
├── dbschema/
├── docs/
│   └── modules/
├── dto/
├── internal/
│   ├── appcli/
│   ├── appdb/
│   ├── jobs/
│   ├── rbacsync/
│   └── systemauth/
├── middleware/
├── migrations/
├── models/
├── repository/
│   ├── media/
│   └── taxonomy/
├── pkg/
│   ├── brevo/
│   ├── cache_clients/
│   ├── dbmigrate/
│   ├── entities/
│   ├── envbool/
│   ├── errors/
│   ├── errcode/
│   ├── httperr/
│   ├── logger/
│   ├── logic/
│   ├── mailtmpl/
│   ├── query/
│   ├── requestutil/
│   ├── response/
│   ├── setting/
│   ├── sqlnamed/
│   ├── supabase/
│   ├── token/
│   └── validate/
├── queues/
├── runtime/
├── scripts/
├── services/
│   ├── auth/
│   ├── rbac/
├── tests/
├── template/
│   └── html/
│       └── email/
└── tracing/
    ├── grafana/
    └── prometheus/
```

## Purpose By Folder
- `.claude/`: local agent skills and assistant automation assets.
- `.context/`: session continuity artifacts and timestamped handoff summaries.
- `.cursor/`: workspace rules, skills, and editor agent metadata.
- `docs/`: generated project discovery snapshot files.
- `.github/`: CI/CD workflows.
- `.gitnexus/`: GitNexus graph index artifacts.
- `api/`: route bootstrap and HTTP entry points.
- `api/system/`: privileged system endpoints (system login, sync-now, scheduler controls).
- `api/v1/`: main external API handlers (auth, me, internal RBAC — e.g. `internal/rbac_handler.go` + `internal/rbac_handler_user_bindings.go` for user–role/permission routes).
- `cmd/`: operational CLI commands.
- `cmd/syncpermissions/`: permission catalog sync command.
- `cmd/syncrolepermissions/`: role-permission sync command.
- `config/`: stage-specific app configuration and initialization glue.
- `constants/`: role/permission constants, domain enums, **`dbschema_name.go`** (PostgreSQL **table/relation names** — single source of truth for `dbschema` + raw SQL), **`media_meta_keys.go`** (JSON keys for Bunny parity: `video_id`, `thumbnail_url`, `embeded_html`), **`bunny_video.go`** / **`bunny_video_status.go`** (Bunny webhook literals + numeric status codes + **`FinishedWebhookBunnyStatus`**; status-string mapping lives in `pkg/media.BunnyStatusString`), **`user_session.go`** (`MaxActiveSessions` for refresh-token device cap), **`auth_token.go`** (`AccessTokenTTL`, `RefreshTokenTTL`, `RememberMeRefreshTTL`), **`register_email_limits.go`** / **`register_http.go`** (registration confirmation email limits + **429** header names), **`rbac_sql.go`** (raw RBAC SQL templates with `%s` table placeholders; **`services/rbac/rbac.go`** `init` fills them via `dbschema`), **`cache_auth.go`** (Redis key prefixes + auth cache TTLs), **`brevo.go`** / **`email_template.go`** (Brevo API URL + HTML template root dir), and **`error_msg.go`** (central error-message / sentinel strings + related limits such as media upload max bytes; **`MsgFileTooLargeUpload`** is shared with `pkg/errcode/messages.go` and `pkg/errors/upload_errors.go` — see file header).
- `dbschema/`: typed namespaces (`RBAC`, `Media`, `Taxonomy`, `System`, `AppUser`) that **return** names from `constants/dbschema_name.go` — no duplicate string literals here; use from `models` `TableName()` and services (e.g. RBAC SQL).
- `docs/`: maintained architecture/API/deploy requirements docs.
- `docs/modules/`: module-level functional docs.
- `docs-will-be-delete/`: moved out of `be-mycourse` to `../temporary-docs/docs-sample-chucnang/docs-will-be-delete/` as shared external docs storage.
- `dto/`: request/query/response transport contracts.
- `internal/`: non-public operational internals.
- `internal/appdb/`: holds the primary PostgreSQL GORM handle for callers that must not import `models` (e.g. `api/system`); set once from `main` after `models.Setup()`.
- `internal/appcli/`: protected CLI flow for system-user registration.
- `internal/jobs/`: **no** loose `*.go` at this root — **`rbac/`** (`interval_sync_loop.go`, `rbac_sync_schedulers.go`), **`media/`** (`media_pending_cleanup_scheduler.go`, …), **`system/`** (HTTP job control).
- `internal/rbacsync/`: DB synchronization logic from constants.
- `internal/systemauth/`: system access token and credential crypto primitives.
- `middleware/`: auth/authz, API-key, system-token, rate-limit, and interceptor middleware.
- `migrations/`: SQL migration files and embed bridge.
- `models/`: GORM model definitions and DB setup helpers; **`models/deleted_at.go`** holds the **`DeletedAt`** soft-delete type alias (**only** `models/*.go` file that imports **`gorm.io/gorm`** — see **`.golangci.yml`**).
- `repository/`: data access (`repository.go` aggregate, `repository/media`, `repository/taxonomy`, **`user_refresh_session.go`** for users JSONB refresh sessions).
- `pkg/`: reusable cross-cutting libraries.
- `pkg/brevo/`: email provider integration wrapper.
- `pkg/cache_clients/`: Redis client setup and lifecycle.
- `pkg/dbmigrate/`: migration runner utility.
- `pkg/entities/`: pure shared domain structs/DTO shapes reused across layers (**no** **`gorm.io/gorm`** / **`database/sql`** / **no functions** — depguard **`restrict_models_pkg_entity_schema_only`** applies here same as **`models/*.go`**).
- `pkg/gormjsonb/auth/`: Postgres JSONB **`Valuer`/`Scanner`** for **`users.refresh_token_session`** (`RefreshTokenSessionMap` — local **`sessionColumnJSONB`** + exported alias per Rule 11).
- `pkg/envbool/`: environment bool parsing helpers.
- `pkg/errors/`: central shared package for reusable functional errors/sentinel vars and typed feature structs (**only** **`Error()`** / **`Unwrap()`** on typed errors) — e.g. **`auth.go`**, **`register_limits.go`**, **`system.go`**, **`media_errors.go`**, **`upload_errors.go`**, **`provider_error.go`** (`ProviderError` body only).
- `pkg/errors_func/`: **functions only** (Rule 19), grouped by feature — **`media/`** (`AsProviderError`, `HTTPStatusForProviderCode`), **`db/`** (`MapRecordNotFound`), **`rbac/`** (`WrapRBACUnknownPermissionID`).
- `pkg/errcode/`: app error code constants and default messages.
- `pkg/httperr/`: centralized HTTP error middleware and typed errors.
- `pkg/logger/`: structured logger setup.
- `pkg/logic/`: **`mapping/`** (model ↔ entity/DTO) and **`utils/`** (generic primitives). **No `helper/` subtree** — media policy lives in **`pkg/media/`**; taxonomy status helper in **`pkg/taxonomy/`**.
- `pkg/media/`: media provider clients (`clients.go` for upload/delete/public URL HTTP, `clients_setting_attach.go` for `NewCloudClientsFromSetting` / B2+Gcore+Bunny Storage wiring, `clients_bunny_get.go` for Bunny Stream GET + JSON parse, `setup.go`, …) **and** media-domain helpers (resolver, metadata, multipart bind, orphan URL scan, local URL codec, `RequireInitialized`, …).
- `pkg/taxonomy/`: taxonomy-only shared helpers (e.g. status normalization).
- `pkg/mailtmpl/`: HTML email template rendering.
- `pkg/query/`: reusable query/filter parsing helpers for list APIs.
- `pkg/requestutil/`: shared HTTP request param/context helpers.
- `pkg/response/`: unified API response envelope helpers.
- `pkg/setting/`: environment + YAML config loading (`setting.go` + `setting_yaml_apply.go` for expand/apply into globals).
- `pkg/sqlnamed/`: named-parameter SQL helper.
- `pkg/supabase/`: Supabase client setup.
- `pkg/token/`: JWT/session token utilities.
- `pkg/validate/`: validator helpers and error flattening.
- `queues/`: queue consumer placeholder.
- `runtime/`: runtime metadata structures.
- `scripts/`: build/deploy helper scripts (including `pm2-reload-with-binary-rollback.sh` for CI — see `docs/deploy.md` Appendix C).
- `services/`: root holds at most **three** `*.go` files (**`make check-architecture`**); shared **`services.go`** aggregate; **`system.go`**. **`services/auth/`** — package **`auth`**: register/login/confirm/refresh, **`GetMe`**, token pair + rotation (`auth.go`, `register_flow.go`, `auth_session_tokens.go`, `auth_refresh_rotation.go`). **`services/rbac/`** — package **`rbac`**: permission/role SQL init + CRUD (`rbac.go`, `rbac_permissions.go`, `rbac_roles.go`, `rbac_seed.go`). **`services/media/`:** `file_service.go` + `file_service_upload.go`, etc. **`services/taxonomy/`**, **`services/cache/`** (includes **`register_email_window.go`** for registration confirmation sliding window). **File naming:** no `helper_` / `_helper` prefixes; see **`docs/patterns.md`**.
- `tests/`: **module-level / integration** Go test packages, shared harnesses, and fixtures (not production code); see `docs/patterns.md` and `README.md` (**Testing**).
- `template/`: template root folder.
- `template/html/`: HTML template grouping.
- `template/html/email/`: email templates.
- `tracing/`: observability artifact container.
- `tracing/grafana/`: Grafana placeholder/readme.
- `tracing/prometheus/`: Prometheus placeholder/readme.
