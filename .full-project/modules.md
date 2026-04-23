# Module Responsibilities

## Implemented Modules
- **Auth module** (`api/v1/auth.go`, `services/auth.go`, `pkg/token`):
  - Register/login/confirm/refresh lifecycle.
- **User self module** (`api/v1/me.go`):
  - Profile and permission introspection for current user.
- **RBAC admin module** (`api/v1/internal_rbac.go`, `services/rbac.go`):
  - Internal CRUD for permissions/roles/user grants.
- **System operations module** (`api/system/routes.go`, `internal/jobs`, `internal/rbacsync`):
  - Privileged login, sync-now, and scheduler management.
- **Taxonomy module** (`api/v1/taxonomy/*`, `services/taxonomy/*`, `repository/taxonomy/*`, `models/taxonomy_*.go`, `dto/taxonomy_*.go`):
  - CRUD and list/filter for `course_levels`, `categories`, `tags`.
  - Uses shared list parsing helper (`pkg/query/filter_parser.go`) and shared request helpers (`pkg/requestutil/params.go`).
  - Uses permission middleware with taxonomy-specific RBAC entries (`P14`-`P25`).

## Planned But Not Implemented (per docs/modules)
- **Course module (phase 02+)**
- **Lesson module (phase 05+)**
- **Enrollment module (phase 11+)**

These planned modules currently have documentation stubs and no route/service/model/migration implementations in source code.

## Ownership and Boundaries
- Middleware + RBAC engine are shared core boundaries and high-risk to modify.
- New domain CRUD should plug into existing route/service/model patterns without changing current RBAC engine behavior.
