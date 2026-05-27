# Session: Taxonomy topics, outcomes, skills (BE)

**Date:** 2026-05-20

## API contract

### Topics (replaces `/categories`)

| Method | Path | Permission |
|--------|------|------------|
| GET/POST/PATCH/DELETE | `/api/v1/taxonomy/topics` | `topic:*` (P18–P21) |

Fields: `id`, `name`, `slug`, `image_file_id`, `image_file_url`, `child_topics`, `status`, `created_by`, timestamps.

### Outcomes

| Method | Path | Permission |
|--------|------|------------|
| GET/POST/PATCH/DELETE | `/api/v1/taxonomy/outcomes` | `course_outcome:*` (P30–P33) |

`short_description` ≤100, `description` string[] ≤8×120, optional `image_file_id`.

### Skills

| Method | Path | Permission |
|--------|------|------------|
| GET/POST/PATCH/DELETE | `/api/v1/taxonomy/skills` | `course_skill:*` (P34–P37) |

`children` tree: max depth 12, max 100 nodes, UUID `id` per node.

## Files touched

- `migrations/000009_taxonomy_topics_outcomes_skills.{up,down}.sql`
- `pkg/taxonomy/{tree_node,tree_validate,description_validate}*.go`
- `internal/taxonomy/{domain,application,infra,delivery}/*`
- `internal/shared/constants/{permissions,dbschema_name}.go`
- `internal/system/application/roles_permission.go`
- `internal/server/wire.go`
- Docs: `database.md`, `modules/taxonomy.md`, `router.md`, `api-overview.md`, `data-flow.md`, `curl_api.md`, `migrations/README.md`

## GitNexus impact

- `CreateCategory` → renamed flow as `CreateTopic`: **LOW** (taxonomy delivery only)
- New `CreateCourseOutcome` / `CreateCourseSkill`: **LOW**

## Post-deploy

```bash
MIGRATE=1 go run .
go run ./cmd/syncpermissions
go run ./cmd/syncrolepermissions
```

Users with cached JWT need re-login for `topic:*` permission names.

## API docs sync (2026-05-20 follow-up)

- `docs/api_swagger.yaml` — topics/outcomes/skills paths + schemas (`TreeNode`, …)
- `docs/api-dog-import.json` — regenerated via `ruby scripts/generate-apidog-postman.rb`
- `scripts/generate-apidog-postman.rb` — reads request/query examples from swagger only (no hardcoded payloads)
- `docs/curl_api.md`, `docs/return_types.md`, `README.md`, `docs/folder-structure.md`, `docs/modules/media.md`

## Quality gate

- `go fmt ./...` ✅
- `go vet ./...` ✅
- `go test ./pkg/taxonomy/...` ✅
