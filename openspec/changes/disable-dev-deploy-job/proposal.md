## Why

Development pushes should continue to run backend quality checks and produce the build artifact without automatically deploying that artifact to the VPS. Temporarily disabling only the deployment job prevents unintended server changes while retaining the deployment procedure for later reactivation and keeping repository documentation consistent with the active workflow.

## What Changes

- Comment every non-empty line of the `deploy` job in `.github/workflows/deploy-dev.yml` instead of deleting or rewriting the job.
- Preserve the complete checkout, artifact download, SSH, backup, rsync, PM2 health-gate, and rollback workflow text in place.
- Leave the `test` and `build` jobs, workflow triggers, concurrency settings, imported actions, and deployment helpers unchanged.
- Keep the workflow valid so pushes to `master` still execute `test` followed by `build`, but no deployment job is registered or run.
- Audit and synchronize all relevant backend documentation that currently describes automatic dev deployment, including `README.md`, `docs/deploy.md`, and `docs/requirements.md`, so it states that deployment is temporarily paused while test and build remain active.
- Execute the complete implementation lifecycle required by `AGENTS.md`: validate and reuse the conversation Session ID, create or reuse the exact backend session context, inspect all existing contexts and relevant documentation/skills/Git state, review GitNexus applicability before editing, force-sync GitNexus after editing, run every required backend quality gate, perform the post-implementation review, and record the final learning and handoff state.
- Keep application source, APIs, dependencies, deployment scripts, secrets, and server configuration unchanged.

## Capabilities

### New Capabilities

- `development-ci-pipeline`: Defines the observable backend development pipeline while automatic VPS deployment is paused and preserves the disabled deployment procedure for restoration.

### Modified Capabilities

None.

## Impact

- Primary implementation file: `.github/workflows/deploy-dev.yml`.
- Governance-required synchronization may update relevant documentation, GitNexus-generated metadata, and `.context/session-<SESSION_ID>.md`; it must not introduce unrelated changes.
- GitHub Actions continues to test and build the backend on pushes to `master`.
- VPS SSH, binary backup/copy, PM2 reload, health checks, and rollback are not executed while the job remains commented.
- Backend validation includes formatting, linting, `make test-all`, `make check-all`, tests, build checks, GitNexus force-sync/change detection, YAML validation, scoped diff review, strict OpenSpec validation, and final CI/CD readiness review.
- No application code, API, dependency, deployment helper, secret, or remote server state changes.
