# Dependencies Snapshot


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Core Runtime Libraries
- `github.com/gin-gonic/gin`: HTTP server/router.
- `gorm.io/gorm` + `gorm.io/driver/postgres`: ORM and PostgreSQL driver.
- `github.com/golang-jwt/jwt/v5`: JWT handling.
- `github.com/go-playground/validator/v10`: request validation.
- `github.com/redis/go-redis/v9`: Redis integration.
- `github.com/golang-migrate/migrate/v4`: SQL migration runner.
- `go.uber.org/zap`: logging.

## Integration Libraries
- `github.com/supabase-community/supabase-go`: Supabase client.
- Brevo integration uses direct HTTP wrapper in `pkg/brevo`.

## Internal Dependency Hotspots
- Most reused internal packages:
  - `pkg/errcode`
  - `pkg/setting`
  - `pkg/response`
  - `models`
  - `services`

**Media / Bunny (Sub 09 + Sub 12):** public contract is `dto.UploadFileResponse` (no **`origin_url`** in JSON — Sub 12) + `media_files` / `metadata_json`; policy helpers in `pkg/media/media_resolver.go`; HTTP in `pkg/media/clients.go`. Authoritative write-up: **`docs/modules/media.md`** (and `docs/api_swagger.yaml` for OpenAPI consumers).

## Build/Run Tooling
- Go 1.25 module.
- Makefile/shell scripts for build.
- PM2 ecosystem file for deployment process orchestration.
- GitHub Actions deploy workflow under `.github/workflows`.
