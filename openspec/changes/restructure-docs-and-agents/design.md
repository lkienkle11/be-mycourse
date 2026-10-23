## Context

See `proposal.md` - Why / What Changes for the full motivation and change list. This document covers only the technical decisions needed to execute that plan without creating new drift or dead links.

Relevant current state, confirmed by direct file reads (not just filenames) during exploration:

- `docs/` has ~29 files; 9 under `docs/modules/`. Of those 9, `user.md`, `lesson.md`, and `enrollment.md` are pointer stubs with no corresponding `internal/` package — they state this explicitly.
- Module density varies by an order of magnitude: `authorization.md` is 59 lines but every line is a load-bearing invariant; `instructor.md` is 367 lines of genuinely non-obvious state-machine and ordering rules; `media.md` is 563 lines, much of it code-derivable field-mapping tables.
- `README.md`'s "Documentation Convention (Mandatory)" table already only lists 14/29 real files — the sync process has already silently fallen behind once.
- `migrations/README.md` has ~10 hardcoded links into the module docs being merged; this file was initially assumed to be an unaffected satellite doc and was only found to need edits via a backlink `git grep` pass.
- Two Go source comments (`internal/shared/middleware/auth_jwt.go:85`, `internal/shared/setting/setting.go:78`) point at doc paths that this change deletes.
- GitNexus has no editable doc-allowlist config (`.gitnexus/` contains only `meta.json` and a binary graph store) — it re-derives whatever doc association it needs at `analyze` time, so the only required GitNexus action is one re-`analyze --embeddings` after all file changes land.

## Goals / Non-Goals

**Goals:**
- Reach a fixed, 4-file `docs/` structure that cannot drift into the same triplicated-flow / triplicated-route-list problems the current structure has, by construction (one place per concern).
- Leave zero dead links: every backlink found via `git grep` (docs, README tables, migrations/README.md, Go comments) is updated in the same change, not left for later.
- Preserve every non-obvious business rule currently documented, even where that means a FEATURES.md entry runs well past a nominal "15-20 line" budget (`instructor`, `taxonomy`).
- Make the `AGENTS.md` sections that reference documentation consistent with the new structure, without touching unrelated sections.

**Non-Goals:**
- No application code behavior changes (the two Go edits are comment-only).
- No `fe-mycourse` changes — tracked as a separate follow-up change.
- No change to GitNexus configuration, Global Memory Bank rules, Tooling rule protection, Commit rules, Branch rename rules, or the secrets policy.
- Not attempting a hard per-entry line cap in `FEATURES.md` — module complexity varies too much for a uniform budget to hold without losing real rules.

## Decisions

**1. Module-level `FEATURES.md` granularity, not micro-feature-slug.**
be-mycourse's domains are entangled (e.g. course review touches Course + Authorization + Instructor in one flow), so a micro-feature-slug-per-entry scheme (one entry per narrow user-facing feature) would force the same cross-cutting rule to be repeated across multiple entries. One entry per bounded-context module keeps each rule written once. Alternative considered: keep the external reference document's micro-feature-slug scheme as-is — rejected because it was designed against a toy LMS with cleanly separated features, which doesn't match this codebase's actual coupling.

**2. Collapse `user`/`lesson`/`enrollment` into `auth`/`course`, don't give them their own entries.**
These three already state in their own current doc files that no standalone package backs them. Giving them FEATURES.md entries would re-introduce the same "module that isn't a module" fiction the current structure has. Their content (the `/me` endpoints, in-course lesson/enrollment behavior) moves into the `auth` and `course` entries respectively.

**3. Variable entry length per module, not a fixed line budget.**
Confirmed by full reads of all 9 current module docs: `rbac` compresses to ~40 lines without loss, but `instructor`'s approve-ordering safety invariant and state-priority rules, and `taxonomy`'s write-sync/optimistic-lock rules, cannot be cut to 15-20 lines without deleting rules that have no other home. Alternative considered: enforce a uniform short budget and push overflow into code comments — rejected because these are cross-function invariants (e.g. "step 2 must not run before step 1's DB write commits"), not something a single function's comment can carry.

**4. Keep Mermaid diagrams for error-code-sensitive flows; drop them elsewhere.**
Sequence diagrams earn their place only where the branching itself is the hard-to-hold-in-your-head part (auth, slug-conflict 3007, optimistic-lock 3005). Everywhere else, prose in the FEATURES.md entry is sufficient and cheaper to keep in sync than a diagram.

**5. Add `docs/SHARED.md` (not in the original external reference document).**
The reference document's BE structure omitted a shared/reusable-infra file, but `fe-mycourse` already has one (`docs/reusable-assets.md`) and its README treats checking it as mandatory before adding any new utility. Without a BE equivalent, the "Critical Reuse Rule" has nothing concrete to point agents at. `SHARED.md` mirrors that pattern.

**6. Delete `curl_api.md` outright rather than merge it.**
Its content (hand-written curl examples) is now redundant with the generated `docs/api_swagger.yaml` / `docs/api-dog-import.json`. Merging it into FEATURES.md would just relocate duplication that generation already solves. `README.md`'s reference to it (smoke-test examples, line ~167) is updated to point at the generated Swagger instead.

**7. Sequencing: fix backlinks in the same change, not as follow-up cleanup.**
All three backlink groups (README.md, migrations/README.md, two Go comments) are updated as part of this change's task list rather than deferred, because deferring them is exactly how the current drift (e.g. `router.md`'s empty Locale Configuration section) happened in the first place.

**8. AGENTS.md rule changes are scoped to sections with confirmed, specific problems — not a rewrite.**
Each of the 8 touched sections maps to a concrete issue found during exploration (e.g. Exit Code Rule currently can't distinguish a pre-existing CI failure from one the task caused; Session Bootstrap's external-file mechanism is unnecessary when the Claude Code runtime already exposes a session ID). Sections with no identified problem (GitNexus blocks, Memory Bank rules, Tooling protection, Commit rules, Branch rename rules) are left untouched to avoid introducing new risk where none was found.

**9. Any rule text that branches `be-mycourse` / `fe-mycourse` must stay directly portable to `fe-mycourse/AGENTS.md`.**
`fe-mycourse/AGENTS.md` was confirmed (byte-for-byte diff) to be a near-identical copy of `be-mycourse/AGENTS.md` before this change, apart from the `AI Agent — dev test account` block. Editing a shared section in `be-mycourse` only, without writing it so it already branches correctly for both repos, silently makes the two files drift apart with no mechanism to catch it (there is deliberately no shared/imported file between the two repos — a prior, separate decision). So every touched rule that has repo-specific content must say `For be-mycourse: ... / For fe-mycourse: ...` inline, using `fe-mycourse`'s *current* file names only where they concretely exist (e.g. `docs/reusable-assets.md`), and a generic "as currently structured" placeholder — explicitly marked as such, with a named list of which sections to tighten later — where `fe-mycourse` has no fixed-file equivalent yet (since `fe-mycourse/docs/` restructuring is an out-of-scope follow-up, not yet decided). Copying the affected `be-mycourse/AGENTS.md` sections verbatim into `fe-mycourse/AGENTS.md` is intentionally left as a separate, not-yet-scheduled action — this decision only requires that the `be-mycourse` text be *ready* to copy, not that the copy happen now.

**10. Session/context-lifecycle duplication: keep the full version in one place, trim the others to a cross-reference — never remove an obligation while trimming.**
`Critical Dual Session Context Lifecycle Rule` is the authoritative, step-by-step version for both the in-scope and out-of-scope paths. `Critical Pre-Conversation Session Bootstrap Rule`'s tail (lines ~390-394) and `Critical First Action: Read Existing Project Context` both restate the in-scope half of it in less detail; `Critical Dual Session Context Lifecycle Rule`'s own out-of-scope branch restates what `Mandatory External Context Update for Read-Only Repository Tasks` covers in full. The fix is a straight cross-reference trim, not a content rewrite: every obligation currently stated in a "brief" copy must still be true after the trim (just stated once, pointed to from the shorter sections) — if a brief copy ever contains an obligation the full version doesn't have, that obligation must be moved into the full version first, not silently dropped. Alternative considered: merge everything into one giant section — rejected, because `Session Bootstrap` (how to get a Session ID) and `Critical First Action` (what to do with it on first read) are genuinely different moments in the task lifecycle and deserve to stay separate sections; only the *restated lifecycle steps* are redundant, not the sections themselves.

**11. `Subagent Model and Cost Restriction Rule`'s misplaced be/fe paragraphs move to `Initial Phase`, not deleted outright.**
The two paragraphs (574-588: "read the repository carefully... identify module location...") are topically about project understanding, not model/cost limits, and overlap with `Initial Phase: Project Understanding`'s existing guidance. Moving (not just deleting) preserves the content in case it has detail `Initial Phase` doesn't already carry (e.g. the fe-mycourse bullet list mentions "components, hooks" explicitly, which the current `Initial Phase` fe-mycourse line doesn't call out) — any such detail gets folded into `Initial Phase` during the move, not lost.

**12. `docs/ARCHITECTURE.md`'s Data Access Patterns bullets trim to cross-references, not deletion.**
The SQL-console-logging and audit-timestamps bullets in `docs/ARCHITECTURE.md:353-363` restate facts whose authoritative, fuller version lives in `docs/DATA-MODEL.md` (schema-level framing) and `docs/SHARED.md` (reusable-asset framing) respectively. `ARCHITECTURE.md` keeps a one-line pointer for each (so a reader scanning Data Access Patterns still knows these exist and where to look), rather than removing all mention — this matches decision 7's "fix the backlink in place" philosophy rather than creating a new dead end.

## Risks / Trade-offs

- **[Risk]** Merging 25 files into 4 by hand risks silently dropping a rule during consolidation, especially from the denser modules (`instructor`, `taxonomy`, `media`). → **Mitigation**: tasks.md requires each source file to be read in full immediately before its content is merged (already done once during exploration; re-verify at merge time), and the old file is only deleted after its replacement section exists in the new file.
- **[Risk]** `FEATURES.md` entries without a line cap could regress back toward sprawl over time. → **Mitigation**: the cap is removed deliberately, but entries still follow the "only write what code can't say for itself" principle (e.g. `media.md`'s pure field-mapping tables are dropped, not carried over) — this is a content-quality norm, not a mechanical limit.
- **[Risk]** `AGENTS.md`'s Critical Reuse/Dedup consolidation changes a strict "redo everything" policy agents currently follow into a judgment call ("semantic ownership"). This is less mechanically enforceable. → **Mitigation**: accepted trade-off — the strict version was already identified as a problem (it incentivizes avoiding reuse checks rather than encouraging them), and the proposal explicitly calls this out as a deliberate policy change, not a pure rewording.
- **[Risk]** Deleting `IMPLEMENTATION_PLAN_EXECUTION.md` and the `.vscode` plan file removes tracked git history content. → **Mitigation**: both are confirmed dead (the `.vscode` file references an entirely obsolete `§0`-`§11` rule system); git history retains them if ever needed, so deletion from the working tree is safe.
- **[Risk]** GitNexus's project-context association with docs is not configurable, so there's no way to directly verify it picked up the new files pre-merge. → **Mitigation**: the mandated post-change `npx gitnexus analyze --embeddings` re-derives this automatically; no special handling needed beyond running it once at the end.
- **[Risk]** Trimming the session/context-lifecycle duplication (decision 10) touches safety/control-flow rules (Session ID handling, `.context` file placement) — a careless trim could silently drop an obligation, not just remove repeated text. → **Mitigation**: trim only after diffing each "brief" section against the full `Critical Dual Session Context Lifecycle Rule` / `Mandatory External Context Update` section line-by-line to confirm every obligation in the brief version is already present in the full version; if not, add it to the full version before trimming the brief one.
- **[Risk]** Re-verified against the current `fe-mycourse/AGENTS.md` state before each `be-mycourse`-side edit lands, decision 9's "must stay portable" requirement could be missed on a future, unrelated `AGENTS.md` edit (made by a different session with no memory of this decision). → **Mitigation**: accepted as a known process gap, not solved by this change — there is no shared/imported file between the repos by design (prior decision), so this depends on whoever edits `be-mycourse/AGENTS.md`'s shared sections next remembering to branch be/fe inline, same as this change did.

## Migration Plan

1. Build the 4 new files with content merged from the old ones (old files stay in place during this step so nothing is lost mid-merge).
2. Delete the old `docs/*.md` / `docs/modules/*.md` files and the 2 dead root/`.vscode` files, once each old file's content is confirmed present in a new file.
3. Fix every backlink (`README.md`, `migrations/README.md`, 2 Go comments) in the same change.
4. Update the 8 `AGENTS.md` sections and reduce `CLAUDE.md` to `@AGENTS.md`.
5. Run `npx gitnexus analyze --embeddings` once, last.

No rollback tooling is needed beyond normal git revert — this is a docs/rules-only change with no runtime component to roll back.
