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
├── category_handler.go       # HTTP handlers for category CRUD
├── course_level_handler.go   # HTTP handlers for course level CRUD
├── tag_handler.go            # HTTP handlers for tag CRUD
└── routes.go                 # Route registration for /api/v1/taxonomy/*

services/taxonomy/
├── category_service.go       # Business logic for categories
├── course_level_service.go   # Business logic for course levels
└── tag_service.go            # Business logic for tags
```

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
  └─ api/v1/taxonomy/*_handler.go   (validate input, bind DTO)
       └─ services/taxonomy/*_service.go  (business rules, DB queries)
            └─ models / database (Postgres via GORM or raw SQL)
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
