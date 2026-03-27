# MyCourse Backend

Backend scaffold aligned to the monolith layout in `36.md` (inspired by `openedu-core`).

## Quick Start

1. Ensure PostgreSQL and Redis are running.
2. Update credentials in `config/app.ini`.
3. Run:

```bash
go mod tidy
go run .
```

4. Verify:

```bash
curl http://localhost:8080/api/v1/health
```

## Structure

- `main.go`: startup flow (settings, db, cache, migrate, bootstrap, queue, router).
- `api/`: router, route groups (`public`, `api/v1`, `api/internal-v1`), API config.
- `middleware/`: request interceptor for auth/permission/tenant hooks.
- `services/`, `dto/`: business layer skeleton.
- `models/`, `models/migrations/`: persistence layer skeleton.
- `cache_clients/`: Redis client bootstrap.
- `queues/`: async layer placeholder (RabbitMQ intentionally excluded).
- `pkg/setting`: INI config loader with stage-based file resolution.
- `config/`: `app.ini` + per-stage config files and bootstrap seeds.
- `tracing/`, `runtime/`: observability and runtime placeholders.
