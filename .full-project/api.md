# API Contracts and Patterns


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Response Contract
- Unified JSON envelope from `pkg/response`:
  - Success: `code=0`, `message`, `data`.
  - Error: non-zero app `code`, message, optional details.
- Health endpoint has dedicated `status` shape.

## Request Validation Contract
- DTO binding tags define transport validation.
- Validation and parser errors are normalized through `pkg/httperr` middleware and `pkg/errcode`.

## Pagination/Filtering Pattern
- GET list APIs should embed `dto.BaseFilter` for consistent pagination/sort/search behavior.
- Existing list implementation pattern is in internal RBAC permission listing.

## Authorization Contract
- JWT permission claims projected into request context (`ctx_permissions`).
- `RequirePermission` enforces permission-name checks with DB fallback path.

## Internal/System Separation
- `/api/internal-v1` is internal API-key protected.
- `/api/system` is privileged system-token domain for synchronization tasks.
