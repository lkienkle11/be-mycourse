# Folder Structure (Root -> Deepest)


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

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
├── .full-project/
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
│   ├── jobs/
│   ├── rbacsync/
│   └── systemauth/
├── middleware/
├── migrations/
├── models/
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
- `.full-project/`: generated project discovery snapshot files.
- `.github/`: CI/CD workflows.
- `.gitnexus/`: GitNexus graph index artifacts.
- `api/`: route bootstrap and HTTP entry points.
- `api/system/`: privileged system endpoints (system login, sync-now, scheduler controls).
- `api/v1/`: main external API handlers (auth, me, internal RBAC).
- `cmd/`: operational CLI commands.
- `cmd/syncpermissions/`: permission catalog sync command.
- `cmd/syncrolepermissions/`: role-permission sync command.
- `config/`: stage-specific app configuration and initialization glue.
- `constants/`: role/permission constants, domain enums, and **`error_msg.go`** (central error-message / sentinel strings + related limits such as media upload max bytes; **`MsgFileTooLargeUpload`** is shared with `pkg/errcode/messages.go` and `pkg/errors/upload_errors.go` — see file header).
- `dbschema/`: shared DB table-name namespace helpers.
- `docs/`: maintained architecture/API/deploy requirements docs.
- `docs/modules/`: module-level functional docs.
- `docs-will-be-delete/`: moved out of `be-mycourse` to `../temporary-docs/docs-sample-chucnang/docs-will-be-delete/` as shared external docs storage.
- `dto/`: request/query/response transport contracts.
- `internal/`: non-public operational internals.
- `internal/appcli/`: protected CLI flow for system-user registration.
- `internal/jobs/`: in-memory scheduler loops for periodic sync.
- `internal/rbacsync/`: DB synchronization logic from constants.
- `internal/systemauth/`: system access token and credential crypto primitives.
- `middleware/`: auth/authz, API-key, system-token, rate-limit, and interceptor middleware.
- `migrations/`: SQL migration files and embed bridge.
- `models/`: GORM model definitions and DB setup helpers.
- `pkg/`: reusable cross-cutting libraries.
- `pkg/brevo/`: email provider integration wrapper.
- `pkg/cache_clients/`: Redis client setup and lifecycle.
- `pkg/dbmigrate/`: migration runner utility.
- `pkg/entities/`: pure shared entity structs reused across layers.
- `pkg/envbool/`: environment bool parsing helpers.
- `pkg/errors/`: central shared package for reusable functional errors/sentinel vars and typed feature error structs.
- `pkg/errcode/`: app error code constants and default messages.
- `pkg/httperr/`: centralized HTTP error middleware and typed errors.
- `pkg/logger/`: structured logger setup.
- `pkg/logic/`: shared domain-agnostic helper logic grouped by concern.
- `pkg/mailtmpl/`: HTML email template rendering.
- `pkg/query/`: reusable query/filter parsing helpers for list APIs.
- `pkg/requestutil/`: shared HTTP request param/context helpers.
- `pkg/response/`: unified API response envelope helpers.
- `pkg/setting/`: environment + YAML config loading.
- `pkg/sqlnamed/`: named-parameter SQL helper.
- `pkg/supabase/`: Supabase client setup.
- `pkg/token/`: JWT/session token utilities.
- `pkg/validate/`: validator helpers and error flattening.
- `queues/`: queue consumer placeholder.
- `runtime/`: runtime metadata structures.
- `scripts/`: build/deploy helper scripts.
- `services/`: core business logic and orchestration.
- `tests/`: **module-level / integration** Go test packages, shared harnesses, and fixtures (not production code); see `.full-project/patterns.md` and `README.md` (**Testing**).
- `template/`: template root folder.
- `template/html/`: HTML template grouping.
- `template/html/email/`: email templates.
- `tracing/`: observability artifact container.
- `tracing/grafana/`: Grafana placeholder/readme.
- `tracing/prometheus/`: Prometheus placeholder/readme.
