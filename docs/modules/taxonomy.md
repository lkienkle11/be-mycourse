# Taxonomy Module

The taxonomy module (`internal/taxonomy/`) handles classification reference data for courses: **categories**, **course levels**, and **tags**.

---

## Directory Layout

```
internal/taxonomy/
├── domain/
│   └── (entity types)
├── application/
│   └── taxonomy_service.go      # TaxonomyService: CRUD for categories, tags, course levels
├── infra/
│   ├── gorm_category_repo.go
│   ├── gorm_tag_repo.go
│   ├── gorm_course_level_repo.go
│   └── gorm_shared.go           # Shared list query helpers
└── delivery/
    ├── handler.go                # HTTP handlers for all three sub-domains
    ├── routes.go                 # Route registration under /api/v1/taxonomy
    ├── dto.go                    # Request/response DTOs
    └── mapping.go                # Domain → DTO mapping
```

---

## Responsibilities

| Domain | Description |
|--------|-------------|
| **Category** | Hierarchical or flat subject groupings for courses. Optionally linked to a media file (`image_file_id` FK into `media_files`) |
| **Course Level** | Difficulty designations (e.g. Beginner, Intermediate, Advanced) |
| **Tag** | Free-form keyword labels for discovery and search |

---

## API Endpoints

All routes are under `/api/v1/taxonomy/` and require `Authorization: Bearer <token>`. Write operations require the appropriate permission.

### Course Levels

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/levels` | `course_level:read` | List all course levels |
| POST | `/taxonomy/levels` | `course_level:create` | Create a new course level |
| PATCH | `/taxonomy/levels/:id` | `course_level:update` | Update a course level |
| DELETE | `/taxonomy/levels/:id` | `course_level:delete` | Delete a course level |

### Categories

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/categories` | `category:read` | List all categories (paginated) |
| POST | `/taxonomy/categories` | `category:create` | Create a new category |
| PATCH | `/taxonomy/categories/:id` | `category:update` | Update a category |
| DELETE | `/taxonomy/categories/:id` | `category:delete` | Delete a category |

### Tags

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/tags` | `tag:read` | List all tags (paginated) |
| POST | `/taxonomy/tags` | `tag:create` | Create a new tag |
| PATCH | `/taxonomy/tags/:id` | `tag:update` | Update a tag |
| DELETE | `/taxonomy/tags/:id` | `tag:delete` | Delete a tag |

---

## Category Image Contract

Categories can have an associated image via `image_file_id` (UUID of a `media_files` row).

- **Create/Update:** JSON body includes `image_file_id`.
- **Validation:** The server validates the referenced file's kind, status, and MIME type via the `MediaFileValidator` interface (injected from `MediaService.LoadValidatedProfileImageFile`).
- **Response:** Includes a nested `image` object with public file fields.
- **Orphan cleanup:** When `image_file_id` is replaced or the category is deleted, the old file ID is enqueued for deferred cloud deletion via `OrphanEnqueuer.EnqueueOrphanCleanupForFileID`.

---

## Data Flow

```
HTTP Request
  └─ internal/taxonomy/delivery/handler.go  (bind DTO, validate)
       └─ internal/taxonomy/application/TaxonomyService  (business logic)
            ├─ (image validation) → MediaService via MediaFileValidator interface
            ├─ (orphan cleanup)  → OrphanEnqueuer via OrphanImageEnqueuer interface
            └─ internal/taxonomy/infra/  (GORM repositories)
                 └─ PostgreSQL
                      └─ HTTP Response (standard envelope)
```

---

## Cross-Domain Dependencies

Taxonomy depends on two other domains, both injected as interfaces in `internal/server/wire.go`:

| Interface | Implemented by | Purpose |
|-----------|---------------|---------|
| `MediaFileValidator` | `MediaService.LoadValidatedProfileImageFile` | Validate `image_file_id` points to a usable image file |
| `OrphanImageEnqueuer` | `OrphanEnqueuer.EnqueueOrphanCleanupForFileID` | Queue old image files for deferred cloud deletion |

---

## Implementation Reference

| Concern | Location |
|---------|----------|
| TaxonomyService | `internal/taxonomy/application/taxonomy_service.go` |
| GORM repositories | `internal/taxonomy/infra/` |
| HTTP handlers | `internal/taxonomy/delivery/handler.go` |
| Route registration | `internal/taxonomy/delivery/routes.go` |
| DTOs | `internal/taxonomy/delivery/dto.go` |
| Cross-domain adapters | `internal/server/wire.go` (`taxMediaFileValidator`, `taxOrphanEnqueuer`) |
| DB migrations | `migrations/000002_taxonomy_domain.*` |
