---
name: progress-tracking
description: Track and report progress of an in-scope task according to how it is orchestrated - an OpenSpec change (tick tasks.md after verification), a superpowers plan (tick plan checkboxes, keep spec and plan files out of the fixed docs set), or neither (session context file or a short todo list). Use during implementation of any in-scope task and before reporting it done.
---

# Progress tracking

Never report a step or the task as done before running its verification. Say plainly what ran and what did not.

## OpenSpec change

- `openspec/changes/<name>/tasks.md` is the source of truth.
- Tick `- [x]` right after each task, and only after the verification the task states has run.
- A task that can complete only with the final answer stays unticked, with the reason stated.
- If the work exposes a design problem, stop and propose an edit to `design.md` or `tasks.md` instead of changing direction.
- When all tasks are done, suggest archiving and moving the row in `docs/ALL-SPRINTS.md` to Done. Never archive on your own.

## superpowers skills

- The plan file's checkboxes and the todo list are the source of truth; tick each step as it finishes and honour the skill's verification gates.
- These skills save specs and plans under `docs/superpowers/`, which would add files to the fixed documentation set. Save them under `.context/` or the scratchpad instead, and record only a row in `docs/ALL-SPRINTS.md` (item, owner, status).
- A commit step in a skill runs only under the Commit rules in `AGENTS.md`.
- Do not mix OpenSpec and superpowers for the same task; use one.

## Neither

- Small task: keep progress in the current `.context/session-<SESSION_ID>.md`.
- Multi-step task: keep a short todo list in the conversation and update it after each step.
