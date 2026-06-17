# Taxonomy Module

_Last audited: 2026-06-16 (list `taxonomyListTotal` skip-count optimization)._

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
│   ├── repos_crud_helper.go # Shared list/create helpers; taxonomyListTotal skips COUNT on last page
│   └── jsonb_types.go       # JSONB scanners for tree nodes and description arrays
└── delivery/
    ├── handler.go           # Thin handlers per resource
    ├── handler_helpers.go   # listTaxonomyItems, shared mutation error mapping
    ├── routes.go
    └── dto.go

internal/shared/taxonomy/
├── tree_node.go             # TreeNode JSON shape
├── tree_slug.go             # NormalizeTreeSlugs — derive slug from name (recursive)
├── tree_validate.go         # Depth, node count, UUID id, duplicate slug checks
└── description_validate.go  # Outcome description paragraph limits

internal/shared/utils/
└── slug.go                  # SlugifyName — shared slug algorithm (matches FE slugifyName)
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
- `page`, `per_page`, `sort_by`, `sort_desc`, optional `status`, optional `include_images` (default `true`; `false` skips `media_files` hydration on topics/outcomes list)
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

**Create / update body:** `name`, optional `status`, optional `image_file_id`, optional `child_topics` (tree array). **`slug` is not accepted on write** — the service derives it from `name` via `utils.SlugifyName` (same rules as FE `slugifyName`).

**Tree nodes:** same `TreeNode` JSON shape for read and write; `slug` is optional on write and always present on read responses. Max depth **12**, max **100** nodes per tree.

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

**Create / update body:** `name`, optional `children` (tree), optional `status`. Slug is server-computed from `name` (and from each tree node name).

### Course Levels / Tags

`/taxonomy/levels` and `/taxonomy/tags` with `course_level:*` and `tag:*`. Each resource supports `GET /full`, soft `DELETE /:id`, and hard `DELETE /:id/hard` (same permission as delete).

**Soft delete:** `deleted_at` Unix seconds on all five taxonomy tables. Default list/get exclude soft-deleted rows. Slug uniqueness is enforced only among active rows (partial unique index).

**List performance:** `taxonomyList` in `repos_crud_helper.go` runs `Find` first; `taxonomyListTotal` skips a separate `COUNT(*)` when `len(rows) < per_page` (total = `(page-1)*per_page + len(rows)`). Topics/outcomes hydrate `image_file_url` via a second batched `media_files` query (`listTaxonomyWithImageURLs`) unless `include_images=false`.

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
