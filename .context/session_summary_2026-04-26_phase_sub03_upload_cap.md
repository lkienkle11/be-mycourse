# Session handoff — Phase Sub 03 (upload cap 2 GiB)

**Date:** 2026-04-26  
**Scope:** `phase-sub-03-task-01` … `phase-sub-03-task-10` (be-mycourse)

## What shipped
- **Constants:** **`constants/error_msg.go`** — canonical file for shared **error/sentinel strings** and related limits (see file header). Media: `MaxMediaUploadFileBytes` (2 GiB) **and** `MsgFileTooLargeUpload` (**single** string for both `pkg/errcode` `FileTooLarge` default JSON `message` and `pkg/media.ErrFileExceedsMaxUploadSize`). Former `constants/upload_limits.go` merged here.
- **Gin:** `api/router.go` → `MaxMultipartMemory = 64 << 20` (64 MiB).
- **Sentinel:** `pkg/media/upload_errors.go` → `ErrFileExceedsMaxUploadSize = errors.New(constants.MsgFileTooLargeUpload)` — **not** in `services/media/` (removed `services/media/errors.go`).
- **Handler:** `api/v1/media/file_handler.go` — early reject when `FileHeader.Size` known and over cap; `errors.Is(err, pkgmedia.ErrFileExceedsMaxUploadSize)` → HTTP **413** + `errcode.FileTooLarge` (alias `pkgmedia` because handler package is also named `media`). Missing `file` unchanged (**400** + `BadRequest`).
- **Service:** `services/media/file_service.go` — header check, `io.LimitReader(max+1)` + `ReadAll`, returns `pkgmedia.ErrFileExceedsMaxUploadSize`.
- **Errcode:** `pkg/errcode` → `FileTooLarge = 2003`; `messages.go` uses **`constants.MsgFileTooLargeUpload`** (same literal as sentinel — never duplicate in `messages.go`).
- **FE mirror:** `fe-mycourse/src/types/api.ts` → `ApiErrorCode.FileTooLarge` aligned with backend.

## Docs / snapshots touched
- `IMPLEMENTATION_PLAN_EXECUTION.md` (Phase Sub 03 section), `README.md`, `docs/*` (media, deploy, requirements, return_types, curl_api, architecture), `.full-project/*` (data-flow, router, api-overview, modules, reusable-assets).

## Repo scan
- Only **media** handlers use `FormFile`; no other multipart upload entry points.

## Next
- Plan sequencing: proceed to **`phase-02-start`** when ready.
- After deploy: confirm nginx **HTTPS** `server` for API still has `client_max_body_size` ≥ **2G**.

## Quality gate (local)
- `gofmt`, `go test ./...`, `go vet ./...` — pass.
