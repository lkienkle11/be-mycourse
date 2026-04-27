# Sub05 correction pass (2026-04-27)

- User-requested correction applied:
  - Removed `provider` from public response contract (`dto.UploadFileResponse` and mapper response output).
  - Moved DB<->entity mapping logic out of `services/media/file_service.go`.
  - New mapping location: `pkg/logic/mapping/media_model_mapping.go`.
- Updated docs/snapshots:
  - `docs/modules/media.md`
  - `.full-project/modules.md`
  - `IMPLEMENTATION_PLAN_EXECUTION.md`
- Validation:
  - `gofmt -w ...`
  - `go test ./...` passed.
