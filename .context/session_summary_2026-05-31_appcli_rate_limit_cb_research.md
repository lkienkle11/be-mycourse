# APPCLI Rate Limit + Circuit Breaker — GitNexus Research (pre-code)

## Symbols reuse
| Symbol | Action |
|--------|--------|
| `rateBucket` / allow logic | Consolidate → `internal/shared/ratelimit` |
| `RateLimitLocal`, `RateLimitSystemIP` | Thin wrappers over `InMemoryStore` |
| `response.AbortFail`, `errors.TooManyRequests` | Reuse for HTTP 429/503 |
| `machineIdentityPath()` pattern | Reuse for CLI rate-limit file path |
| `shareddb.StdDB().PingContext` | DB probe for CB |
| `cache.Redis` + `RedisAvailable()` | Optional CB state persistence |

## Symbols change (risk)
| Symbol | Risk | d=1 callers |
|--------|------|-------------|
| `RateLimitLocal` | LOW | router groups (internal) |
| `RateLimitSystemIP` | LOW | router system group |
| `InitRouter` | LOW | `main` |
| `MaybeRunSystemLogin` | LOW | `main` |
| `MaybeRunRegisterNewSystemUser` | LOW | `main` |
| `main` bootstrap | MEDIUM | process entry |

## Docs gap
- NFR-1.1: `/api/system` window documented as **3 seconds**; code uses **3 minutes** (`RateLimitSystemIP(10, 3)`).

## Processes
- HTTP: `InitRouter` → middleware stack → rate limiters → handlers
- CLI: `main` → `MaybeRun*` → credentials → service
