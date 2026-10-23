## Why

`AGENTS.md` of `be-mycourse` is 60,339 bytes. Codex loads only the first 32,768 bytes (32 KiB) of project instructions, so about 27,600 bytes of rules are never seen by Codex. The earlier `restructure-docs-and-agents` change (53/53) made it worse: it added `For be-mycourse` / `For fe-mycourse` branches and "as currently structured" placeholders, and kept rules that are false in this repository.

`fe-mycourse` solved the same problem in its change `restructure-docs-and-agents` (`AGENTS.md` 56,831 to 23,283 bytes, three skills, a skill-selection rule). That result and the hand-off document `/Users/kienlt/Documents/projects/mycourse-full/temporary-docs/claude-doanchat/Handoff - fe-mycourse to be-mycourse (restructure-docs-and-agents).md` define what BE must copy. The two repositories must share the same text everywhere except the GitNexus block and `Repository specifics`.

Defects found in the current BE `AGENTS.md` (hand-off section 5):

1. Commit rules say a Husky `pre-commit` hook runs `make check-all` and a `commit-msg` hook runs commitlint. BE has no `.husky`, no `core.hooksPath`, no installed hook and no commitlint.
2. "Never store secrets" conflicts with "persist the final answer verbatim in `lastResponse.md`".
3. Final Phase requires both `make test-all` and `make check-all`; `check-all` already includes `test-all`.
4. Placeholders "as currently structured" for `fe-mycourse` waste bytes.
5. "Thoroughly read all documentation and all skills" is unbounded.
6. `docs/architecture.md` is tracked in lowercase while every document links to `ARCHITECTURE.md` (works on macOS, breaks on Linux).

## What Changes

- Rewrite `AGENTS.md` below 32,768 bytes (target about 27 KB): shared sections identical to `fe-mycourse/AGENTS.md` (SKILLS rule with three mandatory triggers, Global Memory Bank Rules with `[redacted]` persistence, Implementation Workflow Rules), GitNexus block unchanged byte for byte, and a BE `Repository specifics` section holding the docs map, `make check-all` gate, routing, size limit, sync targets, the `AI Agent - dev test account` block (placeholders only) and the Swagger rule.
- Add three skills under `.ai/skills/`: `session-context` and `branch-rename` (copied from `fe-mycourse`, verbatim), and `docs-sync` (copied; only the final `Repository specifics` section replaced by the BE mapping: `FEATURES.md#<module>`, `ARCHITECTURE.md`, `DATA-MODEL.md`, `SHARED.md`, `ALL-SPRINTS.md`, Swagger generator).
- Commit rule without Husky: run `make check-all` after the task and again immediately before every commit; title rules are a convention (no commitlint in BE).
- Rename tracked `docs/architecture.md` to `docs/ARCHITECTURE.md` with a two-step `git mv`.
- Add `docs/ALL-SPRINTS.md` (To do / In progress / Done), seeded from the deleted Cursor plan `.vscode/e-learning_crud_backend_8f5ca286_reset_all_pending.plan.md` and the OpenSpec change list (file already written in the working tree).
- Update stale references in `.cursor/rules/rule-before-plan-code.mdc` and `README.md`; remove `.cursor/skills/session-context-handoff/` (its `.context/session_summary_<timestamp>.md` contradicts the session-file rule).
- Inventory the skills visible to BE (project and agent levels) and record whether functional and non-functional backend skills exist.
- No change under `internal/`, `cmd/`, `migrations/`, `Makefile`, CI workflows, lint or test configuration, `CLAUDE.md`, or OpenSpec skills and commands.

## Capabilities

### New Capabilities

None. `skip_specs: true`: this change edits agent instructions and documentation only.

### Modified Capabilities

None.

## Impact

- Files: `AGENTS.md`, `.ai/skills/{session-context,branch-rename,docs-sync}/SKILL.md`, `docs/ARCHITECTURE.md` (rename), `docs/ALL-SPRINTS.md`, `README.md`, `.cursor/rules/rule-before-plan-code.mdc`, `.cursor/skills/session-context-handoff/` (delete), `CLAUDE.md` (GitNexus stats line only, rewritten by `npx gitnexus analyze`).
- Risk: a rule lost while condensing. Mitigation: obligation map in `design.md` checked line by line, key-phrase grep list, GitNexus block hash comparison.
- Branch: `docs/shrink-agents-md-under-32kib`, created from `feat/add-user-slug-management` so the six unpushed local commits are kept (that branch's upstream is `[gone]`). Nothing is pushed.
