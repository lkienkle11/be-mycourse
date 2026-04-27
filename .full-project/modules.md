# Module Responsibilities

## Implemented Modules
- **Auth module** (`api/v1/auth.go`, `services/auth.go`, `pkg/token`):
  - Register/login/confirm/refresh lifecycle.
- **User self module** (`api/v1/me.go`):
  - Profile and permission introspection for current user.
- **RBAC admin module** (`api/v1/internal/*`, `services/rbac.go`):
  - Internal CRUD for permissions/roles/user grants.
- **System operations module** (`api/system/routes.go`, `internal/jobs`, `internal/rbacsync`):
  - Privileged login, sync-now, and scheduler management.
- **Taxonomy module** (`api/v1/taxonomy/*`, `services/taxonomy/*`, `repository/taxonomy/*`, `models/taxonomy_*.go`, `dto/taxonomy_*.go`):
  - CRUD and list/filter for `course_levels`, `categories`, `tags`.
  - Uses shared list parsing helper (`pkg/query/filter_parser.go`) and shared request helpers (`pkg/requestutil/params.go`).
  - Uses permission middleware with taxonomy-specific RBAC entries (`P14`-`P25`).
- **Media upload module** (`api/v1/media/*`, `services/media/*`, `dto/media_file.go`, `pkg/entities/file.go`, `models/media_file.go`, `repository/media/file_repository.go`):
  - Unified upload/file API for file + video branches with methods `GET/POST/PUT/DELETE/OPTIONS`, plus `GET /media/videos/:id/status` and public `POST /webhook/bunny`.
  - Uses provider clients/adapters in `pkg/media/*` for Local/B2/Gcore/Bunny URL generation and cloud upload.
  - Provider source-of-truth is server config (`setting.MediaSetting.AppMediaProvider`), not client request payload/query.
  - Uses shared resolver helpers in `pkg/logic/helper/media_resolver.go`; service layer remains orchestration-only.
  - Uses `helper.DefaultMediaProvider` from `pkg/logic/helper/media_metadata.go`; service no longer owns provider default helper.
  - Generic metadata parsing primitives are extracted to `pkg/logic/utils/media_metadata.go` and reused by media helper.
  - Uses mapper helpers in `pkg/logic/mapping` so handlers always return DTO (`dto.UploadFileResponse`) instead of raw entity.
  - Uses helper `pkg/logic/helper/DecodeLocalURLToken` for local token decode (no non-CRUD decode utility in service layer).
  - Metadata is inferred by backend and returned as typed metadata (`ImageMetadata`, `VideoMetadata`, `DocumentMetadata`) from `pkg/entities/file.go`.
  - SDK clients are initialized at app startup via `pkg/media.Setup()` in `main.go`.
  - Persists upload metadata in `media_files` and syncs create/update/delete + webhook duration updates.
  - DB model<->entity mapping is handled in `pkg/logic/mapping/media_model_mapping.go` (not inside service layer).
  - Uses permission middleware with media RBAC entries (`P26`-`P29`).
  - Converts Bunny numeric video status to stable API strings via `pkg/logic/utils/bunny_status.go`.
  - Enforces **2 GiB** max per uploaded `file` part (`constants.MaxMediaUploadFileBytes` in **`constants/error_msg.go`**); handler + service guards; HTTP **413** + `errcode.FileTooLarge` (**2003**) on violation (see `docs/modules/media.md`, `docs/deploy.md` for proxy sizing).

## Planned But Not Implemented (per docs/modules)
- **Course module (phase 02+)**
- **Lesson module (phase 05+)**
- **Enrollment module (phase 11+)**

These planned modules currently have documentation stubs and no route/service/model/migration implementations in source code.

## Ownership and Boundaries
- Middleware + RBAC engine are shared core boundaries and high-risk to modify.
- New domain CRUD should plug into existing route/service/model patterns without changing current RBAC engine behavior.

## Testing (repository-wide)

- **All tests** (unit/module-level/integration) and shared harnesses: repository root **`tests/`** — see `tests/README.md` and `.full-project/patterns.md`.
