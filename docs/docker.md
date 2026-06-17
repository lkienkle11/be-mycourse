# Docker Compose & manual container deploy (backend)

Alternative to the PM2 + binary path in [`docs/deploy.md`](deploy.md). **CI/CD (`.github/workflows/*`) is unchanged** — this is for local development and optional VPS manual deploy.

---

## Environment matrix

| Environment | `STAGE` | Env files (`env_file`) | Host port | Image tag |
|-------------|---------|------------------------|-----------|-----------|
| local | `local` | `.env` + `.env.local` | 8080 | `mycourse-io-be:local` |
| dev | `dev` | `.env` + `.env.dev` | 8080 | `mycourse-io-be:dev` |
| staging | `staging` | `.env` + `.env.staging` | 8080 | `mycourse-io-be:staging` |
| prod | `prod` | `.env` + `.env.prod` | 8080 | `mycourse-io-be:prod` |

Copy templates before first run:

```bash
cp .env.example .env
cp .env.local.example .env.local    # or .env.dev.example → .env.dev, etc.
```

Fill **`DATABASE_URL`** / **`REDIS_ADDR`** with your **cloud** Postgres and Redis URLs (default path). Real `.env*` files stay gitignored.

---

## Quick start (local)

**Linux / macOS / WSL:**

```bash
cd be-mycourse
cp .env.example .env && cp .env.local.example .env.local
# Edit .env / .env.local — DATABASE_URL, REDIS_ADDR, JWT_SECRET, …

./scripts/docker/compose-up.sh local
./scripts/docker/health-check.sh local
./scripts/docker/compose-down.sh local
```

**Windows 10 / 11 (CMD hoặc PowerShell)** — cần [Docker Desktop](https://www.docker.com/products/docker-desktop/) + WSL2 backend (khuyến nghị):

```cmd
cd be-mycourse
copy .env.example .env
copy .env.local.example .env.local

scripts\docker\compose-up.cmd local
scripts\docker\health-check.cmd local
scripts\docker\compose-down.cmd local
```

PowerShell trực tiếp (tương đương):

```powershell
.\scripts\docker\compose-up.ps1 local
.\scripts\docker\health-check.ps1 local
.\scripts\docker\compose-down.ps1 local
```

Health endpoint (same as PM2 deploy script): `GET http://127.0.0.1:8080/api/v1/health`

---

## Scripts (`scripts/docker/`)

| Script | Purpose |
|--------|---------|
| `compose-up.sh` / `.ps1` / `.cmd` | Verify env files → `docker compose up --build -d` |
| `compose-down.sh` / `.ps1` / `.cmd` | Tear down stack |
| `build-image.sh` / `.ps1` / `.cmd` | Build image only (no run) |
| `health-check.sh` / `.ps1` / `.cmd` | Poll health URL (timeout **90s**) |
| `local-infra.sh` / `.ps1` | **Disabled** — commented Postgres/Redis helpers |
| `swarm-deploy.sh` / `.ps1` / `.cmd` | Swarm demo only — **not for CI/tests** |

- **Unix:** `.sh` + `_lib.sh`
- **Windows 10/11:** `.ps1` + `_lib.ps1`; `.cmd` gọi PowerShell (dùng được từ CMD)
- Shared constants: health URL/timeout khớp `scripts/pm2-reload-with-binary-rollback.sh`

---

## Dockerfile (aligned with CI)

- **Quality gates:** CI runs **`make test-all`** before build; locally run **`make check-all`** before shipping. The Docker image only runs the **production CGO build** (same as CI **`build`** job).
- **Builder:** `golang:1.25.0-bookworm` + `libvips-dev`, `libhdf5-dev`, `pkg-config`
- **Build:** `CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o mycourse-io-be-<STAGE> .`
- **Runtime:** `debian:bookworm-slim` + `libvips42`, `libhdf5-103-1`, `curl`
- **Defaults:** `CGO_ENABLED=1`, `MIGRATE=1` (matches `ecosystem.config.cjs`)
- **HEALTHCHECK:** `curl -fsS http://127.0.0.1:8080/api/v1/health`

---

## Optional local Postgres / Redis

`docker/compose.*.yml` includes **commented-out** `postgres` and `redis` services. Default stacks use cloud URLs from env. To run on-box DB/cache:

1. Uncomment `postgres` / `redis` in the target compose file
2. Point `.env.<stage>` at `postgres:5432` / `redis:6379` hostnames
3. Uncomment helpers in `scripts/docker/local-infra.sh`

---

## VPS manual deploy (example: dev)

```bash
git pull
cp .env.dev.example .env.dev   # first time only; edit secrets
./scripts/docker/build-image.sh dev
docker compose -f docker/compose.dev.yml up -d
./scripts/docker/health-check.sh dev
```

Existing PM2/Nginx setup remains valid until you switch traffic intentionally.

---

## Swarm stack (blue-green demo — not for testing)

Files: `docker/stack.{local,dev,staging,prod}.yml` — **2 replicas**, `start-first` rolling update, rollback on failure.

```bash
docker swarm init   # once, operator only
./scripts/docker/swarm-deploy.sh dev
```

**Do not** run `docker swarm` commands as part of automated test gates for this repo.

---

## Compose project names

| Env | Project name |
|-----|--------------|
| local | `mycourse-be-local` |
| dev | `mycourse-be-dev` |
| staging | `mycourse-be-staging` |
| prod | `mycourse-be-prod` |

Running staging + prod on one host: override host ports in compose if both stacks need to run simultaneously.

---

## Related docs

- [`docs/deploy.md`](deploy.md) — primary PM2 + CI runbook
- [`README.md`](../README.md) — quick start and env setup
- `.env.dev.example` — new template for `STAGE=dev`
