## Context

See `proposal.md`. Source of truth for the shared text: `fe-mycourse/AGENTS.md` (from `## SKILLS` through `# Implementation Workflow Rules`) and `fe-mycourse/.ai/skills/{session-context,branch-rename,docs-sync}/SKILL.md`. Hand-off document: `/Users/kienlt/Documents/projects/mycourse-full/temporary-docs/claude-doanchat/Handoff - fe-mycourse to be-mycourse (restructure-docs-and-agents).md`.

Measured BE state (2026-10-22): `AGENTS.md` 60,339 bytes (GitNexus block 5,495 bytes, markers `<!-- gitnexus:start -->` / `<!-- gitnexus:end -->`); `CLAUDE.md` 5,508 bytes; `docs/` = `architecture.md` (tracked lowercase), `FEATURES.md`, `DATA-MODEL.md`, `SHARED.md`, `deploy.md`, `docker.md`, `security-public-seo-notes.md`, `course-collaboration-handoff-2026-06-04.md`, `observability/`, generated `api_swagger.yaml` and `api-dog-import.json`; `Makefile` has `check-all` and `test-all` (`check-all` includes `test-all`); no Git hook, no commitlint; project skills `.ai/skills/{cache-strategy-implementer,logic-n-1-optimize,postgresql-optimization}`; OpenSpec has 10 unarchived changes (8 complete, 2 in progress).

## Goals / Non-Goals

**Goals:** `AGENTS.md` < 32,768 bytes with every obligation kept, moved verbatim to a skill, or deliberately removed and listed; shared text identical to FE; correct Commit rule; docs map and links valid.

**Non-Goals:** no change to code, Makefile, CI, lint or test tooling (Tooling rule protection); no Git hook installation; no archive of BE OpenSpec changes; no change to `CLAUDE.md` beyond the GitNexus stats; no re-opening of BE `restructure-docs-and-agents`; no trimming of `.context/` reading.

## Decisions

1. **Copy, do not re-author.** Shared sections are copied from `fe-mycourse/AGENTS.md`; only the GitNexus block (BE's own, `be-mycourse` URIs and stats) and `Repository specifics` differ. Verified by diffing the two files outside those two parts.
2. **GitNexus block kept byte for byte.** The BE block is extracted before editing, its sha256 recorded, and compared after the rewrite (`npx gitnexus analyze` later changes only the stats line).
3. **`Repository specifics (be-mycourse)`** contains: documentation map (`ARCHITECTURE.md`, `FEATURES.md`, `DATA-MODEL.md`, `SHARED.md`, `ALL-SPRINTS.md`; satellites `deploy.md`, `docker.md`, `security-public-seo-notes.md`, `observability/`, dated handoff note, generated `api_swagger.yaml` / `api-dog-import.json`); routing (`grep -n "^## " docs/FEATURES.md`; check `SHARED.md` before adding anything); quality gate (`make check-all`, which already contains `test-all`; no hook and no commitlint, title rules are convention); size limit; sync targets (table to `DATA-MODEL.md` including the permission catalog appendix); the `AI Agent - dev test account` block (variable names and placeholders only); the Swagger rule (`ruby generate-apidog-postman.rb`, then fix the `API:` line of the matching `FEATURES.md` entry), each inside its own marker pair.
4. **Skills.** `session-context` and `branch-rename` are copied verbatim. `docs-sync` is copied and only its final `Repository specifics` section replaced (BE mapping plus the Swagger step). The skill-selection rule is copied unchanged; the inventory of BE-visible skills is recorded in the session file and in `docs/ALL-SPRINTS.md` if a gap exists.
5. **Branch.** `docs/shrink-agents-md-under-32kib` created from the current branch tip, keeping the six unpushed commits. The upstream of the old branch is `[gone]`; pushing is not part of this change.
6. **Case-insensitive rename.** `git mv docs/architecture.md docs/ARCHITECTURE.tmp.md && git mv docs/ARCHITECTURE.tmp.md docs/ARCHITECTURE.md`; then repository-wide search for `architecture.md` references.
7. **`ALL-SPRINTS.md`** follows the FE schema (To do / In progress / Done), is history plus roadmap, exempt from per-task docs update, and replaces the deleted `.vscode` plan. Already created in the working tree.
8. **`.cursor`.** Only references are updated in `rule-before-plan-code.mdc` (docs list to the fixed set, plan target to `docs/ALL-SPRINTS.md`, reusable assets to `docs/SHARED.md`); its other directives are the owner's and are not edited. `session-context-handoff` skill is deleted.

### Obligation map (old BE `AGENTS.md` section, line to new location)

| Old section (line) | New location |
|---|---|
| SKILLS (103) | SKILLS (`skill-selection-rule`) + 3 triggers |
| Memory location, pre-response load (112, 118) | Global Memory Bank Rules, `memory-bank-load-rule` |
| End-of-turn persistence, verbatim (129, 140) | `memory-bank-persistence-rule` (adds `[redacted]`) |
| MCP tool contract (149) | tool list line |
| Agent rule boundaries (159), Tooling rule protection (170) | same names, markers |
| Commit rules (186) | `commit-rule` + Repository specifics (no Husky) |
| Branch rename (199) | skill `branch-rename` |
| AI Agent dev test account (226) | Repository specifics (own marker pair) |
| Scope and Applicability (250) | same name |
| Session Bootstrap (287), Propagation (394), First Action (413), Dual Lifecycle (423) | skill `session-context` + invariants |
| Initial Phase (500) | Initial Phase (routing + SKILLS) |
| Subagent Model and Cost (530) | same name |
| Middle Phase (552), Reuse and Deduplication (563) | Implementation Rule; Reuse and Deduplication Rule |
| Final Phase (578) | `final-phase-rule` (single `make check-all`) |
| Documentation and Context Sync (597), Full Doc Replacement (735), Language Consistency (758) | Documentation and Context Sync + skill `docs-sync` |
| Swagger (616) | Repository specifics (own marker pair) |
| Mandatory External Context Update (631), Learning Summary (680) | skill `session-context` |
| Final Response Requirement (710) | same name (+ skills picked) |
| Sensitive Test Accounts and Environment Secrets Policy (774) | same name |

## Risks / Trade-offs

- A condensed rule loses an obligation: obligation map filled and verified, key-phrase grep, hash of the GitNexus block.
- Session procedure depends on one trigger because Codex does not auto-load `.ai/skills`: the trigger names the exact path and `AGENTS.md` keeps the five invariants.
- `CLAUDE.md` duplicates the GitNexus block (auto-managed): accepted.
- No hook enforces the pre-commit gate: stated honestly in the Commit rule.

## Migration Plan

1. Branch, change artifacts. 2. Inventory skills. 3. Create skills. 4. Rewrite `AGENTS.md`. 5. Rename `ARCHITECTURE.md`, fix references, `.cursor`, `README.md`. 6. `make check-all`, `npx gitnexus analyze --embeddings`, re-check size. 7. Session file, Memory Bank. Rollback: `git checkout` of the listed files. Nothing is committed or pushed without an explicit request.

## Open Questions

- Whether the six local commits get pushed from this new branch (user decision).
- Whether completed BE OpenSpec changes should be archived (separate change).

## Extension

Extension (2026-10-23, user-approved): the same Degraded Environment Rule, Implementation Rule, two skills (`pre-implementation-checkpoint`, `progress-tracking`, identical to FE), triggers, and BE implementation notes were added. Shared text stays identical to `fe-mycourse/AGENTS.md`. Not added by decision: a "never commit or push unless asked" rule (user still deciding).
