# Patterns and Conventions

## Architectural Patterns
- Layered monolith: `api` -> `services` -> `models`.
- Cross-cutting concerns isolated in `middleware` and `pkg/*`.
- RBAC policy-as-code through constants + DB sync.

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
- `pkg/logic/util/*`: generic, cross-feature helpers without domain coupling (e.g., raw metadata primitive conversion/image probing).
- Rule of placement: if logic can be reused across multiple modules, move to `util`; keep orchestration-specific flow logic in `helper`.

## Operational Patterns
- Sync CLI commands for RBAC catalogs.
- Optional in-memory periodic jobs for sync.
- Redis cache-aside for auth/me pathways.
