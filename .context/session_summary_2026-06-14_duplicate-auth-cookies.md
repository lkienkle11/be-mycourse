# Session Summary — Duplicate Auth Cookies Fix

**Date:** 2026-06-14  
**Scope:** `be-mycourse` + `fe-mycourse`  
**Checklist:** `temporary-docs/tieu-chuan-check-be-fe`

## Task

Fix duplicate `access_token` / `refresh_token` / `session_id` cookies on dev domain after client silent refresh.

## Root cause

Client refresh: BE `Set-Cookie` with `domain=""` (host-only `api.*`) + FE `syncAuthSessionCookiesAction` (`AUTH_COOKIE_DOMAIN` parent domain).

## Files changed

### BE
| File | Change |
|------|--------|
| `internal/auth/delivery/handler.go` | `AUTH_COOKIE_DOMAIN`; `writeAuthCookies`; clear legacy host-only |
| `internal/shared/setting/setting.go` | `App.AuthCookieDomain` |
| `internal/shared/setting/setting_yaml_apply.go` | load/expand domain |
| `config/app*.yaml` | `auth_cookie_domain: "${AUTH_COOKIE_DOMAIN}"` |
| `.env.example`, `.env.dev.example` | document env |
| `docs/modules/auth.md` | cookie domain note |

### FE
| File | Change |
|------|--------|
| `src/api/instance.ts` | `persistRefreshedAuthSession` server-only |
| `docs/flow.md` | §4.4 cookie update table |

## GitNexus

- Research: `.context/gitnexus_research_2026-06-14_duplicate-auth-cookies.md`
- Impact: `setAuthCookies` LOW; `persistRefreshedAuthSession` LOW
- detect_changes: expected auth/login/refresh/logout flows only

## Quality gates

| Command | Result |
|---------|--------|
| BE `golangci-lint run` | PASS |
| BE `go test ./...` (changed pkgs) | PASS |
| BE `go build ./...` | PASS |
| FE `npm run lint:biome` | PASS |
| FE `npm run lint` | PASS |
| FE `npm run build` | PASS |

## Deploy (required)

Set on **both** BE and FE hosted environments (same value as FE — see `fe-mycourse/docs/deploy.md`):

```bash
AUTH_COOKIE_DOMAIN=yourdomain.net
```

Users with existing duplicates: clear cookies once for parent domain and `api.<parent-domain>`.

## Security / docs hygiene

- Replaced hardcoded deploy domains with placeholders (`cdn.yourdomain.com`, `api.yourdomain.com`, `yourdomain.com`) in all tracked docs, `.env.*.example`, context notes, `AGENTS.md`, `CLAUDE.md`.
- Real domain values belong only in non-committed `.env` / server config.

## Manual verify

1. Login → one cookie set per name on parent domain
2. Delete `access_token` → trigger API → refresh → still one set (no `api.*` duplicate)
3. SSR protected page after client refresh → still authenticated
