## Context

See `proposal.md` for motivation and `specs/development-ci-pipeline/spec.md` for the behavioral contract. The backend development workflow currently defines `test`, `build`, and `deploy` as a dependency chain. The change affects CI configuration and the documentation that describes it, and its implementation must follow the complete repository lifecycle in `AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Remove `jobs.deploy` from the active YAML structure by commenting its complete block.
- Preserve the job text and ordering so reactivation requires only removing the added comment prefixes.
- Keep a valid workflow in which `test` and `build` retain their current definitions and dependency.
- Keep current-state CI/CD documentation, GitNexus state, and the single session context consistent with the implemented workflow.

**Non-Goals:**

- Do not alter triggers, concurrency, test/build behavior, artifacts, imported actions, secrets, scripts, application code, or server state.
- Do not replace the job with `if: false`, rename it, or delete any deployment line.
- Do not execute a deployment or change remote VPS files.
- Do not create a second session ID or write implementation state to another session context.

## Decisions

1. Prefix every non-empty line from the top-level `deploy:` key through the end of that job with `# `. This removes the job from parsed YAML while preserving every original line after the prefix.
2. Keep blank lines in their existing positions. Existing explanatory comments remain in place and receive an additional comment prefix with the rest of the block.
3. Use commenting rather than `if: false`. The user explicitly requires the deploy code to remain commented, and an inactive job would still be registered in the workflow graph.
4. Treat the pause as a CI behavior change and describe it with the new `development-ci-pipeline` capability instead of using `skip_specs`; the scenarios provide observable acceptance criteria without specifying parser or shell implementation details.
5. Update `README.md`, `docs/deploy.md`, and `docs/requirements.md` after the workflow edit so every current-state statement says `test` → `build`, with automatic deploy paused and the commented block retained for restoration.
6. Follow the repository workflow as part of apply: validate the external Session ID, reuse exactly one `.context/session-<SESSION_ID>.md`, read the required context/docs/skills/Git state, document that symbol impact is not applicable to YAML/docs, force-sync GitNexus, run all backend gates, review the final committable diff, and record learning/handoff evidence.
7. Validate the resulting file as YAML, assert the active job graph, reconstruct the pre-change deploy block by stripping only the added prefixes, and inspect a path-scoped diff. These checks jointly prove valid syntax, inactive deployment, and lossless preservation.

## Risks / Trade-offs

- [The build artifact is still uploaded although deployment is disabled] → Retain this intentionally because the requested scope keeps the build job unchanged.
- [A partial comment could leave deploy steps active or invalidate YAML] → Comment from `deploy:` through the end of the complete job and validate the final YAML.
- [Manual reactivation can introduce indentation mistakes] → Preserve original indentation and content behind uniform comment prefixes.
- [Documentation could still imply that pushes deploy automatically] → Search all relevant backend documentation and replace every conflicting current-state statement.
- [Repository-wide gates are broader than a YAML-only change] → Run them because `AGENTS.md` requires them and report any pre-existing failure without weakening tooling.
- [GitNexus may report no changed symbols for YAML/docs] → Record that symbol impact is not applicable, then still force-sync the index and run final change detection as required.

## Migration Plan

1. Complete the session/context, repository-understanding, Git, and GitNexus readiness checks required by `AGENTS.md`.
2. Comment the complete backend `deploy` block without altering any retained text.
3. Synchronize the affected backend CI/CD documentation with the paused state.
4. Force-sync GitNexus, run the backend quality gates, validate YAML and OpenSpec, and review the complete diff and CI/CD readiness.
5. Record the final state and learning in the one current backend session context, then provide the required handoff.
6. Roll back by removing only the added `# ` prefixes, restoring the corresponding active-deploy wording in documentation, and rerunning the same validation lifecycle.
