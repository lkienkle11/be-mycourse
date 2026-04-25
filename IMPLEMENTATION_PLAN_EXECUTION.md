# IMPLEMENTATION_PLAN_EXECUTION

## Discovery Summary
- Performed zero-code discovery only; no application source code was modified.
- Completed mandatory root-file read and protected/read-only checks.
- Ran `npx gitnexus analyze --force` before discovery and used GitNexus-backed exploration for all subsequent lanes.
- Executed 8 discovery lanes:
  - S1 folder structure/traversal
  - S2 API map
  - S3 data flow
  - S4 modules/dependencies
  - S5 DB schema+migration impact
  - S6 RBAC/permission matrix impact
  - S7 reusable-assets deep scan
  - S8 testing/validation surface
- Aggregated lane outputs and resolved conflicts:
  - Conflict: MCP instability in some lanes; resolved by fallback to GitNexus CLI with source validation.
  - Conflict: route graph extraction incomplete in GitNexus route tools; resolved using direct router/handler evidence.
- `.context/` exists but currently empty; ingestion/reconstruction/validation/integration completed as no-op with no missing artifacts.

## Folder Structure
- Full root-to-deepest tree and per-folder purpose are documented in:
  - `.full-project/folder-structure.md`
- Coverage includes workspace-level hidden folders and all source/ops folders.

## Module Responsibilities
- Current implemented domains:
  - auth
  - user self
  - internal RBAC admin
  - system operations/synchronization
- Planned domains (not yet implemented): course/lesson/enrollment + full e-learning/commerce interactions.
- Detailed module map is in:
  - `.full-project/modules.md`

## Data Flow
- Current verified end-to-end flows:
  - register
  - login
  - confirm email
  - refresh token/session
  - me profile read
  - permission check middleware fallback
  - system sync now/scheduler controls
- Detailed flow artifacts are in:
  - `.full-project/data-flow.md`
  - `.full-project/logic-flow.md`

## Related Features
- Shared security/authorization core:
  - `middleware/auth_jwt.go`
  - `middleware/rbac.go`
  - `services/rbac.go`
  - `constants/permissions.go`
  - `constants/roles_permission.go`
  - `internal/rbacsync/*`
- Shared response/error core:
  - `pkg/response/*`
  - `pkg/errcode/*`
  - `pkg/httperr/*`
- Shared transport/model patterns:
  - `dto/BaseFilter`
  - GORM + migration pipeline.

## Task Analysis

### 6.1 Objective/Constraints/Outcome
- Objective: prepare strict pre-implementation discovery + actionable implementation plan for full e-learning CRUD rollout.
- Constraints:
  - keep RBAC/middleware engine behavior intact
  - no role additions/removals
  - no mutation of protected/read-only reference docs
  - no coding before explicit approval
- Expected outcome: phase-by-phase plan with reuse-first mapping and exact change surfaces.

### 6.2 System Mapping (Detailed For Phase 01-03)

#### Phase 01 - Taxonomy (course levels, categories, tags)
- **Files to add (planned):**
  - `migrations/000002_taxonomy_domain.up.sql`
  - `migrations/000002_taxonomy_domain.down.sql`
  - `dbschema/taxonomy.go`
  - `models/taxonomy/course_level.go`
  - `models/taxonomy/category.go`
  - `models/taxonomy/tag.go`
  - `dto/taxonomy/course_level_dto.go`
  - `dto/taxonomy/category_dto.go`
  - `dto/taxonomy/tag_dto.go`
  - `repositories/taxonomy/course_level_repository.go`
  - `repositories/taxonomy/category_repository.go`
  - `repositories/taxonomy/tag_repository.go`
  - `services/taxonomy/course_level_service.go`
  - `services/taxonomy/category_service.go`
  - `services/taxonomy/tag_service.go`
  - `api/v1/taxonomy/course_level_handler.go`
  - `api/v1/taxonomy/category_handler.go`
  - `api/v1/taxonomy/tag_handler.go`
  - `api/v1/taxonomy/routes.go`
  - `pkg/query/filter_parser.go` (new reusable parsing/whitelist helper)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount taxonomy sub-routes from `api/v1/taxonomy/routes.go`)
  - `repositories/repository.go` (refactor repository root module to host taxonomy/course/course_edit repository composition and constructor wiring)
  - `constants/permissions.go` (add taxonomy permission IDs/names)
  - `constants/roles_permission.go` (map taxonomy permissions to roles)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/taxonomy/course_level_repository.go`:
    - `ListCourseLevels(...)`
    - `CreateCourseLevel(...)`
    - `GetCourseLevelByID(...)`
    - `UpdateCourseLevel(...)`
    - `DeleteCourseLevel(...)`
  - `repositories/taxonomy/category_repository.go`:
    - `ListCategories(...)`, `CreateCategory(...)`, `GetCategoryByID(...)`, `UpdateCategory(...)`, `DeleteCategory(...)`
  - `repositories/taxonomy/tag_repository.go`:
    - `ListTags(...)`, `CreateTag(...)`, `GetTagByID(...)`, `UpdateTag(...)`, `DeleteTag(...)`
  - `services/taxonomy/course_level_service.go`:
    - `ListCourseLevels(...)`, `CreateCourseLevel(...)`, `UpdateCourseLevel(...)`, `DeleteCourseLevel(...)`
  - `services/taxonomy/category_service.go`:
    - `ListCategories(...)`, `CreateCategory(...)`, `UpdateCategory(...)`, `DeleteCategory(...)`
  - `services/taxonomy/tag_service.go`:
    - `ListTags(...)`, `CreateTag(...)`, `UpdateTag(...)`, `DeleteTag(...)`
  - `api/v1/taxonomy/*_handler.go`:
    - `listCourseLevels`, `createCourseLevel`, `updateCourseLevel`, `deleteCourseLevel`
    - `listCategories`, `createCategory`, `updateCategory`, `deleteCategory`
    - `listTags`, `createTag`, `updateTag`, `deleteTag`
  - `pkg/query/filter_parser.go`:
    - `ParseListFilter(...)`
    - `BuildSortClause(...)`
    - `BuildSearchClause(...)`
- **Types to implement (planned):**
  - `models/taxonomy/course_level.go`: `CourseLevel`
  - `models/taxonomy/category.go`: `Category`
  - `models/taxonomy/tag.go`: `Tag`
  - `dto/taxonomy/course_level_dto.go`: `CreateCourseLevelRequest`, `UpdateCourseLevelRequest`, `CourseLevelFilter`, `CourseLevelResponse`
  - `dto/taxonomy/category_dto.go`: `CreateCategoryRequest`, `UpdateCategoryRequest`, `CategoryFilter`, `CategoryResponse`
  - `dto/taxonomy/tag_dto.go`: `CreateTagRequest`, `UpdateTagRequest`, `TagFilter`, `TagResponse`
  - `repositories/repository.go`: `Repository` root struct refactor with domain repository members + constructor dependency injection shape
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - `dto.BaseFilter`
    - `middleware.RequirePermission`
    - `pkg/response` + `pkg/errcode` + `pkg/httperr`
    - `pkg/sqlnamed.Postgres` (nếu dùng raw query)
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/query/filter_parser.go` (không đặt trong service layer);
    - Phase 02/03 bắt buộc dùng lại asset này thay vì tạo helper mới trong service.

#### Phase 02 - Course shell (course base CRUD + list/filter)
- **Files to add (planned):**
  - `migrations/000003_course_shell.up.sql`
  - `migrations/000003_course_shell.down.sql`
  - `dbschema/course.go`
  - `models/course/course.go`
  - `dto/course/course_dto.go`
  - `repositories/course/course_repository.go`
  - `services/course/course_service.go`
  - `api/v1/course/course_handler.go`
  - `api/v1/course/routes.go`
  - `pkg/policy/course_policy.go` (new reusable ownership/role policy asset)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount `api/v1/course/routes.go`)
  - `repositories/repository.go` (extend root repository module with `course` repository member and builder wiring)
  - `constants/permissions.go` (if missing required course shell perms)
  - `constants/roles_permission.go` (course-shell mapping)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/course/course_repository.go`:
    - `CreateCourse(...)`
    - `GetCourseByID(...)`
    - `ListCourses(...)`
    - `UpdateCourse(...)`
    - `SoftDeleteCourse(...)`
  - `services/course/course_service.go`:
    - `CreateCourse(instructorID uint, req dto.CreateCourseRequest) (...)`
    - `GetCourseByID(courseID uint, actorID uint, roleSet map[string]struct{}) (...)`
    - `ListCourses(filter dto.CourseFilter, actorID uint, roleSet map[string]struct{}) (...)`
    - `UpdateCourse(courseID uint, actorID uint, req dto.UpdateCourseRequest) (...)`
    - `DeleteCourse(courseID uint, actorID uint) error`
  - `api/v1/course/course_handler.go`:
    - `createCourse`, `getCourse`, `listCourses`, `updateCourse`, `deleteCourse`
  - `pkg/policy/course_policy.go`:
    - `CanManageCourse(actorID uint, courseInstructorID uint, roleSet map[string]struct{}) bool`
    - `CanViewCourse(actorID uint, courseInstructorID uint, roleSet map[string]struct{}, isPublished bool) bool`
- **Types to implement (planned):**
  - `models/course/course.go`: `Course`
  - `dto/course/course_dto.go`: `CreateCourseRequest`, `UpdateCourseRequest`, `CourseFilter`, `CourseResponse`
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - taxonomy IDs/relations từ Phase 01 models
    - `dto.BaseFilter`, response/error stack, permission middleware, `pkg/query/filter_parser.go`
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/policy/course_policy.go` (shared policy layer);
    - service chỉ gọi policy + repository, không chứa reusable policy helper.

#### Phase 03 - Course edits/versioning (draft/edit lifecycle)
- **Files to add (planned):**
  - `migrations/000004_course_edits.up.sql`
  - `migrations/000004_course_edits.down.sql`
  - `dbschema/course_edit.go`
  - `models/course_edit/course_edit.go`
  - `dto/course_edit/course_edit_dto.go`
  - `repositories/course_edit/course_edit_repository.go`
  - `services/course_edit/course_edit_service.go`
  - `api/v1/course_edit/course_edit_handler.go`
  - `api/v1/course_edit/routes.go`
  - `pkg/workflow/course_edit_state_machine.go` (new reusable state transition asset)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount `api/v1/course_edit/routes.go`)
  - `services/course/course_service.go` (if publishing flow updates `published_edit_id`)
  - `repositories/repository.go` (extend root repository module with `course_edit` repository member and builder wiring)
  - `constants/permissions.go` and `constants/roles_permission.go` (approval/versioning perms for admin/instructor if missing)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/course_edit/course_edit_repository.go`:
    - `CreateCourseEdit(...)`
    - `GetCourseEditByID(...)`
    - `ListCourseEdits(...)`
    - `UpdateCourseEdit(...)`
    - `SetCourseEditStatus(...)`
    - `PublishCourseEdit(...)`
  - `services/course_edit/course_edit_service.go`:
    - `CreateCourseEdit(courseID uint, actorID uint, req dto.CreateCourseEditRequest) (...)`
    - `GetCourseEdit(editID uint, actorID uint) (...)`
    - `ListCourseEdits(courseID uint, actorID uint, filter dto.CourseEditFilter) (...)`
    - `UpdateCourseEdit(editID uint, actorID uint, req dto.UpdateCourseEditRequest) (...)`
    - `SubmitCourseEdit(editID uint, actorID uint) error`
    - `ApproveCourseEdit(editID uint, adminID uint, req dto.ApproveCourseEditRequest) error`
    - `RejectCourseEdit(editID uint, adminID uint, req dto.RejectCourseEditRequest) error`
  - `api/v1/course_edit/course_edit_handler.go`:
    - `createCourseEdit`, `getCourseEdit`, `listCourseEdits`, `updateCourseEdit`, `submitCourseEdit`, `approveCourseEdit`, `rejectCourseEdit`
  - `pkg/workflow/course_edit_state_machine.go`:
    - `ValidateCourseEditTransition(from, to string) error`
    - `IsTerminalCourseEditState(state string) bool`
- **Types to implement (planned):**
  - `models/course_edit/course_edit.go`: `CourseEdit`
  - `dto/course_edit/course_edit_dto.go`: `CreateCourseEditRequest`, `UpdateCourseEditRequest`, `CourseEditFilter`, `ApproveCourseEditRequest`, `RejectCourseEditRequest`, `CourseEditResponse`
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - `CanManageCourse` + `CanViewCourse` (from `pkg/policy/course_policy.go`)
    - `pkg/query/filter_parser.go`
    - response/error/validation stack
    - `dto.BaseFilter`
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/workflow/course_edit_state_machine.go` (không đặt trong service);
    - các phase sau (section/lesson/quiz workflow) phải tái sử dụng hoặc mở rộng module workflow này.

#### Repository Root Refactor (applies across Phase 01-03)
- **Target file:** `repositories/repository.go`
- **Refactor objective:**
  - tách hoàn toàn repository layer khỏi `models/`,
  - đặt repository layer tại `repositories/<domain>/`,
  - dùng `repositories/repository.go` làm composition root cho tất cả domain repositories.
- **Planned refactor items:**
  - convert current minimal `Repository` into a structured container with domain members (`taxonomy`, `course`, `course_edit`).
  - add constructor wiring that initializes each domain repository with shared DB dependency.
  - expose typed accessors (or public fields) so services resolve repositories via the root module instead of creating ad-hoc repository instances.
  - keep backward compatibility path for existing code using `NewRepository()` via repository root module during transition.
- **Planned outputs tied to this refactor:**
  - Phase 01 registers taxonomy repository set.
  - Phase 02 extends root repository with course repository.
  - Phase 03 extends root repository with course-edit repository.

### Non-target Modules (unless mandatory minimal touch)
- core auth middleware internals
- system token middleware engine
- queue placeholder logic

### 6.3 Cross-check with `.full-project`
- Cross-checked all planning assumptions with newly written snapshot files.
- Reuse baseline is anchored in `.full-project/reusable-assets.md`.

### 6.3.1 Reusability Check (Mandatory)
- Existing reusable foundations confirmed:
  - `dto.BaseFilter`
  - `RequirePermission`
  - response/error envelopes
  - RBAC permission resolution/service patterns
  - SQL named-parameter helper and schema namespace pattern.

### 6.3.2 Reuse Enforcement
- Any phase implementation must reuse existing shared assets where available.
- New shared logic is allowed only when a genuine gap exists and must be placed in proper module/folder (no dumping into one utility file).

### 6.4 Technical Direction
- Add e-learning schema via staged migrations.
- Add domain models/DTO/services/routes per phase complexity.
- Extend RBAC catalog with `P14+` and role mappings without altering role identities.
- Keep ownership/permission checks at service + middleware boundary.

## Reusability Strategy

### Existing Assets To Reuse
- Authorization and session:
  - `AuthJWT`, `RequirePermission`, `PermissionCodesForUser`, `UserHasAllPermissions`.
- API contracts:
  - `pkg/response`, `pkg/errcode`, `pkg/httperr`.
- List/query patterns:
  - `dto.BaseFilter`, sort/search whitelist approach.
- SQL utility patterns:
  - `pkg/sqlnamed.Postgres`, `dbschema` namespace approach.

### New Reusable Assets To Create (When Implementing)
- Domain schema namespaces under `dbschema/` per new bounded context.
- Ownership-check helpers for instructor/learner/admin permissions.
- Shared DTO fragments for pagination/filtering and common identifiers.
- Shared query builders for reusable complex list/search operations.

### Reusability Validation Before Implement
- For each phase core, map each CRUD/query to:
  - existing reusable asset (reuse),
  - or new reusable asset (create once, reuse later).
- Update `.full-project/reusable-assets.md` whenever reusable logic is introduced/changed.

## CRUD/Query Mapping (Phase 01-12)

| Phase | Domain | Planned CRUD/Query | Reuse Now | New Asset Needed |
|---|---|---|---|---|
| 01 | Taxonomy | levels/categories/tags CRUD + list/filter | `BaseFilter`, `RequirePermission`, response/error stack | taxonomy schema/model/service/route set |
| 02 | Course shell | course CRUD + list/filter | same as above + ownership check pattern | course base model/DTO/service |
| 03 | Course edits/versioning | edits CRUD + state transitions | auth/RBAC services + response stack | edit state machine helpers |
| 04 | Metadata+junction | category/tag/level binding queries | `sqlnamed` pattern, list patterns | reusable join query helpers |
| 05 | Sections/lessons | tree CRUD + reorder | response/error + permission gates | ordering/reorder shared helper |
| 06 | Text/video/subtitle | content CRUD + media metadata | validation patterns + ownership checks | content payload helpers |
| 07 | Quiz authoring | nested quiz/question/choice CRUD | dto/validation patterns | nested aggregate transaction helper |
| 08 | Course series | series CRUD + ordered course mapping | list/filter + RBAC checks | ordered M:N helper |
| 09 | Coupons/scope | coupon CRUD + condition queries | sql helper + error codes | scope predicate/query builder |
| 10 | Orders/items | order/item CRUD + aggregates | response pagination + auth context | order aggregate query helpers |
| 11 | Enrollments | enrollment CRUD + ownership checks | permission + ownership patterns | enrollment uniqueness helper |
| 12 | Progress/attempts/reviews | progress/review CRUD + analytics queries | auth/session + list patterns | progress/review shared computation helpers |

## Action Plan (Pre-Approved Design Only)

### Planned File Surfaces By Layer
- **Migrations**
  - Add new migration pairs from `000002_*` onward in `migrations/`.
  - Estimated LoC (SQL): 1800-3000 across all phases.
- **Models**
  - Add domain models in `models/` for each phase domain.
  - Estimated LoC: 800-1400.
- **Repositories (separated from models)**
  - Add domain repositories in `repositories/<domain>/`.
  - Refactor `repositories/repository.go` as repository composition root and constructor wiring module.
  - Estimated LoC: 500-900.
- **DTO**
  - Add request/query/response DTOs in `dto/`.
  - Estimated LoC: 700-1200.
- **Services**
  - Add domain business logic files in `services/`.
  - Estimated LoC: 1800-3200.
- **Routes/Handlers**
  - Add/extend route registration and handlers in `api/v1/`.
  - Estimated LoC: 1200-2200.
- **RBAC catalog**
  - Extend permission constants and role mappings:
    - `constants/permissions.go`
    - `constants/roles_permission.go`
  - Estimated LoC: 150-300.
- **Sync/ops usage**
  - Reuse existing sync commands and document run-order after permission updates.

### Planned Logic Sequence Per Phase Core
1. DDL/migration changes.
2. Model + Repository + DTO definitions.
3. Service logic (ownership, business constraints, repository orchestration).
4. Route wiring + middleware permissions.
5. Query optimization and list/filter behavior.
6. Documentation sync (`.full-project` + `.context`).

## Discovery Phases (1->5) and Output Artifacts
- Phase 1 Architecture (S1+S4):
  - `.full-project/architecture.md`
  - `.full-project/folder-structure.md`
- Phase 2 Documentation (S7):
  - `.full-project/modules.md`
  - `.full-project/patterns.md`
  - `.full-project/reusable-assets.md`
- Phase 3 API (S2+S6):
  - `.full-project/api-overview.md`
  - `.full-project/router.md`
  - `.full-project/api.md`
- Phase 4 Data flow (S3+S8):
  - `.full-project/data-flow.md`
  - `.full-project/logic-flow.md`
- Phase 5 Targeted code reading (S5 + hot paths):
  - `.full-project/dependencies.md`
  - DB/RBAC impact captured in this plan and reusable inventory.

## Post-Approval Discipline (Reminder)
- After explicit approval only:
  - maintain strict phase order (`phase-NN-start -> core -> end`)
  - keep docs synchronized in markdown files (not chat-only)
  - update reusable-assets whenever shared logic is added/changed
  - run lint/type/build checks and stop immediately on unresolved failures
  - no assumption-driven or partial implementation on blockers.

## Phase Execution Status

### Phase 01 (Taxonomy) - END CHECKPOINT COMPLETED
- Review status:
  - Verified Phase 01 scope includes migration, model, DTO, repository, service, and route wiring for taxonomy CRUD.
  - Verified RBAC extensions for taxonomy permissions (`P14`-`P25`) and role mapping integration.
- Test status:
  - Executed `go test ./...` successfully on 2026-04-23 after Phase 01 implementation.
  - Result: pass (no package test failures; most packages currently have no test files).
- Documentation sync status:
  - Updated `.full-project/api-overview.md` with taxonomy endpoint inventory.
  - Updated `.full-project/router.md` with taxonomy route registration topology.
  - Updated `.full-project/data-flow.md` with taxonomy CRUD flow.
  - Updated `.full-project/modules.md` to mark taxonomy as implemented.
  - Updated `.full-project/reusable-assets.md` with newly introduced reusable helpers.
- Reusable-assets closure:
  - Added reusable assets for `pkg/query/filter_parser.go` and `pkg/requestutil/params.go`.
  - Updated gap analysis to remove taxonomy from missing-domain list.

### Next Gate
- Phase 01 is closed.
- Ready to begin `phase-02-start` only after this checkpoint is accepted.

## Phase Sub 01 Execution Update (2026-04-25)

### Task 01 / Task 04 - Continuous Discovery + Sync
- Re-read active context from `.context/` and existing implementation snapshot docs.
- Re-ran GitNexus indexing with `gitnexus analyze --force` before making refactor changes.
- Performed GitNexus impact checks for symbols touched in this sub-phase (`SetupRedis`, `CourseLevel`, `Category`, `taxonomyNS`) and proceeded with only LOW-risk results.
- Synced this file with the exact refactor surfaces completed in Task 02 and Task 03.

### Task 02 - Move `cache_clients` into `pkg` + shared entities example
- Moved Redis client package from `cache_clients/redis.go` to `pkg/cache_clients/redis.go`.
- Updated all imports referencing the old location:
  - `main.go`
  - `services/cache/auth_user.go`
- Introduced shared entities package `core/entities` and migrated two taxonomy modules as reusable examples:
  - `core/entities/course_level.go`
  - `core/entities/category.go`
- Models remain the persistence layer and own table mapping (`TableName`) while reusing shared fields from core entities:
  - `models/taxonomy_course_level.go`
  - `models/taxonomy_category.go`

### Task 03 - Split `dbschema/taxonomy.go`
- Split taxonomy schema namespace into focused files under `dbschema/`:
  - `taxonomy_namespace.go`
  - `taxonomy_course_levels.go`
  - `taxonomy_categories.go`
  - `taxonomy_tags.go`
- Removed the previous monolithic file `dbschema/taxonomy.go`.
- Existing call sites remain stable (`dbschema.Taxonomy.*`) so no consumer import changes were required.

### Validation
- Executed `go test ./...` successfully after all refactors.

## Phase Sub 01 Rework Update (2026-04-25 - sync 2)

### Request-driven architecture correction
- Enforced strict boundary: `core/entities` now contains only pure type definitions (no table-name mapping, no dbschema dependency).
- Moved table mapping responsibility back to models only:
  - `models/taxonomy_course_level.go` defines model wrapper + `TableName()`.
  - `models/taxonomy_category.go` defines model wrapper + `TableName()`.
- Updated taxonomy service/repository layers to consume `models` types again so ORM-facing flows stay inside model boundary.

### Subtask 01 + Subtask 04 sync loop
- Re-ran `gitnexus analyze --force`.
- Re-read `.context` and execution plan documentation before rework.
- Ran impact checks for touched symbols (`CourseLevel`, `Category`) and continued with LOW risk only.
- Re-synchronized docs in `.full-project/*` and this execution plan to reflect corrected architecture boundary.

### Validation for rework
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 3)

### Task 02 (requested in this cycle) - Extract pagination math to shared utils
- Added reusable pagination builder:
  - `pkg/logic/utils/pagination.go` with `utils.BuildPage(page, perPage, totalItems)`.
- Refactored handlers to stop manual `totalPages` math and use shared utility:
  - `api/v1/taxonomy/course_level_handler.go`
  - `api/v1/taxonomy/category_handler.go`
  - `api/v1/taxonomy/tag_handler.go`
  - `api/v1/internal_rbac.go`

### Task 03 (requested in this cycle) - Apply refactor across all manual page calculations
- Completed cross-module replacement for all current manual pagination calculations in API handlers.
- Confirmed no remaining manual `totalPages` blocks outside shared utility implementation.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force` and impact checks before/around edits.
- Re-read context and synchronized docs after implementation.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 7, remove redundant wrappers)

### Task 02 + Task 03 (redefined for this cycle) - Direct util/helper usage in internal RBAC handler
- Removed redundant local wrappers from `api/v1/internal/rbac_handler.go`:
  - `parseUintParam(...)`
  - `parsePermissionIDParam(...)`
- Updated all call-sites in the same file to call shared helpers directly:
  - `utils.ParseUintPathParam(...)`
  - `helper.ParsePermissionIDParam(...)`
- This keeps one source of truth and avoids thin pass-through wrappers.

### Risk handling note
- GitNexus impact before change:
  - wrapper `parseUintParam`: CRITICAL
  - wrapper `parsePermissionIDParam`: HIGH
- Mitigation applied: wrapper removal only; preserved parsing behavior by direct calls to the same underlying helpers.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force`.
- Re-ran impact checks and validated no behavior regression with test/build.
- Synchronized `.full-project/reusable-assets.md` and this execution plan.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 6, param util/helper extraction)

### Task 02 + Task 03 (redefined for this cycle) - Extract repeated param parsing logic
- User-requested logic from old `api/v1/internal_rbac.go` was extracted and reused in current internal module:
  - uint path param parser -> `pkg/logic/utils/params.go` (`ParseUintPathParam`)
  - permission id parser -> `pkg/logic/helper/permission.go` (`ParsePermissionIDParam`)
- Refactored internal RBAC handlers to use shared helpers:
  - `api/v1/internal/rbac_handler.go`
- Refactored existing shared request util to delegate to new common parser:
  - `pkg/requestutil/params.go` now uses `utils.ParseUintPathParam`, extending reuse to taxonomy handlers.

### Risk handling note
- GitNexus impact:
  - `parseUintParam`: CRITICAL (many direct dependents)
  - `parsePermissionIDParam`: HIGH
  - `pkg/requestutil.ParseUintParam`: CRITICAL
- Mitigation applied: preserved external behavior and only replaced duplicated parsing internals with shared helpers.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force` before edits.
- Performed impact analysis for all modified symbols.
- Updated `.full-project/reusable-assets.md` and this plan to keep docs synchronized with code changes.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 4, user-requested redo)

### Task 02 + Task 03 (redefined for this cycle) - Add `Tag` into core entity and refactor
- Added pure shared entity:
  - `core/entities/tag.go`
- Refactored `models/taxonomy_tag.go` to keep ORM table mapping in model and reuse core entity fields via embedding.
- Updated `services/taxonomy/tag_service.go` to initialize nested entity payload (`Tag: entities.Tag{...}`) for create flow.

### Task 01 + Task 04 loop for this redo
- Re-ran `gitnexus analyze --force` before the refactor cycle.
- Ran impact checks for `Tag`, `CreateTag`, and related model surfaces, then applied changes with scope control.
- Re-synced `.full-project/reusable-assets.md` and this plan after code updates.

### Validation for this redo
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 5, internal route modularization)

### Task 02 + Task 03 (redefined for this cycle) - Move internal RBAC API into `api/v1/internal`
- Moved internal RBAC handlers from monolithic `api/v1/internal_rbac.go` into:
  - `api/v1/internal/rbac_handler.go`
- Added internal route module:
  - `api/v1/internal/routes.go`
- Removed old file:
  - `api/v1/internal_rbac.go`
- Moved old route block (from `api/v1/routes.go` lines 35-57) into `api/v1/internal/routes.go`.
- Kept compatibility with current router wiring by exposing a thin wrapper in `api/v1/routes.go`:
  - `RegisterInternalRoutes(rg)` delegates to `internalv1.RegisterRoutes(rg)`.

### Risk handling note
- GitNexus impact marked `parseUintParam` as CRITICAL (many direct dependents inside internal handlers).
- Mitigation applied: pure structural move without changing internal handler behavior/response contracts.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force`.
- Ran impact checks before edit (`RegisterInternalRoutes`, `parseUintParam`).
- Re-synced `.full-project` docs and this execution plan after refactor.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)
