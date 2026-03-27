# MyCourse Backend

Backend scaffold for E-learning platform using Gin + Gorm + PostgreSQL and Clean Architecture.

## Quick Start

1. Ensure PostgreSQL is running.
2. Update `config/config.yaml` database credentials.
3. Run:

```bash
go mod tidy
go run ./cmd/api
```

4. Verify:

```bash
curl http://localhost:8080/health
```

## Structure

- `cmd/api`: composition root and server startup.
- `internal/core/domain`: domain entities.
- `internal/core/ports`: service/repository interfaces.
- `internal/repository`: persistence adapters.
- `pkg/logger`, `pkg/token`: reusable infrastructure packages.
- `docs`: architecture, schema, API specs, and module docs.
