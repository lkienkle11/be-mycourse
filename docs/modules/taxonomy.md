# Taxonomy Module

_Last audited: 2026-07-12 (multilingual hybrid: translation tables + JSONB tree `translations` + optimistic `row_version`)._

The taxonomy module (`internal/taxonomy/`) handles classification reference data for courses: **course topics**, **course outcomes**, **course skills**, **course levels**, and **tags**.

Phase multilingual uses a **hybrid** model:

| Layer | Storage | Localized fields |
|-------|---------|------------------|
| Root rows | 5 `*_translations` tables | `name` (or outcome `short_description` + `description[]`) |
| Tree nodes | Inline `translations` map inside JSONB (`child_topics` / `children`) | `name` per locale |
| Identity | Root + node `id` + canonical `slug` | **Not** localized; course/instructor FKs stay ID-only |

Default data locale: **`en`** (backfill from canonical fields). FE route locales remain `en`/`vi` only — no `ja` routing.

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
│   ├── repos_crud_helper.go # Shared list/create/get helpers; taxonomyListTotal skip-count
│   └── jsonb_types.go       # JSONB scanners for tree nodes and description arrays
└── delivery/
    ├── handler.go           # Thin handlers per resource
    ├── handler_helpers.go   # listTaxonomyItems, mapTaxonomyMutationError (incl. 409/3005)
    ├── routes.go
    └── dto.go

internal/shared/taxonomy/
├── tree_node.go             # TreeNode JSON shape (+ translations map)
├── tree_slug.go             # NormalizeTreeSlugs — derive slug from canonical name
├── tree_validate.go         # Depth, node count, UUID id, duplicate slug checks
└── description_validate.go  # Outcome description paragraph limits

internal/shared/i18n/        # Content locale helpers (NOT mailtmpl email locale)
├── locale.go                # CanonicalizeLocale (write), NegotiateReadLocale / ResolveText (read)

internal/shared/utils/
└── slug.go                  # SlugifyName — shared slug algorithm (matches FE slugifyName)
```

List endpoints use **`internal/shared/httpx.ListPaginated`** where applicable (same pattern as media list).

### Reuse map (do not duplicate)

| Need | Reuse / extend | Do not create |
|------|----------------|---------------|
| Slug | `utils.SlugifyName` + `tree_slug.go` | Second slug util |
| Tree validate | `tree_validate.go` / `description_validate.go` | Parallel validators |
| List/get infra | `taxonomyList` / `taxonomyGetByID` in `repos_crud_helper.go` | New get-by-id repo from scratch |
| Optimistic lock | Course `optimisticUpdate` + `expected_row_version` → HTTP 409 + app code **3005** | `expected_updated_at` |
| Email locale en/vi | Keep `mailtmpl.NormalizeLanguageCode` for email only | Use mailtmpl for taxonomy BCP47 |
| Permissions | Existing `*:read` / `*:update` (P14–P37) | Separate “manage translations” permission |
| Tx write sync | `db.Transaction` inline (course/instructor pattern) | Shared `WithTx` package |

---

## Multilingual contract

### Canonical ↔ default locale (`en`)

| Role | Source | Used for |
|------|--------|----------|
| Canonical field (`name` / `short_description` / `description`, node `name`) | Root row / node field | Slug derive, search/sort phase 1, last-resort fallback |
| Default translation (`locale=en` / `translations.en`) | Translation table / map | Display when falling back to default locale |

**Write sync + conflict (root, outcome fields, tree nodes alike):**

| Payload | Behavior |
|---------|----------|
| Both canonical and `translations.en` present **and different** | **Reject 4xx** — no silent preference |
| Only canonical | Mirror to `en` |
| Only `translations.en` | Mirror to canonical |
| Both present and equal | Accept |
| Locale other than `en` | Write translation / `translations[locale]` only; do not touch canonical |

After every successful write: `canonical text == en translation text` (assert in service/tests). Persist both in one transaction.

**Slug:** regenerate only from **canonical `name`** when that name changes (`SlugifyName`). Do not regenerate slug when only non-`en` translations change. Updating `en` name mirrors canonical → slug may regenerate.

### Locale: write vs read (separate APIs)

**Write (persist):**

- Canonicalize BCP47 per `docs/database.md` § Taxonomy locale rules: language lowercase, region uppercase, script titlecase if supported; **do not strip region**.
- After canonicalize, locale string length must be **≤ 16** (`VARCHAR(16)` on translation tables); longer → **4xx** (never rely on DB constraint text).
- Upsert / `UNIQUE (resource_id, locale)` uses **canonicalized** locale (`pt-br` and `pt-BR` → same row).
- Empty/invalid locale → **4xx** (no remap to `en`).
- **Collision after canonicalize:** if two distinct raw keys in the same `translations` map canonicalize to the same locale (e.g. `en-us` + `en-US`) and their payloads differ → **4xx** (deterministic validation error; never silent last-write-wins via map iteration). Identical payloads after canonicalize may collapse to one entry.
- **Per-locale content limits (all locales, not only canonical `en`):** non-empty `NodeTranslation.Name` → 1–255 runes; outcome translation with content → `short_description` 1–100 and `ValidateDescriptionParagraphs` (max 8 × 120). Empty both fields for a locale may be dropped; description-without-short still **4xx**.
- Tree node `translations[*].name` with content use the same 1–255 limit (`ValidateTree` walks translation maps).
- No hard language whitelist for storage (API accepts any valid BCP47 ≤16). FE admin add-locale is preset-dropdown-only.

**Read (negotiate + fallback):**

```text
exact locale
→ base language (if requested has region and exact missed; e.g. en-US → en)
→ default locale (en)
→ canonical field
```

- Missing/empty `locale` query on read → default `en` (not 4xx).
- Invalid format on read → default `en` + optional debug log (not 4xx).
- **Outcomes (localized):** resolve a **whole** translation row (exact → base → en → canonical). `short_description` and `description[]` always come from the **same** chosen locale/canonical source. Do **not** mix `resolved_locale=vi` with `description` from `en`.

**Do not** use one “normalize then fall back to en” function for both write and read.

### Public / localized vs admin editable

**Public / picker / instructor chips (localized):**

- Familiar shape: `id`, `slug`, `status`, resolved `name` (or outcome fields).
- Optional: `resolved_locale` (debug/rollout).
- Consumers do **not** need the full translations map.

**Admin get-for-edit (`GET .../:id?view=edit`):**

- Canonical fields + full `translations` map (outcome: per-locale `short_description` + `description[]`).
- Tree nodes: `id`, `slug`, canonical `name`, full `translations`, recursive `children`.
- Metadata: `available_locales`; `row_version` for optimistic lock.
- Admin list may still localize display columns via `locale`; **edit always loads** `view=edit` detail (list row is insufficient).

### Optimistic lock

- Column `row_version` on each of the five taxonomy root tables (mirror course/media).
- Write body: `expected_row_version` (same name as course).
- Stale → domain sentinel → HTTP **409** + app code **`Conflict` (3005)**.
- Every successful write (including translation-only) bumps `row_version` + `updated_at`.

---

## Responsibilities

| Domain | Description |
|--------|-------------|
| **CourseTopic** | `course_topics` + `course_topic_translations`; optional image; nested `child_topics` JSONB with per-node `translations` |
| **CourseOutcome** | `course_outcomes` + `course_outcome_translations` (`short_description`, `description[]`); optional image |
| **CourseSkill** | `course_skills` + `course_skill_translations`; nested `children` JSONB with per-node `translations` |
| **CourseLevel** | `course_levels` + `course_level_translations` |
| **Tag** | `tags` + `tag_translations` |

---

## API Endpoints

All routes are under `/api/v1/taxonomy/` and require `Authorization: Bearer <token>`.

### List / full query contract

- `page`, `per_page`, `sort_by`, `sort_desc`, optional `status`, optional `include_images` (default `true`)
- typed search: `search_by` + `search_value` — **search/sort still use canonical fields** (phase 1)
- **`locale`** (optional): content locale for resolved labels; missing → `en`; invalid format on read → `en`
- allowed `search_by` values:
  - levels/topics/skills/tags: `name`, `slug`
  - outcomes: `short_description`

### Get by ID (CREATE in this phase)

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/{resource}/:id` | `*:read` | Detail by id. Reuses infra `taxonomyGetByID`. |

Query:

| Param | Behavior |
|-------|----------|
| `locale` | Localized/public shape (resolved text + optional `resolved_locale`) |
| `view=edit` | Admin editable DTO: canonical + full `translations` + tree translations + `row_version` |

Pattern mirrors course `include_outline` / taxonomy `include_images` optional flags.

### Course Topics

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| GET | `/taxonomy/topics` | `topic:read` | List active topics (paginated, localized by `locale`) |
| GET | `/taxonomy/topics/full` | `topic:read` | List including soft-deleted |
| GET | `/taxonomy/topics/:id` | `topic:read` | Get by id (`locale` / `view=edit`) |
| POST | `/taxonomy/topics` | `topic:create` | Create topic |
| PATCH | `/taxonomy/topics/:id` | `topic:update` | Update topic (active only) |
| DELETE | `/taxonomy/topics/:id` | `topic:delete` | Soft-delete topic |
| DELETE | `/taxonomy/topics/:id/hard` | `topic:delete` | Hard-delete topic (+ orphan image cleanup) |

**Create / update body:** canonical `name` and/or `translations` map; optional `status`, `image_file_id`, `child_topics` (tree with optional per-node `translations`); **`expected_row_version`** on update; **`slug` not accepted on write** — derived from canonical `name` via `utils.SlugifyName`.

**Tree nodes:** `TreeNode` = `{ id, name, slug, translations?, children[] }`. Max depth **12**, max **100** nodes per tree. Slug from canonical node `name`.

### Course Outcomes / Skills / Levels / Tags

Same list `/full` / **GET `/:id`** / POST / PATCH / soft+hard DELETE pattern with resource permissions:

| Resource | Permission prefix |
|----------|-------------------|
| Outcomes | `course_outcome:*` (P30–P33) |
| Skills | `course_skill:*` (P34–P37) |
| Levels | `course_level:*` (P14–P17) |
| Tags | `tag:*` (P22–P25) |

Outcome body: canonical `short_description` / `description[]` and/or per-locale translations; image; status; `expected_row_version` on update. Per-locale outcome translations that are present in the payload must include non-empty `short_description` (partial description-only → **4xx**). On update, submitted `translations` map replaces the full set of locales (missing keys are deleted in the same transaction).

Skill body: canonical `name` and/or translations; optional `children` tree; status; `expected_row_version` on update.

**Soft delete:** `deleted_at` Unix seconds. Default list/get exclude soft-deleted. Slug uniqueness among active rows only (partial unique index).

**List performance:** `taxonomyList` Find-first; `taxonomyListTotal` skips `COUNT(*)` when `len(rows) < per_page`. Topics/outcomes hydrate `image_file_url` unless `include_images=false`.

**Mutation errors:** validation → 4xx / `2001` or `3001`; canonical↔`en` conflict → 4xx; stale `expected_row_version` → **409 / 3005** via `mapTaxonomyMutationError`.

---

## Image contract (topics and outcomes)

- **Create/Update:** optional `image_file_id` (UUID of a `media_files` row).
- **Read:** responses return both `image_file_id` and `image_file_url` (resolved from `media_files.url`).
- **Validation:** `MediaFileValidator` → `MediaService.LoadValidatedProfileImageFile`.
- **Orphan cleanup:** on **hard delete** or when `image_file_id` is replaced on update → `OrphanImageEnqueuer.EnqueueOrphanCleanupForFileID`. Soft delete does **not** enqueue orphan cleanup.

---

## Cross-Domain Dependencies

| Interface / consumer | Purpose |
|----------------------|---------|
| `MediaFileValidator` / `OrphanImageEnqueuer` | Image FK + orphan cleanup (wired in `wire.go`) |
| Instructor expertise / application / profile chips | Localized join on translation tables + **`locale` threaded from delivery → service → repository** (same fallback as taxonomy read); FK IDs unchanged |
| Course editor pickers (FE) | Call taxonomy list with `locale` |

---

## Non-goals (phase 1)

- Per-locale slug / SEO URL
- Multilingual full-text search
- Splitting tree nodes into relational tables
- Changing course/instructor relation tables
- FE `ja` routing / `messages/ja.ts`
- Accept-Language middleware (use query `locale`)
- Inventing `expected_updated_at` instead of `expected_row_version`

---

## DB migrations

| Version | Change |
|---------|--------|
| `000002` | Original `categories`, tags, levels |
| `000006` | `categories.image_file_id` FK |
| `000009` | Rename `categories` → `course_topics`, add `child_topics`, tables `course_outcomes` / `course_skills`, permissions P18–P37 |
| `000012` | Soft delete + partial unique slug indexes |
| `000032` | Five `*_translations` tables + `en` backfill; JSONB tree `translations.en` patch (**fail-fast** if non-null tree is not a JSON array / invalid nodes); `row_version` on five taxonomy roots (mirror `000020`). **Shipped in source**; apply on deploy before enabling localized reads/writes. |

See `docs/database.md` § Taxonomy for full schema.
