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
| **Conversation log** | Each message turn: what user asked / said, what AI responded, how user reacted |
| **Interaction patterns** | Did the AI take user feedback seriously? Did it learn from mistakes? Did it dismiss requests? |

> Do **NOT** copy-paste raw file contents or long code blocks into the summary.
> Summarize changes in plain language (e.g., "Added JWT middleware to auth routes").

**Additional extraction rules for Conversation Log & Interaction Analysis:**

For **Conversation Log**:
- Number each turn sequentially (Turn 1, Turn 2, …).
- For **User**: capture the intent, tone, and clarity of the request — not just a keyword. E.g., "Yêu cầu tạo API upload file, chưa rõ storage backend cần dùng S3 hay local."
- For **AI**: describe concretely what was produced or decided — files written, plan proposed, question asked. E.g., "Đề xuất 2 phương án storage, viết draft file_service.go với local strategy."
- For **User reaction**: be honest. Examples:
  - "Hài lòng, confirm tiếp tục."
  - "Chỉ trích: AI viết sai tên field, yêu cầu sửa lại."
  - "Im lặng / không phản hồi rõ ràng."
  - "Bổ sung thêm yêu cầu chưa đề cập trước đó."

For **Interaction Analysis — AI Behavior**:
- Be **honest and self-critical**. Do not default to "Có" for positive traits.
- "Rút kinh nghiệm" = true only if AI fixed the exact issue raised on the **next** response, not just acknowledged it.
- "Xem nhẹ yêu cầu" = true if AI said it would do something but the output didn't reflect it, or if AI skipped steps the user explicitly asked for.
- "Tự ý thêm / bớt" = true if AI added unrequested features OR omitted explicitly requested parts.

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

## Conversation Log
<!-- Record ALL exchange turns in the session. Each turn has 3 parts. -->
<!-- Ghi lại TẤT CẢ các lượt trao đổi trong session. Mỗi lượt gồm 3 phần. -->

### Turn 1
- **User:** <Summarize the user's intent, tone, and clarity of request — not just keywords. / Tóm tắt ý định, cảm xúc, mức độ rõ ràng của yêu cầu — không chỉ ghi từ khóa>
- **AI:** <Describe concretely what was produced or decided — files written, plan proposed, question asked. / Mô tả cụ thể output đã tạo ra — file nào được viết, kế hoạch nào được đề xuất, câu hỏi nào được đặt ra>
- **User reaction:** <How did the user respond? Satisfied / requested fix / criticized / silent / confirmed to continue. / Người dùng phản ứng thế nào? Hài lòng / yêu cầu sửa lại / chỉ trích / im lặng / xác nhận tiếp tục>

### Turn 2
- **User:** ...
- **AI:** ...
- **User reaction:** ...

<!-- Repeat for every exchange turn. Even if session is short (< 3 turns), still record all. -->
<!-- Lặp lại cho mỗi lượt trao đổi. Dù session ngắn (< 3 lượt), vẫn ghi đủ. -->

## Interaction Analysis
<!-- Đánh giá tổng thể về chất lượng tương tác trong session này. Phải trung thực, không tô hồng. -->

### AI Behavior
- **Nghiêm túc thực hiện yêu cầu:** <Có/Không — ví dụ cụ thể trong session>
- **Rút kinh nghiệm từ sai lầm:** <Có/Không — khi bị chỉ ra lỗi, AI có thực sự sửa đúng hướng không? Hay lặp lại lỗi cũ?>
- **Xem nhẹ / bỏ qua yêu cầu người dùng:** <Có/Không — ví dụ nếu có: AI nói "đã làm" nhưng thực ra không làm, hoặc làm khác với yêu cầu>
- **Tự ý thêm / bớt so với yêu cầu:** <Có/Không — ví dụ cụ thể>

### Lessons Learned for Next Session
- <Điều AI nên nhớ để không lặp lại sai lầm trong session tiếp theo>
- <Phong cách giao tiếp người dùng: chi tiết / ngắn gọn / hay thay đổi yêu cầu / ưa giải thích>
- <Bất kỳ pattern nào trong cách người dùng phản hồi giúp AI phục vụ tốt hơn>

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

**Lessons from last session (AI self-reflection):**
- <Paste Lessons Learned for Next Session content verbatim>
- AI behavior flags: Nghiêm túc=<Yes/No> | Rút kinh nghiệm=<Yes/No> | Xem nhẹ=<Yes/No> | Tự ý thêm bớt=<Yes/No>

Ready to continue. Where would you like to start?
```

> The "Lessons from last session" block reminds the AI of behavioral pitfalls from the previous session so it can avoid repeating them.

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
