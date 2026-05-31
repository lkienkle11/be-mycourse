# Session summary — System config bcrypt alignment

**Date:** 2026-05-31  
**Task:** Align `system_app_config` secrets with bcrypt-at-rest (cost 14); CLI verify via `auth/infra.CheckPassword`.

## GitNexus findings

- **`cliVerifyAppPassword`:** LOW risk; d=1 caller `runRegister` → `MaybeRunRegisterNewSystemUser` → `main`.
- **`CheckPassword`:** Now called from `appcli/cliVerifyAppPassword` (was zero callers before).
- **`SystemLogin` / `RegisterPrivilegedUser`:** No algorithm change; HMAC/JWT still use hash strings as key material via `SystemCryptoAdapter` → `cryptox`.
- Reuse path confirmed: no new bcrypt helper; `appcli` → `auth/infra.CheckPassword` allowed by arch lint (`anyProjectDeps: true`).

## Code changes

| File | Change |
|------|--------|
| `internal/appcli/register_system_user.go` | Replaced `subtle.ConstantTimeCompare` with `authinfra.CheckPassword`; removed `crypto/subtle` |
| `internal/system/application/service.go` | Comments on `SystemLogin`, `RegisterPrivilegedUser`, `VerifySystemAccessToken` (bcrypt-14 key material) |

## Docs synced

- `docs/database.md` — three config columns → bcrypt hash cost 14
- `docs/requirements.md` — FR-4.2/4.3, NFR-2.5; fixed stale source paths
- `docs/sequence_diagrams.md` — system login diagram + CLI bcrypt verify step; removed `systemauth` references
- `docs/deploy.md` — bcrypt-14 generation + re-register after `app_system_env` rotation
- `docs/reusable-assets.md` — document `CheckPassword` reuse from appcli
- `docs/modules.md` — system crypto semantics
- `docs/return_types.md` — fix `SystemAccessTokenTTL` reference
- `.env.example` — bcrypt-14 note for config columns

## Deploy note

1. Seed `system_app_config` id=1 with bcrypt-14 hashes for `app_cli_system_password`, `app_system_env`, `app_token_env`.
2. Run CLI registration with `CLI_REGISTER_NEW_SYSTEM_USER=true`.
3. **After rotating `app_system_env`:** existing `system_privileged_users` are invalid — re-run CLI registration.

```bash
python3 -c "import bcrypt; print(bcrypt.hashpw(b'your-secret', bcrypt.gensalt(rounds=14)).decode())"
```

## Quality gates (all PASS)

```bash
golangci-lint cache clean && golangci-lint run          # 0 issues
make check-architecture                                  # OK
make check-dupl                                          # 0 clone groups
make check-layout                                        # OK
go build ./...                                           # OK
npx gitnexus analyze --force                             # OK (4330 nodes)
```

No out-of-scope fixes required — all gates passed on first run.

## Manual verification (operator)

1. Wrong CLI app password → `"Failure: invalid app password."`
2. Correct password → register privileged user OK
3. `POST /api/system/login` → JWT; protected `/api/system/*` works
4. Rotate `app_system_env` without re-register → login fails until CLI re-run
