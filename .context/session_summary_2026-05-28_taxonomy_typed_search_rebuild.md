# Session: Taxonomy typed search rebuild (BE)

**Date:** 2026-05-28

## Why this rebuild

- Previous implementation was discarded.
- Rebuilt from clean working tree with reuse-first approach.

## Reuse-first implementation

- Reused existing shared list-search utility:
  - `internal/shared/utils/filter.go` -> `BuildSearchClause`
- Taxonomy repo search now maps taxonomy filter input into shared `BaseFilter` search shape and reuses that helper.
- No duplicate standalone search parser was introduced.

## Contract changes

- Taxonomy list query contract changed from `search` to strict `search_by` + `search_value`.
- Status logic remains unchanged.
- Allowed `search_by` values:
  - levels/topics/skills/tags: `name`, `slug`
  - outcomes: `short_description`

## Code changes

- `internal/taxonomy/delivery/dto.go`
  - `TaxonomyBaseFilter`: `SearchBy`, `SearchValue`
- `internal/taxonomy/domain/taxonomy.go`
  - `TaxonomyFilter`: `SearchBy`, `SearchValue`
- `internal/taxonomy/delivery/handler.go`
  - `toFilter` maps `search_by`/`search_value`
- `internal/taxonomy/infra/repos.go`
  - status filter extracted to shared helper function in-file
  - search now uses `shared/utils.BuildSearchClause`

## Docs synchronized

- `docs/modules/taxonomy.md`
- `docs/curl_api.md`
- `docs/return_types.md`
- `docs/api_swagger.yaml`
- Regenerated `docs/api-dog-import.json` via `ruby scripts/generate-apidog-postman.rb` after Swagger edits.

## Verification

- `golangci-lint cache clean && golangci-lint run` ✅
- `make check-architecture` ✅
- `make check-dupl` ✅
- `make check-layout` ⚠️ existing make target issue (`layoutguard: ./... matched no packages`)
- `go run ./tools/layoutguard ./` ✅
- `go vet ./...` ✅
- `go test ./...` ✅
- `go build ./...` ✅

## GitNexus

- Reindexed before and after edits with `npx gitnexus analyze --force`.
- Symbol impact before edits:
  - `TaxonomyBaseFilter`: LOW
  - `toFilter`: LOW
  - `TaxonomyFilter`: LOW
  - `applyTaxonomyFilter`: LOW
  - `applyOutcomeSearch`: LOW
  - `BuildSearchClause`: LOW
