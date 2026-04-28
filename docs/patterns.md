# Patterns and Conventions


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Architectural Patterns
- Layered monolith: `api` -> `services` -> `models`.
- Cross-cutting concerns isolated in `middleware` and `pkg/*`.
- RBAC policy-as-code through constants + DB sync.
- Shared **error/sentinel message strings** (and small related numeric caps) belong in **`constants/error_msg.go`** — see file header; numeric JSON codes stay in `pkg/errcode`. If the same sentence is both API default `message` and `errors.New` text, **`pkg/errcode/messages.go` must use the constant** from `error_msg.go` (example: `MsgFileTooLargeUpload` ↔ `FileTooLarge`).
- **Centralized functional errors:** all reusable functional/sentinel errors (`var Err...`) and typed domain errors (`struct ...Error`) must be placed in **`pkg/errors`**. Domain modules (including `services/*`, `repository/*`, `pkg/media/*`) must import from `pkg/errors` instead of declaring duplicate error vars/types.

## Error / ErrorCode Convention (Mandatory)
- Reusable/sentinel/typed errors must be defined in `pkg/errors`.
- Message strings must be centralized in `constants` and referenced by `pkg/errcode/messages.go` (do not duplicate literal text).
- Every business/functional error must have a corresponding numeric code in `pkg/errcode/codes.go`.
- Never set error code/message directly inside `services/*`, `repository/*`, or other feature modules. Those layers only return/propagate errors from `pkg/errors` and map via `pkg/errcode`.

## Constants Placement Convention (Mandatory)
- All constants must be declared under `constants/*` (messages, status, limits, parameters, default values, error-related constants, and any other shared constant).
- Do not declare module-level business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, or other feature folders.
- If a new constant is needed, create/extend the appropriate file in `constants/` and import it where used.
- Keep `pkg/errcode/codes.go` and `pkg/errcode/messages.go` as mapping tables only; source message literals and shared numeric/enum constants must come from `constants/*` whenever applicable.

### Error Implementation Examples (for AI agents)
- **Typed provider error pattern**: follow `pkg/errors/provider_error.go` (`ProviderError`, `AsProviderError`, `HTTPStatusForProviderCode`) for upstream/provider integrations that need stable `code` mapping.
- **Sentinel upload error pattern**: follow `pkg/errors/upload_errors.go` where sentinel `Err...` uses shared message constant from `constants/error_msg.go`, and handler maps it to code in `pkg/errcode`.
- Minimal checklist when adding new errors:
  1. Add message constant in `constants/error_msg.go`.
  2. Add numeric code in `pkg/errcode/codes.go`.
  3. Wire default message in `pkg/errcode/messages.go` using that constant.
  4. Declare sentinel/typed error in `pkg/errors/*`.
  5. In handler boundary, map error -> `errcode` (never hardcode in service/repository).

## API Patterns
- Standardized response envelope via `pkg/response`.
- DTO-centric bind/validate in handlers (`dto/*`).
- List/filter APIs should rely on `dto.BaseFilter`.

## Security Patterns
- JWT-based auth with permission projection in claims.
- Route-group-level security boundaries:
  - public/authenticated/internal/system.
- Permission middleware checks by `resource:action` names.

## Data Access Patterns
- GORM as primary ORM.
- Raw SQL only when needed (RBAC joins/deletes), with named-param helper (`pkg/sqlnamed`).
- Embedded SQL migrations using `golang-migrate`.

## Helper vs Util Convention
- `pkg/logic/helper/*`: feature-scoped helpers tied to one bounded domain (e.g., media resolver/provider flow).
- `pkg/logic/utils/*`: generic, cross-feature helpers without domain coupling (e.g., raw metadata primitive conversion/image probing).
- Rule of placement: if logic can be reused across multiple modules, move to `utils`; keep orchestration-specific flow logic in `helper`.
- Import alias consistency: when helper modules use `pkg/logic/utils`, function calls must use the imported alias (`utils.*`) to avoid compile-time `undefined` errors from stale aliases such as `util.*`.
- Naming rule: common-purpose function names (e.g. parse/url/normalize/generic transformers) belong in `pkg/logic/utils`; feature-intent function names (e.g. processLearning/decodeVideo and module-specific flows) belong in `pkg/logic/helper`.

## Type Placement Convention
- From now on, for every new code written in any module under `pkg/*` that contains logic handling, all newly introduced reusable types must be created in `pkg/entities` (new file/module or existing entity module), not inline in that logic package file.
- Do not declare new reusable types inside logic-orchestration layers such as `repository/*`, `services/*`, or ad-hoc feature logic files.
- All shared/new types must be defined in **`pkg/entities`** (new module file or existing module file) and referenced from other layers.
- Exception for tests: any type created for test scope can be declared directly inside files under `tests/` when it is only used by those tests.
- Repository exception remains limited: only the repository initialization type may be declared directly in `repository/*`; other reusable/shared types still belong to `pkg/entities`.

## Tests directory (`tests/`)
- **Module-level tests** (integration or black-box packages that import `mycourse-io-be`, shared test harnesses, fixtures used across features, or any test code intentionally kept out of production packages) **belong under repository root `tests/`** (see `tests/README.md`).
- Prefer **`tests/`** when a test suite spans multiple packages or mirrors a user-facing “module” of the API rather than a single `.go` file.
- **All tests** must live under repository root `tests/`; do not add colocated `*_test.go` files next to production implementation files.

## Operational Patterns
- Sync CLI commands for RBAC catalogs.
- Optional in-memory periodic jobs for sync.
- Redis cache-aside for auth/me pathways.
