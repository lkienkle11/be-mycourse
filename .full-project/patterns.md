# Patterns and Conventions

## Architectural Patterns
- Layered monolith: `api` -> `services` -> `models`.
- Cross-cutting concerns isolated in `middleware` and `pkg/*`.
- RBAC policy-as-code through constants + DB sync.
- Shared **error/sentinel message strings** (and small related numeric caps) belong in **`constants/error_msg.go`** — see file header; numeric JSON codes stay in `pkg/errcode`. If the same sentence is both API default `message` and `errors.New` text, **`pkg/errcode/messages.go` must use the constant** from `error_msg.go` (example: `MsgFileTooLargeUpload` ↔ `FileTooLarge`).

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

## Tests directory (`tests/`)
- **Module-level tests** (integration or black-box packages that import `mycourse-io-be`, shared test harnesses, fixtures used across features, or any test code intentionally kept out of production packages) **belong under repository root `tests/`** (see `tests/README.md`).
- Prefer **`tests/`** when a test suite spans multiple packages or mirrors a user-facing “module” of the API rather than a single `.go` file.
- **Colocated `*_test.go`** next to implementation remains acceptable for small unit tests; use `tests/` when the test is clearly cross-cutting or a dedicated test module.

## Operational Patterns
- Sync CLI commands for RBAC catalogs.
- Optional in-memory periodic jobs for sync.
- Redis cache-aside for auth/me pathways.
