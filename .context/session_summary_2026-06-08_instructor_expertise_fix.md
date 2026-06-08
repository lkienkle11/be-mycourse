# Session summary — instructor expertise POST/GET fix (2026-06-08)

## Scope

Fix instructor expertise APIs:

- `POST /api/v1/instructors/:id/expertise/topics` — 500 (`course_topic_id` NOT NULL)
- `GET .../expertise/topics` and `GET .../expertise/skills` — PascalCase JSON without `name`/`slug`; FE showed `#undefined`
- `GET` after JOIN fix — `deleted_at` ambiguous SQL error

## Root causes

| Symptom | Cause |
|---------|--------|
| POST 500 | Drifted DB: legacy `course_topic_id` NOT NULL while app inserts `topic_id` |
| FE `#undefined` | Domain structs lacked `json` tags + taxonomy join fields |
| GET ambiguous `deleted_at` | `activeScope()` unqualified on JOIN queries |

## Code changes (reuse-only — no new response DTOs)

| File | Change |
|------|--------|
| `migrations/000017_*` | Drop legacy FK cols; NOT NULL + FK on `topic_id`/`skill_id` |
| `internal/instructor/domain/instructor.go` | `json` tags + `Name`/`Slug` on `ExpertiseTopic`/`ExpertiseSkill` |
| `internal/instructor/infra/rows.go` | Explicit `gorm:"column:..."` on expertise rows |
| `internal/instructor/infra/repos_map.go` | `expertiseTopicRowToDomain` / `expertiseSkillRowToDomain` |
| `internal/instructor/infra/repos.go` | JOIN taxonomy; alias-qualified soft-delete; explicit SELECT + inline scan; `getExpertiseByID` reload after create |
| `AGENTS.md`, `CLAUDE.md` | Dev test account `user01@yopmail.com` / `Test@1234` |

## Docs sync

- `docs/database.md`, `docs/modules/instructor.md`, `docs/modules.md`, `docs/deploy.md`
- `docs/return_types.md` — expertise JSON example
- `docs/api_swagger.yaml` — `InstructorExpertiseTopic`/`Skill` schemas + list/create responses
- `docs/curl_api.md` — §18 smoke step 4 (expertise GET/POST)
- `migrations/README.md`
- `docs/api-dog-import.json` — regenerated via Ruby script

## GitNexus

- Pre-edit impact: `ListExpertise` — LOW risk (handlers unchanged)
- End: `npx gitnexus analyze --force` — 5,111 nodes / 14,821 edges
- `detect_changes(scope=all)` — expected instructor expertise processes; no missing d=1 callers

## Quality gates (final batch — all PASS)

| Gate | Result |
|------|--------|
| `golangci-lint cache clean && golangci-lint run` | 0 issues |
| `make check-architecture` | OK |
| `make check-dupl` | 0 clones |
| `make check-layout` | OK |
| `go test ./...` | PASS |
| `go build ./...` | PASS |
| `ruby scripts/generate-apidog-postman.rb` | OK (130586 bytes) |

## Manual verification

Login: `user01@yopmail.com` / `Test@1234` on `localhost:8080`.

```bash
# GET expertise topics (user_id=14)
curl -sS 'http://localhost:8080/api/v1/instructors/14/expertise/topics' \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Expected: snake_case `topic_id`, joined `name` + `slug`. Restart `go run .` after code changes.

## Deploy notes

- Apply **`000017`** on drifted DBs (`MIGRATE=1` or manual SQL). See **`docs/deploy.md`** troubleshooting.
- Run after **`000015`** when legacy columns still exist.
