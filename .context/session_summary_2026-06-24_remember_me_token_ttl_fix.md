# Session Summary — Remember-me refresh token TTL fix

**Date:** 2026-06-24

## TTL policy

- `remember_me = false` or email confirm → **3 days** (`RefreshTokenTTL`)
- `remember_me = true` → **30 days** (`RememberMeRefreshTTL`)

## API contract

Login / confirm / refresh JSON: **only** `access_token`, `refresh_token`, `session_id`.
No `remember_me` or `refresh_ttl_seconds` in response body.

`TokenPairResult.RememberMe` and `RefreshTTL` remain internal for session DB + `Set-Cookie` Max-Age.

## Docs synced

`requirements.md`, `modules/auth.md`, `curl_api.md`, `return_types.md`, `data-flow.md`, `sequence_diagrams.md`, `api_swagger.yaml`, `api-dog-import.json`
