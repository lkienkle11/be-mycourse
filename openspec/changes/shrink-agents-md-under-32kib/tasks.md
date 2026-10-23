## 0. Classification, session, branch

- [x] 0.1 Classify as in-scope (edits agent rules, skills, docs; no `internal/`, `cmd/`, `migrations/`, Makefile or CI file). Verify the planned file set has no such path.
- [x] 0.2 Read `AI_SESSION_ID_FILE` (outside every project), confirm one non-empty ID; never generate a new one.
- [x] 0.3 Read the complete `.context/` folder, resolve exactly one `.context/session-<SESSION_ID>.md`.
- [x] 0.4 Confirm branch `docs/shrink-agents-md-under-32kib` (created from `feat/add-user-slug-management`, six local commits kept). Do not push.
- [x] 0.5 Load Memory Bank for project `be-mycourse` (30 s limit, no retry).

## 1. Skill inventory (BE)

- [x] 1.1 List `name` and `description` of every `SKILL.md` in project folders (`.ai`, `.agents`, `.agent`, `.claude`, `.codex`, `.cursor`, `.gemini`, `.antigravity`) and agent folders; de-duplicate by name. Record counts and whether functional and non-functional backend skills exist; note gaps in `docs/ALL-SPRINTS.md` To do.

## 2. Skills

- [x] 2.1 Copy `fe-mycourse/.ai/skills/session-context/SKILL.md` and `branch-rename/SKILL.md` verbatim; verify with `diff`.
- [x] 2.2 Copy `docs-sync/SKILL.md` and replace only the final section by `Repository specifics (be-mycourse)` (module entries of `FEATURES.md`, `ARCHITECTURE.md`, `DATA-MODEL.md`, `SHARED.md`, `ALL-SPRINTS.md`, Swagger step). Verify `diff` differs only in that section.

## 3. AGENTS.md

- [x] 3.1 Save the BE GitNexus block (`<!-- gitnexus:start -->` to `<!-- gitnexus:end -->`) and its sha256 to the scratchpad.
- [x] 3.2 Build the new file: GitNexus block + shared sections copied from `fe-mycourse/AGENTS.md` + `Repository specifics (be-mycourse)` with `AI Agent - dev test account` (placeholders only) and Swagger rule in their own marker pairs.
- [x] 3.3 Verify: `wc -c` < 32768; every `<!-- begin:x:start -->` has a unique `<!-- end:x:end -->`; GitNexus block hash unchanged; shared part identical to FE outside the two differing parts; key phrases present (`AI_SESSION_ID_FILE`, `session-<SESSION_ID>`, `30-second`, `lastResponse.md`, `make check-all`, `committable Git surface`, `semantic ownership`, `functional`, `non-functional`, `docs-sync`, `[redacted]`, `generate-apidog-postman.rb`); no mention of Husky as an active hook.
- [x] 3.4 Walk the obligation map in `design.md` line by line against the old file (`git show HEAD:AGENTS.md`).

## 4. Docs and references

- [x] 4.1 Two-step `git mv docs/architecture.md docs/ARCHITECTURE.md`; grep and fix every `architecture.md` reference.
- [x] 4.2 Keep `docs/ALL-SPRINTS.md` consistent with `docs-sync` (In progress holds this change).
- [x] 4.3 Update `README.md` documentation index and `.cursor/rules/rule-before-plan-code.mdc` references (docs set, plan target `docs/ALL-SPRINTS.md`, `docs/SHARED.md`); delete `.cursor/skills/session-context-handoff/`.
- [x] 4.4 Verify no dead links to removed or renamed files (`grep -rn`).

## 5. Quality gates and sync

- [x] 5.1 `make check-all`; fix by cause; never touch tooling.
- [x] 5.2 `npx gitnexus analyze --embeddings`; re-check `wc -c AGENTS.md`.
- [x] 5.3 Update `.context/session-<SESSION_ID>.md` (skills picked, learnings) and Memory Bank (`activeContext`, `progress`, `lastResponse`).
- [x] 5.4 Review the diff; no commit and no push unless the user asks.

## Extension (done)

- [x] Extension (2026-10-23, user-approved): the same Degraded Environment Rule, Implementation Rule, two skills (`pre-implementation-checkpoint`, `progress-tracking`, identical to FE), triggers, and BE implementation notes were added. Shared text stays identical to `fe-mycourse/AGENTS.md`. Not added by decision: a "never commit or push unless asked" rule (user still deciding).
