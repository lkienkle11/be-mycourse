# Taxonomy Module

The taxonomy module (`internal/taxonomy/`) handles classification reference data for courses: **course topics** (formerly categories), **course outcomes**, **course skills**, **course levels**, and **tags**.

---

## Directory Layout

```
internal/taxonomy/
├── domain/
│   ├── taxonomy.go          # Entities and input types
│   └── repository.go        # Repository interfaces
├── application/
│   └── service.go           # TaxonomyService use-cases
├── infra/
│   ├── repos.go             # GORM repositories (topics, outcomes, skills, tags, levels)
│   └── jsonb_types.go       # JSONB scanners for tree nodes and description arrays
└── delivery/
    ├── handler.go
    ├── routes.go
    └── dto.go

pkg/taxonomy/
├── tree_node.go             # TreeNode JSON shape
├── tree_validate.go         # Depth, node count, UUID id, duplicate slug checks
└── description_validate.go  # Outcome description paragraph limits
```

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

### Course Topics (replaces `/categories`)

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/topics` | `topic:read` | List topics (paginated) |
| POST | `/taxonomy/topics` | `topic:create` | Create topic |
| PATCH | `/taxonomy/topics/:id` | `topic:update` | Update topic |
| DELETE | `/taxonomy/topics/:id` | `topic:delete` | Delete topic |

**Body fields:** `name`, `slug`, `status`, optional `image_file_id`, `child_topics` (tree array).

**Tree node shape:** `{ "id": "<uuid>", "name": "...", "slug": "...", "children": [...] }` — max depth **5**, max **100** nodes per tree.

### Course Outcomes

| Method | Path | Permission |
|--------|------|-----------|
| GET/POST/PATCH/DELETE | `/taxonomy/outcomes` | `course_outcome:*` (P30–P33) |

**Body:** `short_description` (≤100 chars), `description` (string array, ≤8 items × ≤120 chars each), optional `image_file_id`, `status`.

### Course Skills

| Method | Path | Permission |
|--------|------|-----------|
| GET/POST/PATCH/DELETE | `/taxonomy/skills` | `course_skill:*` (P34–P37) |

**Body:** `name`, `slug`, `children` (tree), `status`.

### Course Levels / Tags

Unchanged: `/taxonomy/levels`, `/taxonomy/tags` with `course_level:*` and `tag:*`.

---

## Image contract (topics and outcomes)

- **Create/Update:** optional `image_file_id` (UUID of a `media_files` row).
- **Validation:** `MediaFileValidator` → `MediaService.LoadValidatedProfileImageFile`.
- **Orphan cleanup:** replaced or deleted `image_file_id` → `OrphanImageEnqueuer.EnqueueOrphanCleanupForFileID`.

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
