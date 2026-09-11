## Purpose

Define the observable backend development CI pipeline while automatic VPS deployment is temporarily paused and its restoration path remains intact.

## ADDED Requirements

### Requirement: Development pushes stop after a successful build
The backend development workflow SHALL run its existing test and build jobs for pushes to `master` and SHALL NOT register or execute a deployment job while the pause is active.

#### Scenario: Test and build succeed
- **WHEN** a commit is pushed to `master` and the test job succeeds
- **THEN** the build job produces the existing backend binary artifact and the workflow completes without contacting or modifying the VPS

#### Scenario: Tests fail
- **WHEN** a commit is pushed to `master` and the test job fails
- **THEN** the build job does not run and no deployment operation runs

### Requirement: Deployment procedure remains recoverable
The repository MUST retain the complete backend deployment job text in the workflow as comments so automatic deployment can be restored without reconstructing deleted commands.

#### Scenario: Deployment pause is reviewed
- **WHEN** a maintainer inspects `.github/workflows/deploy-dev.yml`
- **THEN** the checkout, artifact download, SSH setup, host preparation, rollback-helper copy, binary backup, rsync, PM2 health-gate, and rollback instructions remain present as commented YAML

#### Scenario: Deployment is intentionally restored
- **WHEN** a maintainer removes only the pause comment prefixes and validates the workflow
- **THEN** the original deployment job is reconstructed with its dependency on `build`

### Requirement: Operational documentation matches the active pipeline
Backend documentation SHALL state that pushes to `master` currently run test and build only, that automatic VPS deployment is paused, and that the retained deployment block is the restoration source.

#### Scenario: Maintainer follows deployment documentation
- **WHEN** a maintainer reads the backend CI/CD documentation during the pause
- **THEN** the documentation does not claim that a push automatically changes the VPS
