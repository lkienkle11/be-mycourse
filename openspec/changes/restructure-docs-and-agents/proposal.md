## Why

`be-mycourse`'s `docs/` folder has grown to ~29 files with confirmed, content-verified redundancy: the same flows described at three levels of detail (`data-flow.md` / `logic-flow.md` / `sequence_diagrams.md`), the same route list hand-maintained in three places (`api-overview.md`, `router.md`, `curl_api.md`), a bootstrap sequence duplicated between `architecture.md` and `deploy.md`, and three "module" docs (`user.md`, `lesson.md`, `enrollment.md`) that are pure pointer stubs saying the module doesn't actually exist as a separate package. `README.md`'s own "Documentation Convention (Mandatory)" doc-index table already only lists 14 of the 29 real files, and `docs/router.md` has a real drift bug (an empty `## Locale Configuration` section whose content now lives in a different file) — concrete proof the current doc-sync process does not reliably hold up in practice. `CLAUDE.md` separately duplicates the entire GitNexus block that `AGENTS.md` already has, instead of just importing it. None of this is hypothetical cleanup — every redundancy and drift point listed here was confirmed by reading the actual file contents, not just filenames.

Fixing this now (rather than continuing to add module docs under the current structure) stops the drift from compounding and gives the AI agent workflow a stable, finite set of files to read and update instead of a 29-file surface that nobody — human or agent — keeps fully in sync.

**Follow-up finding (post-implementation review):** after the 4-file `docs/` restructure and the first round of `AGENTS.md` edits landed, a second review pass (prompted by the file still measuring 866 lines, barely below the original 884) found two more confirmed redundancy clusters that were not in the original exploration:
- `AGENTS.md`'s own "Implementation Workflow Rules" section restates the same rule twice in two places: once briefly, once in full, for both the in-scope session-context lifecycle (`Critical Pre-Conversation Session Bootstrap Rule`'s tail + `Critical First Action` both restate what `Critical Dual Session Context Lifecycle Rule` already covers in full) and the out-of-scope lifecycle (`Critical Dual Session Context Lifecycle Rule`'s out-of-scope branch restates what `Mandatory External Context Update for Read-Only Repository Tasks` already covers in full). Separately, `Subagent Model and Cost Restriction Rule` contains two be-mycourse/fe-mycourse paragraphs about reading repo structure that belong under `Initial Phase: Project Understanding`, not under a model/cost-pricing rule.
- The new `docs/ARCHITECTURE.md`'s "Data Access Patterns" section independently restates two facts (SQL console logging thresholds, audit-timestamp migration/helpers) that already have a fuller, authoritative home in `docs/DATA-MODEL.md` and `docs/SHARED.md` respectively — confirmed via a full read of all 4 new docs files, not just a skim.

Both are addressed in this same change (see new "What Changes" items and tasks group 9-11) rather than a separate change, since they're a direct continuation of the same redundancy-reduction goal this change already exists for.

## What Changes

- Collapse `be-mycourse/docs/` from ~29 files down to 4 fixed files: `docs/ARCHITECTURE.md`, `docs/FEATURES.md`, `docs/DATA-MODEL.md`, `docs/SHARED.md` (new).
- `docs/FEATURES.md` holds ~6 real module-level entries (`auth` absorbing `user`, `course` absorbing `lesson`+`enrollment`, `taxonomy`, `instructor`, `media`, `rbac`, `authorization`), each sized to the module's actual complexity rather than a fixed line budget — dense modules like `instructor` and `taxonomy` keep their non-obvious business rules (state-machine priority order, optimistic-lock conflict codes) in full.
- `docs/DATA-MODEL.md` holds the schema overview plus the permission catalog as its own appendix.
- `docs/SHARED.md` is a new file cataloguing reusable BE infra/utilities, mirroring the pattern already used by `fe-mycourse/docs/reusable-assets.md`.
- Mermaid sequence diagrams are preserved (not flattened to prose) for flows where the diagram carries error-code-specific branching (auth, slug conflict 3007, optimistic lock 3005).
- **BREAKING (docs only)**: `docs/curl_api.md` is deleted; its content is superseded by the already-generated `docs/api_swagger.yaml` + `docs/api-dog-import.json`.
- All remaining current `docs/*.md` and `docs/modules/*.md` files are merged into the 4 new files, then deleted.
- Delete two confirmed-dead tracked files: `IMPLEMENTATION_PLAN_EXECUTION.md` (root) and `.vscode/e-learning_crud_backend_8f5ca286_reset_all_pending.plan.md`.
- Fix every real backlink found via `git grep` so the restructuring does not create new dead links: `README.md`'s doc-index table and three scattered path references, `migrations/README.md`'s ~10 hardcoded cross-references into the old module docs, and two Go source code comments (`internal/shared/middleware/auth_jwt.go:85`, `internal/shared/setting/setting.go:78`).
- `CLAUDE.md` is reduced to a single `@AGENTS.md` import, removing its duplicated GitNexus block.
- `AGENTS.md` sections "Initial Phase: Project Understanding", "Full Documentation Replacement Rule", "Swagger and API Documentation Rule", and "Critical Reuse Rule" are updated to reference the new 4-file structure instead of the old broad/scattered one.
- `AGENTS.md` "Session Bootstrap" section is updated to prefer the native runtime session ID (`CLAUDE_CODE_SESSION_ID`) before falling back to the manual external-file bootstrap mechanism.
- `AGENTS.md` "Exit Code Rule" is updated to distinguish task-caused failures from pre-existing and environment errors, instead of forcing a fix for every non-zero exit regardless of cause.
- `AGENTS.md` "Critical Reuse Rule" and "Critical Deduplication Rule" (currently two near-duplicate sections) are consolidated into one, replacing the "redo the entire solution from scratch on any duplicate" policy with a targeted "merge the duplicated piece" policy judged by semantic ownership rather than structural similarity.
- `AGENTS.md`'s overlapping "Final Phase" sections (Quality Gates, Exit Code Rule, Post-Implementation Code Review) are consolidated into one cohesive end-of-task flow.
- No other `AGENTS.md` sections change (GitNexus blocks, Global Memory Bank rules, Agent rule boundaries, Tooling rule protection, Commit rules, Branch rename rules, dev test account block, and the secrets policy are explicitly out of scope for this change).
- Re-run `npx gitnexus analyze --embeddings` once, after all file changes land, to refresh the index (existing mandated step, not a new rule).
- **(Already applied during follow-up review, documented here retroactively)**: fixed a real internal contradiction in `AGENTS.md`'s Session Bootstrap section (the native-runtime-session-ID paragraph was worded as replacing the file-based mechanism instead of seeding it, contradicting the unchanged validation checklist); added the missing `fe-mycourse` branch to the Reuse and Deduplication Rule (`docs/reusable-assets.md`) and to the Swagger and API Documentation Rule / Full Documentation Replacement Rule (both were left be-mycourse-only after the first edit pass); added an explicit note marking `fe-mycourse`'s "as currently structured" phrasing as a deliberate placeholder (not an oversight) with a named list of exactly which sections must be tightened if/when `fe-mycourse` adopts an equivalent fixed docs structure.
- **(New, pending)** Consolidate `AGENTS.md`'s session/context-lifecycle duplication: `Critical Pre-Conversation Session Bootstrap Rule` and `Critical First Action: Read Existing Project Context` each restate what `Critical Dual Session Context Lifecycle Rule` already covers in full — trim the two shorter restatements to a cross-reference instead of repeating the lifecycle steps. Symmetrically, `Critical Dual Session Context Lifecycle Rule`'s out-of-scope branch restates what `Mandatory External Context Update for Read-Only Repository Tasks` already covers in full — trim that restatement too.
- **(New, pending)** Move the two be-mycourse/fe-mycourse "read the repository carefully, identify module/folder structure" paragraphs out of `Subagent Model and Cost Restriction Rule` (where they don't belong — that section is about model tier/cost limits) into `Initial Phase: Project Understanding` (where equivalent guidance already lives), removing the duplication between the two sections.
- **(New, pending)** Trim `docs/ARCHITECTURE.md`'s "Data Access Patterns" section: its SQL-console-logging bullet and audit-timestamps bullet currently restate facts that already have a fuller, authoritative version in `docs/DATA-MODEL.md` and `docs/SHARED.md` respectively — reduce both bullets to one-line cross-references.

## Capabilities

### New Capabilities

_None._ This change does not add, remove, or modify any application-level behavior, API, or runtime capability of `be-mycourse`. It only restructures documentation files and the AI-agent workflow instructions that govern how docs are kept in sync.

### Modified Capabilities

_None._ No spec-level (application) requirement changes. `skip_specs: true` is set in this change's `.openspec.yaml` accordingly.

## Impact

- **Affected files (deleted)**: ~25 of the current ~29 `docs/*.md` and `docs/modules/*.md` files, `docs/curl_api.md`, `IMPLEMENTATION_PLAN_EXECUTION.md`, `.vscode/e-learning_crud_backend_8f5ca286_reset_all_pending.plan.md`.
- **Affected files (created)**: `docs/ARCHITECTURE.md`, `docs/FEATURES.md`, `docs/DATA-MODEL.md`, `docs/SHARED.md`.
- **Affected files (edited)**: `CLAUDE.md`, `AGENTS.md` (8 original sections + follow-up fixes to Session Bootstrap/Reuse Rule/Swagger Rule/Full Doc Replacement Rule, plus pending consolidation of session/context-lifecycle duplication and the misplaced Subagent Model content), `docs/ARCHITECTURE.md` (pending: Data Access Patterns trim), `README.md` (doc-index table + 3 path references), `migrations/README.md` (~10 cross-references), `internal/shared/middleware/auth_jwt.go` (1 comment), `internal/shared/setting/setting.go` (1 comment).
- **No application code behavior changes** — Go source edits are comment-only.
- **GitNexus**: index must be re-analyzed (`--embeddings`) once after this change lands, since doc content it associates with the project context shifts.
- **Out of scope**: `fe-mycourse` gets an equivalent but separate treatment in a follow-up change — the two repos are kept intentionally independent with no shared files.
