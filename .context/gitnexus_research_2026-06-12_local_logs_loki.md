# GitNexus Research Note — Local Logs + Loki (pre-code)

## Scope

- Repository: `be-mycourse`
- Task: local file logging upgrade for Grafana Alloy/Loki while preserving existing logger package and middleware reuse
- Date: 2026-06-12

## Discovery Checklist Status

- Read baseline docs/context done:
  - `docs/patterns.md`, `docs/reusable-assets.md`, `README.md`, `IMPLEMENTATION_PLAN_EXECUTION.md`
  - reference doc: `temporary-docs/tai-lieu-setup-docs-local-tuong-thich-grafana-loki/tai-lieu-setup-be-local-logs-loki.md`
  - `.context` session summaries scanned for logging/system baseline
- Git baseline done:
  - branch: `feat/lavinmq-mq-bootstrap`
  - reviewed `git log --oneline -20`
  - reviewed target-area diff against `master` for `internal/shared/logger`, `internal/shared/middleware`, `internal/shared/setting`
- GitNexus research done:
  - `list_repos`, `query`, `context`, `impact` for required symbols
  - stale warning observed in `gitnexus://repo/be-mycourse/context`
  - index refreshed with `npx gitnexus analyze --force`

## Docs Gap List (current code/docs vs target Loki plan)

1. Docs still describe `LOG_FILE_PATH` single file as Filebeat/ELK sink; no Alloy-first guidance.
2. Missing log path strategy by OS (`user` vs `service` mode, `LOG_DIR` override).
3. Missing split between app/business log stream and access log stream (`app.log` vs `access.log`).
4. Missing file rotation policy fields (`max_size`, backups, age, compress).
5. Missing Loki JSON contract details (`ts` RFC3339Nano UTC, `kind`, `log_file`, label guidance).
6. Missing operator doc for Alloy config and Grafana Cloud Loki pipeline.

## Symbol Context + Blast Radius (upstream)

### `InitFromSettings`

- Location: `internal/shared/logger/init.go`
- d=1 caller(s): `main`
- Risk: LOW
- Notes: startup bootstrap path only.

### `Init`

- Location: `internal/shared/logger/init.go`
- d=1 caller(s): `InitFromSettings` (+ logger tests)
- Risk: LOW
- Notes: core logger factory; test coverage already exists and must be extended.

### `RequestLogger`

- Location: `internal/shared/middleware/request_logger.go`
- d=1 caller(s): `server.InitRouter`
- Risk: LOW
- Notes: middleware-level behavior change will affect all HTTP routes.

### `appendJSONFileCoreIfConfigured`

- Location: `internal/shared/logger/init.go`
- d=1 caller(s): `Init`
- Risk: LOW
- Notes: this is the legacy single-file tee compatibility path.

### `applyYAMLLoggingGlobals`

- Location: `internal/shared/setting/setting_yaml_apply.go`
- d=1 caller(s): `applyYAMLToGlobals`
- Risk: LOW
- Notes: config shape contract for all env/yaml logging fields.

### `identityFilePath` (for XDG consolidation)

- Location: `internal/shared/machineidentity/identity.go`
- d=1 caller(s): `IdentityFilePath`, `loadOrCreateFileSecret`, `loadFileSecret`
- Risk: LOW
- Notes: transitive APPCLI consumers exist; path behavior must stay backward compatible.

### `cliRateLimitPath` (for XDG consolidation)

- Location: `internal/shared/ratelimit/file.go`
- d=1 caller(s): `DefaultCLIFileStore`
- Risk: LOW
- Notes: only rate-limit file path derivation should change.

## Process/Flow Findings

- Startup logger flow:
  - `main` -> `setting.Setup` -> `applyYAMLLoggingGlobals` -> `logger.InitFromSettings` -> `Init` -> optional file sink -> `zap.ReplaceGlobals`.
- HTTP access logging flow:
  - `main` -> `server.InitRouter` -> `middleware.RequestLogger` -> per-request request_id enrichment + access line emission.
- Context correlation flow already in place via `logger.WithRequestID` / `logger.FromContext`; must be reused (no parallel middleware/logger stack).

## Reuse List (must keep/extend)

- `internal/shared/logger/`: `Init`, `InitFromSettings`, `Sync`, `WithRequestID`, `FromContext`.
- `internal/shared/middleware/request_logger.go`: extend fields/leveling in place.
- `internal/shared/setting/setting.go` + `setting_yaml_apply.go`: extend logging config schema in place.
- Existing `LOG_FILE_PATH` behavior: retain as legacy compatibility path.

## Expected Touch Surface

- Code:
  - `internal/shared/xdgx/*` (new helper)
  - `internal/shared/machineidentity/identity.go`
  - `internal/shared/ratelimit/file.go`
  - `internal/shared/setting/setting.go`
  - `internal/shared/setting/setting_yaml_apply.go`
  - `internal/shared/logger/*`
  - `internal/shared/middleware/request_logger.go`
  - logger/middleware tests
- Config/docs:
  - `config/app*.yaml`, `.env.example`
  - `docs/observability/loki-alloy.md`, `config/alloy/be-mycourse.example.alloy`
  - sync `docs/patterns.md`, `docs/reusable-assets.md`, `README.md`, `IMPLEMENTATION_PLAN_EXECUTION.md`, and deploy/operator docs

## Risk Summary

- No HIGH/CRITICAL risks for planned edited symbols in this phase (all LOW in GitNexus impact output).
- Primary migration risk is behavior drift in logging contract or path resolution; mitigated by keeping legacy mode and adding tests.
