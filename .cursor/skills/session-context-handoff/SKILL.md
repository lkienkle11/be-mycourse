---
name: session-context-handoff
description: >
  Saves in-progress work to .context/session_summary_<timestamp>.md when the user types /save,
  creating a NEW timestamped file each time (never overwrites).
  When the user types /load, reads the MOST RECENT session file and restores context.
  Triggers on: /save, /load, "session summary", "context handoff", "lưu context", "tải context",
  or any mention of .context/session_summary.
---

# Skill: Session Context Handoff (`/save` / `/load`)

> **ABSOLUTE RULES — READ BEFORE PROCEEDING:**
> - `/save` → **CREATE A NEW FILE** `.context/session_summary_<YYYY-MM-DD_HHmmss>.md` — **NEVER overwrite an existing file**
> - `/load` → **READ ONLY** the most recent `.context/session_summary_*.md` — **NEVER write or modify any file**
> - Both commands support Vietnamese and English trigger phrases equally.

---

## Command `/save` — Persist current session context

### Trigger phrases (any of these)
- `/save`
- `save context`
- `lưu context`
- `lưu lại`
- `save session`

### What to do — step by step

#### Step 1 — Collect information from the entire conversation

Scan through ALL messages in the current session and extract:

| Field | What to capture |
|-------|-----------------|
| **Overview** | What is the main goal of this session? (2–3 sentences) |
| **Completed** | Tasks or sub-tasks that are fully done |
| **In-progress** | Tasks started but not finished; include current state |
| **Files touched** | Every file created, modified, or deleted; include a short reason |
| **Errors / fixes** | Bugs or errors encountered — mark as FIXED or OPEN |
| **Key decisions** | Architecture choices, trade-offs made, reasons for choosing an approach |
| **Next steps** | Concrete actions to take when resuming — ordered by priority |
| **Blockers** | Anything preventing progress (missing info, waiting on review, etc.) |

> Do **NOT** copy-paste raw file contents or long code blocks into the summary.
> Summarize changes in plain language (e.g., "Added JWT middleware to auth routes").

#### Step 2 — Determine the save path

1. The `.context/` directory lives at the **project root** (same folder as `go.mod` for backend, or `package.json` for frontend).
2. Compute the current datetime as `YYYY-MM-DD_HHmmss` (e.g., `2026-04-20_153045`).
3. The new file path will be: `.context/session_summary_<YYYY-MM-DD_HHmmss>.md`
   - Example: `.context/session_summary_2026-04-20_153045.md`

#### Step 3 — Create the directory if needed

- Check if `.context/` exists. If not, create it.
- Do **NOT** delete or rename any existing files in `.context/`.

#### Step 4 — Write the new file using the template below

Use the exact template structure. Fill every section — if a section has nothing to report, write `(none)` rather than leaving it blank.

#### Step 5 — Confirm to the user

After writing, tell the user:
```
Session saved to .context/session_summary_<datetime>.md
```
Also show a one-line summary of what was captured (e.g., "3 tasks completed, 2 in-progress, next step: implement X").

---

### Template for `.context/session_summary_<YYYY-MM-DD_HHmmss>.md`

```markdown
# Session Summary

> Saved: <YYYY-MM-DD HH:mm:ss>
> Branch: <current git branch, or "unknown" if not determinable>
> Project: <project name, e.g. be-mycourse>

## Overview
<!-- 2–3 sentences: what was the main goal of this session, what approach was taken -->

## Completed
- [x] <task description — be specific enough to understand without context>
- [x] ...

## In-progress
- [ ] <task description + current state, e.g. "Implement payment service — service layer done, controller not started">
- [ ] ...

## Files created / modified
- `path/to/file.go` — <what changed and why>
- `path/to/another.go` — <what changed and why>

## Errors / fixes
- **[FIXED]** `<error message or description>` — <root cause> → <how it was fixed>
- **[OPEN]** `<error message or description>` — <suspected cause, what was tried>

## Key decisions
- <Decision made> — <reason / trade-off>
- ...

## Blockers
- (none) or list blockers with context

## Next steps (priority order)
1. <Most urgent action — include file/function/command if relevant>
2. ...
3. ...

## Notes
<!-- Technical constraints, env-specific details, important reminders -->
- ...
```

---

## Command `/load` — Restore context from a previous session

### Trigger phrases (any of these)
- `/load`
- `load context`
- `tải context`
- `tiếp tục từ session trước`
- `continue from last session`
- `load last session`

### What to do — step by step

> ⚠️ **DO NOT create, write, modify, or delete any file during `/load`.**
> This command is strictly READ-ONLY.

#### Step 1 — Find the most recent session file

1. Look inside `.context/` at the project root.
2. List all files matching the pattern `session_summary_*.md`.
3. Sort them **lexicographically descending** (ISO datetime format sorts correctly this way).
4. Select the **first** (most recent) file.

#### Step 2 — Handle missing files

**If `.context/` does not exist OR no `session_summary_*.md` files are found:**
- Reply exactly:
  ```
  No session files found in .context/. Nothing to load — start a new session and use /save to record it.
  ```
- Stop. Do not attempt to read anything else.

#### Step 3 — Read and present the loaded context

Read the selected file and present a structured recap to the user using this exact format:

```
✅ Context loaded from .context/session_summary_<filename>

📌 Project: <project>  |  Branch: <branch>  |  Saved: <datetime>

**Overview:**
<Paste the Overview section content verbatim>

**Completed:**
<List all [x] items>

**In-progress:**
<List all [ ] items with their current state>

**Files in scope:**
<List all files from "Files created / modified">

**Open errors / blockers:**
<List OPEN errors and any blockers>

**Next steps:**
1. <step 1>
2. <step 2>
...

Ready to continue. Where would you like to start?
```

#### Step 4 — Wait for user instruction

Do **NOT** automatically start executing any next step.
Wait for the user to explicitly say what to do next.

---

## Notes & edge cases

- **Multiple saves in one session** are encouraged. Each `/save` creates a new timestamped file, building a history.
- **Which file does `/load` pick?** Always the lexicographically largest filename (= most recent timestamp). If you want to load an older file, the user can specify the filename explicitly (e.g., `/load session_summary_2026-04-19_100000.md`).
- **Git**: `.context/` may or may not be committed. If not in `.gitignore`, suggest adding it — or committing it intentionally for team sharing.
- **Language**: Vietnamese and English trigger phrases are treated identically.
- **Project root detection**: Use the directory containing `go.mod` (backend) or `package.json` (frontend) as the project root.
