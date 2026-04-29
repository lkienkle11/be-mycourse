# Phase Sub 07 — Orphan image cleanup (2026-04-29)

## Scope
Implement the "orphan image object" cleanup pipeline: when a business record holding
an image URL field (categories.image_url, users.avatar_url, future course/lesson
image fields) is deleted or the URL is replaced, the referenced cloud object must be
scheduled for deletion via the existing `media_pending_cloud_cleanup` queue.

## Key conventions enforced (corrected mid-session)
- `*_helper` suffix → ONLY in `pkg/logic/helper/`, never in `services/`
- Service files in `services/` do NOT use `_helper` suffix
- Cross-service imports avoided: taxonomy service calls `repository/media` indirectly
  through `services/media.EnqueueOrphanImageCleanup`, which is a single domain function
  (not a shared cross-service util)
- All tests in `tests/` (no colocated `_test.go` in production packages)

## New files
| File | Purpose |
|------|---------|
| `pkg/logic/helper/media_url_orphan.go` | Pure URL parser: URL → (provider, objectKey, bunnyVideoID, ok) |
| `pkg/logic/helper/media_jsonb_scan.go` | JSONB walker: collects image URLs from nested JSONB payloads (skeleton for Phase 05+) |
| `services/media/orphan_cleanup.go` | `EnqueueOrphanImageCleanup(imageURL string) bool` — DB-lookup + parse fallback + InsertPendingCleanup |
| `tests/sub07_orphan_image_test.go` | 11 tests for URL parser and JSONB scanner |

## Modified files
| File | Change |
|------|--------|
| `repository/media/file_repository.go` | Added `GetByURL(url string)` method |
| `services/taxonomy/category_service.go` | `DeleteCategory` reads+enqueues image_url; `UpdateCategory` enqueues superseded image_url |
| `IMPLEMENTATION_PLAN_EXECUTION.md` | Added § Phase Sub 07 |
| `.full-project/data-flow.md` | Added Orphan Image Cleanup Flow section |
| `.full-project/reusable-assets.md` | Added 4 new asset entries |

## Transaction / compensation policy
1. DB delete/update commits FIRST
2. `EnqueueOrphanImageCleanup` called AFTER successful commit
3. Enqueue failure = cloud object temporarily stranded (acceptable, no data loss)
4. Worker retries with exponential backoff, marks `failed` after max attempts
5. No 2-phase commit (cloud providers cannot join Postgres transactions)

## Current image field inventory
| Domain | Field | Cleanup wired? |
|--------|-------|----------------|
| categories | image_url | ✅ DELETE + UPDATE |
| users | avatar_url | ❌ No update endpoint yet — TODO when UpdateProfile added |
| courses (future) | cover_image, thumbnail, banner | ❌ To wire in Phase 02+ |
| lessons/sections (future) | JSONB content URLs | ❌ ScanJSONBForImageURLs skeleton ready |

## Quality gate
- `go build ./...` ✅
- `go test ./...` ✅ (11 sub07 tests + all existing pass)

## Next phase
`phase-02-start` — course shell domain (CRUD 1 bảng chính, list/filter).
