<!-- begin:lifecycle-rule:start -->
# Lifecycle (read first)

Read this whole file before the first tool call, then follow this order for every request. The sections and skills named here hold the details; do not skip a step because an earlier one looks small.

1. **Classify** the request as in scope (changes the project) or out of scope (read-only): "Scope and Applicability Rule".
2. **Session** (in scope): read `.ai/skills/session-context/SKILL.md` before reading or changing any project file.
3. **Understand:** pick skills ("SKILLS") and read documentation by routing ("Initial Phase").
4. **Plan**, when one is required (the user asked for it, an OpenSpec change exists, or the agent tooling has its own plan gate): record it and wait for explicit approval.
5. **Pre-implementation checkpoint:** the moment right before the first create, edit, or delete of any file outside `.context/`, `docs/`, and `openspec/`. It applies whether or not step 4 happened. Read `.ai/skills/pre-implementation-checkpoint/SKILL.md` and follow it; only a few-line change with no behavior change, or a task that touches only those three folders, skips it.
6. **Implement** ("Implementation Rule"), tracking progress with `.ai/skills/progress-tracking/SKILL.md`.
7. **Finish:** "Final Phase", then "Documentation and Context Sync", then the "Final Response Requirement".
<!-- end:lifecycle-rule:end -->

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **be-mycourse** (7901 symbols, 24418 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/be-mycourse/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/be-mycourse/context` | Codebase overview, check index freshness |
| `gitnexus://repo/be-mycourse/clusters` | All functional areas |
| `gitnexus://repo/be-mycourse/processes` | All execution flows |
| `gitnexus://repo/be-mycourse/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

## SKILLS

<!-- begin:skill-selection-rule:start -->
Skills are instructions packaged for a kind of work. Use them as follows; user instructions and `AGENTS.md` / `CLAUDE.md` always override a skill, a skill never overrides Tooling rule protection, and when two skills conflict the workflow in this file wins.

1. **When:** before any action of an in-scope task (after the session-context trigger below, because the session identity comes first) and, for an out-of-scope task, whenever a skill clearly fits (for example code review). Subagents do not re-pick: the main agent passes them the paths of the skills it picked.
2. **Discover by index only.** List the frontmatter (`name` and `description`) of every `SKILL.md` in the project skill folders (`.ai/skills`, `.agents/skills`, `.agent/skills`, `.claude/skills`, `.codex/skills`, `.cursor/skills`, `.gemini/skills`, `.antigravity/skills`) and in the skill folders of the agent tooling itself (`~/.claude/skills`, `~/.codex/skills`, `~/.agents/skills`, `~/.cursor/skills`, `~/.gemini/skills`, and the skills of installed plugins). De-duplicate by name (project over user level; ignore plugin cache copies and older versions). Do not read a skill body that was not picked.
3. **Pick the relevant skills in three categories** and say "none" for a category with no fit: functional (what the task builds, e.g. UI, layout, responsive design for frontend; folder structure, transactions for backend), non-functional (performance, security, avoiding re-renders, N+1 queries, sound structure), and workflow (skills for the rules in this file that the task will trigger, e.g. documentation update, commit, branch rename, session context, OpenSpec).
4. **Read each picked `SKILL.md` in full**, announce `Using <skill> (<category>) to <purpose>`, and follow it. Skills of another agent are read as plain instructions; use tool-specific ones only when the tool exists.
5. **Evidence:** record the picked skills (name, category, path) in the session context file and list them in the final response.

**Mandatory triggers (hard rules):**

If a skill path named anywhere in this file does not exist or cannot be read, look for a skill with the same `name` in the other skill folders listed in item 2 and use it; only when none exists apply the Degraded Environment Rule.

- Before reading or changing any project file for an implementation task, and before writing any session state for any repository task: read `.ai/skills/session-context/SKILL.md` and follow it. Never create a Session ID, a `.context` file, or an external context file from memory of this summary.
- When the user asks to rename the current Git branch: read `.ai/skills/branch-rename/SKILL.md` and follow it.
- Right before the first source edit (Lifecycle step 5) of an in-scope task that is not a few-line, no-behavior change, whether or not a plan was approved: read `.ai/skills/pre-implementation-checkpoint/SKILL.md` and follow it.
- During any in-scope task and before reporting it done: follow `.ai/skills/progress-tracking/SKILL.md`.
- After an in-scope task that changes behavior, routes, shared code, or architecture: follow `.ai/skills/docs-sync/SKILL.md`.
<!-- end:skill-selection-rule:end -->

# Global Memory Bank Rules

Mandatory for every task and every response. Shared memory root: `/private/tmp/ai-agent/memory`. `projectName` is the Git repository root directory name (or the working directory name when there is no Git root). Never store secrets, credentials, tokens, private keys, or raw sensitive values in memory.

<!-- begin:memory-bank-load-rule:start -->
## Pre-response load

Before task work or composing any answer, and again immediately after every successful context compaction, load all memory files for the project:

1. Run `memory-bank-cli memory_bank_read_all --project <projectName>`. Run no preliminary `list_projects` or `list_project_files` command.
2. On CLI failure, use the `memory-bank` MCP server: `list_project_files`, then concurrent `memory_bank_read` calls.
3. If both fail, read the files directly from `/private/tmp/ai-agent/memory/<projectName>/`.
4. If the project has no memory yet, continue and initialize it during persistence.

Every CLI or MCP operation has a hard 30-second limit. Never retry a failed or timed-out operation in the same turn.
<!-- end:memory-bank-load-rule:end -->

<!-- begin:memory-bank-persistence-rule:start -->
## End-of-turn persistence

The last operation before every final answer persists the durable context: the user's goal and constraints, decisions and rationale, changes made, commands/tests and outcomes, current state, blockers, and next steps.

1. Use `memory-bank-cli memory_bank_update` (existing files) or `memory_bank_write` (new files). Run no preliminary list command.
2. On CLI failure, use the MCP `memory_bank_update` / `memory_bank_write`; only if both fail, write directly under `/private/tmp/ai-agent/memory/<projectName>/`.
3. Keep `activeContext.md` current, append or update `progress.md`, and update other files when their durable facts change.
4. Persist the complete final answer verbatim in `lastResponse.md` (first to last character, all Markdown, code blocks, links, and whitespace preserved; overwrite on every response). Finish drafting the answer first, persist that exact payload, then emit it without further edits. Never replace it with a summary. Before persisting, replace any value covered by the Sensitive Test Accounts and Environment Secrets Policy (credentials, tokens, keys, private endpoints, database URLs, cookies, certificates) with `[redacted]` in the persisted copy only; the answer shown to the user is unchanged.
5. Never claim persistence unless the write succeeded. The same 30-second limit and no-retry rule apply.
<!-- end:memory-bank-persistence-rule:end -->

MCP tools of the `memory-bank` server: `memory_bank_read`, `memory_bank_write`, `memory_bank_update`, `list_projects`, `list_project_files`.

<!-- begin:agent-rule-boundary-rule:start -->
## Agent rule boundaries

Every new rule added to `AGENTS.md` must be wrapped in a uniquely named boundary: `<!-- begin:<rule-name>:start -->` immediately before the rule section and `<!-- end:<rule-name>:end -->` immediately after it.
<!-- end:agent-rule-boundary-rule:end -->

<!-- begin:tooling-rule-protection:start -->
## Tooling rule protection

- Never modify, disable, weaken, remove, or replace tools or configuration that check, lint, test, validate, analyze, format, build, or otherwise enforce code quality.
- Never bypass them in code, configuration, scripts, commands, Git hooks, or implementation structure (suppressions, exclusions, ignored paths, wrapper commands, conditional skips, and equivalents are prohibited).
- Follow the architecture and flow the existing rules enforce. Do not create arbitrary paths, modules, packages, dependencies, or imports only to make an implementation pass. Refactor or correct the implementation instead.
- If compliance is genuinely impossible after the implementation is complete: (1) explain which rule would need to change, why, and the consequences; (2) stash the entire implementation with a descriptive message; (3) stop and wait for the user. If the user rejects the change, reapply the stash and fix the implementation until it complies. If the user approves, reapply the stash and implement the approved rule change together with the original code.
- Never change a protected rule or its tooling without the user's explicit approval through this process.
<!-- end:tooling-rule-protection:end -->

<!-- begin:commit-rule:start -->
## Commit rules

- Write every commit title entirely in English, as one line: `<type>: <title>` (Conventional Commits). No scope, no parentheses, no body, no bullet list, no footer.
- The title must be meaningful and specific enough to explain the change without a description, and be between 15 and 72 characters inclusive.
- Before a commit, run `git branch --show-current`. If the branch is protected (`main`, `master`, `dev`, `release*`) or not named `<type>/<english-kebab-summary>`, propose one valid name (AskUserQuestion; on a protected branch use `git switch -c`, never rename it) and commit only after the user approves.
- This repository has no installed Git hook: nothing runs automatically on commit. Run the quality gate named in "Repository specifics" after the task and again immediately before every commit on the exact tree being committed, then validate the message as described there. Never bypass a hook or gate (no `--no-verify`).
<!-- end:commit-rule:end -->

<!-- begin:degraded-environment-rule:start -->
## Degraded Environment Rule

When a required component is missing or unusable (a skill file, GitNexus, the question tool, subagents, documentation, a test setup, a quality gate): do not skip it silently. (1) Say what is missing. (2) Use the fallback below. (3) If there is no fallback, stop and ask the user. (4) Never claim anything ran or was verified that did not.

- **Skill file missing, unreadable, or at another path:** first look for the same `name` in the skill folders listed in SKILLS item 2; only if none exists apply the minimum here and offer to copy the skill. `session-context`: stop before modifying project files. `pre-implementation-checkpoint`: ask once (AskUserQuestion) about a new branch name (on a protected branch use `git switch -c`, never rename it), about tests only when there is no test setup (skip is Recommended), and about extra subagents only when the task is heavy. `progress-tracking`: tick the OpenSpec `tasks.md` or plan checkbox right after its verification, otherwise keep a todo list in the session file. `docs-sync`: edit only the touched documentation entries and replace outdated text. `branch-rename`: propose one `<type>/<english-kebab-summary>` name and wait for approval; never push.
- **GitNexus unavailable, stale, or UNKNOWN / 0 callers:** use `grep` for callers and blast radius, and state that GitNexus did not answer.
- **Question tool absent:** ask the same question directly, with the Recommended option named.
- **No subagents available:** do the research yourself.
- **Documentation missing, or an entry not found:** read the code directly, say there is no documentation, and propose the entry in the final response.
- **No test setup or gate cannot run:** follow the checkpoint for tests, and report the gate as not run with the reason.
- **Quality gate, documentation map, or Repository specifics absent or incomplete (for example this file was copied into another repository):** infer the gate from `package.json`, `Makefile`, or CI configuration, state the assumption, and ask when two commands are plausible; read the code when there is no documentation map.
- **Not a Git repository, no `origin`, or no Git access:** skip the branch, commit, and `branch-rename` steps, say so, and never invent a baseline.
- **OpenSpec CLI or `openspec/` absent:** use the plan or session-file branch of `progress-tracking`; do not create `openspec/`.
<!-- end:degraded-environment-rule:end -->

# Implementation Workflow Rules

## Scope and Applicability Rule

Classify every request first, before the Session ID, `.context`, or any other lifecycle step.

**In scope** (implementation): creating, modifying, deleting, debugging, refactoring, testing, formatting, linting, building, migrating, or generating code or project-owned technical files; running checks as part of validating such a change; updating project documentation only to match such a change.

**Out of scope** (read-only or advisory): review, audit, analysis, explanation, inspection, planning, brainstorming, estimation, general questions, standalone writing, and running diagnostics only to answer a question.

- Review, inspection, planning, documentation, and test runs become in scope only when the same request also asks for a project change. In a mixed request, apply these rules to the implementation part only.
- When uncertain, treat the task as out of scope. Reading files or `.context` does not make a task in scope.
- Out-of-scope tasks never write to any project `.context` file. Quality gates are not run for them.

## Session and Context Rule

Details live in `.ai/skills/session-context/SKILL.md`. Invariants that always hold:

- One stable Session ID per conversation, read from `AI_SESSION_ID_FILE` (outside every project); never regenerated by a topic, task, branch, retry, compaction, or new message.
- In-scope tasks write session state only to `<project>/.context/session-<SESSION_ID>.md` (one file per Session ID, reused).
- Out-of-scope tasks that need writable state write only to the external `AI_SESSION_CONTEXT_FILE`, never into a project.
- The main agent passes the same Session ID file and context path to every subagent; subagents never create or select another file.
- If the Session ID or context path cannot be established consistently, stop before modifying project files.

## Initial Phase: Project Understanding

Runs only after the task is classified as in scope.

1. Read the complete project `.context` folder, then resolve the session file as described in the session skill.
2. Apply the SKILLS rule above: discover by index, pick the functional, non-functional, and workflow skills, and read the picked ones.
3. Read documentation by routing, not in bulk: follow "Repository specifics" below. Never read the whole `docs/` folder.
4. Use GitNexus to inspect the project, and review all changed code in Git.
5. Understand the repository structure (modules, folders, components, hooks, utilities, types, services) so new work follows it.
6. Use subagents only when they speed up research (codebase structure, docs and GitNexus review, Git changes, quality-gate failures). The default cap is 4 subagents (`N` = 4), and do not spawn the maximum by default. If the task is heavy enough that more research subagents would help, do not exceed `N` on your own: first tell the user the task is heavy and that more subagents cost more tokens, then ask (AskUserQuestion, or a direct question when that tool is absent) with options such as keep 4 (Recommended), raise to a larger number you name, or no subagents. A number the user states in the request replaces the default `N`.

## Subagent Model and Cost Restriction Rule

Subagents may use only Auto mode, a model of the same family at the same or lower tier than the main agent, or another low-cost model. Never spawn a model whose listed input price is greater than USD 3 or whose listed output price is greater than USD 15. If the model identity, tier, or price cannot be verified, use Auto or the lowest-cost model. The main agent coordinates findings, avoids duplicated effort, and owns the final solution.

## Implementation Rule

Implement the requested feature completely, following the existing folder structure. All work is written in English.

- **Scope:** change only what the task needs. Required companions (tests, documentation, callers broken by the change, shared-assets entries) are in scope. Unrelated refactors, renames, cleanups, extra files, and commands with side effects nobody asked for are not: list them under follow-ups, and ask before an out-of-scope edit the task cannot work without.
- **Match the surrounding code:** same naming, comment density, and idiom as the file you edit; read its neighbours first.
- **Ask when it is the user's call:** if a requirement is unclear, or two designs exist and the user should choose, stop and ask with options and a Recommended one (AskUserQuestion, or a direct question when absent). Do not ask what the code or documentation can answer; state the assumption instead.
- **Tests** (when the repository has a test setup and the user did not skip them): add or update tests when behavior changes; for a bug, write a failing test first; run the narrowest relevant test after each unit of work and the full gate at the end. Never edit, delete, or skip an existing test to make it pass. Name untested paths in the final response.
- **Safe by default:** validate input at system boundaries, never log or hardcode secrets, and handle errors with the pattern the repository already uses (see Repository specifics).
- **Completion:** never report a step or the task as done before running its verification; say what ran and what did not.

<!-- begin:reuse-and-deduplication-rule:start -->
## Reuse and Deduplication Rule

- Before adding any component, function, hook, utility, service, or type, check the shared-assets file named in "Repository specifics" and the surrounding code. Use an existing resource whenever one fits.
- Do not create an equivalent of an existing resource. If duplication is found, merge the duplicated piece into one shared implementation by reusing, extending, or composing the existing one; do not redo the whole solution.
- Judge duplication by semantic ownership, not structural similarity: code that looks alike but represents different concepts or owners stays separate; shared fields or behavior of the same concept are merged.
<!-- end:reuse-and-deduplication-rule:end -->

<!-- begin:final-phase-rule:start -->
## Final Phase: Sync, Quality Gates, and Review

1. Force-sync GitNexus (`npx gitnexus analyze`, preserving embeddings as described in the GitNexus section).
2. Run the quality gates named in "Repository specifics". Fix only the side of the project the task belongs to, unless the task requires cross-repository changes.
3. Handle every non-zero exit code by cause:
   - Caused by this task: fix it and rerun all gates until they pass.
   - Pre-existing or unrelated: report it with the evidence and fix it only when it would break CI/CD for this change.
   - Environment or tooling failure: report it; do not change tooling or rules to hide it.
4. Never claim a command passed unless it was run and completed successfully.
5. Review all code written or modified: it follows the existing architecture and folder structure, reuses existing resources, adds no duplicate logic, touches no unrelated files, hardcodes no secrets, and is ready for CI/CD.
<!-- end:final-phase-rule:end -->

## Documentation and Context Sync

After an in-scope task, re-read the documentation entries you changed and the current session context file, then make documentation and session state match the final code (follow the `docs-sync` skill):

- Update only the entries of the docs files named in "Repository specifics" that describe what the task changed. Replace outdated content instead of appending; leave no conflicting old and new descriptions. Add what is missing, correct what differs from the code, delete what no longer exists.
- Keep each documentation file in its existing language and style; do not translate it or mix languages.
- Dated handoff notes and the sprints log are history and are exempt from replacement.
- Write session state and the end-of-task learning summary ("What I Learned Today" and "What I Should Learn or Do Next Time", based only on what actually happened; structure in the session-context skill) to the current `.context/session-<SESSION_ID>.md`, replacing outdated content.

## Final Response Requirement

For an in-scope task report: what was implemented; files changed; skills picked with their category (functional, non-functional, workflow); existing resources reused; duplicate logic merged or avoided; documentation, GitNexus, and session-context files updated; quality gates run, which passed, which failed and how they were fixed; the pre-implementation checkpoint outcome (branch asked or renamed, test decision, subagent decision) or why it was skipped; remaining risks and follow-ups.

For an out-of-scope task report only what applies: what was examined, confirmed findings, limitations and unverified assumptions, recommended next steps, and whether `AI_SESSION_CONTEXT_FILE` was updated. Do not report implementation or gate results that did not happen.

## Sensitive Test Accounts and Environment Secrets Policy

- Inspect for secrets only the committable Git surface: tracked files, files in the current change set, and untracked files not excluded by `.gitignore`, `.git/info/exclude`, global ignore, or sparse-checkout. Check with `git ls-files`, `git status --short`, and `git check-ignore -v -- <path>`. If inclusion cannot be verified, do not read the file for scanning and report that.
- Do not read ignored, excluded, external, or machine-local files (including the external session files) merely to search for secrets, and do not claim a broader check than was done. Reading skill folders outside the repository is skill discovery, not a secrets scan, and is allowed.
- When reviewing a diff, inspect staged and unstaged tracked changes together with non-ignored untracked files that may be committed.
- Never write credentials, tokens, API keys, passwords, private endpoints, database URLs, cookies, certificates, or values derived from `.env` into tracked or committable files. Mention environment variables by name only.
- Use clearly fake placeholders in examples (`example@example.com`, `YOUR_API_KEY_HERE`, `DATABASE_URL_PLACEHOLDER`).

# Repository specifics (be-mycourse)

<!-- begin:repository-specifics-rule:start -->
**Documentation map** (fixed set; add entries, never new files):

| Need | File |
|---|---|
| Layering, bounded contexts, composition root, HTTP request path, security, error codes, quality gate targets | `docs/ARCHITECTURE.md` |
| One entry per module (auth, course, taxonomy, instructor, media, rbac, authorization): rules, flows, `API:` line | `docs/FEATURES.md` |
| Schema, relationships, migration history, permission catalog (appendix) | `docs/DATA-MODEL.md` |
| Reusable packages, functions, types, utilities | `docs/SHARED.md` |
| Features to do, in progress, done (history, not read by default) | `docs/ALL-SPRINTS.md` |

Satellites: `docs/deploy.md`, `docs/docker.md`, `docs/security-public-seo-notes.md`, `docs/observability/`, dated handoff notes. API contract: `docs/api_swagger.yaml` is the hand-edited source; `docs/api-dog-import.json` is generated from it, never edit it by hand.

**Routing:** run `grep -n "^## " docs/FEATURES.md`, read only the entries for the touched modules; read `DATA-MODEL.md` for tables, columns, migrations, or permissions and `ARCHITECTURE.md` only when the task crosses modules. Check `docs/SHARED.md` before adding any package, function, type, or utility.

**Quality gate and commits:** `make check-all` (= `go fmt`, `test-all`, `build-nocgo`, `build`; `test-all` is allowed alone for intermediate runs). Run it after the task and again immediately before every commit on the exact tree being committed. There is no Git hook and no commitlint in this repository, so the title rules in Commit rules are a convention to follow by hand. In a restricted sandbox `make test-all` / `check-all` (golangci-lint cache clean, Go build cache) and `npx gitnexus analyze` can fail with permission errors: run them outside it.

**Implementation notes:** keep the layering (`make check-layout` and `make check-architecture` enforce it); every migration has both up and down; use a transaction for multi-table writes; return errors through the existing error envelope and error codes; authorize through the existing RBAC middleware; follow the Swagger rule below after API changes.

**Size limit:** keep this file under 32,768 bytes (Codex loads only 32 KiB of project instructions). Put long procedures in a skill under `.ai/skills/` and keep only a trigger here.

**Sync targets:** a module change updates its `FEATURES.md` entry; tables, columns, migrations, and permissions update `DATA-MODEL.md`; shared code updates `SHARED.md`; cross-cutting changes update `ARCHITECTURE.md`. On archiving an OpenSpec change, move its line in `ALL-SPRINTS.md` to Done. The four fixed docs files describe only the current state; `migrations/README.md` and dated handoff notes keep their history.
<!-- end:repository-specifics-rule:end -->

<!-- begin:swagger-api-docs-rule:start -->
## Swagger and API Documentation Rule

After modifying any Swagger YAML file run `ruby generate-apidog-postman.rb` (the script lives at `scripts/generate-apidog-postman.rb`), then update the `API:` line of the matching `docs/FEATURES.md` entry so it cites real, current endpoint paths. `docs/api_swagger.yaml` and the generated `docs/api-dog-import.json` are the source of truth for request and response examples; there is no hand-written curl file.
<!-- end:swagger-api-docs-rule:end -->

<!-- begin:dev-test-account-rule:start -->
## AI Agent - dev test account

For local API smoke tests (`http://localhost:8080` after `go run .`) use the shared admin account of the dev database (sysadmin, permissions P1-P68). Read the credentials from your local environment or ask the user; they are never written to a tracked file. Placeholders only:

```bash
curl -sS -X POST 'http://localhost:8080/api/v1/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"email":"DEV_TEST_ACCOUNT_EMAIL","password":"DEV_TEST_ACCOUNT_PASSWORD","remember_me":false}'
# then: -H "Authorization: Bearer $ACCESS_TOKEN"
```
<!-- end:dev-test-account-rule:end -->
