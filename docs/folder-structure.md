# Folder Structure

## Full Directory Tree

```text
be-mycourse/
├── .claude/                        # Agent skills (GitNexus, etc.)
├── .context/                       # Session continuity artifacts
├── .cursor/                        # Workspace rules, editor skills
├── .github/
│   └── workflows/                  # CI/CD (deploy-dev.yml)
├── .gitnexus/                      # GitNexus graph index
├── cmd/
│   ├── syncpermissions/            # CLI: upsert permissions from constants
│   └── syncrolepermissions/        # CLI: rebuild role_permissions from constants
├── config/                         # app.yaml + app-<STAGE>.yaml
├── docs/
│   └── modules/                    # Per-domain module documentation
├── internal/
│   ├── appcli/                     # CLI: register first system user
│   ├── auth/
│   │   ├── domain/                 # User (pure), repository interfaces, errors, token TTL
│   │   ├── application/            # AuthService, cache_keys.go, email_limits.go
│   │   ├── infra/                  # userRow, gormjsonb, user_repo, user_query, crypto, session_limits
│   │   └── delivery/               # HTTP handlers, routes, DTOs, mapping
│   ├── media/
│   │   ├── domain/                 # File, MediaGateway port, repos interfaces, bunny/meta types
│   │   ├── application/            # MediaService, service_upload_helpers.go
│   │   ├── infra/                  # storage_gateway, repos, cloud clients, metadata, webhooks
│   │   ├── delivery/               # HTTP handlers, routes, DTOs, handler_helpers via httpx
│   │   └── jobs/                   # Cleanup scheduler, OrphanEnqueuer, cleanup_constants.go
│   ├── rbac/
│   │   ├── domain/                 # Permission, Role entities
│   │   ├── application/            # RBACService
│   │   ├── infra/                  # GORM repos, sql_templates.go
│   │   └── delivery/               # handler.go, handler_helpers.go, handler_mutations.go
│   ├── system/
│   │   ├── domain/
│   │   ├── application/            # SystemService, catalog.go, roles_permission.go
│   │   ├── infra/                  # GORM repos, crypto
│   │   ├── delivery/               # HTTP handlers, routes
│   │   └── jobs/                   # RBAC permission + role-permission sync schedulers
│   ├── taxonomy/
│   │   ├── domain/
│   │   ├── application/            # TaxonomyService
│   │   ├── infra/                  # GORM repos
│   │   └── delivery/               # HTTP handlers, routes, DTOs
│   ├── shared/
│   │   ├── brevo/                  # Brevo SMTP email client + constants.go
│   │   ├── cache/                  # Redis client setup (go-redis v9)
│   │   ├── constants/              # Cross-domain constants (5 files only)
│   │   │   ├── dbschema_name.go    # PostgreSQL table/relation names
│   │   │   ├── error_msg.go        # Reusable error message strings + numeric caps
│   │   │   ├── media.go            # Media limits, multipart constants
│   │   │   ├── permissions.go      # AllPermissions catalog (permission IDs)
│   │   │   └── register_http.go    # Registration rate-limit HTTP header names
│   │   ├── db/                     # GORM setup, PostgreSQL connection, migrations runner
│   │   ├── errors/                 # ErrXXX sentinel vars, error code constants
│   │   ├── logger/                 # Uber Zap bootstrap, WithRequestID, FromContext
│   │   │   └── logger_test.go
│   │   ├── mailtmpl/               # HTML email templates + constants.go
│   │   ├── middleware/             # Gin middleware: CORS, AuthJWT, RBAC, rate limit, request logger
│   │   ├── response/               # Unified response envelope helpers
│   │   ├── setting/                # YAML config loading + env substitution
│   │   ├── gormx/                  # FirstWhere, CreateAndThen, audit timestamps, soft-delete scope
│   │   ├── timex/                  # NowUnix and nullable epoch helpers (audit columns)
│   │   ├── cryptox/                # Credential HMAC, system JWT helpers
│   │   ├── httpx/                  # Paginated list handler helper
│   │   ├── token/                  # JWT generation and validation
│   │   ├── taxonomy/               # TreeNode + tree/description validators (taxonomy JSONB)
│   │   ├── httperr/                # Gin error middleware + panic recovery
│   │   ├── parsebool/              # Loose bool parsing (env, YAML, forms)
│   │   ├── machineidentity/        # Enrollment file + OS fingerprint → hybrid binding material
│   │   ├── mediaquery/             # Shared media file-ID → public URL hydration helpers
│   │   ├── utils/                  # Generic utilities: image encode, random, fingerprint, request param helpers
│   │   │   └── webp_test.go
│   │   └── validate/               # Request validation helpers
│   └── server/
│       ├── wire.go                 # Dependency injection: constructs all Services + Handlers
│       └── router.go               # Gin router setup, route group mounting
├── migrations/                     # Versioned SQL migration files (embedded)
├── pkg/
│   └── supabase/                   # Supabase HTTP client + DB helpers (optional integration)
├── scripts/                        # Deploy helper scripts (pm2-reload-with-binary-rollback.sh)
├── go.mod
├── go.sum
└── main.go                         # Process entry point
```

---

## Purpose By Path

### Root-level files

| Path | Purpose |
|------|---------|
| `main.go` | Process entry: settings, logger, DB, Redis, migrations, wiring, background jobs, HTTP server |
| `go.mod` | Go module definition (`mycourse-io-be`, Go 1.25) |

### `internal/auth/`

| Path | Purpose |
|------|---------|
| `domain/` | Pure `User` (no GORM), `RefreshTokenSessionMap`, repository interfaces, errors, token TTLs |
| `application/` | `AuthService`: register, login, confirm, refresh, GetMe, UpdateMe, DeleteMe. `cache_keys.go`, `email_limits.go` |
| `infra/` | `userRow` + mappers, `gormjsonb.go` (JSONB scanner), `user_repo.go` (user + refresh session repos), `user_query.go`, bcrypt `crypto.go`, `session_limits.go` |
| `delivery/` | `Handler` (HTTP handlers), `routes.go`, request/response DTOs, mapping |

### `internal/media/`

| Path | Purpose |
|------|---------|
| `domain/` | `File`, `MediaGateway` port, repository interfaces, Bunny status/webhook/meta types |
| `application/` | `MediaService` (injected `MediaGateway`), `service_upload_helpers.go` |
| `infra/` | `storage_gateway.go` (implements `MediaGateway`), `repos.go`, B2/Bunny clients, metadata, webhooks, multipart open/validate |
| `delivery/` | `Handler` + `MediaGateway` for multipart/metadata; `routes.go`, DTOs, multipart bind helpers in `mapping.go` |
| `jobs/` | `OrphanEnqueuer`, cleanup scheduler, `GlobalCounters`, `cleanup_constants.go` |

### `internal/rbac/`

| Path | Purpose |
|------|---------|
| `domain/` | `Permission` and `Role` entities |
| `application/` | `RBACService`: permission CRUD, role CRUD, user-role/direct-permission bindings |
| `infra/` | GORM repos for permissions, roles, user-roles, user-permissions; `sql_templates.go` |
| `delivery/` | `handler.go`, `handler_helpers.go`, `handler_mutations.go` — RBAC admin API under `/api/internal-v1/rbac` |

### `internal/taxonomy/`

| Path | Purpose |
|------|---------|
| `domain/` | Taxonomy entities and repository interfaces |
| `application/` | `TaxonomyService`, `service_helpers.go` (shared create/update/delete) |
| `infra/` | `repos.go`, `repos_crud_helper.go`, `jsonb_types.go` |
| `delivery/` | `handler.go`, `handler_helpers.go` (shared list/mutation HTTP), `routes.go` |

### `internal/instructor/`

| Path | Purpose |
|------|---------|
| `domain/` | Applications, profiles, expertise, tickets; repository interface |
| `application/` | `InstructorService` + roster/apps/profiles/expertise/tickets; ports for RBAC, auth cache, media |
| `infra/` | `GormRepository`, rows, profile JSONB |
| `delivery/` | Split handlers (`handler_expertise_*`, `handler_ticket`, …), `routes.go`; list handlers use `internal/shared/httpx.ListPaginated` directly |

Wiring: `internal/server/wire_instructor.go`, `wire_instructor_adapters.go`, `wire_core.go`, `router.go` (`instdelivery.RegisterRoutes`).

### `internal/system/`

| Path | Purpose |
|------|---------|
| `domain/` | `SystemCrypto` port, repository interfaces |
| `application/` | `SystemService` (injected `SystemCrypto`): permission sync, role-permission sync, scheduler control, system login |
| `infra/` | GORM repos, `crypto.go` + `crypto_ports.go` adapter |
| `delivery/` | `Handler`, `routes.go` — system API under `/api/system` |
| `jobs/` | RBAC permission-sync and role-permission-sync schedulers (`sync_schedulers.go`) — ticker-driven, started/stopped via `/api/system` endpoints |

### `internal/shared/`

| Path | Purpose |
|------|---------|
| `constants/` | **Only 5 files** — cross-domain constants. All domain-specific constants live inside their own domain package |
| `db/` | `shareddb.Setup()`, `shareddb.Conn()`, `MigrateDatabase()` |
| `cache/` | `cache.SetupRedis()`, `cache.Redis` global Redis client |
| `setting/` | `setting.Setup()`, config structs (`ServerSetting`, `DatabaseSetting`, `MediaSetting`, `LogSetting`, …) |
| `logger/` | `logger.InitFromSettings()`, `logger.Sync()`, `logger.FromContext()`, `logger.WithRequestID()` |
| `middleware/` | `AuthJWT`, `RequirePermission`, `RequireInternalAPIKey`, `RequireSystemAccessToken`, `RateLimitLocal`, `RateLimitSystemIP`, `CircuitBreakerMiddleware`, `BeforeInterceptor`, `RequestLogger` |
| `response/` | `response.OK`, `response.Created`, `response.WriteByStatus`, `response.OKPaginated`, `response.Fail`, `response.AbortFail`, `response.Health` |
| `token/` | JWT sign/parse for access and refresh tokens |
| `validate/` | Validator setup, error flattening for Gin binding |
| `taxonomy/` | `TreeNode`, `ValidateTree`, `ValidateDescriptionParagraphs` |
| `httperr/` | `Middleware`, `Recovery`, `HTTPError`, `Abort` |
| `parsebool/` | `Loose`, `EnvEnabled` — env/YAML boolean strings |
| `machineidentity/` | `LoadOrCreateMachineIdentityMaterial`, `LoadMachineIdentityMaterial`, `BuildHybridMachineBindingMaterial`, `IdentityFilePath`; platform files `fingerprint_{linux,darwin,windows,other}.go` |
| `mediaquery/` | Shared avatar/media file-ID URL hydration helpers reused across bounded contexts without importing media/domain |
| `utils/` | `CurrentUserID`, `ParseUUIDParam`, `ParseUintParam`, `ParseUUIDPathParam`, `ParseUintPathParam`, `ParsePermissionIDParam`, `RoutePermission`, `EncodeWebP`, `ContentFingerprint`, `ParseBoolLoose` (delegates to `parsebool`), `SameStringSet`, `UniqueUint`, `NilIfBlank`, `NilIfZeroUint`, `NormalizeJSON` |
| `brevo/` | Brevo SMTP HTTP wrapper + `constants.go` |
| `mailtmpl/` | HTML email template rendering + `constants.go` |
| `errors/` | Sentinel `Err*` vars and error code constants |
| `ratelimit/` | Fixed-window counters: `InMemoryStore` (HTTP), `FileStore` (APPCLI) |
| `resilience/` | Global circuit breaker, DB probe, optional Redis state |

### `internal/appcli/`

CLI flows for system administration: register privileged user (`CLI_REGISTER_NEW_SYSTEM_USER=1`), obtain system JWT (`CLI_SYSTEM_LOGIN=1`), and restore legacy SQL dumps into the UUID-v7 schema (`CLI_IMPORT_LEGACY_DATA=1` + `CLI_IMPORT_LEGACY_DATA_DUMP=/absolute/path/backup-*.sql`). Import logic lives in `import_legacy_data*.go` (parse INSERT statements, remap numeric ids → UUID v7, ULID `user_code`, write `*.idmap.json` beside the dump). `cli_guard.go` enforces circuit breaker + file-backed rate limit (5 ops / 3 min) before credential prompts. Machine enrollment + OS fingerprint live in `internal/shared/machineidentity/` (imported by register/login/guard).

### `internal/server/`

| Path | Purpose |
|------|---------|
| `wire.go` | Constructs all repos, services, handlers, and cross-domain adapters |
| `router.go` | Builds Gin engine, attaches global middleware, mounts all route groups |

### `pkg/`

| Path | Purpose |
|------|---------|
| `pkg/supabase/` | Supabase client initialization and DB helpers |

### `cmd/`

| Path | Purpose |
|------|---------|
| `cmd/syncpermissions/` | Upsert `permissions` rows from `constants.AllPermissions` |
| `cmd/syncrolepermissions/` | Rebuild `role_permissions` from `constants.RolePermissions` |

### `migrations/`

Versioned SQL migration files. Applied by `golang-migrate` via `shareddb.MigrateDatabase()`.

### `config/`

`app.yaml` and stage-specific overrides (`app-dev.yaml`, `app-prod.yaml`). Values prefixed with `$` are substituted from environment variables at startup.
