# Dependencies

## Core Runtime

| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.12.0 | HTTP server and router |
| `gorm.io/gorm` | v1.31.1 | ORM for PostgreSQL |
| `gorm.io/driver/postgres` | v1.6.0 | GORM PostgreSQL driver (via pgx) |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | JWT sign and validation (HS256) |
| `github.com/go-playground/validator/v10` | v10.30.1 | Request struct validation |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis client (cache for auth/me) |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | SQL migration runner |
| `go.uber.org/zap` | v1.27.1 | Structured logging |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config file parsing |
| `golang.org/x/crypto` | v0.49.0 | bcrypt password hashing |

## Cloud / Media

| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/Backblaze/blazer` | v0.7.2 | Backblaze B2 SDK (file upload/delete) |
| `github.com/l0wl3vel/bunny-storage-go-sdk` | v1.0.0 | BunnyCDN Storage SDK |
| `github.com/G-Core/gcorelabscdn-go` | v1.0.37 | Gcore CDN API client |
| `github.com/h2non/bimg` | v1.1.9 | WebP encoding via libvips (CGO) |
| `golang.org/x/image` | v0.39.0 | WebP decoder for `image.DecodeConfig` |

## Email / Auth

| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/supabase-community/supabase-go` | v0.0.4 | Supabase auth client |
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading |
| `github.com/google/uuid` | v1.6.0 | UUID generation (session IDs) |

## HTTP Middleware

| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-contrib/cors` | v1.7.6 | CORS middleware for Gin |
| `github.com/gin-contrib/gzip` | v1.2.5 | gzip compression middleware for Gin |

## Build / Misc

| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/jmoiron/sqlx` | v1.4.0 | Extended SQL helpers (where needed beyond GORM) |
| `golang.org/x/term` | v0.42.0 | TTY detection for CLI prompts |

---

## Internal Dependency Architecture

The project follows DDD with strict layer boundaries. No circular imports are allowed.

```
cmd/*             → internal/shared/*, internal/rbac/*, internal/system/*
main.go           → internal/server/*, internal/shared/*, internal/media/*, pkg/*
internal/server/  → all internal/<domain>/* (composition root)
internal/<domain>/delivery/     → internal/<domain>/application/, internal/shared/*
internal/<domain>/application/  → internal/<domain>/domain/, internal/<domain>/infra/, internal/shared/*
internal/<domain>/infra/        → internal/<domain>/domain/, internal/shared/*
internal/<domain>/domain/       → (no internal imports)
internal/shared/middleware/     → internal/shared/response, internal/shared/token, internal/shared/errors
pkg/*             → (no internal imports — standalone utilities)
```

Cross-domain dependencies (e.g. Auth calling RBAC) use **interface adapters** defined at the consuming domain and wired in `internal/server/wire.go`.

---

## Key Internal Packages

| Package | Role |
|---------|------|
| `internal/shared/response` | Standard JSON envelope — all handlers use this |
| `internal/shared/errors` | Shared sentinel errors and error codes |
| `internal/shared/middleware` | Auth JWT, permission check, rate limit, request logger |
| `internal/shared/token` | JWT generation and validation |
| `internal/shared/setting` | YAML + env config loading |
| `internal/shared/logger` | Uber Zap bootstrap and per-request context |
| `internal/shared/db` | PostgreSQL GORM setup and migration runner |
| `internal/shared/cache` | Redis client |
| `pkg/httperr` | Gin error middleware and panic recovery |
| `pkg/envbool` | Parse environment boolean strings |
| `pkg/supabase` | Supabase client init and helpers |

---

## Build Requirements

- **Go 1.25+**
- **CGO_ENABLED=1** and **`libvips-dev`**, **`libhdf5-dev`**, **`pkg-config`** — required for WebP image encoding (`bimg`) on Ubuntu 24.04+ runners (**`hdf5.pc`** is needed by **matio**, an optional libvips dependency). Use `make build` for CGO builds.
- `CGO_ENABLED=0` — supported for CI review builds; WebP encoding returns HTTP 503 (stub returns `ErrImageEncodeBusy`).

```bash
# CGO build (production)
make build

# No-CGO build (CI / review)
make build-nocgo

# Or directly
CGO_ENABLED=1 go build -o mycourse-io-be .
```

---

## Tooling

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis (configured in `.golangci.yml`) |
| `golang-migrate` | SQL migration runner |
| `PM2` | Process manager for deployment |
| GitHub Actions | CI/CD pipeline (`.github/workflows/deploy-dev.yml`) |
| `npx gitnexus` | Code intelligence graph (impact analysis, query, context) |
