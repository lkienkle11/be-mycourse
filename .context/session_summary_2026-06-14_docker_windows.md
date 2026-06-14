# Session addendum — Docker Windows scripts (BE)

**Date:** 2026-06-14  
**Scope:** `scripts/docker/*.ps1`, `*.cmd`, `_lib.ps1` — mirror bash helpers; no Go changes.

## Added

| File | Role |
|------|------|
| `_lib.ps1` | Shared: env validation, compose invoke, health poll (curl.exe / 90s timeout) |
| `compose-up.ps1`, `compose-down.ps1`, `build-image.ps1`, `health-check.ps1`, `swarm-deploy.ps1` | PowerShell entrypoints |
| `*.cmd` | CMD wrappers → `powershell -ExecutionPolicy Bypass -File …` |
| `local-infra.ps1` | Disabled stub (mirror `.sh`) |

## Docs synced

- `docs/docker.md` — Windows 10/11 quick start (CMD + PowerShell)
- `docs/folder-structure.md`, `README.md`

## Windows notes

- Requires **Docker Desktop** (WSL2 backend recommended)
- Health check uses **curl.exe** (built into Windows 10+) — same semantics as `_lib.sh`
- `<env>` = `local` \| `dev` \| `staging` \| `prod`

## Quality gates

Re-run after this addendum: golangci-lint, make checks, go test, go build — infra-only, expect PASS.
