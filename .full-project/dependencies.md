# Dependencies Snapshot

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

## Build/Run Tooling
- Go 1.25 module.
- Makefile/shell scripts for build.
- PM2 ecosystem file for deployment process orchestration.
- GitHub Actions deploy workflow under `.github/workflows`.
