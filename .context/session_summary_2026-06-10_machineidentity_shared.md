# Machine identity → internal/shared/machineidentity

## GitNexus impact (pre-move)

| Symbol | Risk | d=1 callers |
|--------|------|-------------|
| `LoadOrCreateMachineIdentityMaterial` | LOW | `runRegister` |
| `LoadMachineIdentityMaterial` | LOW | `runLogin` |
| `buildHybridMachineBindingMaterial` | LOW | identity loaders, `stableMachineMaterialForRateLimit` |

## Changes

| Action | Path |
|--------|------|
| **New package** | `internal/shared/machineidentity/` — `identity.go`, `fingerprint.go`, `fingerprint_{linux,darwin,windows,other}.go`, `binding_test.go` |
| **Deleted from appcli** | `machine_identity.go`, `machine_fingerprint*.go`, `machine_binding_test.go` |
| **Updated callers** | `register_system_user.go`, `login_system_user.go`, `cli_guard.go` → import `machineidentity` |
| **Kept in appcli** | `machine_secret.go` (`DeriveMachineSecret` — depends on `system/infra` adapter) |

### Exported API (`machineidentity`)

- `LoadOrCreateMachineIdentityMaterial`
- `LoadMachineIdentityMaterial`
- `BuildHybridMachineBindingMaterial`
- `IdentityFilePath`

## Quality gates

| Gate | Result |
|------|--------|
| `golangci-lint run` | PASS (0 issues) |
| `make check-architecture` | PASS |
| `make check-dupl` | PASS (0 clones) |
| `make check-layout` | PASS |
| `go test ./...` | PASS |
| `go build ./...` | PASS |

## Docs synced

- `docs/requirements.md` (FR-4.2 source paths)
- `docs/architecture.md`, `docs/folder-structure.md`, `docs/modules.md`
- `docs/reusable-assets.md` (new asset entry)
- `docs/sequence_diagrams.md`, `docs/logic-flow.md`, `docs/deploy.md`
