---
name: pre-implementation-checkpoint
description: Ask the user, in one AskUserQuestion call, the questions that must be answered before the first source edit of an in-scope task - whether to create a new branch name now, whether to skip or install a test library when the repository has no test setup, and how many research subagents to use when the task is heavy. Use after the plan is approved and before the first source edit.
---

# Pre-implementation checkpoint

Runs once per task, after any plan approval and before the first source edit. Collect every question below that applies and ask them in ONE AskUserQuestion call (up to 4 questions; one direct message when the tool is absent). Drop a question that does not apply. Mark the default option "(Recommended)". Record the answers in the session context file and repeat them in the final response.

## Skip the whole checkpoint when

- The task is out of scope, or the user already answered these questions earlier in this conversation.
- The change is a few lines and does not change behavior (typo, wording, comment, formatting).

## 1. Branch

Skip only this question when the current branch already names this task.

Options: "Yes, create a new branch name now (Recommended)" and "No, implement first; name the branch later".

On Yes:

1. Propose exactly one name following `.ai/skills/branch-rename/SKILL.md`: `<type>/<english-kebab-summary>`, `type` from the repository's commitlint config or Conventional Commits, derived from the user's task when there are no local commits.
2. Ask in a second call: "Accept this name?" with "Yes (Recommended)" and "No, I will type my own" (the user types it in Other). Use the name the user typed.
3. If the current branch is protected (`main`, `master`, `dev`, `release*`), create the new branch with `git switch -c <name>` and never rename the protected branch. Otherwise rename with `git branch -m -- '<name>'`. Verify with `git branch --show-current`.
4. Never push and never change the upstream.

On No: implement first and say in the final response that the branch is still unnamed.

## 2. Tests

Ask only when the task changes behavior and the repository has no test setup. Detect a setup by looking for one of: a `test` script plus its configuration in `package.json`, `*_test.go` files, `pytest.ini`, `tox.ini`, a `tests/` folder, a test dependency in `pom.xml` or `build.gradle`.

- Has a setup: do not ask; write tests as the Implementation Rule says.
- Has none: ask "Skip writing tests for this whole task (Recommended)" or "Install a test library: the agent picks one that fits the stack and writes the tests".
- Installing counts as the user's explicit approval to add that tooling, and only to add it; Tooling rule protection still forbids weakening existing tooling.
- When skipped, say so in the final response.

## 3. Subagents

Ask only when the task is heavy enough that more research subagents would help. The default cap is 4. Options: "Keep 4 (Recommended)", a larger number you name, "No subagents". Tell the user first that more subagents cost more tokens. A number the user already stated replaces the default.
