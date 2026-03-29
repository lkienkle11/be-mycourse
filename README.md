# MyCourse Backend

Backend scaffold aligned to the monolith layout in `36.md` (inspired by `openedu-core`).

## Quick Start

1. Ensure Redis is running.
2. Copy `.env.example` to `.env`, set `STAGE` if needed, and fill Supabase keys used in `config/app.yaml` or `config/app-<STAGE>.yaml`:
   - `supabase.url` placeholders → `SUPABASE_URL`
   - `SUPABASE_SERVICE_ROLE_KEY`
   - `SUPABASE_DB_URL` (pooler or direct)
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
- `pkg/setting`: YAML config with per-stage files and `.env` map substitution.
- `config/`: `app.yaml` + `app-<STAGE>.yaml` and env examples.
- `tracing/`, `runtime/`: observability and runtime placeholders.
