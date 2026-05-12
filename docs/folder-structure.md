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
│   │   ├── domain/                 # User entity, UserRepository interface, domain errors, token TTL constants
│   │   ├── application/            # AuthService, cache_keys.go, email_limits.go
│   │   ├── infra/                  # GormUserRepository, GormRefreshSessionRepository, crypto, session_limits.go
│   │   └── delivery/               # HTTP handlers, routes, DTOs, mapping
│   ├── media/
│   │   ├── domain/                 # File entity, repository interfaces, errors, bunny_status_codes.go, bunny_webhook.go, meta_keys.go
│   │   ├── application/            # MediaService, service_upload_helpers.go
│   │   ├── infra/                  # GORM repos, B2/Bunny cloud clients, metadata parsers, webhook validation
│   │   ├── delivery/               # HTTP handlers, routes, DTOs
│   │   └── jobs/                   # Cleanup scheduler, OrphanEnqueuer, cleanup_constants.go
│   ├── rbac/
│   │   ├── domain/                 # Permission entity, Role entity
│   │   ├── application/            # RBACService
│   │   ├── infra/                  # GORM repos, sql_templates.go
│   │   └── delivery/               # HTTP handlers, routes
│   ├── system/
│   │   ├── domain/
│   │   ├── application/            # SystemService, catalog.go, roles_permission.go
│   │   ├── infra/                  # GORM repos, crypto
│   │   ├── delivery/               # HTTP handlers, routes
│   │   └── jobs/                   # (scheduler placeholders)
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
│   │   ├── token/                  # JWT generation and validation
│   │   ├── utils/                  # Generic utilities: image encode, random, fingerprint
│   │   │   └── webp_test.go
│   │   └── validate/               # Request validation helpers
│   └── server/
│       ├── wire.go                 # Dependency injection: constructs all Services + Handlers
│       └── router.go               # Gin router setup, route group mounting
├── migrations/                     # Versioned SQL migration files (embedded)
├── pkg/
│   ├── envbool/                    # Environment bool parsing (true/1/yes/on)
│   ├── httperr/                    # Gin error middleware, panic recovery
│   └── supabase/                   # Supabase HTTP client + DB helpers
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
| `domain/` | `User` entity, `UserRepository` and `RefreshSessionRepository` interfaces, domain errors, `AccessTokenTTL` / `RefreshTokenTTL` |
| `application/` | `AuthService`: register, login, confirm, refresh, GetMe, UpdateMe, DeleteMe. `cache_keys.go`, `email_limits.go` |
| `infra/` | `GormUserRepository`, `GormRefreshSessionRepository`, bcrypt hashing, session limit constants |
| `delivery/` | `Handler` (HTTP handlers), `routes.go`, request/response DTOs, mapping |

### `internal/media/`

| Path | Purpose |
|------|---------|
| `domain/` | `File` entity, `FileRepository` and `PendingCleanupRepository` interfaces, domain errors, Bunny status codes, webhook types, metadata key constants |
| `application/` | `MediaService`: create/update/delete/list files, batch delete, video status, Bunny webhook handling |
| `infra/` | `GormFileRepository`, `GormPendingCleanupRepository`, B2/BunnyCDN SDK clients, metadata parsing, WebP encoding |
| `delivery/` | `Handler` (HTTP handlers), `routes.go`, `RegisterWebhookRoutes`, DTOs |
| `jobs/` | `OrphanEnqueuer`, cleanup scheduler, `GlobalCounters`, `cleanup_constants.go` |

### `internal/rbac/`

| Path | Purpose |
|------|---------|
| `domain/` | `Permission` and `Role` entities |
| `application/` | `RBACService`: permission CRUD, role CRUD, user-role/direct-permission bindings |
| `infra/` | GORM repos for permissions, roles, user-roles, user-permissions; `sql_templates.go` |
| `delivery/` | `Handler`, `routes.go` — RBAC admin API under `/api/internal-v1/rbac` |

### `internal/taxonomy/`

| Path | Purpose |
|------|---------|
| `domain/` | Taxonomy entity types |
| `application/` | `TaxonomyService`: categories, tags, course levels |
| `infra/` | GORM repos with shared list query helpers |
| `delivery/` | `Handler`, `routes.go` — taxonomy CRUD under `/api/v1/taxonomy` |

### `internal/system/`

| Path | Purpose |
|------|---------|
| `application/` | `SystemService`: permission sync, role-permission sync, scheduler control, system login |
| `infra/` | GORM repos (`AppConfig`, `PrivilegedUser`), `PermissionSyncer`, `RolePermissionSyncer`, crypto |
| `delivery/` | `Handler`, `routes.go` — system API under `/api/system` |

### `internal/shared/`

| Path | Purpose |
|------|---------|
| `constants/` | **Only 5 files** — cross-domain constants. All domain-specific constants live inside their own domain package |
| `db/` | `shareddb.Setup()`, `shareddb.Conn()`, `MigrateDatabase()` |
| `cache/` | `cache.SetupRedis()`, `cache.Redis` global Redis client |
| `setting/` | `setting.Setup()`, config structs (`ServerSetting`, `DatabaseSetting`, `MediaSetting`, `LogSetting`, …) |
| `logger/` | `logger.InitFromSettings()`, `logger.Sync()`, `logger.FromContext()`, `logger.WithRequestID()` |
| `middleware/` | `AuthJWT`, `RequirePermission`, `RequireInternalAPIKey`, `RequireSystemAccessToken`, `RateLimitLocal`, `RateLimitSystemIP`, `BeforeInterceptor`, `RequestLogger` |
| `response/` | `response.OK`, `response.Created`, `response.OKPaginated`, `response.Fail`, `response.AbortFail`, `response.Health` |
| `token/` | JWT sign/parse for access and refresh tokens |
| `validate/` | Validator setup, error flattening for Gin binding |
| `utils/` | `EncodeWebP`, `ContentFingerprint`, random helpers |
| `brevo/` | Brevo SMTP HTTP wrapper + `constants.go` |
| `mailtmpl/` | HTML email template rendering + `constants.go` |
| `errors/` | Sentinel `Err*` vars and error code constants |

### `internal/server/`

| Path | Purpose |
|------|---------|
| `wire.go` | Constructs all repos, services, handlers, and cross-domain adapters |
| `router.go` | Builds Gin engine, attaches global middleware, mounts all route groups |

### `internal/appcli/`

CLI flow for registering the first system user (`CLI_REGISTER_NEW_SYSTEM_USER=1`).

### `pkg/`

| Path | Purpose |
|------|---------|
| `pkg/envbool/` | Parse environment booleans (`true`, `1`, `yes`, `on`, …) |
| `pkg/httperr/` | Gin middleware for centralized error handling and panic recovery |
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
