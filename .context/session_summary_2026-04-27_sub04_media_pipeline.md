# Session handoff — Phase Sub 04 (media pipeline)

## Done (code)
- B2: `effectiveB2Bucket()` (YAML `media.b2_bucket` / `MediaSetting` then env bucket from `NewCloudClientsFromEnv`); public `URL` + `BuildPublicURL` use `JoinURLPathSegments(cdn, bucket, key)` when bucket set.
- Keys: `pkg/logic/helper/media_upload_keys.go` — `ResolveMediaUploadObjectKey`, `BuildB2ObjectKey` (8 digits + sanitized name); `file_service` uses helper; Bunny `ObjectKey` = API GUID from client.
- Bunny: POST create → read body → PUT upload; `ProviderError` + errcodes **9010–9014**; handler maps with **502** for **9012–9014**.
- `VideoMetadata.video_provider` + `BuildTypedMetadata` field wiring.
- Tests: `tests/sub04_media_pipeline_test.go` (no tests under `pkg/logic/utils` for this work).

## Docs / env
- `IMPLEMENTATION_PLAN_EXECUTION.md` — Sub 04 tasks 01–10 (Đầu/Giữa/Cuối).
- `docs/modules/media.md`, `.full-project/reusable-assets.md`, five `.env*.example` comments for `MEDIA_B2_BUCKET`.

## Not in scope
- Poll/webhook upload pipeline vs `temporary-docs/chuc-nang-upload` — still absent from workspace baseline.

## Verify
- From repo root: `go build ./...` and `go test ./...` (includes `tests/`).
