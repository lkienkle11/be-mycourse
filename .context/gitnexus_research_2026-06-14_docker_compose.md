# GitNexus research — Docker Compose (BE)

**Date:** 2026-06-14  
**Task:** Add Dockerfile + compose/stack + scripts (no Go symbol edits)

## Index

- Repo: `be-mycourse`, indexed 2026-06-12 (may be stale vs HEAD — refresh at Phase 3)
- Embeddings: 0 → `npx gitnexus analyze --force` (no `--embeddings`)

## Queries

| Query | Findings |
|-------|----------|
| `deploy build health check docker startup` | `Health` handler in `internal/shared/response/response.go`; `InitRouter` mounts `/api/v1/health`; `scripts/pm2-reload-with-binary-rollback.sh`; `docs/deploy.md` |
| `context(Health)` | Caller: `mountAPITree` in `internal/server/router.go` — no code change needed for Docker |

## Reuse (do not duplicate)

| Asset | Reuse for Docker |
|-------|------------------|
| CI build | `CGO_ENABLED=1 go build -trimpath -ldflags="-s -w"` + libvips/HDF5 apt packages |
| Health URL | `http://127.0.0.1:8080/api/v1/health` from pm2 script default |
| Health timeout | `ROLLBACK_HEALTH_TIMEOUT_SEC=90` from pm2 script |
| Env merge | `.env` + `.env.<STAGE>` via `setting.Setup()` |
| PM2 env | `CGO_ENABLED=1`, `MIGRATE=1` from `ecosystem.config.cjs` |
| Stages | `local`, `dev`, `staging`, `prod` + `config/app-<STAGE>.yaml` |

## Symbols changed

**None** — infra-only files (Dockerfile, compose, scripts, env templates, docs).

## Risk

| Area | Level | Notes |
|------|-------|-------|
| Go symbols | LOW | No application code edits |
| CGO runtime libs | MEDIUM | Must pin libvips/HDF5 in slim runtime image |
| Secrets in image | LOW | `.dockerignore` excludes `.env*`; compose uses `env_file` at run time |

## Docs gap (pre-implementation)

| Doc | Gap |
|-----|-----|
| `docs/deploy.md` | Mentions optional Docker install (Step 6) but no compose workflow |
| `.env.dev.example` | Missing while `config/app-dev.yaml` exists |
| README | No Docker quick-start |
| `.context/` | No prior Docker session; one note suggests CI Docker image as future optimization |

## Git baseline

- Branch: feature work on `feat/course-submit-quiz-validation` (ahead 1)
- No existing `Dockerfile` / `docker/` in repo
- CI: `.github/workflows/deploy-dev.yml` — **do not edit**

## Phase 2 file list

- `Dockerfile`, `.dockerignore`, `.env.dev.example`
- `docker/compose.{local,dev,staging,prod}.yml`
- `docker/stack.{local,dev,staging,prod}.yml`
- `scripts/docker/_lib.sh`, `compose-up.sh`, `compose-down.sh`, `build-image.sh`, `health-check.sh`, `local-infra.sh`, `swarm-deploy.sh`
