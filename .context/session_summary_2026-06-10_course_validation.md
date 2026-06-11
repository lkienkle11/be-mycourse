# Session Summary — Course Field Validation (2026-06-10)

## Scope
BE + FE validation for course create, basic-info PATCH, outline entities, and QUIZ preview guard.

## GitNexus research
See `.context/gitnexus_research_2026-06-10_course_validation.md`.

## BE changes
- `internal/shared/utils/text_rules.go` + tests
- `internal/shared/validate/text_rules.go` + register in `validate.go`
- Stricter `internal/course/delivery/dto.go`
- `courseTitleAndSlug` min 5 chars; new domain errors
- `handler_instructor.go` trim + required basic-info mapping
- `validateSubLessonPayload` + `filterPreviewOutline` QUIZ preview guard

## Quality gates (all PASS)
- `golangci-lint run` — 0 issues
- `make check-architecture` — OK
- `make check-dupl` — OK
- `make check-layout` — OK
- `go test ./...` — PASS
- `go build ./...` — PASS
- `ruby scripts/generate-apidog-postman.rb` — OK

## Docs synced
- `docs/modules/course.md`, `docs/return_types.md`, `docs/reusable-assets.md`, `docs/api-dog-import.json`

## Manual smoke (operator)
- Login `user01@yopmail.com` / `Test@1234`
- POST create with short title → 400
- PATCH basic-info missing fields → 400
- QUIZ sub-lesson `is_preview: true` → 400
