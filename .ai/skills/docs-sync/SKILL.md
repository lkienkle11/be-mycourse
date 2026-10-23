---
name: docs-sync
description: Update project documentation after an in-scope task that changes behavior, routes, shared code or architecture. Map each change to its target docs file, edit only the touched entries, replace outdated text instead of appending, keep each file's language, verify paths and links, and move the sprint-log row when an OpenSpec change is archived. Use after any in-scope implementation task before the final response.
---

# Docs sync

Keeps the fixed documentation set in `docs/` equal to the code after a task. It does not add files and does not append changelogs.

## When to use

After any in-scope task that changes behavior, routes, shared code or architecture; and when an OpenSpec change is archived.

## Procedure

1. List what the task changed: `git status --short` and `git diff --stat` (plus untracked files).
2. Map every change to a target file with the table in "Repository specifics".
3. Open only the matching entry (or section) and edit it: replace outdated sentences with the current behavior, add what is missing, delete what no longer exists. Never append a "changelog" paragraph or leave old and new descriptions side by side. If several files describe the same fact, update them all consistently (or keep it in one place and point to it).
4. Keep each file's existing language, tone and format. Do not translate a file and do not mix languages inside one file unless it is multilingual by design.
5. Verify: every backticked repository path exists (`test -e`), every markdown link and anchor in the files you touched resolves, no heading is duplicated across docs files, and removed or renamed symbols no longer appear (`grep -rn <name> docs`).
6. When an OpenSpec change is archived, move its row in `docs/ALL-SPRINTS.md` from `In progress` to `Done` (date = the change's creation date, summary in one line, archive folder name in the last column).

## Do not

- Create a new docs file. The set is fixed.
- Rewrite dated handoff notes or the history rows of `docs/ALL-SPRINTS.md` (they are history).
- Copy code into docs; point to paths and describe only what the code cannot say (rules, reasons, flows across files, gotchas).
- Document credentials, tokens, keys, private endpoints or values derived from `.env` (variable names only; placeholders such as `YOUR_API_KEY_HERE` in examples).

## Repository specifics (be-mycourse)

Change to target file:

| Change | Target |
|--------|--------|
| Module rule, flow, endpoint behavior, permission of a module | `docs/FEATURES.md`, the `## <module>` entry of that module (grep `^## `) |
| Table, column, index, constraint, migration, permission catalog (appendix) | `docs/DATA-MODEL.md` |
| Reusable package, function, type, utility, constant | `docs/SHARED.md` |
| Layering, composition root, HTTP request path, security, error codes, quality gate targets, delivery surface | `docs/ARCHITECTURE.md` |
| Deployment, Docker, observability, public SEO notes | `docs/deploy.md`, `docs/docker.md`, `docs/observability/`, `docs/security-public-seo-notes.md` |
| Work started / finished, new backlog item | `docs/ALL-SPRINTS.md` |

Module slugs: `auth`, `course`, `taxonomy`, `instructor`, `media`, `rbac`, `authorization`. Keep each entry's `API:` line citing real, current endpoint paths.

Swagger: after modifying a Swagger YAML file run `ruby generate-apidog-postman.rb` (the script is `scripts/generate-apidog-postman.rb`), then fix the `API:` line of the matching `FEATURES.md` entry. `docs/api_swagger.yaml` is the hand-edited source; `docs/api-dog-import.json` is generated from it: never edit it by hand.

The four fixed files (`ARCHITECTURE.md`, `FEATURES.md`, `DATA-MODEL.md`, `SHARED.md`) describe only the current state. `migrations/README.md` and dated handoff notes under `docs/` keep their history.
