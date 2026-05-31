# Taxonomy Module

The taxonomy module (`internal/taxonomy/`) handles classification reference data for courses: **course topics** (formerly categories), **course outcomes**, **course skills**, **course levels**, and **tags**.

---

## Directory Layout

```
internal/taxonomy/
├── domain/
│   ├── taxonomy.go          # Entities and input types
│   └── repository.go        # Repository interfaces (incl. generic list patterns)
├── application/
│   ├── service.go           # TaxonomyService use-cases
│   └── service_helpers.go   # Shared create/update/delete, slug+status entities, orphan image hook
├── infra/
│   ├── repos.go             # GORM repositories (topics, outcomes, skills, tags, levels)
│   ├── repos_crud_helper.go # Shared list/create helpers to avoid duplicated repo code
│   └── jsonb_types.go       # JSONB scanners for tree nodes and description arrays
└── delivery/
    ├── handler.go           # Thin handlers per resource
    ├── handler_helpers.go   # listTaxonomyItems, shared mutation error mapping
    ├── routes.go
    └── dto.go

internal/shared/taxonomy/
├── tree_node.go             # TreeNode JSON shape
├── tree_validate.go         # Depth, node count, UUID id, duplicate slug checks
└── description_validate.go  # Outcome description paragraph limits
```

List endpoints use **`internal/shared/httpx.ListPaginated`** where applicable (same pattern as media list).

---

## Responsibilities

| Domain | Description |
|--------|-------------|
| **CourseTopic** | Subject groupings for courses (`course_topics` table). Optional `image_file_id` FK and nested `child_topics` JSONB tree |
| **CourseOutcome** | Learning outcomes with `short_description`, `description` (string array JSONB), optional image |
| **CourseSkill** | Skill taxonomy root row with nested `children` JSONB tree (same node shape as `child_topics`) |
| **CourseLevel** | Difficulty designations (e.g. Beginner, Intermediate, Advanced) |
| **Tag** | Free-form keyword labels for discovery and search |

---

## API Endpoints

All routes are under `/api/v1/taxonomy/` and require `Authorization: Bearer <token>`.

List query contract:
- `page`, `per_page`, `sort_by`, `sort_desc`, optional `status`
- typed search: `search_by` + `search_value`
- allowed `search_by` values:
  - levels/topics/skills/tags: `name`, `slug`
  - outcomes: `short_description`

### Course Topics (replaces `/categories`)

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/topics` | `topic:read` | List active topics (paginated) |
| GET | `/taxonomy/topics/full` | `topic:read` | List topics including soft-deleted |
| POST | `/taxonomy/topics` | `topic:create` | Create topic |
| PATCH | `/taxonomy/topics/:id` | `topic:update` | Update topic (active only) |
| DELETE | `/taxonomy/topics/:id` | `topic:delete` | Soft-delete topic |
| DELETE | `/taxonomy/topics/:id/hard` | `topic:delete` | Hard-delete topic (+ orphan image cleanup) |

**Body fields:** `name`, `slug`, `status`, optional `image_file_id`, `child_topics` (tree array).

**Tree node shape:** `{ "id": "<uuid>", "name": "...", "slug": "...", "children": [...] }` — max depth **12**, max **100** nodes per tree.

### Course Outcomes

| Method | Path | Permission |
|--------|------|-----------|
| GET/POST/PATCH/DELETE | `/taxonomy/outcomes` | `course_outcome:*` (P30–P33) |
| GET | `/taxonomy/outcomes/full` | `course_outcome:read` | List including soft-deleted |
| DELETE | `/taxonomy/outcomes/:id/hard` | `course_outcome:delete` | Hard delete (+ orphan image) |

**Body:** `short_description` (≤100 chars), `description` (string array, ≤8 items × ≤120 chars each), optional `image_file_id`, `status`.

### Course Skills

| Method | Path | Permission |
|--------|------|-----------|
| GET/POST/PATCH/DELETE | `/taxonomy/skills` | `course_skill:*` (P34–P37) |
| GET | `/taxonomy/skills/full` | `course_skill:read` | List including soft-deleted |
| DELETE | `/taxonomy/skills/:id/hard` | `course_skill:delete` | Hard delete |

**Body:** `name`, `slug`, `children` (tree), `status`.

### Course Levels / Tags

`/taxonomy/levels` and `/taxonomy/tags` with `course_level:*` and `tag:*`. Each resource supports `GET /full`, soft `DELETE /:id`, and hard `DELETE /:id/hard` (same permission as delete).

**Soft delete:** `deleted_at` Unix seconds on all five taxonomy tables. Default list/get exclude soft-deleted rows. Slug uniqueness is enforced only among active rows (partial unique index).

---

## Image contract (topics and outcomes)

- **Create/Update:** optional `image_file_id` (UUID of a `media_files` row).
- **Read:** responses return both `image_file_id` and `image_file_url` (resolved from `media_files.url`).
- **Validation:** `MediaFileValidator` → `MediaService.LoadValidatedProfileImageFile`.
- **Orphan cleanup:** on **hard delete** or when `image_file_id` is replaced on update → `OrphanImageEnqueuer.EnqueueOrphanCleanupForFileID`. Soft delete does **not** enqueue orphan cleanup.

---

## Cross-Domain Dependencies

| Interface | Implemented by | Purpose |
|-----------|---------------|---------|
| `MediaFileValidator` | `MediaService.LoadValidatedProfileImageFile` | Validate image file FK |
| `OrphanImageEnqueuer` | `OrphanEnqueuer.EnqueueOrphanCleanupForFileID` | Deferred cloud cleanup |

Wiring: `internal/server/wire.go` (`taxMediaFileValidator`, `taxOrphanEnqueuer`).

---

## DB migrations

| Version | Change |
|---------|--------|
| `000002` | Original `categories`, tags, levels |
| `000006` | `categories.image_file_id` FK |
| `000009` | Rename `categories` → `course_topics`, add `child_topics`, tables `course_outcomes` / `course_skills`, permissions P18–P37 |
