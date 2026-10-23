---
name: branch-rename
description: Rename the current local Git branch from the combined intent of its local-only commits (resolve the origin baseline, derive exactly one Conventional Commit style branch name, show the proposal, rename with git branch -m only after approval unless --force). Use only when the user asks to rename the current local Git branch.
---

# Branch rename

Apply this procedure only when the user asks to rename the current local Git branch.

## Rules

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
