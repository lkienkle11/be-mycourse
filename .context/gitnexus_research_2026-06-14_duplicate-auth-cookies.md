# GitNexus Research — Duplicate Auth Cookies (dev domain)

**Date:** 2026-06-14  
**Task:** Fix duplicate `access_token` / `refresh_token` / `session_id` after client silent refresh  
**Checklist:** `temporary-docs/tieu-chuan-check-be-fe` (discovery phase)

## Root cause

Client-side refresh writes cookies **twice** with different `Domain`:

| Writer | Domain | When |
|--------|--------|------|
| BE `setAuthCookies` (`domain=""`) | `api.<parent-domain>` (host-only) | Browser XHR `POST /auth/refresh` |
| FE `syncAuthSessionCookiesAction` | `<parent-domain>` (`AUTH_COOKIE_DOMAIN`) | After refresh, Server Action |

Login/confirm and server-side refresh do **not** duplicate — BE `Set-Cookie` never reaches the browser (server-to-server).

## Symbols

| Symbol | Repo | Action | Risk |
|--------|------|--------|------|
| `setAuthCookies` | BE | Use `AUTH_COOKIE_DOMAIN` | LOW |
| `clearAuthCookies` | BE | Same domain + clear legacy host-only | LOW |
| `persistRefreshedAuthSession` | FE | Server-only (skip on client after BE fix) | LOW |
| `setAuthSessionCookies` | FE | **Reuse** — no change | HIGH if touched |

## Proposed fix

1. BE: `AUTH_COOKIE_DOMAIN` in settings → `setAuthCookies` / `clearAuthCookies`
2. BE: When parent domain set, also expire legacy host-only cookies (`domain=""`)
3. FE: `persistRefreshedAuthSession` only when `isServer()` — client relies on BE `Set-Cookie`
4. Env: `AUTH_COOKIE_DOMAIN=yourdomain.net` on BE + FE (must match; see deploy docs)

## d=1 callers

None for BE handler-internal change. FE interceptor-only change.
