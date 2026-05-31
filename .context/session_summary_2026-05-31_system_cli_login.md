# Session summary — System CLI login + machine binding

**Date:** 2026-05-31  
**Task:** Migrate system login from HTTP to CLI (`CLI_SYSTEM_LOGIN`), add machine binding, remove `POST /api/system/login`.

## GitNexus (pre-code)

- Subagent + main thread: stale index (4 commits); source-verified d=1 callers for `SystemLogin` → handler (removed), `RegisterPrivilegedUser` → `runRegister`, `MatchCount` → `SystemLogin` only.
- Reuse: `SystemCryptoAdapter`, `cryptox`, `authinfra.CheckPassword`, appcli TTY helpers.
- Research note: `.context/session_summary_2026-05-31_system_cli_login_research.md`

## Code changes

| File | Change |
|------|--------|
| `migrations/000014_system_user_machine_binding.*.sql` | `machine_secret` column |
| `internal/system/domain/system.go` | `PrivilegedUser.MachineSecret` |
| `internal/system/domain/repository.go` | `FindByCredentials` replaces `MatchCount` |
| `internal/system/infra/repos.go` | Map column; `FindByCredentials` |
| `internal/system/application/service.go` | `machineSecret` param; binding check |
| `internal/shared/constants/error_msg.go` | `MsgSystemMachineBindingFailed` |
| `internal/shared/errors/system.go` | `ErrSystemMachineBindingFailed` |
| `internal/appcli/system_cli_common.go` | Shared TTY + `newSystemService` |
| `internal/appcli/machine_identity.go` | Local identity file load/create |
| `internal/appcli/machine_secret.go` | `DeriveMachineSecret` via adapter |
| `internal/appcli/register_system_user.go` | Machine binding on register |
| `internal/appcli/login_system_user.go` | `CLI_SYSTEM_LOGIN` flow |
| `internal/system/delivery/routes.go` | Removed `/login` |
| `internal/system/delivery/handler.go` | Removed HTTP login handler/DTOs |
| `main.go` | Wire `MaybeRunSystemLogin` |
| `.env.example` | `CLI_SYSTEM_LOGIN=false` |

## Docs synced

- `docs/requirements.md` FR-4.1–4.4, `docs/sequence_diagrams.md` §8, `docs/curl_api.md` §5.0
- `docs/api_swagger.yaml`, `docs/api-dog-import.json` (regenerated)
- `docs/router.md`, `docs/api-overview.md`, `docs/database.md`, `docs/deploy.md`
- `docs/return_types.md`, `docs/reusable-assets.md`, `docs/modules.md`, `docs/folder-structure.md`, `docs/architecture.md`, `README.md`

## Quality gates (all PASS — first run)

```bash
golangci-lint cache clean && golangci-lint run    # 0 issues
make check-architecture                            # OK
make check-dupl                                    # 0 clone groups
make check-layout                                  # OK
go test ./...                                      # OK
go build ./...                                     # OK
npx gitnexus analyze --force                       # OK (4342 nodes)
```

No unrelated fixes required.

## Deploy notes

1. Apply migration `000014`; re-register privileged users on each host (`CLI_REGISTER_NEW_SYSTEM_USER=1`).
2. Login: `SYSTEM_TOKEN=$(CLI_SYSTEM_LOGIN=1 go run .)` — stdout only JWT (TTL **90s**).
3. Backup `~/.config/mycourse/machine_identity` (or `$XDG_CONFIG_HOME/mycourse/machine_identity`).
4. After `app_system_env` rotation or host change → re-register.

## Hybrid binding update (2026-05-31)

Replaced file-only HMAC input with **hybrid** material:

`v1|file:<enrollment_secret>|mid:<machine-id>|hw:<hardware_uuid>|host:<hostname>|plat:<goos/goarch>`

→ `CredentialHMACHex(app_system_env, hybrid)` → `machine_secret` in DB.

**Re-register required** after this change (old bindings used file content only).

New files: `machine_fingerprint.go`, `machine_fingerprint_{linux,darwin,windows,other}.go`, `machine_binding_test.go`.

```bash
CLI_REGISTER_NEW_SYSTEM_USER=1 go run .
TOKEN=$(CLI_SYSTEM_LOGIN=1 go run .)
curl -H "Authorization: Bearer $TOKEN" -X POST http://localhost:8080/api/system/permission-sync-now
# POST /api/system/login → expect 404
```
