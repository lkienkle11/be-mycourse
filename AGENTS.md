<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **be-mycourse** (7861 symbols, 23180 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

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
Before making any code or documentation changes, read and follow:
1. All relevant skill files under `.ai/skills/`

<!-- begin:repository-memory-bank-rule:start -->
# Global Memory Bank Rules

These rules are mandatory for every task and every response in every repository.

## Memory location and project identity

- The shared memory root is `/private/tmp/ai-agent/memory`.
- Use the current Git repository root directory name as `projectName`. If there is no Git root, use the current working directory name.
- Never store secrets, credentials, tokens, private keys, or raw sensitive values in memory.

## Required pre-response load

Before doing task work or composing any user-facing answer, load all memory files for the current project.

1. Use the installed Memory Bank CLI first and load the project in one command: `memory-bank-cli memory_bank_read_all --project <projectName>`. Do not run preliminary `list_projects` or `list_project_files` commands during the normal path.
2. After agent context compaction completes successfully, immediately reload every Memory Bank file for the current project before continuing task work or composing a response. Use the same CLI-first fallback sequence defined in this section.
3. Every Memory Bank CLI or MCP operation has a hard 30-second limit. Do not retry a failed or timed-out operation in the same turn.
4. If the CLI is unavailable, times out, or fails, fall back to the `memory-bank` MCP server. Call `list_project_files`, then issue the required `memory_bank_read` calls concurrently rather than serially.
5. If both CLI and MCP fail, read every file directly from `/private/tmp/ai-agent/memory/<projectName>/`.
6. If the project has no memory yet, continue the task and initialize it during the required persistence step.

## Required end-of-turn persistence

The final operation before sending each final answer must persist all durable context needed to continue later: the user's goal and constraints, decisions and rationale, changes made, commands/tests and outcomes, current state, blockers, and next steps.

1. Prefer the CLI: use `memory-bank-cli memory_bank_update` for existing files and `memory-bank-cli memory_bank_write` for new files. Do not run preliminary list commands.
2. Every Memory Bank CLI or MCP operation has a hard 30-second limit. Do not retry a failed or timed-out operation in the same turn.
3. If a CLI write fails, fall back to `memory_bank_update` or `memory_bank_write` through MCP.
4. Only if both CLI and MCP fail may you write directly under `/private/tmp/ai-agent/memory/<projectName>/`.
5. Keep `activeContext.md` current and append or update `progress.md`; update other memory files when their durable facts change.
6. Because no tools can run after a final response is emitted, perform this persistence immediately before the final response. Do not claim persistence unless the write succeeded.

## Verbatim final-response persistence

- For every response, persist the complete final answer verbatim in `/private/tmp/ai-agent/memory/<projectName>/lastResponse.md`.
- `lastResponse.md` must contain exactly the final answer shown to the user, from its first character through its last character, preserving all Markdown, code blocks, links, punctuation, whitespace, and line breaks.
- Never replace the verbatim response with a summary, paraphrase, context note, progress report, or selected highlights. Updates to `activeContext.md` and `progress.md` are supplementary and do not satisfy this requirement.
- Before emitting the final answer, first finish drafting it as an immutable payload. Persist that exact payload using the CLI-first fallback chain, then emit the same payload without any further edits. This ordering is required because tools cannot run after the final answer has been emitted.
- Overwrite `lastResponse.md` on every response so it always contains the most recent complete assistant final answer.
- A response is not ready to send until its exact payload has been persisted successfully. Do not claim that the verbatim response was saved unless the write succeeded.

## MCP tool contract

- `memory_bank_read`: read a memory bank file.
- `memory_bank_write`: create a memory bank file.
- `memory_bank_update`: update an existing memory bank file.
- `list_projects`: list available projects.
- `list_project_files`: list files within a project.
<!-- end:repository-memory-bank-rule:end -->

<!-- begin:agent-rule-boundary-rule:start -->
### Agent rule boundaries

- Every new rule added to `AGENTS.md` must be enclosed in a uniquely named boundary using exactly this structure:

  `<!-- begin:<rule-name>:start -->`
  `<!-- end:<rule-name>:end -->`

- The opening marker must appear immediately before the rule section, and the matching closing marker must appear immediately after it.
<!-- end:agent-rule-boundary-rule:end -->

<!-- begin:tooling-rule-protection:start -->
### Tooling rule protection

- Never modify, disable, weaken, remove, or replace tools or configuration used to check, lint, test, validate, analyze, format, build, or otherwise enforce code quality.
- Never bypass these tools or their rules in code, configuration, scripts, commands, Git hooks, or implementation structure. Suppressions, exclusions, ignored paths, wrapper commands, conditional skips, and equivalent workarounds are prohibited.
- Code must closely follow the intended architecture and flow enforced by the existing rules. Do not create arbitrary paths, modules, packages, dependencies, or imports merely to make an implementation pass.
- In most cases, enforcement rules and their tooling must not be changed. Refactor or correct the implementation instead.
- If compliance is genuinely impossible after the implementation has been completed:
  1. Explain clearly to the user which rule would need to change, why the implementation cannot comply, and the consequences of changing it.
  2. Stash the entire implementation with a specific, descriptive stash message.
  3. Stop and wait for the user to decide whether the rule may be changed.
  4. If the user rejects the rule change, reapply the stash and refactor or correct the implementation until it complies. Do not bypass the rule.
  5. If the user explicitly approves the rule change, reapply the stash and implement the approved rule change together with the original code.
- Never change a protected rule or its tooling without the user's explicit approval through this process.
<!-- end:tooling-rule-protection:end -->

<!-- begin:commit-rule:start -->
### Commit rules

- Write every commit title entirely in English.
- Use the one-line Conventional Commit format `<type>: <title>`.
- Do not include a scope or any parentheses between the type and the colon.
- The commit must contain only the title line. Do not add a body, extended description, bullet list, or footer.
- Make the title meaningful and specific enough to explain the change without additional description.
- Keep the complete commit title between 15 and 72 characters inclusive.
- Before every commit, allow the Husky `pre-commit` hook to run `make check-all`; do not bypass the hook with `--no-verify`.
- Allow the Husky `commit-msg` hook to validate the message with commitlint.
<!-- end:commit-rule:end -->

<!-- begin:branch-rename-rule:start -->
### Branch rename rules

Apply this rule only when the user asks to rename the current local Git branch.

- Inspect every local commit that is reachable from `HEAD` but not present on the relevant `origin` baseline. Derive the branch name from the combined intent of all those commits, not only the latest commit.
- Refresh remote-tracking information with `git fetch origin` before comparing when possible. If the fetch fails, use the existing local `origin/*` refs and clearly state that the remote baseline may be stale.
- Resolve the comparison baseline in this order:
  1. The current branch's configured upstream when it belongs to `origin`.
  2. `origin/<current-branch>` when that ref exists.
  3. The default branch referenced by `refs/remotes/origin/HEAD`.
- Record the resolved origin ref and its full commit SHA, then inspect the complete local-only range with `git log --reverse --format='%H%x09%s' <origin-ref>..HEAD`.
- Before proposing or performing a rename, show the current branch, the resolved origin ref and full SHA, and the number and subjects of the local-only commits used to derive the name.
- Generate exactly one branch name for each repository being evaluated. Never provide multiple candidates or alternatives for the same repository.
- When a rename request covers multiple repositories, analyze each repository independently from its own origin baseline and local-only commits. The generated branch names may differ; never force repositories to share a name merely for consistency.
- Use a Commitlint/Conventional Commits action type such as `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build`, `ci`, `chore`, or `revert`.
- Without a ticket, use `<type>/<english-kebab-case-summary>`.
- When the user's request contains `ticket <number>`, use `<type>/#<number>-<english-kebab-case-summary>`.
- Keep the name in English and lowercase. Use hyphens between summary words; do not use spaces or underscores.
- Keep the summary concise, specific, and understandable. Prefer three to seven summary words and keep the complete branch name at no more than 72 characters.
- If there are no local-only commits, derive the name from the user's explicit task description. If neither the commits nor the request provide enough intent for one meaningful name, ask for the missing intent instead of guessing.
- Without `--force`, do not rename the branch immediately. Show the single proposed name and wait for the user's explicit approval.
- When the rename request includes `--force`, treat it as approval to rename the current local branch immediately without a confirmation round.
- Use `git branch -m -- '<new-name>'` for the rename, then verify the result with `git branch --show-current`.
- The user's `--force` flag bypasses confirmation only. It does not authorize overwriting an existing local branch, pushing the renamed branch, deleting a remote branch, or changing upstream configuration.
- Never push, delete or rename a remote branch, or modify upstream tracking unless the user explicitly requests that separate action.
<!-- end:branch-rename-rule:end -->

## AI Agent — dev test account

Use this shared admin account for local API smoke tests (`http://localhost:8080` after `go run .`):

| Field | Value |
|-------|--------|
| Email | `DEV_TEST_ACCOUNT_EMAIL` |
| Password | `DEV_TEST_ACCOUNT_PASSWORD` |
| Role | sysadmin/admin (full P1–P58 on dev DB) |

```bash
# Login → capture access_token
curl -sS -X POST 'http://localhost:8080/api/v1/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"email":"DEV_TEST_ACCOUNT_EMAIL","password":"DEV_TEST_ACCOUNT_PASSWORD","remember_me":false}'

# Example: instructor expertise (user id 14)
curl -sS 'http://localhost:8080/api/v1/instructors/14/expertise/topics' \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```


# Implementation Workflow Rules

## Scope and Applicability Rule

These rules apply only to **implementation tasks that require creating, modifying, deleting, debugging, refactoring, testing, formatting, linting, building, migrating, generating, or otherwise changing code or project-owned technical files inside a software project, repository, workspace, or working tree**.

An in-scope code task includes, but is not limited to:

* Writing, implementing, changing, fixing, deleting, or refactoring source code.
* Debugging project code, configuration, builds, tests, runtime behavior, integrations, or CI/CD failures when the task requires changing project files.
* Creating, updating, or deleting tests, fixtures, mocks, schemas, migrations, generators, scripts, or project configuration.
* Running formatters, linters, tests, builds, migrations, generators, or CI/CD checks as part of implementing, validating, or fixing an actual project change.
* Creating or modifying project-owned documentation only when it is required to describe, support, or synchronize with code or technical project files changed by the current implementation task.
* Producing an implementation plan only when the user asks the agent to proceed with an actual code or project-file change in the same task.

These rules do **not** apply to read-only, advisory, explanatory, review-only, or non-implementation tasks. Out-of-scope tasks include, but are not limited to:

* Reviewing code without being asked to modify, fix, refactor, test, or otherwise change project files.
* Reading or inspecting a repository only to explain its implementation, architecture, dependencies, Git state, runtime behavior, code quality, or possible improvements.
* Auditing, analyzing, or commenting on code, pull requests, diffs, commits, designs, architecture, security, performance, accessibility, or maintainability when no project change is requested.
* Answering general programming, product, infrastructure, architecture, or technology questions, including questions that use code snippets supplied in the conversation.
* Creating examples, pseudocode, isolated snippets, interview answers, tutorials, explanations, or demonstrations that are not being applied to an existing project.
* General planning, brainstorming, estimation, roadmaps, task breakdowns, or architecture discussions, even when they concern a software project, unless the same task explicitly requires implementing a project change.
* Writing standalone documents outside the project, such as external proposals, reports, essays, emails, presentations, policies, product descriptions, meeting notes, or generic technical documentation.
* Rewriting, translating, summarizing, proofreading, or formatting user-provided text when the task does not modify code or project-owned technical files.
* Running commands only to inspect information, collect diagnostics, review status, or answer a question when no implementation, fix, refactor, or test change is requested.

For every request, the agent must first classify the task before invoking the Session ID or `.context` lifecycle:

1. If the task is out of scope, the agent may read existing project `.context` files when they are useful for review, explanation, analysis, or project understanding. It must not create, modify, rename, delete, or append to any `.context` file inside the project for that out-of-scope task. If any writable session state is required for the out-of-scope repository-related task, regardless of the task type or the kind of information being persisted, the agent must establish or reuse the conversation Session ID and write only to an external machine-local or temporary context directory outside every project. Do not run implementation quality gates merely because this rule file has `alwaysApply: true`. Complete the task normally.
2. If the task explicitly requires creating, modifying, deleting, debugging, refactoring, or testing code or project-owned technical files, apply every relevant rule in this file before interacting with project content.
3. If one request contains both in-scope implementation work and out-of-scope review, explanation, planning, or standalone writing, apply these rules only to the implementation portion.
4. Code review is not automatically an in-scope task. It becomes in scope only when the user also asks the agent to implement fixes, modify files, refactor code, add or update tests, or perform another project change based on the review.
5. Repository inspection is not automatically an in-scope task. It becomes in scope only when inspection is necessary to carry out an explicitly requested project change.
6. A plan is not automatically an in-scope task. It becomes in scope only when the same request includes or directly authorizes implementation of code or project-file changes.
7. Documentation is not automatically an in-scope task. It becomes in scope only when the document is a project-owned technical file that must be created or changed as part of the current implementation task.
8. Running tests, linters, formatters, builds, or diagnostics is not automatically in scope. It becomes in scope only when performed as part of implementing, validating, debugging, refactoring, or fixing an actual project change.
9. When classification is uncertain, default to out of scope unless the user clearly requests a change to code or project-owned technical files. Reading repository files or project `.context` files does not by itself make the task in scope. If any session state must be written for such a task, use only the external context lifecycle defined below.

## Critical Pre-Conversation Session Bootstrap Rule

For an in-scope implementation task, before the agent reads or changes project files for the purpose of implementing, debugging, refactoring, testing, or validating the requested project change, invokes a project subagent, or runs a project command for that implementation work, the conversation host or bootstrap process must establish one stable Session ID for the entire conversation.

This Session ID must be created outside every project, repository, workspace, Desktop project folder, and working tree. It must never be written into a project-level temporary file, source file, configuration file, `.context` file, `.env` file, documentation file, Git metadata file, or any other file located inside a project.

The canonical external temporary location must be resolved using the current operating system's temporary directory:

```text
<OS_TEMP_DIR>/ai-agent-sessions/<conversation-bootstrap-key>/session-id
```

Resolve `<OS_TEMP_DIR>` as follows:

* Windows: use `%TEMP%`, then `%TMP%`, then the platform API for the current user's temporary directory.
* macOS and Linux: use `$TMPDIR`, then `/tmp`.
* Other operating systems: use the runtime or platform API that returns the current user's temporary directory.

The resolved directory must be user-writable and outside every project, repository, workspace, and working tree.

The exact resolved path must be stored in the process-level environment variable:

```text
AI_SESSION_ID_FILE
```

The bootstrap process must follow this exact lifecycle:

1. Before the first task action of a new conversation, create one unpredictable conversation bootstrap key outside all projects.
2. Resolve `AI_SESSION_ID_FILE` to `<OS_TEMP_DIR>/ai-agent-sessions/<conversation-bootstrap-key>/session-id` using the operating-system-specific temporary directory rules above.
3. Create the parent directory outside all projects with permissions restricted to the current user.
4. If the file does not exist, generate exactly one stable Session ID and write it to that file atomically.
5. If the file already exists, read and reuse the existing Session ID. Never overwrite it with a different value during the same conversation.
6. Keep `AI_SESSION_ID_FILE` unchanged and reuse the Session ID from that exact file for every later message, task, subtask, tool call, and subagent in the same conversation.
7. A topic change, repository change, branch change, task change, date change, tool restart, retry, compaction, or new user message does not create a new Session ID.
8. A new Session ID may be created only for a genuinely new conversation or when the user explicitly orders a Session ID reset.

A recommended POSIX shell bootstrap for macOS and Linux is:

```bash
set -eu

: "${TMPDIR:=/tmp}"
: "${AI_CONVERSATION_BOOTSTRAP_KEY:?AI_CONVERSATION_BOOTSTRAP_KEY must be created by the conversation host}"

export AI_SESSION_ID_FILE="${TMPDIR%/}/ai-agent-sessions/${AI_CONVERSATION_BOOTSTRAP_KEY}/session-id"
mkdir -p "$(dirname "$AI_SESSION_ID_FILE")"
chmod 700 "$(dirname "$AI_SESSION_ID_FILE")"

if [ ! -s "$AI_SESSION_ID_FILE" ]; then
  umask 077
  temporary_file="${AI_SESSION_ID_FILE}.tmp.$$"
  uuidgen | tr '[:upper:]' '[:lower:]' > "$temporary_file"
  mv "$temporary_file" "$AI_SESSION_ID_FILE"
fi

export AI_SESSION_ID="$(tr -d '\r\n' < "$AI_SESSION_ID_FILE")"
```

A recommended PowerShell bootstrap for Windows is:

```powershell
$ErrorActionPreference = "Stop"

if (-not $env:AI_CONVERSATION_BOOTSTRAP_KEY) {
    throw "AI_CONVERSATION_BOOTSTRAP_KEY must be created by the conversation host"
}

$tempRoot = [System.IO.Path]::GetTempPath()
$sessionDirectory = Join-Path $tempRoot "ai-agent-sessions\$($env:AI_CONVERSATION_BOOTSTRAP_KEY)"
$env:AI_SESSION_ID_FILE = Join-Path $sessionDirectory "session-id"

[System.IO.Directory]::CreateDirectory($sessionDirectory) | Out-Null

if (-not (Test-Path -LiteralPath $env:AI_SESSION_ID_FILE) -or
    (Get-Item -LiteralPath $env:AI_SESSION_ID_FILE).Length -eq 0) {
    $temporaryFile = "$($env:AI_SESSION_ID_FILE).tmp.$PID"
    ([guid]::NewGuid().ToString().ToLowerInvariant()) |
        Set-Content -LiteralPath $temporaryFile -NoNewline -Encoding utf8
    Move-Item -LiteralPath $temporaryFile -Destination $env:AI_SESSION_ID_FILE -Force
}

$env:AI_SESSION_ID = (Get-Content -LiteralPath $env:AI_SESSION_ID_FILE -Raw).Trim()
```

On Windows, the bootstrap must use Windows path APIs such as `Join-Path` or `System.IO.Path`. It must not construct Windows paths by assuming POSIX separators.

The agent must validate all of the following before continuing:

* `AI_SESSION_ID_FILE` is defined.
* The resolved path is outside every project, repository, workspace, and working tree involved in the task.
* The file exists and contains exactly one non-empty stable Session ID.
* `AI_SESSION_ID`, when provided, matches the value stored in `AI_SESSION_ID_FILE`.
* The same resolved path and Session ID are supplied to every subagent.

The agent must never generate a replacement Session ID merely because `AI_SESSION_ID` is missing from the current shell. It must first read the value from `AI_SESSION_ID_FILE`.

The agent must never search project directories for the external Session ID file. It must read only the exact path stored in `AI_SESSION_ID_FILE`.

If `AI_SESSION_ID_FILE` is missing, empty, inaccessible, located inside a project, points to a different Session ID than the one already established for the conversation, or cannot be passed consistently to subagents, the agent must stop before modifying project files. It must repair the external bootstrap state when possible. It must not guess, silently create a project-local fallback, reuse an unrelated `.context` file, or create multiple session context files.

The external temporary Session ID file is the authoritative identity source for the conversation.

For in-scope implementation tasks, the writable session context must be the current project's `.context/session-<SESSION_ID>.md`.

For any out-of-scope repository-related task that requires writable session state, the writable session context must remain outside every project in the external context location defined below. This includes, but is not limited to, review, audit, analysis, explanation, planning, advisory, inspection, research, investigation, comparison, documentation review, architecture review, security review, performance review, accessibility review, maintainability review, pull-request review, diff review, commit review, dependency review, configuration review, and repository-status inspection.

## Critical Agent and Subagent Session Propagation Rule

The main agent owns the conversation Session ID and must propagate the same identity to every subagent.

Before a subagent reads or changes any project content, the main agent must provide it with:

* The exact `AI_SESSION_ID_FILE` path.
* The exact `AI_SESSION_ID` value read from that file.
* The exact current session context file path, whether it is the in-project coding context file or the external read-only-task context file.
* An explicit prohibition against generating a new Session ID, creating another session context file, or writing to any session context file other than the exact path supplied by the main agent.

Every subagent must verify that its Session ID matches the value in `AI_SESSION_ID_FILE`. A subagent must never derive a Session ID from timestamps, task names, repository names, branch names, process IDs, message text, random values, or existing `.context` filenames.

Subagents may read project `.context` files as references and may read the current session context file when needed. They must not independently create, rename, replace, or select a session context file. Unless the main agent explicitly delegates a specific context edit, only the main agent may write the current session context file.

When a delegated context edit is allowed, the subagent may edit only the exact current session context file supplied by the main agent. Concurrent writes are forbidden. The main agent must serialize context updates and remain responsible for the final consolidated content.

Any result returned by a subagent that used a different Session ID, created a second session context file, selected a context file by guessing, or wrote to another session's context file must be rejected and redone using the correct Session ID and exact context path.

## Critical First Action: Read Existing Project Context

For any task involving an existing repository, including implementation, review-only, explanation, analysis, debugging, refactoring, or testing, the agent may read the complete existing project `.context` folder when it is available and relevant.

The purpose of this read is to understand prior decisions, conventions, failures, constraints, unresolved issues, and reusable project knowledge.

For an in-scope implementation task:

* The agent must read the complete existing project `.context` folder before repository-based planning, modification, testing, or implementation.
* After the full read, the agent must resolve exactly one writable project session context file for the current Session ID:

```text
<project>/.context/session-<SESSION_ID>.md
```

* If that exact file already exists, reuse and update it.
* If it does not exist, create it exactly once after confirming that no matching file exists.
* Do not create a second project session context file for the same Session ID.

For review-only, explanation-only, analysis-only, planning-only, audit-only, or other out-of-scope repository tasks:

* Existing project `.context` files may be read as references.
* Project `.context` files must remain read-only.
* Do not create or update `<project>/.context/session-<SESSION_ID>.md` for the out-of-scope task.
* If any writable session state is required for the task, use the external session context lifecycle below, regardless of what kind of repository-related work is being performed or what information must be persisted.

## Critical Dual Session Context Lifecycle Rule

The writable session context location depends on the classified task.

### In-Scope Coding and Implementation Tasks

For each affected project, use exactly one project session context file for the current Session ID:

```text
<project>/.context/session-<SESSION_ID>.md
```

Lifecycle:

1. Read the Session ID from `AI_SESSION_ID_FILE`.
2. Read the complete existing project `.context` folder.
3. Search for files associated with the exact complete Session ID.
4. If exactly one matching file exists, reuse it.
5. If no matching file exists, create exactly one file named `session-<SESSION_ID>.md`.
6. If multiple matching files exist, do not create another file; resolve the duplication before continuing.
7. Reuse the exact resolved path for all later coding messages, implementation subtasks, quality gates, documentation updates, learning summaries, agents, and subagents for that project in the same chat.
8. Pass the exact project session context path to every participating subagent.

Forbidden for in-scope coding tasks:

* Creating a new context file for each message, task, branch, feature, date, or retry.
* Using a context file owned by another Session ID.
* Selecting a context file by filename similarity, date, size, or semantic guess.
* Writing coding-task session history to an external context file instead of the required project session context file.
* Allowing a subagent to create or select another context file.

### All Out-of-Scope Repository-Related Tasks

If any writable session state is required for any out-of-scope repository-related task, use exactly one external context file outside every project:

```text
<OS_TEMP_DIR>/ai-agent-sessions/<conversation-bootstrap-key>/context/session-<SESSION_ID>.md
```

A persistent machine-local directory may be used instead:

```text
Windows: %LOCALAPPDATA%\ai-agent-sessions\<conversation-bootstrap-key>\context\session-<SESSION_ID>.md
macOS:   $HOME/Library/Application Support/ai-agent-sessions/<conversation-bootstrap-key>/context/session-<SESSION_ID>.md
Linux:   ${XDG_STATE_HOME:-$HOME/.local/state}/ai-agent-sessions/<conversation-bootstrap-key>/context/session-<SESSION_ID>.md
```

Store the resolved external paths in:

```text
AI_SESSION_CONTEXT_DIR
AI_SESSION_CONTEXT_FILE
```

Lifecycle:

1. Create or reuse the conversation Session ID whenever any writable external session state is required for the out-of-scope repository-related task.
2. Resolve an external directory outside every project and working tree.
3. Create exactly one external file named `session-<SESSION_ID>.md`.
4. Reuse that exact file for later out-of-scope repository tasks in the same chat.
5. Do not write state from any out-of-scope repository-related task into any project `.context` file.

Before the final response of an in-scope implementation task, verify:

* The Session ID still matches `AI_SESSION_ID_FILE`.
* Exactly one `<project>/.context/session-<SESSION_ID>.md` exists for each affected project.
* No external context file was used as the coding-task session record.
* No context file was created for a different or guessed Session ID.
* No subagent created or modified an unauthorized context file.

Before the final response of any out-of-scope repository-related task:

* If no writable external session state was required and `AI_SESSION_CONTEXT_FILE` was not created, do not create it merely for finalization.
* If `AI_SESSION_CONTEXT_FILE` already exists or was required during the task, update that exact external file with the final consolidated state before responding.
* Record the task purpose, repository or project examined, files or areas reviewed, relevant findings, decisions, unresolved questions, risks, limitations, and recommended follow-up actions.
* Include only facts established during the task. Do not invent files read, checks run, findings, conclusions, or completed work.
* Replace outdated or contradictory content in the external context instead of appending an unstructured transcript.
* Reuse the exact same external context file for later out-of-scope repository tasks in the same chat.
* Do not write this information into any project `.context` file.
* Verify that `AI_SESSION_CONTEXT_FILE` remains outside every project, repository, workspace, and working tree.
* Verify that the Session ID still matches `AI_SESSION_ID_FILE`.
* Verify that no second external context file or Session ID was created by the main agent or a subagent.

## Initial Phase: Project Understanding

This phase runs only after the task has already been classified as an in-scope implementation task. Its repository-reading requirements must never be used to pull a review-only, explanation-only, planning-only, or advisory request into scope.

After reading the complete existing project `.context` folder and resolving the current project session context file, the agent must thoroughly read all project documentation and all project skills inside `.ai/skills` to understand the full project scope, conventions, reusable workflows, and task-specific instructions.

If `.ai/skills` exists, the agent must inspect every relevant skill before planning or implementing any change.

Use GitNexus to work faster and to inspect the project context.

The agent must also review all changed code in Git to understand exactly what has been modified, then update the documentation accordingly.

Use subagents when they help speed up research, codebase exploration, documentation review, or Git change inspection.

The agent may spawn up to 4 subagents during the research and understanding phase, but must not spawn the maximum number by default. Spawn only the number of subagents that are actually needed for the task.

Subagents must be used purposefully. For example:

* One subagent may inspect the backend codebase structure.
* One subagent may inspect the frontend codebase structure.
* One subagent may review `.context`, documentation, and GitNexus files.
* One subagent may inspect Git changes, modified files, quality-gate failures, or CI/CD-related issues.

## Subagent Model and Cost Restriction Rule

Subagents may use only:

* Auto mode;
* A model in the same model family and at the same or lower capability tier than the main agent;
* Or another low-cost model.

For example:

* If the main agent is running in Cursor Auto mode, subagents must use Auto mode or a lower-cost option.
* If the main agent is GPT-5.6, subagents may use GPT-5.6, GPT-5.5, GPT-5.4, or a lower-tier model.

The agent must not spawn any model whose listed price exceeds either of these limits:

* Input price greater than USD 3.
* Output price greater than USD 15.

If model identity, capability tier, or pricing cannot be verified, the agent must use Auto mode or the lowest-cost available model instead.

The main agent remains responsible for coordinating all subagent findings, avoiding duplicated effort, deciding the final implementation approach, and ensuring the final solution is correct.

For `be-mycourse` tasks:

* Read the `be-mycourse` repository carefully.
* Understand the project structure.
* Identify where each module is located.
* Identify where relevant code, folders, utilities, types, and services are placed.
* Make sure any new work follows the existing structure.

For `fe-mycourse` tasks:

* Read the `fe-mycourse` repository carefully.
* Understand the project structure.
* Identify where each module is located.
* Identify where relevant code, folders, components, hooks, utilities, types, and services are placed.
* Make sure any new work follows the existing structure.

## Middle Phase: Implementation

Implement the requested feature completely.

The implementation must follow the existing folder structure and reuse existing utility functions, components, types, services, hooks, helpers, and shared logic whenever possible.

If a required utility does not exist, create a new reusable utility and place it in the correct folder so that future phases can reuse it.

All work must be written in English.

## Critical Reuse Rule

When implementing a solution, the agent must use existing resources, code, components, functions, methods, utilities, services, hooks, and types whenever they already exist in the codebase.

The agent is not allowed to create a new resource, code block, component, function, method, utility, service, hook, or type if an equivalent one already exists.

If duplicate implementation is found, the entire solution must be rejected, and the agent must read the project again and redo the work from the beginning.

## Critical Deduplication Rule

If a type, function, method, component, utility, service, hook, or code block from feature A already exists, and feature B duplicates the same logic or structure, the agent must merge the duplicated logic into a shared implementation.

If multiple types or functions share common fields or behavior, the agent must reuse, extend, compose, or inherit from the existing implementation instead of duplicating it.

Do not leave duplicated types, duplicated functions, duplicated methods, duplicated components, duplicated utilities, or duplicated business logic in the codebase.

If duplication is found during review, the solution must be rejected and redone from the beginning.

## Final Phase: Sync, Quality Gates, Review, and Documentation

At the end of an in-scope code task, the agent must force-sync GitNexus when GitNexus exists or is required by the project.

Then run the required quality gates based on the side of the project being changed.

For Backend tasks:

* Run formatting.
* Run linting.
* Run `make test-all`.
* Run `make check-all`.
* Run tests.
* Run build checks.
* Make sure the backend CI/CD pipeline will not break.

For Frontend tasks:

* Run `npm run check-all`.
* Fix any errors.
* Make sure the frontend CI/CD pipeline will not break.

If the task is Backend-related, fix Backend issues only.

If the task is Frontend-related, fix Frontend issues only.

Do not randomly fix the opposite side unless the task explicitly requires cross-repository changes.

## Exit Code Rule

If any command exits with a non-zero status code, the agent must fix the errors gradually until all issues are resolved.

This includes errors that are not directly related to the current task if they can break CI/CD.

After fixing the errors, the agent must rerun all required quality gates until everything passes.

The agent must not claim that a command passed unless it was actually run and completed successfully.

## Post-Implementation Code Review Requirement

After implementation, the agent must inspect and review all code written or modified during the current task. This section applies only after an in-scope implementation task and must not be used to classify a standalone review-only request as in scope.

The review must verify:

* The implementation follows the existing architecture.
* The implementation follows the existing folder structure.
* Existing utilities, components, functions, services, hooks, and types were reused where possible.
* No duplicate logic was introduced.
* No unrelated files were modified.
* No secrets or credentials were hardcoded.
* The code is ready for CI/CD.

## Documentation and Context Sync

After completing an in-scope implementation, the agent must fully sync, update, or delete project documentation and GitNexus files so that they match the latest source code exactly. Session-specific state and learning for the coding task must be written only to the current project's `.context/session-<SESSION_ID>.md`. External `AI_SESSION_CONTEXT_FILE` is reserved for review-only or other out-of-scope repository tasks that require writable state.

Updating documentation means:

* Scan the entire source code.
* If something exists in code or features but is missing from documentation, add it to the relevant documentation.
* If code and features are implemented one way but the documentation describes a different approach, update the documentation to match the code.
* If something is documented but does not exist in the code or implemented features, remove it from the documentation.
* Update only the current project's `.context/session-<SESSION_ID>.md` for coding-task session state and learning.
* Update all relevant GitNexus files.

For `be-mycourse` tasks, update GitNexus files inside the `be-mycourse` project.

For `fe-mycourse` tasks, update GitNexus files inside the `fe-mycourse` project.

## Swagger and API Documentation Rule

If any Swagger YAML file is modified, the agent must also run:

```bash
ruby generate-apidog-postman.rb
```

After running the generator, the agent must update all related API documentation and GitNexus files to match the generated output and the latest source code. Update only the current project's `.context/session-<SESSION_ID>.md` for coding-task session state.

## Mandatory External Context Update for Read-Only Repository Tasks

This section applies to every repository-related task that does not modify project-owned files. This includes, but is not limited to, review, audit, analysis, explanation, planning, advisory work, inspection, research, investigation, comparison, documentation review, architecture review, security review, performance review, accessibility review, maintainability review, pull-request review, diff review, commit review, dependency review, configuration review, Git-state inspection, test-result inspection, build-result inspection, CI/CD inspection, and other read-only or non-implementation repository work.

If the task did not require writable state and no external context file was created, do not create one only to satisfy this section.

If `AI_SESSION_CONTEXT_FILE` exists or writable external state was required, the main agent must update that exact file before the final response.

The update must consolidate the complete current state and include, when applicable:

### Task Scope

* What the user asked the agent to inspect, review, audit, explain, plan, or analyze.
* Which repository, project, branch, diff, commit, pull request, modules, files, or subsystems were actually examined.
* Which areas were intentionally excluded or could not be inspected.

### Findings and Evidence

* Confirmed findings and the evidence supporting them.
* Code-quality, architecture, security, performance, accessibility, maintainability, testing, documentation, integration, or operational observations relevant to the task.
* Findings that were dismissed after investigation and why.
* Uncertain findings that still require verification.

### Decisions and Recommendations

* Conclusions reached during the task.
* Recommendations provided to the user.
* Trade-offs, assumptions, constraints, and reasons behind those recommendations.
* Priority and expected impact when the task involves multiple findings.

### Unresolved Items and Next Steps

* Remaining risks, unknowns, missing evidence, incomplete coverage, or blocked checks.
* Files, commands, environments, people, systems, or information required for further verification.
* Specific follow-up actions that should be performed next.

The external context update must be based only on what actually happened during the task.

The agent must not claim that it reviewed files, ran commands, verified behavior, found vulnerabilities, or completed checks that it did not actually perform.

Only the exact external file referenced by `AI_SESSION_CONTEXT_FILE` may receive this out-of-scope repository-task summary.

Do not create or update `<project>/.context/session-<SESSION_ID>.md` for any out-of-scope repository-related task.

Subagents may contribute findings, but only the main agent may perform the final consolidated external context update unless a serialized edit to the exact `AI_SESSION_CONTEXT_FILE` was explicitly delegated.


## Mandatory End-of-Task Learning Summary

After an in-scope code task reaches implementation, review, quality-gate, GitNexus, and project-documentation completion, the agent must update the current project's `.context/session-<SESSION_ID>.md`.

The update must summarize the complete final state of the session and must include at least these sections:

### What I Learned Today

Record concise, concrete, reusable lessons learned during the current task, including:

* Newly discovered project architecture, conventions, dependencies, or workflows.
* Important implementation decisions and the reasons behind them.
* Existing resources that were successfully reused.
* Mistakes, failed approaches, command failures, incorrect assumptions, or duplicated work that should not be repeated.
* Quality-gate, CI/CD, documentation, GitNexus, security, or integration lessons.
* Any user corrections or preferences that materially affect future work in this project.

### What I Should Learn or Do Next Time

Record specific improvements and follow-up guidance for the next task, including:

* Unresolved risks, technical debt, missing coverage, or follow-up actions.
* Files, modules, documentation, skills, or commands that should be inspected earlier next time.
* Better investigation, reuse, implementation, testing, or review strategies.
* Project-specific traps and assumptions that must be verified instead of guessed.

The learning summary must be based on what actually happened during the task. Do not invent completed work, successful commands, failures, decisions, or lessons.

Only the current project's `.context/session-<SESSION_ID>.md` may receive the coding-task learning summary. If it does not yet exist, create it exactly once according to the in-scope coding context lifecycle. Reuse the same file for later coding messages in the same chat. Creating another coding-task context file for the same Session ID is forbidden without explicit user permission.

## Final Response Requirement

At the end of an in-scope code task, the agent must report:

* What was implemented.
* Which files were changed.
* Which existing resources were reused.
* Which duplicate logic was merged or avoided.
* Which documentation and GitNexus files were updated, and which `<project>/.context/session-<SESSION_ID>.md` file was used for the coding task.
* Which quality gates were run.
* Which commands passed.
* Which commands failed and how they were fixed.
* Any remaining risks or follow-up actions.

At the end of any out-of-scope repository-related task, the agent must report only what is relevant to the user's request, including when applicable:

* What was reviewed, inspected, audited, explained, planned, or analyzed.
* The main confirmed findings.
* Important limitations, unverified assumptions, or areas not inspected.
* Recommended actions or next steps.
* Whether an external `AI_SESSION_CONTEXT_FILE` was updated.

Do not report implementation, changed files, quality-gate success, or project `.context` updates for an out-of-scope task unless those actions actually occurred.


## Full Documentation Replacement Rule

After completing any in-scope code task, the agent must re-read all relevant project documentation, the complete project `.context` folder, the current project's `.context/session-<SESSION_ID>.md`, GitNexus files, and project documents before updating documentation and coding-task session state.

The agent must not simply append new notes on top of old documentation.

Instead, the agent must replace all outdated update content with new, accurate content that reflects the entire final state of the codebase after all changes have been made.

This means:

* Remove old update notes that no longer match the current code.
* Replace outdated implementation descriptions with the latest implementation details.
* Rewrite affected documentation sections so they describe the full current behavior, not only the latest diff.
* Make sure the documentation reflects all changed code together as one consistent final state.
* Do not leave conflicting old and new documentation in the same file.
* Do not preserve historical update content unless the document is explicitly meant to be a changelog or migration history.
* If the current project's `.context/session-<SESSION_ID>.md` or a GitNexus file describes a previous implementation, replace the outdated description with the new implementation summary. Do not modify context files owned by other Session IDs.
* If multiple files describe the same feature, update all of them consistently.
* The final documentation must match the source code exactly after the task is complete.

## Documentation Language Consistency Rule

When updating documentation, the agent must preserve and follow the existing language of each documentation file.

For every file inside `docs` and every `README.md` file:

* If the existing document is written in English, update it in English.
* If the existing document is written in Japanese, update it in Japanese.
* If the existing document is written in Vietnamese, update it in Vietnamese.
* If the existing document is written in Chinese, update it in Chinese.
* If the document uses another language, continue using that same language.
* Do not mix languages inside the same document unless the document already intentionally uses multiple languages.
* If a document is multilingual by design, update each section using the language already used by that section.
* Do not translate the entire document unless the task explicitly asks for translation.
* New documentation content must match the existing tone, terminology, formatting style, and language of the target file.

## Sensitive Test Accounts and Environment Secrets Policy

Security and secret-exposure checks apply only to files that belong to the repository's committable Git surface.

Before scanning a file for credentials, secrets, sensitive values, or environment-derived information, the agent must determine whether that file is tracked by Git or can be included in a commit.

A file is in scope for security inspection only when at least one of the following is true:

* The file is already tracked by Git.
* The file is staged, modified, renamed, copied, deleted, or otherwise included in the current Git change set.
* The file is untracked but is not excluded by `.gitignore`, `.git/info/exclude`, global Git ignore rules, sparse-checkout rules, or another active Git exclusion mechanism.
* The file would be included by a normal `git add` operation from the relevant repository scope.

The agent should use Git-aware commands when available, including:

```bash
git ls-files
git status --short
git check-ignore -q -- <path>
git check-ignore -v -- <path>
```

Equivalent platform-specific commands or Git APIs may be used when they provide the same determination.

The following files are out of scope for security inspection unless the user explicitly asks for them to be checked:

* Files ignored by `.gitignore`.
* Files ignored through `.git/info/exclude`.
* Files ignored through global Git ignore configuration.
* Files excluded by sparse-checkout or an equivalent Git mechanism.
* Files outside the current repository or working tree.
* Machine-local files that cannot be committed to the current repository.
* External session files stored under `AI_SESSION_ID_FILE`, `AI_SESSION_CONTEXT_DIR`, or `AI_SESSION_CONTEXT_FILE`.
* Temporary files, caches, generated local artifacts, dependency directories, editor state, and local environment files when Git excludes them.

The agent must not recursively inspect ignored, excluded, external, or machine-local files merely to search for secrets. The existence of those files does not authorize reading their contents.

If Git tracking or ignore status cannot be determined reliably, the agent must not read that file for security scanning. It must report that the file's Git inclusion status could not be verified.

For files inside the committable Git surface:

* Do not write, generate, commit, or document any test account credentials in tracked or committable documentation files, test files, source files, configuration files, examples, fixtures, `README.md`, or other project-owned files.
* Do not write, generate, commit, or document values derived from `.env` keys or environment secrets in tracked or committable project files.
* Never expose real credentials, access tokens, API keys, secret keys, passwords, private endpoints, database URLs, authentication cookies, private certificates, or other sensitive values in tracked or committable docs, tests, examples, comments, fixtures, configuration, source code, or project documentation.
* Inspect only the relevant tracked or committable files changed by the task, unless the user explicitly requests a broader repository security audit.
* When reviewing a diff, inspect staged and unstaged tracked changes together with non-ignored untracked files that may be committed.
* Do not claim that the entire machine, workspace, ignored file set, or excluded file set was checked when only the committable Git surface was inspected.

When examples are required, use clearly fake placeholders only, such as:

* `example@example.com`
* `test-user-placeholder`
* `YOUR_API_KEY_HERE`
* `YOUR_SECRET_HERE`
* `DATABASE_URL_PLACEHOLDER`

If a task requires referencing environment variables, mention only the variable names, not their actual values.
