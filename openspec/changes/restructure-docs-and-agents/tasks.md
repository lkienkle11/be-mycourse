## 1. Build docs/ARCHITECTURE.md

- [x] 1.1 Merge `docs/architecture.md`, `docs/folder-structure.md`, and the "Composition Root"/"HTTP Request Path" bootstrap description duplicated in `docs/deploy.md` Appendix B into one `docs/ARCHITECTURE.md`; verify by diffing the merged sections against each source file to confirm no bootstrap-sequence detail was dropped.
- [x] 1.2 Fold in the still-relevant parts of `docs/patterns.md` (architecture-level patterns only, not per-module ones that belong in FEATURES.md entries) and `docs/requirements.md`; verify every retained claim still matches current code by spot-checking against the `internal/` package layout.
- [x] 1.3 Keep the deploy-specific content that is NOT duplicated (local dev steps, CI/CD, Docker specifics) in `docs/deploy.md` and `docs/docker.md` as-is; verify `docs/ARCHITECTURE.md` does not re-describe them, only cross-references. (`docs/deploy.md` Appendix B's stale duplicated table was replaced with a pointer to `docs/ARCHITECTURE.md`, since it used obsolete package names.)

## 2. Build docs/FEATURES.md

- [x] 2.1 Create the `auth` entry by merging `docs/modules/auth.md` with the `/me`, `/me/permissions` endpoint content from `docs/modules/user.md`; verify every rule present in both source files appears once in the merged entry.
- [x] 2.2 Create the `course` entry by merging `docs/modules/course.md` (if present) with the in-course lesson/enrollment behavior from `docs/modules/lesson.md` and `docs/modules/enrollment.md`; verify no rule from either stub file is lost.
- [x] 2.3 Create the `taxonomy` entry preserving locale canonicalization, write-sync canonical/translation conflict handling, slug-independence, and the 409/3005 optimistic-lock rule from `docs/modules/taxonomy.md`; verify entry length reflects genuine complexity (not forced under a line cap) and keep the Mermaid diagram for the slug-conflict (3007) flow. (No such diagram existed anywhere in the repo for this flow; an ASCII flow substitute was used instead — confirmed via repo-wide search, nothing lost.)
- [x] 2.4 Create the `instructor` entry preserving the state-machine A-H priority order, the 5-day SLA rule, Delta-JSON structural allowlist validation, certificate dedup rule, and the approve-ordering safety invariant from `docs/modules/instructor.md`; verify each rule's exact wording still matches `docs/modules/instructor.md`'s source before deleting it.
- [x] 2.5 Create the `media` entry preserving the upload pipeline, MIME-detection layers, ownership/visibility rules, and Bunny webhook idempotency behavior from `docs/modules/media.md`, deliberately dropping pure field-mapping tables that just restate code; verify by checking the dropped tables are genuinely code-derivable (e.g. via a direct code reference) and not unique business rules.
- [x] 2.6 Create the `rbac` entry from `docs/modules/rbac.md` (permission check flow: JWT fast path + DB fallback, additive-permissions rule, cache/token freshness trade-off); verify entry covers all "Critical Rules" bullets from the source file.
- [x] 2.7 Create the `authorization` entry from `docs/modules/authorization.md` (grant semantics, 6-step decision order, wildcard resource binding, `ReplaceMany` atomicity); verify every invariant in the 59-line source survives given how dense it is relative to its length.
- [x] 2.8 Add an `API:` line to each FEATURES.md entry pointing at the relevant Swagger/Postman path instead of `docs/curl_api.md`; verify by cross-checking against `docs/api_swagger.yaml`.

Additional gap-fill (discovered mid-execution, not in the original per-task list): `docs/requirements.md`, `docs/api-overview.md`, `docs/router.md`, `docs/return_types.md`, `docs/data-flow.md`, `docs/logic-flow.md`, and `docs/sequence_diagrams.md` were also audited and reconciled into `FEATURES.md`/`ARCHITECTURE.md` before deletion — these were part of the original redundancy findings but had not been assigned to a specific build task. Real content merged: auth Mermaid sequence diagrams (register/confirm/login/refresh/logout with full error-code branches), media upload limits, auth rate-limit/OAuth error codes, course owner-only draft-review rule, course DTO field-leakage rules, JWT/RequirePermission middleware flow diagram (→ ARCHITECTURE.md), and a full error-code reference table (→ ARCHITECTURE.md).

## 3. Build docs/DATA-MODEL.md

- [x] 3.1 Merge `docs/database.md` schema content into `docs/DATA-MODEL.md`; verify table/column coverage matches current migrations by cross-checking `migrations/` filenames against documented tables.
- [x] 3.2 Move the permission catalog into its own appendix section in `docs/DATA-MODEL.md` (not squeezed per-table); verify the full P1-P58 permission list is present and matches the "AI Agent — dev test account" reference in `AGENTS.md`/`CLAUDE.md` ("full P1-P58 on dev DB"). **Correction found during this task: the real maximum is P68, not P58 or P40 (database.md's stale heading).** Confirmed via `internal/shared/constants/permissions.go` (68 constants). `AGENTS.md`/`CLAUDE.md`'s dev-account line was corrected to P1–P68 in section 7.

## 4. Build docs/SHARED.md

- [x] 4.1 Audit `internal/shared/` for reusable utilities, middleware, and helpers worth cataloguing; verify by listing exported symbols under `internal/shared/` and cross-checking each has a one-line entry.
- [x] 4.2 Mirror the structure already used by `fe-mycourse/docs/reusable-assets.md` (one line per reusable component/util); verify format consistency by comparing the two files side by side. (BE already had a near-identical-format `docs/reusable-assets.md`; its content was curated/corrected forward rather than rebuilt from scratch, including fixing a stale "Gap Analysis" that claimed already-built features were missing.)

## 5. Delete superseded and dead files

- [x] 5.1 Delete `docs/curl_api.md`; verify `docs/api_swagger.yaml` and `docs/api-dog-import.json` cover its former examples before removal.
- [x] 5.2 Delete all merged-away `docs/*.md` and `docs/modules/*.md` files; verified only the 4 new files plus explicitly-kept satellite docs (`docs/deploy.md`, `docs/docker.md`, `docs/observability/loki-alloy.md`, `docs/security-public-seo-notes.md`, `docs/course-collaboration-handoff-2026-06-04.md`) remain under `docs/`. Also deleted `docs/modules.md` (the old module index, fully superseded) and `docs/dependencies.md` (merged into `ARCHITECTURE.md` first). **Note:** macOS's case-insensitive filesystem meant `docs/architecture.md` and the new `docs/ARCHITECTURE.md` were the same file on disk; fixed via a two-step `git mv` so git correctly tracks it as a rename, not a stray modification.
- [x] 5.3 Delete `IMPLEMENTATION_PLAN_EXECUTION.md` (root) and `.vscode/e-learning_crud_backend_8f5ca286_reset_all_pending.plan.md`; verified no remaining references to the deleted content. One benign match remains: `.cursor/rules/rule-before-plan-code.mdc` is the Cursor workflow rule that *writes* `IMPLEMENTATION_PLAN_EXECUTION.md` in the first place — out of this change's scope, flagged separately to the user.

## 6. Fix backlinks

- [x] 6.1 Rewrite `README.md`'s "Documentation Convention (Mandatory)" doc-index table to list exactly the 4 new `docs/` files (plus the kept satellite docs).
- [x] 6.2 Updated the 3 scattered `README.md` path references.
- [x] 6.3 Updated all `migrations/README.md` cross-references (media/instructor/course/taxonomy/authorization module docs → `docs/FEATURES.md`; `docs/database.md` → `docs/DATA-MODEL.md`).
- [x] 6.4 Updated the doc-path comment in `internal/shared/middleware/auth_jwt.go:85`; confirmed via `make check-all` build success.
- [x] 6.5 Updated the doc-path comment in `internal/shared/setting/setting.go:78`; confirmed via `make check-all` build success.
- [x] 6.6 Ran the final repo-wide backlink sweep (both `docs/`-prefixed and relative `](./...)` link styles) — zero remaining dead references. Additional backlinks found and fixed beyond the original list: `docs/deploy.md` (9 references, including a stale duplicated startup-sequence table replaced with a pointer), `docs/docker.md`, `docs/security-public-seo-notes.md` (2 references), `docs/course-collaboration-handoff-2026-06-04.md` (2 references).

## 7. Update AGENTS.md and CLAUDE.md

- [x] 7.1 Reduce `CLAUDE.md` to a single `@AGENTS.md` import line. **Not durable:** running `npx gitnexus analyze` (task 8.1) automatically rewrote `CLAUDE.md` back to a full self-contained GitNexus block — confirmed this is the tool's by-design behavior (`--skip-agents-md` flag exists specifically to prevent it). Left as gitnexus wrote it rather than re-stripping it, since any future `analyze` run (mandated after every commit) would revert it again. Flagged to the user as a deviation from the original plan — not something this change can durably achieve without always passing `--skip-agents-md` on every future `analyze`, which trades away automatic stat freshness.
- [x] 7.2 Updated AGENTS.md's "Initial Phase: Project Understanding" section to reference the new docs structure.
- [x] 7.3 Updated AGENTS.md's "Full Documentation Replacement Rule" to scope doc sync to the specific `docs/FEATURES.md` module entry; the changelog/migration-history exception is preserved and now explicitly names `migrations/README.md` and dated handoff notes as the exception.
- [x] 7.4 Updated AGENTS.md's "Swagger and API Documentation Rule" to point at the `API:` line in `docs/FEATURES.md`; the `ruby generate-apidog-postman.rb` step is unchanged.
- [x] 7.5 Updated the reuse rule to anchor on checking `docs/SHARED.md` first (done as part of 7.6's consolidation).
- [x] 7.6 Consolidated "Critical Reuse Rule" and "Critical Deduplication Rule" into one "Reuse and Deduplication Rule" section, replacing "redo entire solution from scratch" with "merge the duplicated piece, judged by semantic ownership." Wrapped in `<!-- begin:reuse-and-deduplication-rule:start/end -->` markers per the Agent rule boundaries convention.
- [x] 7.7 Updated "Session Bootstrap" to prefer `CLAUDE_CODE_SESSION_ID` before the manual `AI_SESSION_ID_FILE` fallback; the fallback mechanism itself is untouched.
- [x] 7.8 Updated "Exit Code Rule" (now folded into the Final Phase section, see 7.9) to distinguish task-caused / pre-existing / environment failures.
- [x] 7.9 Consolidated "Final Phase" (Quality Gates + Exit Code Rule + Post-Implementation Code Review) into one "Final Phase: Sync, Quality Gates, and Review" section, wrapped in `<!-- begin:final-phase-rule:start/end -->` markers. Verified every original obligation survived (BE/FE gate split, review checklist items).
- [x] 7.10 Re-read the full edited `AGENTS.md`; all 7 `<!-- begin/end -->` marker pairs verified well-formed via grep; all headings unique, no duplicates or orphaned old section names.

## 8. Final sync and verification

- [x] 8.1 Ran `npx gitnexus analyze --embeddings`. Index succeeded (7,791 nodes, 24,197 edges, 343 clusters, 300 flows); `.gitnexus/meta.json` confirms `stats.embeddings: 3487` (non-zero, preserved). A crash (`libc++abi`/mutex error) occurred after the successful write, during tool exit/cleanup — did not affect the result.
- [x] 8.2 `npx gitnexus status` confirms `Status: up-to-date`, indexed commit matches `git rev-parse HEAD` (`8e8b2d4`).
- [x] 8.3 `make check-all` passed cleanly: all tests pass, `go vet` clean, `golangci-lint run` 0 issues, layout/architecture checks OK, 0 duplicate clone groups, both `build-nocgo` and `build` (CGO) succeeded.

## 9. Follow-up: fe-mycourse portability fixes (retroactive)

Found and fixed during post-implementation review, after task 8 ran. Documented here retroactively per the Full Documentation Replacement Rule's own "replace, don't just append" principle.

- [x] 9.1 Fixed a real contradiction in "Critical Pre-Conversation Session Bootstrap Rule": the native-runtime-session-ID paragraph originally said to use the native ID *instead of* the file-based mechanism, contradicting the unchanged validation checklist a few paragraphs later (which still requires `AI_SESSION_ID_FILE` to exist and calls it "the authoritative identity source"). Reworded so the native ID is the *seed value* written into the file, not a bypass of it; fixed bootstrap lifecycle step 4 to match.
- [x] 9.2 Added the missing `fe-mycourse` branch (`docs/reusable-assets.md`) to the Reuse and Deduplication Rule — it only named `docs/SHARED.md` for be-mycourse after the first edit pass, despite the rule governing both repos' tasks.
- [x] 9.3 Added the missing `For fe-mycourse: ...` branch to the Swagger and API Documentation Rule (previously be-mycourse-only after the first edit pass, silently dropping the original generic fe-applicable wording) and to the Full Documentation Replacement Rule (previously had no fe-mycourse equivalent of its be-mycourse trailing paragraph at all).
- [x] 9.4 Added an explicit note in "Initial Phase: Project Understanding" marking every `fe-mycourse` "as currently structured" reference in the file as a deliberate placeholder, not an oversight — names the exact sections to tighten if/when `fe-mycourse` adopts a fixed docs structure, and explicitly excludes the Reuse and Deduplication Rule's `docs/reusable-assets.md` reference from that placeholder set (that one already names a concrete, current file).

## 10. Consolidate AGENTS.md session/context-lifecycle duplication

- [x] 10.1 Diffed "Critical Pre-Conversation Session Bootstrap Rule"'s tail against "Critical Dual Session Context Lifecycle Rule"; confirmed every obligation already covered there. Trimmed the 2 restated paragraphs (in-scope path + the long out-of-scope task-type list, which also duplicated "Mandatory External Context Update"'s own list) to one cross-reference sentence.
- [x] 10.2 Diffed "Critical First Action: Read Existing Project Context"'s in-scope/out-of-scope bullet lists against "Critical Dual Session Context Lifecycle Rule"; confirmed no unique obligation. Trimmed the resolve/reuse/create-file steps and the out-of-scope forbidden-actions bullets to 2 short paragraphs, keeping only what's genuinely specific to "first action" framing (must read `.context` before in-scope work; out-of-scope reads stay read-only).
- [x] 10.3 Diffed "Critical Dual Session Context Lifecycle Rule"'s out-of-scope final-response checklist (10 bullets) against "Mandatory External Context Update for Read-Only Repository Tasks". Found 5 bullets fully restated there (trimmed to 1 cross-reference bullet), 1 bullet already self-duplicate within the same section's earlier lifecycle steps (dropped, no obligation lost), 1 unique obligation not present anywhere else ("replace outdated/contradictory content instead of appending") — **moved** (not dropped) into "Mandatory External Context Update" itself. Kept 3 bullets specific to Session Bootstrap/Propagation concerns (external-file location, Session ID match, no duplicate files) that "Mandatory External Context Update" doesn't cover.
- [x] 10.4 Re-grepped `<!-- begin:/end: -->` marker pairs after the trims; all 7 pairs still well-formed, correctly nested, no orphaned markers.

## 11. Relocate misplaced Subagent Model content

- [x] 11.1 Moved the be-mycourse/fe-mycourse "read the repository carefully, identify module/folder structure" bullets out of "Subagent Model and Cost Restriction Rule" into "Initial Phase: Project Understanding" (merged into the existing be/fe docs-reading paragraph as one combined "beyond docs, understand repo structure" sentence). Folded in fe-mycourse's explicit "components, hooks" mention rather than dropping it.
- [x] 11.2 Confirmed "Subagent Model and Cost Restriction Rule" reads coherently as model-tier/cost-limit guidance only after the move — ends cleanly on the main-agent-coordination sentence, no orphaned transition.

## 12. Trim docs/ARCHITECTURE.md Data Access Patterns duplication

- [x] 12.1 Reduced the SQL-console-logging bullet in `docs/ARCHITECTURE.md`'s "Data Access Patterns" section to a one-line cross-reference to `docs/SHARED.md`'s "Asset: GORM SQL console logger" entry (verified heading exists at `docs/SHARED.md:85`).
- [x] 12.2 Reduced the audit-timestamps bullet to a one-line cross-reference to `docs/DATA-MODEL.md`'s "Audit timestamps" subsection (verified heading exists at `docs/DATA-MODEL.md:144`) and `docs/SHARED.md`'s `timex`/`gormx` asset entries.
- [x] 12.3 Also trimmed the "table names via constants" bullet to add a cross-reference to `docs/DATA-MODEL.md`'s "Code ↔ table names" section, while touching the section anyway.

## 13. Final re-verification (second pass)

- [x] 13.1 Ran `npx gitnexus analyze --embeddings` again. Result: "Already up to date" — GitNexus indexes against the last Git commit (`8e8b2d4`), and nothing in this change has been committed yet, so there was nothing new for it to pick up. `npx gitnexus status` confirms indexed commit still matches current commit. No action needed until this change is committed.
- [x] 13.2 Re-ran `make check-all` — passed cleanly (tests, `go vet`, `golangci-lint` 0 issues, layout/architecture checks OK, 0 duplicate clone groups, both builds succeeded), as expected since groups 10-12 only touched `AGENTS.md` and `docs/ARCHITECTURE.md`, no code.
- [x] 13.3 Final read-through of the edited session/context sections and the trimmed `docs/ARCHITECTURE.md` Data Access Patterns bullets confirmed no obligation was lost — every trimmed bullet now cross-references a section that was verified (via grep) to actually contain the full obligation. `AGENTS.md` is now 830 lines (was 866 after the first follow-up round, 884 in the original pre-change commit) — net 54 lines shorter than the original, with the session/context and docs-pattern duplication specifically addressed this round.
