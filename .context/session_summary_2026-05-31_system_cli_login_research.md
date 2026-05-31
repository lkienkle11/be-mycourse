# GitNexus research note — System CLI login migration (pre-code)

**Date:** 2026-05-31

## Index

- **Stale** (4 commits behind); re-analyze at end of phase.
- Impact under-reports d=1 for `SystemLogin`/`MatchCount` — source verified callers below.

## Reuse

| Symbol | Reason |
|--------|--------|
| `SystemCryptoAdapter` → `cryptox` | HMAC + JWT |
| `readSecretInput`, `newSystemService`, `cliVerifyAppPassword` | appcli |
| `authinfra.CheckPassword` | CLI app password gate |
| `parsebool.EnvEnabled` | env gates |

## Change + d=1 callers

| Symbol | Callers to update |
|--------|-------------------|
| `SystemLogin` | `handler.systemLogin` (remove), new `login_system_user.go` |
| `RegisterPrivilegedUser` | `runRegister` |
| `MatchCount` | `SystemLogin` only → replace with `FindByCredentials` |

## Risk

- `SystemLogin` / `RegisterPrivilegedUser`: LOW (few callers)
- `SystemCryptoAdapter`: HIGH if changed — **not changing**, only reuse
- `MatchCount` removal: LOW/MEDIUM

## Docs gap

- FR-4.1 still HTTP login; TTL docs say 3600s, code 90s
- No machine binding / CLI login docs
