# `tests/` — all test code

Place **module / integration** Go test packages here (suites that import `mycourse-io-be`, black-box API tests, shared fixtures, cross-feature harnesses). Run from repo root with `go test ./tests/...` or `go test ./...`.

All tests (unit/integration/black-box/shared fixtures) must be placed under this `tests/` tree; do not place new tests next to production packages.

Authoritative docs: `README.md` (**Testing**), `docs/patterns.md`, `docs/requirements.md` (NFR-1.6), `docs/architecture.md` (directory map).

Media / Bunny: `tests/sub04_media_pipeline_test.go` covers mapping + embed URL helpers aligned with **`dto.UploadFileResponse`** parity fields documented in **`docs/modules/media.md`** (Sub 09).
