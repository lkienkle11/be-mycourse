# GitNexus Research — HttpOnly Auth Cookies (BE-02)

**Date:** 2026-06-14  
**Task:** Fix BE-02 — auth cookies `HttpOnly=true` + BE reads session from cookies  
**Checklist:** `temporary-docs/tieu-chuan-check-be-fe/be-mycourse.md` (process adapted for auth security)

## Index

- Repo: `be-mycourse`
- Index used at research time (post-implementation close-out pending `npx gitnexus analyze --force`)

## Symbols changed

| Symbol | Change | Risk (impact) | d=1 callers |
|--------|--------|---------------|-------------|
| `setAuthCookies` | HttpOnly=true; cookie name constants | LOW (0 upstream) | none |
| `clearAuthCookies` | HttpOnly=true | LOW | none |
| `extractSessionCredentials` | **new** — headers or cookies | LOW | RefreshToken, Logout handlers only |
| `RefreshToken` | read cookies; set cookies on success | LOW (0 upstream) | route only |
| `Logout` | read cookies via helper | LOW | route only |
| `extractBearerToken` | fallback `access_token` cookie | LOW — d=1: `requireJWT` | AuthJWT middleware (all protected routes) |

## Processes affected

- Login → Set-Cookie (HttpOnly)
- ConfirmEmail → Set-Cookie (HttpOnly)
- RefreshToken → cookie credentials + Set-Cookie rotation
- Logout → cookie credentials
- AuthJWT → Bearer header **or** access_token cookie

## Docs gap (before fix)

| Doc | Stale content |
|-----|---------------|
| `docs/requirements.md` | non-HttpOnly requirement |
| `docs/modules/auth.md` | non-HttpOnly |
| `docs/data-flow.md` | non-HttpOnly |
| `docs/return_types.md` | non-HttpOnly |

## Reuse

- `middleware.CookieAccessToken` / `CookieRefreshToken` / `CookieSessionID` constants (new in `constants.go`)
- Existing `AuthJWT`, `RefreshSession`, `setAuthCookies` structure — extended, not replaced

## Follow-up (out of scope)

- BE-03: stop returning tokens in JSON body
- BE-04: enable CSRF middleware
- BE-07: Secure cookie tied to TLS config not only RunMode
