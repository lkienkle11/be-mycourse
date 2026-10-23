---
name: session-context
description: Establish and reuse the one stable conversation Session ID and the single session context file (project .context/session-<SESSION_ID>.md for in-scope implementation tasks, external AI_SESSION_CONTEXT_FILE for out-of-scope tasks), pass them unchanged to subagents, update the external context for read-only tasks, and write the end-of-task learning summary. Use before reading or changing any project file for an implementation task, before writing any session state for any repository task, and whenever delegating work to a subagent.
---

# Session and Context

This skill holds the full session-identity and context-file procedure that `AGENTS.md` summarizes in "Session and Context Rule". The text below is the authoritative procedure; `AGENTS.md` and the user's instructions win if they conflict.

## Critical Pre-Conversation Session Bootstrap Rule

For an in-scope implementation task, before the agent reads or changes project files for the purpose of implementing, debugging, refactoring, testing, or validating the requested project change, invokes a project subagent, or runs a project command for that implementation work, the conversation host or bootstrap process must establish one stable Session ID for the entire conversation.

This Session ID must be created outside every project, repository, workspace, Desktop project folder, and working tree. It must never be written into a project-level temporary file, source file, configuration file, `.context` file, `.env` file, documentation file, Git metadata file, or any other file located inside a project.

**Prefer the native runtime session ID as the seed value.** When the agent's own runtime already exposes a stable per-conversation session identifier (for example, Claude Code's `CLAUDE_CODE_SESSION_ID` environment variable), use that value as the Session ID written into `AI_SESSION_ID_FILE` at bootstrap step 4 below, instead of generating a random UUID — it is already stable for the conversation, so there is nothing left to generate. Everything else in this section (the file as the authoritative identity source, the validation checklist, subagent propagation, the stop condition) still applies unchanged; only the source of the *value* changes. Only generate a random UUID when no such native runtime session ID is available.

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
