## 0. Lifecycle gates (AGENTS.md steps 1-4; no project file is edited in this group)

- [x] 0.1 Classify the request as in scope (changes the project) and say so; verify the classification is recorded in `.context/session-<SESSION_ID>.md`
- [x] 0.2 Session: read the Session ID from `AI_SESSION_ID_FILE` (never regenerate), read the whole `.context/` folder, reuse the single `.context/session-<SESSION_ID>.md`; verify exactly one file exists for the ID and the ID equals the file content
- [x] 0.3 Memory bank pre-response load: run `memory-bank-cli memory_bank_read_all --project be-mycourse` (30 s limit, no retry; fallback MCP `memory-bank`, then `/private/tmp/ai-agent/memory/be-mycourse/`); verify the result is noted in the session file, and say plainly if all three failed
- [x] 0.4 Skills: re-list frontmatter of `.ai/skills`, `.agents/skills`, `.claude/skills`, user and plugin skills; confirm or adjust the picks (workflow: `session-context`, `pre-implementation-checkpoint`, `progress-tracking`, `docs-sync`, `openspec-apply-change`; non-functional: `gitnexus-impact-analysis`; functional: none fits, `postgresql-optimization` is a tuning bundle); announce `Using <skill> (<category>) to <purpose>` for each and record name, category and path in the session file
- [x] 0.5 Plan gate: present `proposal.md`, `design.md`, `specs/`, `tasks.md` and wait for the user's explicit approval; verify the approval is quoted in the session file; make no edit outside `.context/`, `docs/`, `openspec/` before it
- [x] 0.6 Confirm the working tree state with `git status --short` and `git branch --show-current`; verify only this change's `openspec/` folder is untracked and nothing else is modified

## 1. Pre-implementation checkpoint (AGENTS.md step 5, before the first source edit)

- [x] 1.1 Read `.ai/skills/pre-implementation-checkpoint/SKILL.md` and ask ONE `AskUserQuestion` call with the questions that apply: (a) new branch name now (the current branch `docs/restructure-docs-and-shrink-agents-md` does not name this task; propose `fix/refresh-session-rotation-race` per `.ai/skills/branch-rename/SKILL.md`, then confirm the name), (b) tests: the repo has `*_test.go` files, so do not ask; write tests as the Implementation Rule says, (c) subagents: ask only if the task is judged heavy (it is not; default cap 4); verify the answers are recorded in the session file and will be repeated in the final response
- [x] 1.2 Apply the branch answer: the current branch is not protected (`main`, `master`, `dev`, `release*`), so rename with `git branch -m -- '<name>'`; never push and never change the upstream; verify with `git branch --show-current`

## 2. GitNexus impact analysis before any symbol edit (CLAUDE.md "Always Do")

- [x] 2.1 Check index freshness (`gitnexus://repo/be-mycourse/context`); if stale run `npx gitnexus analyze` first; verify the index commit equals `git rev-parse --short HEAD`
- [x] 2.2 Run `gitnexus_impact({direction: "upstream"})` for each symbol to be edited or deleted: `SaveSession`, `SaveSessions`, `AddSession`, `RemoveSession`, `SaveWithDB`, `rotateSession`, `sessionRepo`, `RefreshSessionRepository`; where it answers UNKNOWN or 0 callers use `grep -rn` and state that GitNexus did not answer; verify callers match `design.md` Context (`SaveSession` <- `rotateSession`; `SaveSessions` <- `RemoveSession`; `AddSession` <- `issueTokenPair`; `RemoveSession` <- `Logout`; `SaveWithDB` <- `Save`, `ConfirmEmailWithLearnerRole`, `LinkIdentityAndUpdateUser`)
- [x] 2.3 Report the blast radius (direct callers, affected flows, risk level) to the user; stop and warn if any result is HIGH or CRITICAL; verify the report is in the session file

## 3. Tests first (they must fail or not compile before section 4)

- [x] 3.1 Create `internal/auth/application/service_session_test.go`: one `sessionRepo` fake (embeds `domain.RefreshSessionRepository`; implements `AddSession`; `RotateSession` is an in-memory compare-and-swap that records `expectedUUID` and returns `domain.ErrInvalidSession` on mismatch) and a `FindByID` user fake following `slugTestUserRepo`; helper sets and restores `setting.AppSetting.JWTSecret`, builds the refresh token with `token.GenerateRefresh`; no `t.Parallel()`; verify `go vet ./internal/auth/application/` reports only the missing `RotateSession` before section 4
- [x] 3.2 Add `TestRefreshSessionRotatesOnce`, `TestRefreshSessionStaleTokenAfterRotationFails`, `TestRefreshSessionPassesVerifiedUUIDToRotate`, `TestRefreshSessionPropagatesLostRace`, `TestRefreshSessionOtherErrorsUnchanged` exactly as listed in `design.md` Test plan; verify they fail or do not compile now
- [x] 3.3 Create `internal/auth/infra/user_repo_session_test.go` with `TestSessionSQLShape` (string assertions on the four consts) and `TestSaveWithDBOmitsSessionAndSlug` (dry-run handle as in `internal/shared/gormx/hard_delete_test.go`, `Update` callback capturing `Statement.SQL`); verify they fail or do not compile now
- [x] 3.4 Create `internal/auth/delivery/handler_refresh_test.go` with `TestRefreshTokenLostRaceIs401InvalidSession` (stub `AuthUseCase` embedded as in `handler_patch_me_test.go`; cases: `ErrInvalidSession` -> 401 `InvalidSession`, `ErrRefreshTokenExpired` -> 401 `RefreshTokenExpired`, `ErrUserDisabled` -> 403, generic error -> 500, missing headers -> 400); verify it passes against the current handler (it pins the contract the FE depends on)

## 4. Implementation (module `auth` only; match surrounding style; no new helper)

- [x] 4.1 `internal/auth/infra/user_repo.go`: add the four SQL consts (`sqlRotateSession`, `sqlSelectSessionsForUpdate`, `sqlStoreSessions`, `sqlRemoveSession`) and `RotateSession` exactly as in `design.md`; verify `go build ./internal/auth/...` and the SQL-shape test for `sqlRotateSession` passes
- [x] 4.2 `internal/auth/application/service.go` lines 79-85: swap `SaveSession` for `RotateSession` in the `sessionRepo` interface and fix its comment; `internal/auth/application/service_session.go` line 100: call `RotateSession(ctx, user.ID, sessionStr, entry.RefreshTokenUUID, updatedEntry)`; verify the section 3.2 application tests pass
- [x] 4.3 `internal/auth/infra/user_repo.go`: delete `SaveSession` (lines 200-207); verify `grep -rn "SaveSession\b" internal cmd` returns nothing
- [x] 4.4 `AddSession`: use `sqlSelectSessionsForUpdate` and `sqlStoreSessions`; eviction and `MaxActiveSessions` logic untouched; verify `git diff` of the function shows only those two query changes
- [x] 4.5 `RemoveSession`: replace with the single `Exec(sqlRemoveSession, ...)`; delete `SaveSessions` from `user_repo.go` and from `internal/auth/domain/repository.go`, and reword the `RemoveSession` doc comment there; verify `grep -rn "SaveSessions" internal cmd` returns nothing and the `sqlRemoveSession` test passes
- [x] 4.6 `SaveWithDB`: `Omit("slug", "refresh_token_session")` and the new comment; verify `TestSaveWithDBOmitsSessionAndSlug` and `internal/auth/infra/user_slug_test.go` pass
- [x] 4.7 Run `go fmt ./...`, `go build ./...`, `go test ./internal/auth/...`; verify all pass and `gofmt -l internal/auth` prints nothing
- [x] 4.8 Self-review for duplicate or dead code: no leftover `SaveSession`/`SaveSessions` in code, comments or docs; no new helper that duplicates `gormx`; `fmt` import still used; `git diff --stat` touches only files named in `proposal.md` Impact; verify by reading the diff once

## 5. Swagger and documentation (AGENTS.md "Swagger rule" and `docs-sync`; English; replace, never append)

- [x] 5.1 `docs/api_swagger.yaml`: add `"400"`, `"401"`, `"403"` responses to `/api/v1/auth/refresh` as worded in `design.md` item 10; verify YAML parses (`ruby -ryaml -e 'YAML.load_file("docs/api_swagger.yaml", aliases: true)'`)
- [x] 5.2 Run `ruby scripts/generate-apidog-postman.rb`; verify `docs/api-dog-import.json` changed only through the script (`git diff --stat docs/`) and was not hand-edited
- [x] 5.3 `docs/DATA-MODEL.md`: edit line 280 and the lines 296-301 session-rules table as in `design.md` items 1-2; verify `grep -n "SaveSession" docs/DATA-MODEL.md` returns nothing
- [x] 5.4 `docs/FEATURES.md` `## auth` (`grep -n "^## " docs/FEATURES.md`): edit line 22, line 31, refresh diagram line 114 (two lines), logout diagram line 134 as in `design.md` items 3-6; keep the `API:` line accurate; verify mermaid blocks still open/close and `grep -n "SaveRefreshSession" docs/FEATURES.md` returns nothing
- [x] 5.5 `docs/ARCHITECTURE.md` line 307 as in item 7; verify the table row still has 5 columns
- [x] 5.6 `docs/SHARED.md` lines 129-137 as in item 8; verify every backticked path exists (`test -e`) and that the remaining dependency paths are checked, fixing only wrong ones
- [x] 5.7 `docs/ALL-SPRINTS.md`: add the In progress row as in item 9; verify it matches the table format
- [x] 5.8 docs-sync checks (`.ai/skills/docs-sync/SKILL.md` step 5): backticked paths exist, links resolve, no duplicated heading, `grep -rn "SaveSession" docs` returns nothing, docs files keep their language; verify the results are in the session file

## 6. Final phase (AGENTS.md "Final Phase")

- [x] 6.1 Force-sync GitNexus: `npx gitnexus analyze` (add `--embeddings` if `.gitnexus/meta.json` `stats.embeddings` is greater than 0), outside the sandbox if it fails with permissions; verify it exits 0 and note that it may rewrite only the stats sentence in `AGENTS.md` / `CLAUDE.md`
- [x] 6.2 Run `gitnexus_detect_changes({scope: "all"})`; verify changed symbols are exactly those in `proposal.md` Impact and affected flows are login, refresh, logout, confirm-email and OAuth link only
- [x] 6.3 Run the quality gate `make check-all` (`go fmt`, `test-all` = `go test ./...`, `CGO_ENABLED=1 go test ./...`, `go vet`, `golangci-lint cache clean && golangci-lint run`, `check-layout`, `check-architecture`, `check-dupl`; then `build-nocgo` and `build`), outside the sandbox; verify exit 0. Handle every non-zero exit by cause: caused by this change -> fix the code and rerun all gates; pre-existing -> report with evidence; environment -> report. Never edit lint, test or build configuration, never suppress, never `--no-verify`
- [x] 6.4 Run `openspec validate fix-refresh-session-rotation-race --strict --no-interactive`; verify it passes
- [x] 6.5 Optional live smoke test on `http://localhost:8080` (`go run .`) with the dev account from the user's local environment (never written to a tracked file): log in, then send two simultaneous `POST /api/v1/auth/refresh` with identical `X-Refresh-Token` and `X-Session-Id`; verify one 200 and one 401 with code `InvalidSession`, then a refresh with the 200 response's token succeeds. If no credentials are available, leave this task unticked and report it as not run
- [x] 6.6 Final code review checklist: follows existing layering, reuses existing resources, no duplicate logic, no unrelated files, no secrets, ready for CI/CD; verify each item against `git diff`

## 7. Documentation, context sync and report (AGENTS.md "Documentation and Context Sync", "Final Response Requirement")

- [x] 7.1 Re-read every doc entry changed in section 5 and the session file; verify docs equal the final code (symbol names `RotateSession`, `RemoveSession`, `AddSession`, `SaveWithDB` match the code) and no old and new descriptions coexist
- [x] 7.2 Update `.context/session-<SESSION_ID>.md` with the final state, "What I Learned Today" and "What I Should Learn or Do Next Time" based only on what happened; verify exactly one session file exists for the ID
- [x] 7.3 Tick this file's checkboxes only after each task's verification ran (`progress-tracking`); tasks that cannot finish before the final answer (7.4, and 6.5 without credentials) stay unticked with the reason stated
- [x] 7.4 Persist memory (`memory-bank-cli memory_bank_update` / `memory_bank_write`: `activeContext.md`, `progress.md`, `lastResponse.md` verbatim with secrets redacted) as the last operation before the final answer; never claim persistence unless the write succeeded
- [x] 7.5 Final response lists: what was implemented, files changed, skills picked by category, existing resources reused, duplicates avoided or merged, docs / GitNexus / session files updated, gates run with results, checkpoint outcome (branch, tests, subagents), untested paths (true concurrency, `AddSession` lock), remaining risks and follow-ups (de-duplicate the `media` / `course` copies of `gormx.OptimisticUpdate`; revisit a grace window only if logs show lost-response retries)

## 8. Commit and archive (only when the user asks; never on my own)

- [x] 8.1 Before any commit: `git branch --show-current` must be a valid `<type>/<english-kebab-summary>` and not `main`, `master`, `dev`, `release*`; otherwise propose one name through `AskUserQuestion` and commit only after approval
- [x] 8.2 Run `make check-all` again immediately before the commit on the exact tree being committed; verify exit 0
- [x] 8.3 Commit title is English, one line, `<type>: <title>` (Conventional Commits), no scope, no body, no footer from the repo rules, 15-72 characters, meaningful (for example `fix: make refresh token rotation atomic per session`); append the attribution line required by the session setup only if the user's rules allow a footer, otherwise follow the repo rule; verify with `git log -1 --format=%s | wc -c`
- [x] 8.4 After the commit the GitNexus index is stale: run `npx gitnexus analyze` (with `--embeddings` when it had them)
- [ ] 8.5 Do not archive on my own. After merge and deploy, when the user asks, archive the change and move its row in `docs/ALL-SPRINTS.md` from In progress to Done with the change's creation date (2026-10-28) and the archive folder name
