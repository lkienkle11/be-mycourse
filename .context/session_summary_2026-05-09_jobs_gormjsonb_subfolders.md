# Session summary — `internal/jobs` + `pkg/gormjsonb/auth` layout (2026-05-09)

## Code (errors-rule-2 Rules 7 & 9)

- **`internal/jobs/`** — only feature subfolders: **`media/`** (`jobmedia`), **`rbac/`** (`jobrbac`), **`system/`** (`jobsystem`). No `.go` files at `internal/jobs/` root.
- **`pkg/gormjsonb/auth/`** — `RefreshTokenSessionMap` JSONB `Valuer`/`Scanner`; **`models/user.go`** imports as **`gormjsonbauth`**.

## Docs

- Synced **`docs/*`**, **`README.md`**, **`IMPLEMENTATION_PLAN_EXECUTION.md`**, **`.context/session_summary_2026-05-09_errors_rule2_completion.md`** for paths and package names. **Did not** edit `temporary-docs/loi-quang-ra/errors-rule-2.md`.

## Quality gates (run after edits)

- `golangci-lint cache clean`, `golangci-lint run`, `make check-architecture`, `gofmt -w .`, `go vet ./...`, `go test ./...`, `make build-nocgo`.
