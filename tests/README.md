# `tests/` — module-level tests

Place **module / integration** Go test packages here (suites that import `mycourse-io-be`, black-box API tests, shared fixtures, cross-feature harnesses). Run from repo root with `go test ./tests/...` or `go test ./...`.

Narrow **unit tests** may still live as colocated `*_test.go` next to the code under test.

Authoritative docs: `README.md` (**Testing**), `.full-project/patterns.md`, `docs/requirements.md` (NFR-1.6), `docs/architecture.md` (directory map).
