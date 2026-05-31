# APPCLI Rate Limit + Circuit Breaker — Session Summary

**Date:** 2026-05-31

## Scope completed

- Extracted `internal/shared/ratelimit` (fixed-window `AllowFixedWindow`, `InMemoryStore`, `FileStore`)
- Refactored `RateLimitLocal` / `RateLimitSystemIP` to delegate (no HTTP quota change)
- Added `internal/appcli/cli_guard.go` — 5 ops / 3 min file-backed limit + circuit breaker before prompts
- Added `internal/shared/resilience` — CB state machine, DB probe, load tracking, optional Redis
- Added `middleware.CircuitBreakerMiddleware` after `RequestLogger` in `InitRouter`
- Reordered `main.go`: Redis + resilience before APPCLI branch
- Added `ServiceUnavailable = 9018` error code
- Unit tests: `ratelimit/window_test.go`, `resilience/circuitbreaker_test.go`, `appcli/cli_guard_test.go`

## GitNexus

- Pre-code research: `.context/session_summary_2026-05-31_appcli_rate_limit_cb_research.md`
- Post-code: `npx gitnexus analyze --force` — 4473 nodes, 12692 edges

## Files changed (main)

| Area | Files |
|------|-------|
| Rate limit | `internal/shared/ratelimit/*`, `internal/shared/constants/ratelimit.go`, middleware refactor |
| Resilience | `internal/shared/resilience/*`, `internal/shared/middleware/circuitbreaker.go`, `setting` + `config/app.yaml` |
| APPCLI | `internal/appcli/cli_guard.go`, login/register wiring |
| Bootstrap | `main.go`, `internal/server/router.go` |
| Errors | `errcode_codes.go`, `errcode_messages.go`, `constants/error_msg.go` |
| Docs | `requirements.md`, `architecture.md`, `router.md`, `folder-structure.md`, `logic-flow.md`, `curl_api.md`, `return_types.md`, `deploy.md`, `sequence_diagrams.md`, `reusable-assets.md`, `README.md` |

## Quality gates

| Gate | Result |
|------|--------|
| `golangci-lint run` | PASS (0 issues) |
| `make check-architecture` | PASS |
| `make check-dupl` | PASS (0 clone groups) |
| `make check-layout` | PASS |
| `go test ./...` | PASS |
| `go build ./...` | PASS |

## Defaults (YAML `resilience:`)

| Setting | Default |
|---------|---------|
| APPCLI rate limit | 5 ops / 3 min |
| DB probe interval | 5s |
| DB failures to open | 3 |
| Max in-flight HTTP | 200 |
| Half-open probe quota | 3 |
| Open cooldown | 30s |

## Manual verification

Operator should run on environment with DB + bcrypt config:

```bash
CLI_REGISTER_NEW_SYSTEM_USER=1 go run .
TOKEN=$(CLI_SYSTEM_LOGIN=1 go run .)
curl -H "Authorization: Bearer $TOKEN" -X POST http://localhost:8080/api/system/permission-sync-now
```

Verify sixth CLI attempt within 3 minutes prints rate-limit failure on stderr.
