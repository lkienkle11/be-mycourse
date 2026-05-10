# Module: Taxonomy

Handles classification resources: **categories**, **course levels**, and **tags**. These are lightweight reference-data domains used by the Course module to classify and filter content.

---

## Responsibility

| Domain | Description |
|--------|-------------|
| Category | Hierarchical or flat subject groupings for courses |
| Course Level | Difficulty/experience designations (e.g. Beginner, Intermediate, Advanced) |
| Tag | Free-form keyword labels attached to courses for discovery and search |

---

## Directory Layout

```
api/v1/taxonomy/
├── category_handler.go        # HTTP handlers for category CRUD
├── course_level_handler.go    # HTTP handlers for course level CRUD
├── tag_handler.go             # HTTP handlers for tag CRUD
├── handlers_common.go         # Shared generic list/create/update/delete responders
└── routes.go                  # Route registration for /api/v1/taxonomy/* (wires handlers above)

repository/taxonomy/
├── gorm_shared.go             # Shared list query + generic GORM CRUD helpers
└── repositories.go            # CategoryRepository, TagRepository, CourseLevelRepository

services/taxonomy/
├── category_service.go            # Business logic for categories (image_file_id FK + orphan cleanup via media_files)
├── fields.go                      # Trimmed name/slug/status helpers for tag/course level + category PATCH fields
└── tag_course_level_services.go   # Tag + course level list/create/update/delete services

pkg/taxonomy/
└── status.go                      # NormalizeTaxonomyStatus — maps request strings → constants.TaxonomyStatus
```

`services/taxonomy/fields.go` delegates trim/normalize helpers to **`pkg/logic/mapping`** (`TrimmedTaxonomyFields`, `ApplyOptionalTaxonomyNameSlugStatus`) which in turn use **`pkg/taxonomy`**. **Taxonomy** HTTP handlers (`api/v1/taxonomy/*_handler.go`) delegate list/create/update payloads to **`mapping.CategoryListHTTPPayload`**, **`CategoryRowHTTPPayload`**, **`TagListHTTPPayload`**, **`TagRowHTTPPayload`**, **`CourseLevelListHTTPPayload`**, **`CourseLevelRowHTTPPayload`** so **`api/`** never imports **`models`** (depguard `restrict_api`). **`CreateCategory`** builds the insert row via **`mapping.CategoryModelForCreate`**. **`services/taxonomy/*.go`** return **`models.Category`** / **`models.Tag`** / **`models.CourseLevel`**.

**Category image contract:** create/update JSON uses **`image_file_id`** (UUID of a **`media_files`** row). Responses expose nested **`image`** (`dto.MediaFilePublic`). The server validates file kind/status/MIME via **`services/media.LoadValidatedProfileImageFile`**; failures return **`pkg/errors.ErrInvalidProfileMediaFile`** (**`constants.MsgInvalidProfileMediaFile`**). Replacing or deleting a category enqueues **`EnqueueOrphanCleanupForMediaFileID`** (**`internal/jobs/media`**) for the superseded or removed file id.

---

## API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/taxonomy/categories` | `ListCategories` | List all categories |
| POST | `/api/v1/taxonomy/categories` | `CreateCategory` | Create a new category |
| GET | `/api/v1/taxonomy/categories/:id` | `GetCategory` | Get category by ID |
| PUT | `/api/v1/taxonomy/categories/:id` | `UpdateCategory` | Update category |
| DELETE | `/api/v1/taxonomy/categories/:id` | `DeleteCategory` | Delete category |
| GET | `/api/v1/taxonomy/course-levels` | `ListCourseLevels` | List all course levels |
| POST | `/api/v1/taxonomy/course-levels` | `CreateCourseLevel` | Create a course level |
| PUT | `/api/v1/taxonomy/course-levels/:id` | `UpdateCourseLevel` | Update course level |
| DELETE | `/api/v1/taxonomy/course-levels/:id` | `DeleteCourseLevel` | Delete course level |
| GET | `/api/v1/taxonomy/tags` | `ListTags` | List all tags |
| POST | `/api/v1/taxonomy/tags` | `CreateTag` | Create a new tag |
| PUT | `/api/v1/taxonomy/tags/:id` | `UpdateTag` | Update tag |
| DELETE | `/api/v1/taxonomy/tags/:id` | `DeleteTag` | Delete tag |

---

## Data Flow

```
HTTP Request
  └─ api/v1/taxonomy/*_handler.go (+ handlers_common.go)
       └─ services/taxonomy/*  (category_service + tag_course_level_services + fields)
            └─ repository/taxonomy (repositories.go + gorm_shared.go)
                 └─ models / database (Postgres via GORM)
                      └─ HTTP Response  (standard envelope: { code, message, data })
```

---

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `pkg/response` | Standard response envelope |
| `pkg/errcode` | Shared error codes |
| `middleware/auth_jwt` | JWT authentication (write operations require auth) |
| `middleware/rbac` | Permission checks for admin-only operations |

---

## Permissions

| Operation | Required Permission |
|-----------|-------------------|
| List (read) | Public or authenticated user |
| Create / Update / Delete | Admin role or specific taxonomy permission |

---

## Reusable Assets

| Asset | Type | Location | Notes |
|-------|------|----------|-------|
| Category DTO | Data type | `dto/` | Request/response shape for categories |
| CourseLevel DTO | Data type | `dto/` | Request/response shape for course levels |
| Tag DTO | Data type | `dto/` | Request/response shape for tags |
