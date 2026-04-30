## Documentation Convention (Mandatory)

The `docs/` folder is the **primary and authoritative documentation source** for this project.

- **Before starting any task**, read the relevant files in `docs/` (including `docs/architecture.md`, `docs/patterns.md`, `docs/reusable-assets.md`, `docs/data-flow.md`, `docs/api-overview.md`, and any applicable `docs/modules/*.md`).
- If `docs/` already contains sufficient and up-to-date information → **reuse it directly** without re-running full discovery.
- If `docs/` is missing information or outdated → re-run discovery and **update `docs/` before writing code**.
- At the end of each task, sync `docs/` to reflect any changes made (architecture, APIs, reusable assets, patterns, sequences).
- Always check `docs/reusable-assets.md` before proposing any new utility, type, DTO, or helper to avoid duplication.

---

## Phase Sub 07 — Orphan image cleanup (tasks 01→10, closed 2026-04-29)

Single authoritative checklist for plan ids `phase-sub-07-task-01` … `phase-sub-07-task-10`.

### Image field inventory (Task 01) ✅
| Domain | Field | Type | Has CRUD delete/update |
|--------|-------|------|------------------------|
| `categories` | `image_url` VARCHAR(512) | URL string (no FK to media_files) | ✅ DELETE + PATCH |
| `users` | `avatar_url` TEXT | URL string (no FK to media_files) | No update endpoint yet |
| `course_levels` | — | No image field | N/A |
| `tags` | — | No image field | N/A |
| Future courses | `cover_image`, `thumbnail`, `banner` | URL strings | To be added in Phase 02+ |
| Future lessons/sections | JSONB content URLs | Embedded in JSONB | To be added in Phase 05+ |

Orphan risk: URL field points to a B2/Bunny object that is never deleted when the parent record is deleted or the field is replaced.

### Orphan definition + source of truth (Task 02) ✅
- **Orphan object**: a cloud storage object (B2 key or Bunny GUID) that still exists but has no valid owning DB record referencing it.
- **Source of truth for identity**: `media_files` row (preferred — has `provider`, `object_key`, `bunny_video_id` stored). Fallback: parse URL by pattern against runtime `MediaSetting` (CDN base + bucket, or Bunny stream base + library ID).
- **Cleanup queue**: `media_pending_cloud_cleanup` table — same deferred pattern as sub06 replace orphan.

### Cleanup strategy — reuse existing layer (Task 03) ✅
- **Single delete path**: `pkg/media.DeleteStoredObject` routes by provider (B2 → `DeleteB2Object`; Bunny → `DeleteBunnyVideo`; Local → noop).
- **Deferred queue**: `media_pending_cloud_cleanup` via `repository/media.FileRepository.InsertPendingCleanup`. Worker: `internal/jobs/media_pending_cleanup_scheduler.go`.
- **No new providers or delete paths** — all orphan enqueues reuse the existing pipeline.

### hook cleanup — media/files DELETE (Task 04) ✅
Already implemented in sub06: `services/media.DeleteFile` calls `pkg/media.DeleteStoredObject` then `SoftDeleteByObjectKey`. No change needed.

### Course domain skeleton (Task 05) ✅
Course domain not yet in repo. When Phase 02+ adds `courses.cover_image` / `course_edits.thumbnail` etc., call `mediasvc.EnqueueOrphanImageCleanup(oldURL)` after each DELETE/PATCH commit. See `services/media/orphan_cleanup.go` doc comment.

### Taxonomy + user/profile hooks (Task 06) ✅
- `services/taxonomy/category_service.go`:
  - `DeleteCategory`: reads `image_url` before delete, enqueues cleanup after successful DB delete.
  - `UpdateCategory`: captures `prevImageURL` before mutation, enqueues cleanup if `image_url` is replaced.
- `users.avatar_url`: no update endpoint yet — hook must be added to the future `UpdateProfile` service function following the same pattern.
- External/3rd-party URLs (not matching configured CDN/Bunny) are silently skipped (no-op).

### JSONB/nested skeleton (Task 07) ✅
- `pkg/logic/helper/media_jsonb_scan.go`: `ScanJSONBForImageURLs(raw []byte) []string` — recursive walker that collects values under image-field-named keys (`_url`, `image`, `thumbnail`, `cover`, `banner`, `avatar`, `poster`, `icon`).
- Future lesson/quiz domain: call `ScanJSONBForImageURLs(row.ContentJSON)` before cascade delete, then `mediasvc.EnqueueOrphanImageCleanup` for each URL. See TODO comment in the file.

### Transaction + compensation policy (Task 08) ✅
1. **DB delete first, cloud enqueue after commit** — prevents cloud deletes for records that still exist.
2. **Orphan on enqueue failure**: if `InsertPendingCleanup` fails (transient DB error), the cloud object is temporarily stranded. Acceptable — no user data lost; future admin scan or next update cycle will re-trigger.
3. **Worker retry**: exponential backoff `n²` minutes, capped at 30 min, max `constants.MediaCleanupMaxAttempts`. Worker marks row `failed` after max attempts.
4. **No 2-phase commit cloud↔DB**: Postgres cannot enlist B2/Bunny in a distributed transaction. Documented limitation.
5. **Replace path**: `UpdateCategory` captures `prevImageURL` before saving, enqueues ONLY when URL changed AND old URL non-empty — avoids spurious no-op enqueues.

### Quality gate + tests (Task 09) ✅
- `tests/sub07_orphan_image_test.go`: 11 tests covering `ParseImageURLForOrphanCleanup` (empty, local, external, Bunny with/without query params, B2 CDN, B2 without bucket) and `ScanJSONBForImageURLs` (nil, flat, nested, no-url-keys). All PASS.
- `go build ./...` ✅ `go test ./...` ✅

### Final audit (Task 10) ✅
- `categories.image_url` DELETE hook ✅ UPDATE replace hook ✅
- `users.avatar_url` — no update endpoint yet; hook TODO documented in `services/taxonomy/category_service.go` pattern and this section.
- `media_files` DELETE — already covered by sub06 `DeleteFile`.
- Future domains (courses, lessons, quiz) — pattern documented + `ScanJSONBForImageURLs` skeleton ready.
- Transaction policy documented above.
- No HIGH/CRITICAL GitNexus risk: `DeleteCategory` (LOW, 1 direct caller) + `UpdateCategory` (LOW, 1 direct caller).
- `gitnexus_detect_changes` to be run pre-commit.

### Files reference (Sub 07)
- Helpers: `pkg/logic/helper/media_url_orphan.go`, `pkg/logic/helper/media_jsonb_scan.go`
- Service: `services/media/orphan_cleanup.go`
- Repository: `repository/media/file_repository.go` (added `GetByURL`)
- Modified services: `services/taxonomy/category_service.go`
- Tests: `tests/sub07_orphan_image_test.go`

---

## Repository convention — `tests/` (module-level tests)

- Add **module / integration** Go tests, black-box packages importing `mycourse-io-be`, shared fixtures, and cross-feature harnesses under repository root **`tests/`** (see `tests/README.md`, `README.md` **Testing**, `docs/patterns.md`, `docs/requirements.md` NFR-1.6, `docs/architecture.md` directory map).
- **All tests** (including narrow unit tests) must be added under repository root **`tests/`**.

## Phase Sub 05 Execution Update (2026-04-27 - tasks 01->15)

- Completed reset baseline and impact analysis with GitNexus (`analyze --force`, `status`, symbol-level `impact`) before edits; all planned symbol changes reported **LOW** risk.
- Implemented metadata persistence after cloud provider operations:
  - Added migration `000003_media_metadata` for `media_files`.
  - Added `models/media_file.go`, `dbschema/media_*`, `repository/media/file_repository.go`, and repository root wiring.
  - `services/media/file_service.go` now persists create/update metadata and soft-deletes DB row after successful cloud delete (with best-effort compensation when DB write fails after upload).
- Finalized upload file path requirements:
  - B2 public URL remains `<cdn_gcore_base_url>/<bucket_b2_name>/<file_to_see>`.
  - B2 upload object key keeps random 8-digit prefix policy.
  - Bunny pipeline remains `CreateVideo -> UploadVideoContent`, with status API + webhook.
- Webhook/status:
  - `POST /api/v1/webhook/bunny` stays outside auth middleware.
  - Webhook service now updates persisted duration/metadata on finished status and remains idempotent-safe for retry when row is absent.
- Config/env hardening:
  - Added `media.app_media_provider` in all `config/app*.yaml`.
  - Added `MEDIA_APP_PROVIDER` in `.env*.example`.
- Contract sync:
  - Extended upload response mapping with persisted metadata fields (id/kind/provider/filename/mime/size/status/bucket + Bunny metadata).
- Correction pass (same sub05 scope):
  - Removed `provider` from public `UploadFileResponse` to avoid leaking internal provider-routing business logic.
  - Refactored DB<->entity mapping out of `services/media/file_service.go` into `pkg/logic/mapping/media_model_mapping.go` to align with project layering conventions.
  - Kept API behavior and tests green after refactor.
- Quality gate:
  - `gofmt -w ...` and `go test ./...` passed.

## Phase Sub 03 — Upload cap 2 GiB per file (tasks 01–10, 2026-04-26)

### Task 01 — Baseline / discovery / multipart inventory
- Re-read `docs/*`, `.context/*`, `docs/*`, `README.md`, this plan; ran `npx gitnexus analyze --force` + `npx gitnexus status` (index up-to-date).
- **Multipart upload entry points (repo grep `FormFile` / upload):** only `api/v1/media/file_handler.go` (`createFile`, `updateFile`) use `c.FormFile("file")`. No other handlers accept multipart file uploads.
- **Service read path:** `services/media/file_service.go` `CreateFile` (and `UpdateFile` via `CreateFile`) reads the opened part into memory before provider dispatch; guarded by `io.LimitReader(..., MaxMediaUploadFileBytes+1)` + size checks (see tasks 04–05).
- **Other `io.ReadAll`:** `pkg/media/clients.go` uses `io.ReadAll` on **HTTP response bodies** for Bunny/B2 error diagnostics — not a client upload entry point.
- **Policy:** maximum **one** file part per create/update = **2×1024×1024×1024 bytes** (`constants.MaxMediaUploadFileBytes`). Declared oversize → HTTP **413** + `errcode.FileTooLarge` (**2003**). Missing `file` field remains HTTP **400** + `errcode.BadRequest` (**3001**) with explicit message — not confused with oversize.
- **Residual risks:** uploads up to the cap still buffer the full payload in RAM in `CreateFile` (required today for Bunny video `[]byte` path and metadata inference); very large legitimate files increase memory pressure — mitigated by hard cap, Gin `MaxMultipartMemory` spilling most multipart parsing to disk, and infra `client_max_body_size` (task 08).

### Task 02 — Single source of truth
- **`constants/error_msg.go`:** canonical file for **error/sentinel string literals** (and related limits for the same feature). Media upload: `MaxMediaUploadFileBytes` (2 GiB) **and** `MsgFileTooLargeUpload` — **one** string used by both `pkg/errcode` (`FileTooLarge` / 2003 default `message`) and `pkg/errors.ErrFileExceedsMaxUploadSize` (`errors.New`); no duplicate literals in `messages.go` or `upload_errors.go`. File header documents rules for future AI contributors.

### Task 03 — Gin multipart memory
- `api/router.go` `InitRouter`: `router.MaxMultipartMemory = 64 << 20` (64 MiB) with inline comment (parts larger than this spill to temp files during multipart parse).

### Task 04 — Handler early reject
- `file_handler.go`: after successful `FormFile`, if `FileHeader.Size >= 0` and exceeds cap → 413 + `FileTooLarge`.

### Task 05 — Service second line
- `CreateFile`: header size check, `LimitReader` cap+1, reject if `len(payload) > cap`; `SizeBytes` uses actual payload length when header size unknown.

### Task 06 — Other entrypoints
- Confirmed no other `FormFile` / multipart file uploads; policy applies only to media create/update.

### Task 07 — Errcode / message consistency
- `pkg/errcode`: `FileTooLarge = 2003`, default message in `messages.go`; handlers map `pkgmedia.ErrFileExceedsMaxUploadSize` and declared oversize to same code (distinct from missing-file `BadRequest`).
- Sentinel lives in **`pkg/errors/upload_errors.go`** (`errors.New(constants.MsgFileTooLargeUpload)`); **not** under `services/media/` (orchestration only returns the shared sentinel).

### Task 08 — Infra / proxy
- `docs/deploy.md` + `docs/modules/media.md`: reverse proxy (e.g. nginx `client_max_body_size`) must allow **≥ 2G** on the API vhost or the edge returns **413/400** before the app.

### Task 09 — Quality gate + manual test notes
- `gofmt`, `go test ./...`, `go vet ./...` — pass. **Manual:** upload file just under 2 GiB (expect 200 path when auth + provider configured); boundary at cap (expect 413 + code 2003); file slightly over cap with known `Size` (expect early 413); chunked/unknown size with padded stream over cap (expect 413 after service read cap).

### Task 10 — Final audit / handoff
- Checklist satisfied: single constant; Gin memory set; handler + service enforce cap; repo scan clean; err/message split; proxy doc; quality gate. GitNexus re-analyze after doc/code sync. Ready for `phase-02-start` in plan sequencing.

### Phase Sub 03 — Message sync errcode ↔ error_msg (tasks 01–10, 2026-04-26)
- **`constants.MsgFileTooLargeUpload`** is the only literal for upload oversize copy; **`pkg/errcode/messages.go`** uses `constants.MsgFileTooLargeUpload` for `FileTooLarge`; **`pkg/errors/upload_errors.go`** uses the same for `ErrFileExceedsMaxUploadSize`. Renamed from `MediaUploadErrFileExceedsMaxSize` to avoid two strings for one UX. Full docs + package comments updated for downstream AI agents.

### Phase Sub 03 — Constants layout (tasks 01–10 follow-up, 2026-04-26)
- Merged **`constants/upload_limits.go` → `constants/error_msg.go`** so all agent-discoverable **error/sentinel message** literals (and co-located limits) live in one documented file. Updated architecture, README, `docs/*`, `docs/modules/media.md`, `pkg/errcode/messages.go` cross-reference, this plan, `.context` handoff.

### Phase Sub 03 — Re-audit (sentinel + message constant, same tasks 01–10, 2026-04-26)
- Re-validated: `FormFile` only in media handlers; policy unchanged (2 GiB, Gin 64 MiB, handler early reject, service `LimitReader`, deploy/nginx notes, errcode **2003**).
- **Structural fix:** deleted `services/media/errors.go`. Upload oversize sentinel is **`pkg/errors.ErrFileExceedsMaxUploadSize`** in `pkg/errors/upload_errors.go`, built from **`constants.MsgFileTooLargeUpload`** in **`constants/error_msg.go`** — **same** constant as `pkg/errcode/messages.go` → `defaultMessages[FileTooLarge]` (no wording drift between JSON `message` and `errors.Is` sentinel).
- **Import rule:** `api/v1/media` is `package media` → handler imports `mycourse-io-be/pkg/media` as **`pkgmedia`** so it does not collide with the handler’s own package name `media`.
- Docs/snapshots updated again: this plan, `docs/reusable-assets.md`, `.context/session_summary_2026-04-26_phase_sub03_upload_cap.md`; `npx gitnexus analyze --force` after edits.

## Phase Sub 04 — B2/CDN URL, object keys, Bunny split + provider errcodes (tasks 01–10, 2026-04-27)

### Task 01 — Đầu / Giữa / Cuối (baseline + scope) ✅
- **Đầu:** Re-read `AGENTS.md`, `docs/patterns.md`, `docs/modules/media.md`, Sub04 intent: align with project layout (constants → errcode; media-specific keys in `pkg/logic/helper`; generic URL + random in `pkg/logic/utils`; tests under `tests/` only for this slice).
- **Giữa:** Reference docs confirmed at `temporary-docs/chuc-nang-upload/openedu-core-arch.md` (§4.2/§5.5/§5.7) and `temporary-docs/chuc-nang-upload/chuc-nang-bo-sung.md`. Five baseline groups locked: (1) CDN URL `<cdn>/<bucket>/<key>` via `JoinURLPathSegments`; (2) B2 bucket from `setting.MediaSetting.B2Bucket` (not hardcoded); (3) 8-digit prefix for B2 objects only; (4) Bunny 2-step pipeline (CreateVideo POST + UploadContent PUT) — **poll/webhook (openedu-core §5.2/§5.7) deferred to tasks 11–20** per task scope ("2 bước" in task 09); (5) `VideoMetadata` entity has BunnyVideoID/LibraryID/Duration/VideoProvider.
- **Cuối:** This section is the authoritative task checklist for Sub04; implementation matches rows below.

### Task 02 — Đầu / Giữa / Cuối (inventory) ✅
- **Đầu:** Touch list: `pkg/media/clients.go`, `pkg/errors/provider_error.go`, `services/media/file_service.go`, `api/v1/media/file_handler.go`, `pkg/logic/helper/media_upload_keys.go`, `pkg/logic/helper/media_metadata.go`, `pkg/logic/utils/url.go`, `pkg/logic/utils/random.go`, `pkg/entities/file.go`, `pkg/errcode/{codes,messages}.go`, `constants/error_msg.go`, five `.env*.example`, `tests/sub04_media_pipeline_test.go`, `docs/modules/media.md`.
- **Giữa:** GitNexus `impact` on `UploadB2` / `CreateFile` / `UploadBunnyVideo` → LOW blast radius (orchestration + single handler). `api/router.go` webhook group noted in inventory; implementation deferred to tasks 11–20.
- **Cuối:** No tests added under `pkg/logic/utils` (module tests live in `tests/`).

### Task 03 — B2 bucket after `setting.Setup()` ✅
- **Đầu:** `NewCloudClientsFromEnv` stays **env-only** (approved) so blazer can authenticate at startup.
- **Giữa:** `effectiveB2Bucket()` prefers `setting.MediaSetting.B2Bucket` (YAML / expanded env), else env bucket from constructor; used for `b2.Client.Bucket` and for URL path segment.
- **Cuối:** Empty resolved bucket at upload/delete → `ProviderError` **9010** (`B2BucketNotConfigured`).

### Task 04 — `.env*.example` comments ✅
- **Đầu:** All five `/.env*.example` files mention `MEDIA_B2_BUCKET` role in `<cdn>/<bucket>/<object_key>` and that YAML `media.b2_bucket` can override. All `config/app-*.yaml` files have `b2_bucket: "${MEDIA_B2_BUCKET}"`.
- **Giữa / Cuối:** Comments only; no secrets committed.

### Task 05–06 — B2 public URL + tests ✅
- **Đầu:** No duplicated `//`; CDN base trimmed.
- **Giữa:** `pkg/logic/utils.JoinURLPathSegments` builds `URL` in `UploadB2` (`cdn/bucket/key`) and `BuildPublicURL` (B2 default branch). Edge cases: CDN trailing slash, bucket trailing slash, empty bucket (→ `cdn/key` only), leading slash in objectKey (stripped). Tests added: `TestBuildPublicURL_B2_trailingSlashVariants` (4 subtests), `TestBuildPublicURL_B2_emptyBucket`, `TestBuildPublicURL_B2_leadingSlashInKey`.
- **Cuối:** `go test ./tests/...` — 8 tests PASS.

### Task 07 — Eight random digits (B2 key prefix) ✅
- **Đầu:** Cryptographic decimal digits, length 8, using `crypto/rand`.
- **Giữa:** `pkg/logic/utils.GenerateRandomDigits`; `helper.BuildB2ObjectKey` = `digits + "-" + sanitized filename` (B2 only). Bunny path uses empty objectKey (filename becomes title, GUID returned from API).
- **Cuối:** `TestGenerateRandomDigits`: n=8, length check, all-digit check, 20-sample uniqueness check. PASS.

### Task 08 — Resolve upload object key in service ✅
- **Đầu:** Bunny default key remains empty until API returns GUID; local keeps nano-based key; explicit `object_key` still wins.
- **Giữa:** `helper.ResolveMediaUploadObjectKey` used from `CreateFile`; B2 path → `BuildB2ObjectKey`; Bunny path → `""` (Bunny self-generates GUID); Local path → nano-based. `uploadToProvider` routes correctly per provider.
- **Cuối:** `uploadToProvider` receives correct key per provider; `TestResolveMediaUploadObjectKey_byProvider` PASS.

### Task 09 — Bunny Stream create vs upload ✅
- **Đầu:** Two HTTP steps per openedu-core §5.5: `POST …/videos` then `PUT …/videos/{guid}`.
- **Giữa:** Create response body read once then `json.Unmarshal` to `guid`; response `ObjectKey` = **GUID** (not client-supplied placeholder). Return URL = `<BunnyStreamBaseURL>/<libraryID>/<guid>`. GetVideoById status check (§5.5 step 3) and webhook (§5.7) are out-of-scope for this task ("2 bước" scope) → tasks 11–20.
- **Cuối:** Metadata sets `video_guid` (compat alias), `bunny_video_id`, `bunny_library_id`, `video_provider=bunny_stream`.

### Task 10 — Provider errors + HTTP mapping ✅
- **Đầu:** Typed `pkg/errors.ProviderError` with `errcode` **9011–9014** for Bunny; **9010** for missing B2 bucket.
- **Giữa:** All five `UploadBunnyVideo` error branches mapped via `ProviderError`: config missing (9011→500), create failed (9012→502), invalid response/GUID empty (9014→502), upload failed (9013→502). `file_handler.go` uses `AsProviderError` + `HTTPStatusForProviderCode`. Shared default copy in `constants/error_msg.go`; `pkg/errcode/messages.go` references constants. `DeleteBunnyVideo` uses raw `fmt.Errorf` (task scope limited to UploadBunnyVideo); consistent with `deleteFile` handler not calling `AsProviderError`.
- **Cuối:** `VideoMetadata.video_provider` + `BuildTypedMetadata` reads `video_provider` from raw metadata. `go build ./... && go vet ./... && go test ./...` all PASS.

## Phase Sub 04 — Tasks 11–20 (status/webhook/metadata/env/docs/audit)

### Task 11 ✅
- Added `pkg/logic/utils/bunny_status.go`:
  - `type BunnyVideoStatus`
  - status constants `0..8`
  - `StatusString() string`
  - `FinishedWebhookBunnyStatus = 4`
- Added full-case unit tests (including invalid -> `unknown`) in `tests/sub04_media_pipeline_test.go`.

### Task 12 ✅
- Added endpoint: `GET /api/v1/media/videos/:id/status`.
- Added service: `services/media.GetVideoStatus(ctx, videoGUID)`.
- Added DTO: `dto.VideoStatusResponse{status}`.
- Bunny API call uses `CloudClients.GetBunnyVideoByID`.
- Added errcodes for status path:
  - `9015` Bunny video not found
  - `9016` Bunny get-video failed

### Task 13 ✅
- Added DTO `dto/BunnyVideoWebhookRequest`.
- Added handler file `api/v1/media/webhook_handler.go` with `bunnyWebhook`.
- Added route `POST /api/v1/webhook/bunny` mounted before auth middleware via `api/router.go`.
- Follow-up refactor for maintainability: introduced `RegisterNoFilterRoutes` in `api/v1/routes.go` and mount it from a dedicated `/api/v1` no-filter lane in `api/router.go`.

### Task 14 ✅
- Added service `services/media.HandleBunnyVideoWebhook`.
- Only processes status `utils.FinishedWebhookBunnyStatus` (`4`).
- Reuses regex constant `utils.SignBunnyIFrameRegex` in skeleton cleanup path.
- Added explicit TODO for future DB persistence:
  - `// TODO: persist duration xuống bảng files/lessons khi DB ready.`

### Task 15 ✅
- Extended `pkg/entities/file.go`:
  - file-level fields: `BunnyVideoID`, `BunnyLibraryID`, `Duration`, `VideoProvider`.
  - expanded typed `VideoMetadata` with fields:
    - `Width`, `Height`, `Duration`, `Bitrate`, `FPS`, `VideoCodec`, `AudioCodec`, `HasAudio`, `IsHDR`.

### Task 16 ✅
- Updated `helper.BuildTypedMetadata` for `kind=video`:
  - reads Bunny metadata keys (`length`, `width`, `height`, `framerate`, etc.).
  - missing data stays zero-value (no fabricated values).
- Updated `mapping.ToUploadFileResponse` to include top-level video fields from `entities.File`.
- Added mapping test fixture in `tests/sub04_media_pipeline_test.go`.

### Task 17 ✅
- Synced media blocks across 5 env examples:
  - `.env.example`
  - `.env.local.example`
  - `.env.staging.example`
  - `.env.prod.example`
  - `.env.test.example`
- Included full key set and bucket comment:
  - `MEDIA_B2_BUCKET` comment now references URL shape `<gcore_cdn>/<bucket>/<file>`.

### Task 18 ✅
- Synced docs:
  - `docs/modules/media.md`
  - `README.md`
  - `docs/data-flow.md`
  - `docs/modules.md`
  - `docs/api-overview.md`
  - `docs/reusable-assets.md`
  - `IMPLEMENTATION_PLAN_EXECUTION.md` (this section)
- Quality gate:
  - `go fmt ./...` ✅
  - `go vet ./...` ✅
  - `go build ./...` ✅
  - `go test ./...` ✅

### Task 19 ✅ — Audit checklist (code-vs-doc-vs-plan)
- (a) CDN URL format `<cdn>/<bucket>/<file>`: PASS (via `JoinURLPathSegments`; no double slash).
- (b) Bucket source from `setting.MediaSetting.B2Bucket`: PASS.
- (c) B2 object key prefix 8 digits + `-` + sanitized name: PASS.
- (d) Bunny video no 8-digit prefix, GUID-based key/title flow: PASS.
- (e) Bunny pipeline + status endpoint + webhook outside auth middleware: PASS.
- (f) Entity/DTO fields (`BunnyVideoID`, `BunnyLibraryID`, `Duration`, `VideoProvider`) + typed `VideoMetadata`: PASS.
- (g) Five env example files synchronized: PASS.
- (h) docs + `docs/*` + plan synchronized: PASS.
- (i) quality gate clean: PASS.

### Task 20 ✅
- Re-ran index refresh: `gitnexus analyze --force`.
- Ran impact checks for requested symbols:
  - `UploadB2`, `UploadBunnyVideo`, `BuildPublicURL`, `CreateFile` (LOW risk).
  - Note: symbol `BuildObjectKey` is represented in this repo as `BuildB2ObjectKey`.
- GitNexus detect_changes MCP tool is not available through current CLI surface; equivalent verification done via git diff + impact checks + successful quality gate.
- Updated reusable-assets and added session handoff context summary for sub04 tasks 11–20.

---

## Documentation Resync Update (2026-04-26 - full docs pass)

- Re-synced docs folder after sub02 helper/util/settings refactor:
  - `docs/architecture.md` (public media endpoint surface + media module reference)
  - `docs/requirements.md` (added FR-11 Media Upload Gateway with helper/util + settings constraints)
  - `docs/return_types.md` (added media API return-shape section)
  - `docs/curl_api.md` (added media upload/get/update/delete/local-decode cURL section)
  - `docs/modules/media.md` (runtime config source-of-truth + util extraction notes)
- Re-synced root docs:
  - `README.md` (media module description aligned with helper-vs-util convention)
  - this execution plan file

## Phase Sub 02 RESET Update (2026-04-26 - provider helper move + util split + settings source hardening)

### Scope completed for tasks 01->10 in this cycle
- Re-ran baseline/context/doc refresh and GitNexus indexing/status before edits.
- Moved `defaultMediaProvider` out of `services/media/file_service.go` into helper layer (`pkg/logic/helper/media_metadata.go`) as `DefaultMediaProvider`.
- Moved generic functions out of `pkg/logic/helper/media_metadata.go` into util layer (`pkg/logic/utils/parsing.go`; later additions include `ParseBoolLoose`, `ContentFingerprint`):
  - `DetectExtension`
  - `ImageSizeFromPayload`
  - `StringFromRaw`
  - `IntFromRaw`
  - `FloatFromRaw`
  - `NonEmpty`
- Updated media metadata helper to consume util primitives and remain media-feature focused.
- Updated `services/media/file_service.go` call-sites to use `helper.DefaultMediaProvider(...)` (service no longer owns default-provider helper).
- Refactored `pkg/media/clients.go` runtime config reads to `setting.MediaSetting` after `setting.Setup()`:
  - `UploadLocal`, `UploadB2`, `UploadBunnyVideo`, `DeleteBunnyVideo`, `BuildPublicURL`
  - kept approved env-only path in `NewCloudClientsFromEnv`.
- Verified startup chain compatibility (`main.go` -> `setting.Setup()` -> `pkg/media.Setup()` -> `NewCloudClientsFromEnv`) remains intact.
- Synced docs:
  - `README.md`
  - `docs/modules/media.md`
  - `docs/modules.md`
  - `docs/patterns.md`
  - `docs/reusable-assets.md`
  - this plan file

## Phase Sub 02 RESET Update (2026-04-26 - mapper + provider-from-env contract)

### Scope completed for tasks 01->10 in this cycle
- Re-ran baseline and context/docs recovery before implementation:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status`
  - re-read `.context/*`, `docs/*`, `docs/*`, `README.md`, and this plan file.

### Media contract hardening
- Added mapper layer in `pkg/logic/mapping/*_mapping.go` (`media_file_mapping.go`, `taxonomy_category_mapping.go`, `taxonomy_course_level_mapping.go`, `taxonomy_tag_mapping.go`) and routed media handler responses through `mapping.ToUploadFileResponse`.
- Standardized public media payload to `dto.UploadFileResponse` only.
- Removed public `provider` field from `dto.UploadFileResponse` and removed provider from media request DTO fields.
- Updated media handlers to stop accepting provider from client query/body in create/update/get/delete flow.

### Provider source-of-truth update
- Added config field `MediaSetting.AppMediaProvider` in `pkg/setting/setting.go` (`yaml media.app_media_provider`).
- Updated media service provider selection to use server-side config only (`defaultMediaProvider`).
- Client cannot override provider anymore; upload/delete flow keeps behavior based on configured provider + media kind.

### Taxonomy DTO baseline + mapping
- Updated taxonomy handlers (`category`, `course_level`, `tag`) to map all responses via `pkg/logic/mapping`:
  - `CategoryResponse`
  - `CourseLevelResponse`
  - `TagResponse`
- Handlers no longer return raw model/entity payload directly.

### Quality gate + sync
- Executed:
  - `gofmt -w` on changed files
  - `go test ./...` (pass)
  - `go build ./...` (pass)
- Synced docs:
  - `docs/modules/media.md`
  - `docs/modules.md`
  - `docs/data-flow.md`
  - `docs/reusable-assets.md`
  - this plan file
# IMPLEMENTATION_PLAN_EXECUTION


## Global Type Placement Rule (Mandatory)

- For all new code from now on, if a module contains logic handling (including under `pkg/*`, `services/*`, `repository/*`, and similar layers), newly introduced reusable types must be declared in `pkg/entities`.
- Do not declare new reusable/domain types inline inside logic implementation files.
- Use `pkg/entities` for both new and reused domain types (create a new entity module file or extend an existing one), then import those types where needed.

## Global Constants Placement Rule (Mandatory)

- All constants from all features must be centralized under `constants/*`, including setting constants, type constants, enums, status constants, default values, thresholds/limits, and message constants.
- Do not declare business constants directly inside `services/*`, `repository/*`, `api/*`, `pkg/*`, `models/*`, or other feature folders.
- If a new constant is needed, create or extend an appropriate file in `constants/` and import it from there.

## Discovery Summary
- Performed zero-code discovery only; no application source code was modified.
- Completed mandatory root-file read and protected/read-only checks.
- Ran `npx gitnexus analyze --force` before discovery and used GitNexus-backed exploration for all subsequent lanes.
- Executed 8 discovery lanes:
  - S1 folder structure/traversal
  - S2 API map
  - S3 data flow
  - S4 modules/dependencies
  - S5 DB schema+migration impact
  - S6 RBAC/permission matrix impact
  - S7 reusable-assets deep scan
  - S8 testing/validation surface
- Aggregated lane outputs and resolved conflicts:
  - Conflict: MCP instability in some lanes; resolved by fallback to GitNexus CLI with source validation.
  - Conflict: route graph extraction incomplete in GitNexus route tools; resolved using direct router/handler evidence.
- `.context/` exists but currently empty; ingestion/reconstruction/validation/integration completed as no-op with no missing artifacts.

## Folder Structure
- Full root-to-deepest tree and per-folder purpose are documented in:
  - `docs/folder-structure.md`
- Coverage includes workspace-level hidden folders and all source/ops folders.

## Module Responsibilities
- Current implemented domains:
  - auth
  - user self
  - internal RBAC admin
  - system operations/synchronization
- Planned domains (not yet implemented): course/lesson/enrollment + full e-learning/commerce interactions.
- Detailed module map is in:
  - `docs/modules.md`

## Data Flow
- Current verified end-to-end flows:
  - register
  - login
  - confirm email
  - refresh token/session
  - me profile read
  - permission check middleware fallback
  - system sync now/scheduler controls
- Detailed flow artifacts are in:
  - `docs/data-flow.md`
  - `docs/logic-flow.md`

## Related Features
- Shared security/authorization core:
  - `middleware/auth_jwt.go`
  - `middleware/rbac.go`
  - `services/rbac.go`
  - `constants/permissions.go`
  - `constants/roles_permission.go`
  - `internal/rbacsync/*`
- Shared response/error core:
  - `pkg/response/*`
  - `pkg/errcode/*`
  - `pkg/httperr/*`
- Shared transport/model patterns:
  - `dto/BaseFilter`
  - GORM + migration pipeline.

## Task Analysis

### 6.1 Objective/Constraints/Outcome
- Objective: prepare strict pre-implementation discovery + actionable implementation plan for full e-learning CRUD rollout.
- Constraints:
  - keep RBAC/middleware engine behavior intact
  - no role additions/removals
  - no mutation of protected/read-only reference docs
  - no coding before explicit approval
- Expected outcome: phase-by-phase plan with reuse-first mapping and exact change surfaces.

### 6.2 System Mapping (Detailed For Phase 01-03)

#### Phase 01 - Taxonomy (course levels, categories, tags)
- **Files to add (planned):**
  - `migrations/000002_taxonomy_domain.up.sql`
  - `migrations/000002_taxonomy_domain.down.sql`
  - `dbschema/taxonomy.go`
  - `models/taxonomy/course_level.go`
  - `models/taxonomy/category.go`
  - `models/taxonomy/tag.go`
  - `dto/taxonomy/course_level_dto.go`
  - `dto/taxonomy/category_dto.go`
  - `dto/taxonomy/tag_dto.go`
  - `repositories/taxonomy/course_level_repository.go`
  - `repositories/taxonomy/category_repository.go`
  - `repositories/taxonomy/tag_repository.go`
  - `services/taxonomy/course_level_service.go`
  - `services/taxonomy/category_service.go`
  - `services/taxonomy/tag_service.go`
  - `api/v1/taxonomy/course_level_handler.go`
  - `api/v1/taxonomy/category_handler.go`
  - `api/v1/taxonomy/tag_handler.go`
  - `api/v1/taxonomy/routes.go`
  - `pkg/query/filter_parser.go` (new reusable parsing/whitelist helper)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount taxonomy sub-routes from `api/v1/taxonomy/routes.go`)
  - `repositories/repository.go` (refactor repository root module to host taxonomy/course/course_edit repository composition and constructor wiring)
  - `constants/permissions.go` (add taxonomy permission IDs/names)
  - `constants/roles_permission.go` (map taxonomy permissions to roles)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/taxonomy/course_level_repository.go`:
    - `ListCourseLevels(...)`
    - `CreateCourseLevel(...)`
    - `GetCourseLevelByID(...)`
    - `UpdateCourseLevel(...)`
    - `DeleteCourseLevel(...)`
  - `repositories/taxonomy/category_repository.go`:
    - `ListCategories(...)`, `CreateCategory(...)`, `GetCategoryByID(...)`, `UpdateCategory(...)`, `DeleteCategory(...)`
  - `repositories/taxonomy/tag_repository.go`:
    - `ListTags(...)`, `CreateTag(...)`, `GetTagByID(...)`, `UpdateTag(...)`, `DeleteTag(...)`
  - `services/taxonomy/course_level_service.go`:
    - `ListCourseLevels(...)`, `CreateCourseLevel(...)`, `UpdateCourseLevel(...)`, `DeleteCourseLevel(...)`
  - `services/taxonomy/category_service.go`:
    - `ListCategories(...)`, `CreateCategory(...)`, `UpdateCategory(...)`, `DeleteCategory(...)`
  - `services/taxonomy/tag_service.go`:
    - `ListTags(...)`, `CreateTag(...)`, `UpdateTag(...)`, `DeleteTag(...)`
  - `api/v1/taxonomy/*_handler.go`:
    - `listCourseLevels`, `createCourseLevel`, `updateCourseLevel`, `deleteCourseLevel`
    - `listCategories`, `createCategory`, `updateCategory`, `deleteCategory`
    - `listTags`, `createTag`, `updateTag`, `deleteTag`
  - `pkg/query/filter_parser.go`:
    - `ParseListFilter(...)`
    - `BuildSortClause(...)`
    - `BuildSearchClause(...)`
- **Types to implement (planned):**
  - `models/taxonomy/course_level.go`: `CourseLevel`
  - `models/taxonomy/category.go`: `Category`
  - `models/taxonomy/tag.go`: `Tag`
  - `dto/taxonomy/course_level_dto.go`: `CreateCourseLevelRequest`, `UpdateCourseLevelRequest`, `CourseLevelFilter`, `CourseLevelResponse`
  - `dto/taxonomy/category_dto.go`: `CreateCategoryRequest`, `UpdateCategoryRequest`, `CategoryFilter`, `CategoryResponse`
  - `dto/taxonomy/tag_dto.go`: `CreateTagRequest`, `UpdateTagRequest`, `TagFilter`, `TagResponse`
  - `repositories/repository.go`: `Repository` root struct refactor with domain repository members + constructor dependency injection shape
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - `dto.BaseFilter`
    - `middleware.RequirePermission`
    - `pkg/response` + `pkg/errcode` + `pkg/httperr`
    - `pkg/sqlnamed.Postgres` (nếu dùng raw query)
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/query/filter_parser.go` (không đặt trong service layer);
    - Phase 02/03 bắt buộc dùng lại asset này thay vì tạo helper mới trong service.

#### Phase 02 - Course shell (course base CRUD + list/filter)
- **Files to add (planned):**
  - `migrations/000003_course_shell.up.sql`
  - `migrations/000003_course_shell.down.sql`
  - `dbschema/course.go`
  - `models/course/course.go`
  - `dto/course/course_dto.go`
  - `repositories/course/course_repository.go`
  - `services/course/course_service.go`
  - `api/v1/course/course_handler.go`
  - `api/v1/course/routes.go`
  - `pkg/policy/course_policy.go` (new reusable ownership/role policy asset)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount `api/v1/course/routes.go`)
  - `repositories/repository.go` (extend root repository module with `course` repository member and builder wiring)
  - `constants/permissions.go` (if missing required course shell perms)
  - `constants/roles_permission.go` (course-shell mapping)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/course/course_repository.go`:
    - `CreateCourse(...)`
    - `GetCourseByID(...)`
    - `ListCourses(...)`
    - `UpdateCourse(...)`
    - `SoftDeleteCourse(...)`
  - `services/course/course_service.go`:
    - `CreateCourse(instructorID uint, req dto.CreateCourseRequest) (...)`
    - `GetCourseByID(courseID uint, actorID uint, roleSet map[string]struct{}) (...)`
    - `ListCourses(filter dto.CourseFilter, actorID uint, roleSet map[string]struct{}) (...)`
    - `UpdateCourse(courseID uint, actorID uint, req dto.UpdateCourseRequest) (...)`
    - `DeleteCourse(courseID uint, actorID uint) error`
  - `api/v1/course/course_handler.go`:
    - `createCourse`, `getCourse`, `listCourses`, `updateCourse`, `deleteCourse`
  - `pkg/policy/course_policy.go`:
    - `CanManageCourse(actorID uint, courseInstructorID uint, roleSet map[string]struct{}) bool`
    - `CanViewCourse(actorID uint, courseInstructorID uint, roleSet map[string]struct{}, isPublished bool) bool`
- **Types to implement (planned):**
  - `models/course/course.go`: `Course`
  - `dto/course/course_dto.go`: `CreateCourseRequest`, `UpdateCourseRequest`, `CourseFilter`, `CourseResponse`
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - taxonomy IDs/relations từ Phase 01 models
    - `dto.BaseFilter`, response/error stack, permission middleware, `pkg/query/filter_parser.go`
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/policy/course_policy.go` (shared policy layer);
    - service chỉ gọi policy + repository, không chứa reusable policy helper.

#### Phase 03 - Course edits/versioning (draft/edit lifecycle)
- **Files to add (planned):**
  - `migrations/000004_course_edits.up.sql`
  - `migrations/000004_course_edits.down.sql`
  - `dbschema/course_edit.go`
  - `models/course_edit/course_edit.go`
  - `dto/course_edit/course_edit_dto.go`
  - `repositories/course_edit/course_edit_repository.go`
  - `services/course_edit/course_edit_service.go`
  - `api/v1/course_edit/course_edit_handler.go`
  - `api/v1/course_edit/routes.go`
  - `pkg/workflow/course_edit_state_machine.go` (new reusable state transition asset)
- **Files to modify (planned):**
  - `api/v1/routes.go` (mount `api/v1/course_edit/routes.go`)
  - `services/course/course_service.go` (if publishing flow updates `published_edit_id`)
  - `repositories/repository.go` (extend root repository module with `course_edit` repository member and builder wiring)
  - `constants/permissions.go` and `constants/roles_permission.go` (approval/versioning perms for admin/instructor if missing)
- **Files to delete (planned):**
  - none
- **Functions to implement (planned):**
  - `repositories/course_edit/course_edit_repository.go`:
    - `CreateCourseEdit(...)`
    - `GetCourseEditByID(...)`
    - `ListCourseEdits(...)`
    - `UpdateCourseEdit(...)`
    - `SetCourseEditStatus(...)`
    - `PublishCourseEdit(...)`
  - `services/course_edit/course_edit_service.go`:
    - `CreateCourseEdit(courseID uint, actorID uint, req dto.CreateCourseEditRequest) (...)`
    - `GetCourseEdit(editID uint, actorID uint) (...)`
    - `ListCourseEdits(courseID uint, actorID uint, filter dto.CourseEditFilter) (...)`
    - `UpdateCourseEdit(editID uint, actorID uint, req dto.UpdateCourseEditRequest) (...)`
    - `SubmitCourseEdit(editID uint, actorID uint) error`
    - `ApproveCourseEdit(editID uint, adminID uint, req dto.ApproveCourseEditRequest) error`
    - `RejectCourseEdit(editID uint, adminID uint, req dto.RejectCourseEditRequest) error`
  - `api/v1/course_edit/course_edit_handler.go`:
    - `createCourseEdit`, `getCourseEdit`, `listCourseEdits`, `updateCourseEdit`, `submitCourseEdit`, `approveCourseEdit`, `rejectCourseEdit`
  - `pkg/workflow/course_edit_state_machine.go`:
    - `ValidateCourseEditTransition(from, to string) error`
    - `IsTerminalCourseEditState(state string) bool`
- **Types to implement (planned):**
  - `models/course_edit/course_edit.go`: `CourseEdit`
  - `dto/course_edit/course_edit_dto.go`: `CreateCourseEditRequest`, `UpdateCourseEditRequest`, `CourseEditFilter`, `ApproveCourseEditRequest`, `RejectCourseEditRequest`, `CourseEditResponse`
- **Reuse strategy in this phase:**
  - Reuse trực tiếp:
    - `CanManageCourse` + `CanViewCourse` (from `pkg/policy/course_policy.go`)
    - `pkg/query/filter_parser.go`
    - response/error/validation stack
    - `dto.BaseFilter`
  - Nếu thiếu reusable asset:
    - tạo mới ở `pkg/workflow/course_edit_state_machine.go` (không đặt trong service);
    - các phase sau (section/lesson/quiz workflow) phải tái sử dụng hoặc mở rộng module workflow này.

#### Repository Root Refactor (applies across Phase 01-03)
- **Target file:** `repositories/repository.go`
- **Refactor objective:**
  - tách hoàn toàn repository layer khỏi `models/`,
  - đặt repository layer tại `repositories/<domain>/`,
  - dùng `repositories/repository.go` làm composition root cho tất cả domain repositories.
- **Planned refactor items:**
  - convert current minimal `Repository` into a structured container with domain members (`taxonomy`, `course`, `course_edit`).
  - add constructor wiring that initializes each domain repository with shared DB dependency.
  - expose typed accessors (or public fields) so services resolve repositories via the root module instead of creating ad-hoc repository instances.
  - keep backward compatibility path for existing code using `NewRepository()` via repository root module during transition.
- **Planned outputs tied to this refactor:**
  - Phase 01 registers taxonomy repository set.
  - Phase 02 extends root repository with course repository.
  - Phase 03 extends root repository with course-edit repository.

### Non-target Modules (unless mandatory minimal touch)
- core auth middleware internals
- system token middleware engine
- queue placeholder logic

### 6.3 Cross-check with `docs/`
- Cross-checked all planning assumptions with newly written snapshot files.
- Reuse baseline is anchored in `docs/reusable-assets.md`.

### 6.3.1 Reusability Check (Mandatory)
- Existing reusable foundations confirmed:
  - `dto.BaseFilter`
  - `RequirePermission`
  - response/error envelopes
  - RBAC permission resolution/service patterns
  - SQL named-parameter helper and schema namespace pattern.

### 6.3.2 Reuse Enforcement
- Any phase implementation must reuse existing shared assets where available.
- New shared logic is allowed only when a genuine gap exists and must be placed in proper module/folder (no dumping into one utility file).

### 6.4 Technical Direction
- Add e-learning schema via staged migrations.
- Add domain models/DTO/services/routes per phase complexity.
- Extend RBAC catalog with `P14+` and role mappings without altering role identities.
- Keep ownership/permission checks at service + middleware boundary.

## Reusability Strategy

### Existing Assets To Reuse
- Authorization and session:
  - `AuthJWT`, `RequirePermission`, `PermissionCodesForUser`, `UserHasAllPermissions`.
- API contracts:
  - `pkg/response`, `pkg/errcode`, `pkg/httperr`.
- List/query patterns:
  - `dto.BaseFilter`, sort/search whitelist approach.
- SQL utility patterns:
  - `pkg/sqlnamed.Postgres`, `dbschema` namespace approach.

### New Reusable Assets To Create (When Implementing)
- Domain schema namespaces under `dbschema/` per new bounded context.
- Ownership-check helpers for instructor/learner/admin permissions.
- Shared DTO fragments for pagination/filtering and common identifiers.
- Shared query builders for reusable complex list/search operations.

### Reusability Validation Before Implement
- For each phase core, map each CRUD/query to:
  - existing reusable asset (reuse),
  - or new reusable asset (create once, reuse later).
- Update `docs/reusable-assets.md` whenever reusable logic is introduced/changed.

## CRUD/Query Mapping (Phase 01-12)

| Phase | Domain | Planned CRUD/Query | Reuse Now | New Asset Needed |
|---|---|---|---|---|
| 01 | Taxonomy | levels/categories/tags CRUD + list/filter | `BaseFilter`, `RequirePermission`, response/error stack | taxonomy schema/model/service/route set |
| 02 | Course shell | course CRUD + list/filter | same as above + ownership check pattern | course base model/DTO/service |
| 03 | Course edits/versioning | edits CRUD + state transitions | auth/RBAC services + response stack | edit state machine helpers |
| 04 | Metadata+junction | category/tag/level binding queries | `sqlnamed` pattern, list patterns | reusable join query helpers |
| 05 | Sections/lessons | tree CRUD + reorder | response/error + permission gates | ordering/reorder shared helper |
| 06 | Text/video/subtitle | content CRUD + media metadata | validation patterns + ownership checks | content payload helpers |
| 07 | Quiz authoring | nested quiz/question/choice CRUD | dto/validation patterns | nested aggregate transaction helper |
| 08 | Course series | series CRUD + ordered course mapping | list/filter + RBAC checks | ordered M:N helper |
| 09 | Coupons/scope | coupon CRUD + condition queries | sql helper + error codes | scope predicate/query builder |
| 10 | Orders/items | order/item CRUD + aggregates | response pagination + auth context | order aggregate query helpers |
| 11 | Enrollments | enrollment CRUD + ownership checks | permission + ownership patterns | enrollment uniqueness helper |
| 12 | Progress/attempts/reviews | progress/review CRUD + analytics queries | auth/session + list patterns | progress/review shared computation helpers |

## Action Plan (Pre-Approved Design Only)

### Planned File Surfaces By Layer
- **Migrations**
  - Add new migration pairs from `000002_*` onward in `migrations/`.
  - Estimated LoC (SQL): 1800-3000 across all phases.
- **Models**
  - Add domain models in `models/` for each phase domain.
  - Estimated LoC: 800-1400.
- **Repositories (separated from models)**
  - Add domain repositories in `repositories/<domain>/`.
  - Refactor `repositories/repository.go` as repository composition root and constructor wiring module.
  - Estimated LoC: 500-900.
- **DTO**
  - Add request/query/response DTOs in `dto/`.
  - Estimated LoC: 700-1200.
- **Services**
  - Add domain business logic files in `services/`.
  - Estimated LoC: 1800-3200.
- **Routes/Handlers**
  - Add/extend route registration and handlers in `api/v1/`.
  - Estimated LoC: 1200-2200.
- **RBAC catalog**
  - Extend permission constants and role mappings:
    - `constants/permissions.go`
    - `constants/roles_permission.go`
  - Estimated LoC: 150-300.
- **Sync/ops usage**
  - Reuse existing sync commands and document run-order after permission updates.

### Planned Logic Sequence Per Phase Core
1. DDL/migration changes.
2. Model + Repository + DTO definitions.
3. Service logic (ownership, business constraints, repository orchestration).
4. Route wiring + middleware permissions.
5. Query optimization and list/filter behavior.
6. Documentation sync (`docs/ + `.context`).

## Discovery Phases (1->5) and Output Artifacts
- Phase 1 Architecture (S1+S4):
  - `docs/architecture.md`
  - `docs/folder-structure.md`
- Phase 2 Documentation (S7):
  - `docs/modules.md`
  - `docs/patterns.md`
  - `docs/reusable-assets.md`
- Phase 3 API (S2+S6):
  - `docs/api-overview.md`
  - `docs/router.md`
  - `docs/api.md`
- Phase 4 Data flow (S3+S8):
  - `docs/data-flow.md`
  - `docs/logic-flow.md`
- Phase 5 Targeted code reading (S5 + hot paths):
  - `docs/dependencies.md`
  - DB/RBAC impact captured in this plan and reusable inventory.

## Post-Approval Discipline (Reminder)
- After explicit approval only:
  - maintain strict phase order (`phase-NN-start -> core -> end`)
  - keep docs synchronized in markdown files (not chat-only)
  - update reusable-assets whenever shared logic is added/changed
  - run lint/type/build checks and stop immediately on unresolved failures
  - no assumption-driven or partial implementation on blockers.

## Phase Execution Status

### Phase 01 (Taxonomy) - END CHECKPOINT COMPLETED
- Review status:
  - Verified Phase 01 scope includes migration, model, DTO, repository, service, and route wiring for taxonomy CRUD.
  - Verified RBAC extensions for taxonomy permissions (`P14`-`P25`) and role mapping integration.
- Test status:
  - Executed `go test ./...` successfully on 2026-04-23 after Phase 01 implementation.
  - Result: pass (no package test failures; most packages currently have no test files).
- Documentation sync status:
  - Updated `docs/api-overview.md` with taxonomy endpoint inventory.
  - Updated `docs/router.md` with taxonomy route registration topology.
  - Updated `docs/data-flow.md` with taxonomy CRUD flow.
  - Updated `docs/modules.md` to mark taxonomy as implemented.
  - Updated `docs/reusable-assets.md` with newly introduced reusable helpers.
- Reusable-assets closure:
  - Added reusable assets for `pkg/query/filter_parser.go` and `pkg/requestutil/params.go`.
  - Updated gap analysis to remove taxonomy from missing-domain list.

### Next Gate
- Phase 01 is closed.
- Ready to begin `phase-02-start` only after this checkpoint is accepted.

## Phase Sub 01 Execution Update (2026-04-25)

### Task 01 / Task 04 - Continuous Discovery + Sync
- Re-read active context from `.context/` and existing implementation snapshot docs.
- Re-ran GitNexus indexing with `gitnexus analyze --force` before making refactor changes.
- Performed GitNexus impact checks for symbols touched in this sub-phase (`SetupRedis`, `CourseLevel`, `Category`, `taxonomyNS`) and proceeded with only LOW-risk results.
- Synced this file with the exact refactor surfaces completed in Task 02 and Task 03.

### Task 02 - Move `cache_clients` into `pkg` + shared entities example
- Moved Redis client package from `cache_clients/redis.go` to `pkg/cache_clients/redis.go`.
- Updated all imports referencing the old location:
  - `main.go`
  - `services/cache/auth_user.go`
- Introduced shared entities package `core/entities` and migrated two taxonomy modules as reusable examples:
  - `core/entities/course_level.go`
  - `core/entities/category.go`
- Models remain the persistence layer and own table mapping (`TableName`) while reusing shared fields from core entities:
  - `models/taxonomy_course_level.go`
  - `models/taxonomy_category.go`

### Task 03 - Split `dbschema/taxonomy.go`
- Split taxonomy schema namespace into focused files under `dbschema/`:
  - `taxonomy_namespace.go`
  - `taxonomy_course_levels.go`
  - `taxonomy_categories.go`
  - `taxonomy_tags.go`
- Removed the previous monolithic file `dbschema/taxonomy.go`.
- Existing call sites remain stable (`dbschema.Taxonomy.*`) so no consumer import changes were required.

### Validation
- Executed `go test ./...` successfully after all refactors.

## Phase Sub 01 Rework Update (2026-04-25 - sync 2)

### Request-driven architecture correction
- Enforced strict boundary: `core/entities` now contains only pure type definitions (no table-name mapping, no dbschema dependency).
- Moved table mapping responsibility back to models only:
  - `models/taxonomy_course_level.go` defines model wrapper + `TableName()`.
  - `models/taxonomy_category.go` defines model wrapper + `TableName()`.
- Updated taxonomy service/repository layers to consume `models` types again so ORM-facing flows stay inside model boundary.

### Subtask 01 + Subtask 04 sync loop
- Re-ran `gitnexus analyze --force`.
- Re-read `.context` and execution plan documentation before rework.
- Ran impact checks for touched symbols (`CourseLevel`, `Category`) and continued with LOW risk only.
- Re-synchronized docs in `docs/*` and this execution plan to reflect corrected architecture boundary.

### Validation for rework
- `go test ./...` (pass)

## Phase Sub 02 Rework Update (2026-04-25 - startup SDK initialization)

- Refactored media SDK initialization to app startup lifecycle:
  - added `pkg/media/setup.go` with `Setup()` and shared `pkg/media.Cloud`.
  - wired startup call in `main.go` right after cache setup.
- Removed runtime lazy initialization behavior from service:
  - `services/media/file_service.go` no longer does `sync.Once` lazy client creation.
  - service now requires startup initialization and returns explicit error if not configured.
- Kept media flow stateless and no-DB, while making initialization consistent with existing DB/config startup pattern.

### Validation for startup-init refactor
- `go test ./...` (pass)

## Phase Sub 02 Documentation Sync Update (2026-04-25 - strict sync pass)

- Re-synced canonical docs to match final media architecture:
  - startup SDK init in `pkg/media.Setup()` (no runtime lazy constructor in service)
  - runtime dependency check via `pkg/logic/helper/runtime_guard.go`
  - no DB persistence for media resources
- Updated files:
  - `README.md`
  - `docs/modules.md`
  - `docs/reusable-assets.md`
  - `docs/modules/media.md`
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 3)

### Task 02 (requested in this cycle) - Extract pagination math to shared utils
- Added reusable pagination builder:
  - `pkg/logic/utils/pagination.go` with `utils.BuildPage(page, perPage, totalItems)`.
- Refactored handlers to stop manual `totalPages` math and use shared utility:
  - `api/v1/taxonomy/course_level_handler.go`
  - `api/v1/taxonomy/category_handler.go`
  - `api/v1/taxonomy/tag_handler.go`
  - `api/v1/internal_rbac.go`

### Task 03 (requested in this cycle) - Apply refactor across all manual page calculations
- Completed cross-module replacement for all current manual pagination calculations in API handlers.
- Confirmed no remaining manual `totalPages` blocks outside shared utility implementation.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force` and impact checks before/around edits.
- Re-read context and synchronized docs after implementation.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 7, remove redundant wrappers)

### Task 02 + Task 03 (redefined for this cycle) - Direct util/helper usage in internal RBAC handler
- Removed redundant local wrappers from `api/v1/internal/rbac_handler.go`:
  - `parseUintParam(...)`
  - `parsePermissionIDParam(...)`
- Updated all call-sites in the same file to call shared helpers directly:
  - `utils.ParseUintPathParam(...)`
  - `helper.ParsePermissionIDParam(...)`
- This keeps one source of truth and avoids thin pass-through wrappers.

### Risk handling note
- GitNexus impact before change:
  - wrapper `parseUintParam`: CRITICAL
  - wrapper `parsePermissionIDParam`: HIGH
- Mitigation applied: wrapper removal only; preserved parsing behavior by direct calls to the same underlying helpers.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force`.
- Re-ran impact checks and validated no behavior regression with test/build.
- Synchronized `docs/reusable-assets.md` and this execution plan.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 6, param util/helper extraction)

### Task 02 + Task 03 (redefined for this cycle) - Extract repeated param parsing logic
- User-requested logic from old `api/v1/internal_rbac.go` was extracted and reused in current internal module:
  - uint path param parser -> `pkg/logic/utils/params.go` (`ParseUintPathParam`)
  - permission id parser -> `pkg/logic/helper/permission.go` (`ParsePermissionIDParam`)
- Refactored internal RBAC handlers to use shared helpers:
  - `api/v1/internal/rbac_handler.go`
- Refactored existing shared request util to delegate to new common parser:
  - `pkg/requestutil/params.go` now uses `utils.ParseUintPathParam`, extending reuse to taxonomy handlers.

### Risk handling note
- GitNexus impact:
  - `parseUintParam`: CRITICAL (many direct dependents)
  - `parsePermissionIDParam`: HIGH
  - `pkg/requestutil.ParseUintParam`: CRITICAL
- Mitigation applied: preserved external behavior and only replaced duplicated parsing internals with shared helpers.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force` before edits.
- Performed impact analysis for all modified symbols.
- Updated `docs/reusable-assets.md` and this plan to keep docs synchronized with code changes.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 4, user-requested redo)

### Task 02 + Task 03 (redefined for this cycle) - Add `Tag` into core entity and refactor
- Added pure shared entity:
  - `core/entities/tag.go`
- Refactored `models/taxonomy_tag.go` to keep ORM table mapping in model and reuse core entity fields via embedding.
- Updated `services/taxonomy/tag_service.go` to initialize nested entity payload (`Tag: entities.Tag{...}`) for create flow.

### Task 01 + Task 04 loop for this redo
- Re-ran `gitnexus analyze --force` before the refactor cycle.
- Ran impact checks for `Tag`, `CreateTag`, and related model surfaces, then applied changes with scope control.
- Re-synced `docs/reusable-assets.md` and this plan after code updates.

### Validation for this redo
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Rework Update (2026-04-25 - sync 5, internal route modularization)

### Task 02 + Task 03 (redefined for this cycle) - Move internal RBAC API into `api/v1/internal`
- Moved internal RBAC handlers from monolithic `api/v1/internal_rbac.go` into:
  - `api/v1/internal/rbac_handler.go`
- Added internal route module:
  - `api/v1/internal/routes.go`
- Removed old file:
  - `api/v1/internal_rbac.go`
- Moved old route block (from `api/v1/routes.go` lines 35-57) into `api/v1/internal/routes.go`.
- Kept compatibility with current router wiring by exposing a thin wrapper in `api/v1/routes.go`:
  - `RegisterInternalRoutes(rg)` delegates to `internalv1.RegisterRoutes(rg)`.

### Risk handling note
- GitNexus impact marked `parseUintParam` as CRITICAL (many direct dependents inside internal handlers).
- Mitigation applied: pure structural move without changing internal handler behavior/response contracts.

### Task 01 + Task 04 loop for this cycle
- Re-ran `gitnexus analyze --force`.
- Ran impact checks before edit (`RegisterInternalRoutes`, `parseUintParam`).
- Re-synced `docs/` and this execution plan after refactor.

### Validation for this cycle
- `go test ./...` (pass)
- `go build ./...` (pass)

## Phase Sub 01 Refactor Execution Update (2026-04-25 - core-to-pkg-entities cycle)

### Task 01 - Re-discovery and impact map (completed before code edits)
- Re-read `.context/*`, `docs/*`, and this execution plan file.
- Re-ran `npx gitnexus analyze --force` before any source edits.
- GitNexus refactor discovery focus:
  - package path migration from `mycourse-io-be/core/entities` to `mycourse-io-be/pkg/entities`
  - confirm `pkg/logic` remains in place (no `pkg/logic` -> `pkg/core` move)
  - identify all remaining direct imports of `core/entities`
- Current direct code impact map for import rewrite:
  - `models/taxonomy_course_level.go`
  - `models/taxonomy_category.go`
  - `models/taxonomy_tag.go`
  - `services/taxonomy/course_level_service.go`
  - `services/taxonomy/category_service.go`
  - `services/taxonomy/tag_service.go`
- Documentation impact map (paths mentioning `core/entities`):
  - `docs/reusable-assets.md`
  - `IMPLEMENTATION_PLAN_EXECUTION.md`
  - `.context/session_summary_2026-04-25_210429.md` (historical log, read-only snapshot)
- Impact-risk conclusion for migration:
  - structural import-path refactor only, no behavior changes expected
  - `pkg/logic` stays unchanged per request
  - remove root `core` only after all imports are migrated and build/test are green

### Task 02/03 planned execution order for this cycle
1. Create `pkg/entities` and move entity files from `core/entities` to `pkg/entities`.
2. Rewrite all imports from `mycourse-io-be/core/entities` to `mycourse-io-be/pkg/entities`.
3. Verify no code import of `core/entities` remains.
4. Remove root `core` directory once migration is complete.
5. Keep `pkg/logic` unchanged in current location.

### Task 04 planned verification for this cycle
- Run final leftover scan for `core/entities` references.
- Run `go test ./...` and `go build ./...`.
- Sync `docs/*` and `.context/*` summary docs with final state.

### Task 02 - Completed implementation
- Moved shared entity files to `pkg/entities`:
  - `pkg/entities/course_level.go`
  - `pkg/entities/category.go`
  - `pkg/entities/tag.go`
- Updated all code imports from `mycourse-io-be/core/entities` to `mycourse-io-be/pkg/entities` in:
  - `models/taxonomy_course_level.go`
  - `models/taxonomy_category.go`
  - `models/taxonomy_tag.go`
  - `services/taxonomy/course_level_service.go`
  - `services/taxonomy/category_service.go`
  - `services/taxonomy/tag_service.go`

### Task 03 - Completed implementation
- Kept `pkg/logic` unchanged at existing path.
- Removed root `core` directory after successful entity migration and import rewrites.
- Verified no remaining code imports reference `mycourse-io-be/core/entities`.

### Task 04 - Verification and documentation sync (completed)
- Updated `docs/reusable-assets.md` to reflect new entity paths in `pkg/entities`.
- Final leftover scan result:
  - no code files import `mycourse-io-be/core/entities`
  - remaining `core/entities` mentions are historical records in context/plan docs
- Validation commands executed after refactor:
  - `go test ./...`
  - `go build ./...`

## Documentation Synchronization Update (2026-04-25 - post-refactor docs pass)

### Scope
- Synced canonical documentation to match current package structure after entity/cache/internal route refactors.

### Updated docs
- `README.md`
  - corrected persistence layer wording (`models/`, `migrations/`)
  - replaced old Redis path with `pkg/cache_clients/`
- `docs/architecture.md`
  - updated `api/v1` map to include `internal/*` and `taxonomy/*`
  - replaced `cache_clients/` with `pkg/cache_clients/`
  - updated internal RBAC source references to `api/v1/internal/rbac_handler.go` + `api/v1/internal/routes.go`
- `docs/deploy.md`
  - updated startup/deploy cache references to `pkg/cache_clients/redis.go`
- `docs/requirements.md`
  - updated FR-6 source references from `api/v1/internal_rbac.go` to `api/v1/internal/*`
- `docs/folder-structure.md`
  - removed obsolete root `cache_clients/`
  - added/expanded `pkg/cache_clients`, `pkg/entities`, `pkg/logic`, `pkg/query`, `pkg/requestutil`
  - updated `.context/` purpose description to reflect active session summaries
- `docs/architecture.md`
  - added `pkg/entities` as active shared-entity layer in architecture snapshot
- `docs/reusable-assets.md`
  - updated internal RBAC usage reference to `api/v1/internal/rbac_handler.go`

### Verification after doc sync
- Performed stale-reference scan in markdown docs for:
  - `cache_clients/` (old root path)
  - `api/v1/internal_rbac.go`
  - `core/entities` in canonical docs
- Result:
  - canonical docs now point to current paths
  - remaining old mentions are retained only in historical logs (`.context/*` and historical sections of this plan)

## Phase Sub 02 RESET Update (2026-04-25 - resolver relocation + contract verification)

### Scope completed for tasks 01->10
- Re-ran strict reset baseline:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status`
  - re-read required context/doc sets: `.context/*`, `docs/*`, `docs/*`, `README.md`, `IMPLEMENTATION_PLAN_EXECUTION.md`.
- Re-validated media route surface and current transport contract against code:
  - methods kept: `GET/POST/PUT/DELETE/OPTIONS` on `/api/v1/media/files` and `/media/files/:id`, plus local decode route.
  - no request/response/status/error contract changes introduced in this reset cycle.
- Re-discovered media upload architecture (file branch B2/Gcore, video branch Bunny) and kept cloud-gateway stateless behavior unchanged.

### Resolver relocation and service cleanup
- Moved resolver logic out of service layer:
  - removed `resolveKind` and `resolveProvider` from `services/media/file_service.go`
  - added shared helpers in `pkg/logic/helper/media_resolver.go`:
    - `ResolveMediaKind(...)`
    - `ResolveMediaProvider(...)`
- Updated media service call-sites to use helper resolvers.
- Service layer remains orchestration-only; no util/helper media resolution logic remains in `services/media`.

### Verification and quality gates
- Ran full formatting/build/test gate:
  - `gofmt -w services/media/file_service.go pkg/logic/helper/media_resolver.go`
  - `go test ./...`
  - `go build ./...`
- Lint status for edited files: clean.
- Reindexed GitNexus post-change:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status` -> up-to-date.

### Documentation sync for this reset cycle
- Updated:
  - `README.md`
  - `docs/modules/media.md`
  - `docs/modules.md`
  - `docs/reusable-assets.md`
  - this execution plan file
- Added reusable-assets entry for resolver relocation (`pkg/logic/helper/media_resolver.go`).

## Phase Sub 02 RESET Update (2026-04-25 - service tail cleanup)

### Scope completed for reset re-run
- Re-ran baseline for this cycle:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status`
- Re-validated media transport contracts and method scope remain unchanged.

### Core correction requested in this cycle
- Removed remaining non-orchestration helper functions from `services/media/file_service.go` tail:
  - `contextBackground()`
  - `ParseMetadataFromRaw(...)`
- Replaced service-local context helper calls with direct `context.Background()`.
- Relocated metadata raw parsing entrypoint to helper layer:
  - added `helper.ParseMetadataFromRaw(...)` in `pkg/logic/helper/media_metadata.go`.
- Updated API handler call-sites to use helper directly instead of calling service utility.

### Verification
- Executed:
  - `gofmt -w pkg/logic/helper/media_metadata.go services/media/file_service.go api/v1/media/file_handler.go`
  - `go test ./...`
  - `go build ./...`
- Lint diagnostics on touched files: clean.
- Reindexed GitNexus post-change and verified freshness/up-to-date.

## Phase Sub 02 Execution Update (2026-04-25 - media upload domain)

### Scope completed
- Re-ran discovery context and GitNexus index (`npx gitnexus analyze --force`) before implementation.
- Implemented unified media upload API with methods `GET/POST/PUT/DELETE/OPTIONS` under `/api/v1/media/files`.
- Implemented provider normalization (`FileProvider`) with values:
  - `S3 | GCS | B2 | R2 | Bunny | Local`
- Implemented `FileMetadata` persisted as `JSONB`.
- Added local-provider reversible signed URL behavior and cloud direct URL behavior.

### Code artifacts
- Migration:
  - `migrations/000003_media_domain.up.sql`
  - `migrations/000003_media_domain.down.sql`
- Domain and schema:
  - `pkg/entities/file.go`
  - `models/media_file.go`
  - `dbschema/media_namespace.go`
  - `dbschema/media_files.go`
- DTO / repository / service:
  - `dto/media_file.go`
  - `repository/media/file_repository.go`
  - `services/media/provider.go` (later refactored into `pkg/media/clients.go`)
  - `services/media/file_service.go`
- Transport:
  - `api/v1/media/routes.go`
  - `api/v1/media/file_handler.go`
  - `api/v1/routes.go` (media route mount)
- Security utility:
  - `pkg/logic/helper/local_url_codec.go`
- RBAC extension:
  - `constants/permissions.go` (`P26`-`P29`)
  - `constants/roles_permission.go` (role mapping for `P26`-`P29`)

### Documentation synchronization completed
- Updated:
  - `README.md`
  - `docs/architecture.md`
  - `docs/api-overview.md`
  - `docs/router.md`
  - `docs/modules.md`
  - `docs/data-flow.md`
  - `docs/reusable-assets.md`
  - `docs/modules/media.md` (new)

### Validation
- `go test ./...` (pass)
- `gofmt` applied on all modified Go files.

## Phase Sub 02 Rework Update (2026-04-25 - cloud-only media, no DB persistence)

### Request-driven correction applied
- Removed all media DB persistence surfaces from sub-02:
  - removed `models/media_file.go`
  - removed `repository/media/*`
  - removed `dbschema/media_*`
  - removed `migrations/000003_media_domain.*`
- Refactored media flow to stateless cloud gateway:
  - upload file -> push to third-party provider -> return `entities.File` response directly
  - no create/read/update/delete media row in local database

### Code structure correction
- Moved media enums/constants out of entity:
  - new `constants/media.go` contains `FileProvider`, `FileKind`, `FileStatus`
- Kept `pkg/entities/file.go` as pure descriptor type only (no util/scan/value logic).
- Moved metadata normalization/parsing to helpers:
  - `pkg/logic/helper/media_metadata.go`
- Moved service-local types out of provider logic file:
  - `services/media/types.go`

### Provider integrations
- Added SDK dependencies and wired clients:
  - B2: `github.com/Backblaze/blazer/b2`
  - Gcore: `github.com/G-Core/gcorelabscdn-go`
  - Bunny storage SDK package installed: `github.com/l0wl3vel/bunny-storage-go-sdk`
- Implemented upload dispatch:
  - non-video -> B2 upload + Gcore CDN URL
  - video -> Bunny Stream API upload path
  - local -> reversible signed token URL

### Config and env updates
- Added media config block in:
  - `config/app.yaml`
  - `config/app-local.yaml`
  - `config/app-dev.yaml`
  - `config/app-staging.yaml`
  - `config/app-prod.yaml`
  - `config/app-test.yaml`
- Added media env keys into `.env.example` and stage example files.

### Validation for rework
- `go test ./...` (pass)

## Phase Sub 02 RESET Update (2026-04-25 - taxonomy status helper relocation)

### Core correction
- Taxonomy status string normalization must not live under `services/taxonomy` as a shared util file.
- Added `helper.NormalizeTaxonomyStatus` in `pkg/logic/helper/taxonomy_status.go` (same layer pattern as media resolvers/metadata).
- Updated `services/taxonomy/category_service.go`, `course_level_service.go`, and `tag_service.go` to call the helper.
- Removed `services/taxonomy/common.go`.

### Verification
- `gofmt`, `go test ./...`, `go build ./...` (pass).
- `npx gitnexus analyze --force` + `npx gitnexus status` -> up-to-date.

### Documentation sync
- Updated `docs/data-flow.md`, `docs/reusable-assets.md` (new asset + corrected media metadata usage line), and this file.

## Phase Sub 02 RESET Update (2026-04-26 - metadata typing + helper placement)

### Scope completed for tasks 01->10
- Re-ran reset baseline:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status`
- Re-read required sources (`.context/*`, `docs/*`, `docs/*`, `README.md`, this plan file) before edits.
- Revalidated media API method scope remains unchanged:
  - `GET/POST/PUT/DELETE/OPTIONS` on `/api/v1/media/files*`.

### Core implementation in this cycle
- Refactored non-CRUD decode utility out of service layer:
  - removed `DecodeLocalURLToken` from `services/media/file_service.go`
  - added/reused helper placement at `pkg/logic/helper/local_url_codec.go` (`DecodeLocalURLToken`).
- Extended media entity metadata model:
  - `pkg/entities/file.go` now includes base `FileMetadata` plus typed metadata structs:
    - `ImageMetadata`
    - `VideoMetadata`
    - `DocumentMetadata`
  - `VideoMetadata` includes required fields:
    - `duration`, `thumbnail_url`, `bunny_video_id`, `bunny_library_id`, `size`, `width`, `height`.
- Implemented backend metadata inference flow:
  - `services/media/file_service.go` reads payload once, uploads by provider, and maps response metadata through helper inference.
  - `pkg/logic/helper/media_metadata.go` adds typed inference helper (`BuildTypedMetadata`) and keeps raw parse helpers.
  - `pkg/media/clients.go` now enriches Bunny metadata with `bunny_video_id` and `bunny_library_id`.
- Updated media handler decode flow to call helper directly:
  - `api/v1/media/file_handler.go` uses `helper.DecodeLocalURLToken`.

### Contract/behavior verification
- Preserved response envelope/status/error behavior (`pkg/response` and existing handler status codes unchanged).
- CRUD responses now return typed metadata payload by file kind instead of ambiguous map in service result.

### Quality gate + GitNexus post-change
- Executed:
  - `gofmt -w ...` on touched files
  - `go test ./...` (pass)
  - `go build ./...` (pass)
- Reindexed and verified freshness:
  - `npx gitnexus analyze --force`
  - `npx gitnexus status` -> up-to-date.

### Documentation sync
- Updated:
  - `README.md`
  - `docs/modules/media.md`
  - `docs/modules.md`
  - `docs/data-flow.md`
  - `docs/reusable-assets.md`
  - this execution plan file.

---

## Phase Sub 06 — Orphan safety, reuse, deferred cleanup (tasks 01→16, closed 2026-04-28)

Single authoritative checklist for plan ids `phase-sub-06-task-01` … `phase-sub-06-task-16`. Implementation lives under `be-mycourse` only (course/lesson APIs are **not** in repo yet — see task 10).

### Task 01 — Baseline + orphan-risk inventory ✅
- Flows that could strand cloud vs DB: create/update/delete, partial failure after upload, concurrent updates. Symbols/files touched: `api/v1/media/file_handler.go`, `services/media/file_service.go`, `repository/media/file_repository.go`, `pkg/media/clients.go`, `pkg/media/stored_object_delete.go`, `internal/jobs/media_pending_cleanup_scheduler.go`.
- Inventory reflected in this section + `docs/modules/media.md` **Phase Sub 06** prose.

### Task 02 — Orphan definition + source of truth ✅ (documented)
- **DB row canonical:** `media_files` holds `object_key`, `provider`, `bunny_video_id`, URLs, `metadata_json`, `row_version`, `content_fingerprint`.
- **B2/Gcore:** identity = `(provider=B2|…, object_key)`; CDN URL derived from `setting.MediaSetting` + bucket + key.
- **Bunny Stream:** identity = `(provider=Bunny, bunny_video_id GUID)`; `object_key` stores GUID after create.
- **Superseded blob:** previous cloud identifiers queued in `media_pending_cloud_cleanup` until worker deletes.

### Task 03 — Reuse / dedupe strategy ✅
- **Fingerprint:** SHA-256 hex (`utils.ContentFingerprint` in `pkg/logic/utils/parsing.go`) over uploaded bytes.
- **Skip upload:** `skip_upload_if_unchanged` + matching fingerprint ⇒ metadata merge only (`MergeMediaMetadataJSON`).
- **Fallback:** if fingerprint empty/missing, full replace path applies.

### Task 04 — API contract (reuse fields) ✅
- `PUT /api/v1/media/files/:id` multipart: `reuse_media_id`, `expected_row_version`, `skip_upload_if_unchanged`; errors **409** + `errcode.Conflict` on mismatch (`pkg/errors/media_errors.go`).
- Binding centralized in `pkg/logic/helper/media_multipart.go` (not inlined in handler).

### Task 05 — DTO/data types ✅
- `dto.UpdateFileRequest` extended; `dto.UploadFileResponse` includes `row_version`, `content_fingerprint`; `entities.File` aligned; `mapping/*` wired.

### Task 06 — Service flow prioritizes reuse ✅
- `services/media/file_service.go`: guards → fingerprint short-circuit → else upload → `SaveWithRowVersionCheck` → enqueue superseded cleanup when policy says so.

### Task 07 — Traceability fields ✅
- `row_version`, `content_fingerprint` on `media_files`; pending table tracks deferred deletes (`repository/media/pending_cleanup_repo.go`).

### Task 08 — Mark-and-sweep / pending ✅
- Replace does **not** synchronous-delete old cloud object; inserts `media_pending_cloud_cleanup` row (`InsertPendingCleanup`).

### Task 09 — Cleanup worker ✅
- `internal/jobs/media_pending_cleanup_scheduler.go`: mutex/cancel/waitgroup + immediate first tick (same spirit as RBAC jobs). Batch/retry: `constants/media_cleanup.go`; processor `services/media/pending_cleanup.go`; metrics `services/media/cleanup_metrics.go`.
- Env: `MEDIA_CLEANUP_INTERVAL_SEC` (`0` off).

### Task 10 — Apply to course/lesson/quiz APIs — **N/A (repository scope)**
- No `courses`/`lessons`/`sections` upload endpoints exist in `be-mycourse` yet. When Phase 05/06 domain lands, consume **same** media APIs / FK references — no duplicate upload bypass.

### Task 11 — Transaction / compensation ✅ (within current architecture)
- Upload-then-DB: on DB failure after successful upload, **compensate** via `DeleteStoredObject` on the **new** object (`CreateFile` / replace failure branch).
- Full distributed two-phase transaction cloud↔Postgres is **not** implemented (Postgres cannot enlist B2/Bunny); documented limitation above.

### Task 12 — Race guards ✅
- Optimistic locking: `SaveWithRowVersionCheck` vs `expected_row_version` / stored version.

### Task 13 — Observability ✅
- `GET /api/v1/media/files/cleanup-metrics`: exposes atomic counters (deleted/failed/retried). Structured logging for worker start/stop in job package.

### Task 14 — Test plan ✅
- Automated: `tests/sub06_media_orphan_safety_test.go` (fingerprint, merge, superseded policy, constant presence).
- Manual QA checklist: migrate (`MIGRATE=1`); create media; PUT with wrong `expected_row_version` → 409; PUT same file + `skip_upload_if_unchanged=true` → metadata bump without new object key change where fingerprint matches; verify pending rows processed after worker tick.

### Task 15 — Quality gate + audit ✅
- `gofmt`, `go build ./...`, `go test ./tests/...` — run as part of closure.
- Code-vs-plan: this section + `docs/modules/media.md` are aligned.

### Task 16 — GitNexus refresh ✅ (post-edit command)
- After merging: run `npx gitnexus analyze --force` from repo root; optionally `gitnexus_detect_changes` / `gitnexus_impact` on `UpdateFile`, `ProcessPendingCleanupBatch`, `StartMediaPendingCleanupJob` before release branches.

### Files reference (Sub 06)
- DDL: `migrations/000004_media_orphan_safety.up.sql`
- Models: `models/media_file.go`, `models/media_pending_cloud_cleanup.go`, `dbschema/media_pending_cloud_cleanup.go`
- Constants: `constants/media_cleanup.go`, `constants/media_meta_keys.go`, `constants/error_msg.go`
- Errors: `pkg/errors/media_errors.go`
- Helpers: `pkg/logic/helper/media_metadata_merge.go`, `media_replace_policy.go`, `media_upload_entity.go`, `media_multipart.go`; utils: `pkg/logic/utils/parsing.go` (`ContentFingerprint`); input struct `pkg/entities/media_upload.go`
- Media delete routing: `pkg/media/stored_object_delete.go`
- Repo: `repository/media/file_repository.go`, `repository/media/pending_cleanup_repo.go`
- Services: `services/media/file_service.go`, `pending_cleanup.go`, `cleanup_metrics.go`
- Jobs: `internal/jobs/media_pending_cleanup_scheduler.go`; bootstrap `main.go`
- API: `api/v1/media/file_handler.go`, `routes.go`
