---
alwaysApply: true
---

# Implementation Workflow Rules

## Initial Phase: Project Understanding

Before implementing any solution, the agent must thoroughly read the entire `.context` folder and all project documentation to understand the full project scope.

Use GitNexus to work faster and to inspect the project context.

The agent must also review all changed code in Git to understand exactly what has been modified, then update the documentation accordingly.

For `be-mycourse` tasks:

* Read the `be-mycourse` repository carefully.
* Understand the project structure.
* Identify where each module is located.
* Identify where relevant code, folders, utilities, types, and services are placed.
* Make sure any new work follows the existing structure.

For `fe-mycourse` tasks:

* Read the `fe-mycourse` repository carefully.
* Understand the project structure.
* Identify where each module is located.
* Identify where relevant code, folders, components, hooks, utilities, types, and services are placed.
* Make sure any new work follows the existing structure.

## Middle Phase: Implementation

Implement the requested feature completely.

The implementation must follow the existing folder structure and reuse existing utility functions, components, types, services, hooks, helpers, and shared logic whenever possible.

If a required utility does not exist, create a new reusable utility and place it in the correct folder so that future phases can reuse it.

All work must be written in English.

## Critical Reuse Rule

When implementing a solution, the agent must use existing resources, code, components, functions, methods, utilities, services, hooks, and types whenever they already exist in the codebase.

The agent is not allowed to create a new resource, code block, component, function, method, utility, service, hook, or type if an equivalent one already exists.

If duplicate implementation is found, the entire solution must be rejected, and the agent must read the project again and redo the work from the beginning.

## Critical Deduplication Rule

If a type, function, method, component, utility, service, hook, or code block from feature A already exists, and feature B duplicates the same logic or structure, the agent must merge the duplicated logic into a shared implementation.

If multiple types or functions share common fields or behavior, the agent must reuse, extend, compose, or inherit from the existing implementation instead of duplicating it.

Do not leave duplicated types, duplicated functions, duplicated methods, duplicated components, duplicated utilities, or duplicated business logic in the codebase.

If duplication is found during review, the solution must be rejected and redone from the beginning.

## Final Phase: Sync, Quality Gates, Review, and Documentation

At the end of the task, the agent must force-sync GitNexus.

Then run the required quality gates based on the side of the project being changed.

For Backend tasks:

* Run formatting.
* Run linting.
* Run `make test-all`.
* Run `make check-all`.
* Run tests.
* Run build checks.
* Make sure the backend CI/CD pipeline will not break.

For Frontend tasks:

* Run `npm run check-all`.
* Fix any errors.
* Make sure the frontend CI/CD pipeline will not break.

If the task is Backend-related, fix Backend issues only.

If the task is Frontend-related, fix Frontend issues only.

Do not randomly fix the opposite side unless the task explicitly requires cross-repository changes.

## Exit Code Rule

If any command exits with a non-zero status code, the agent must fix the errors gradually until all issues are resolved.

This includes errors that are not directly related to the current task if they can break CI/CD.

After fixing the errors, the agent must rerun all required quality gates until everything passes.

The agent must not claim that a command passed unless it was actually run and completed successfully.

## Code Review Requirement

After implementation, the agent must inspect and review all written and modified code.

The review must verify:

* The implementation follows the existing architecture.
* The implementation follows the existing folder structure.
* Existing utilities, components, functions, services, hooks, and types were reused where possible.
* No duplicate logic was introduced.
* No unrelated files were modified.
* No secrets or credentials were hardcoded.
* The code is ready for CI/CD.

## Documentation and Context Sync

After completing the implementation, the agent must fully sync, update, or delete documentation, `.context` files, and GitNexus files so that they match the latest source code exactly.

Updating documentation means:

* Scan the entire source code.
* If something exists in code or features but is missing from documentation, add it to the relevant documentation.
* If code and features are implemented one way but the documentation describes a different approach, update the documentation to match the code.
* If something is documented but does not exist in the code or implemented features, remove it from the documentation.
* Update all relevant `.context` files.
* Update all relevant GitNexus files.

For `be-mycourse` tasks, update GitNexus files inside the `be-mycourse` project.

For `fe-mycourse` tasks, update GitNexus files inside the `fe-mycourse` project.

## Swagger and API Documentation Rule

If any Swagger YAML file is modified, the agent must also run:

```bash
ruby generate-apidog-postman.rb
```

After running the generator, the agent must update all related API documentation, `.context` files, and GitNexus files to match the generated output and the latest source code.

## Final Response Requirement

At the end of the task, the agent must report:

* What was implemented.
* Which files were changed.
* Which existing resources were reused.
* Which duplicate logic was merged or avoided.
* Which documentation, `.context`, and GitNexus files were updated.
* Which quality gates were run.
* Which commands passed.
* Which commands failed and how they were fixed.
* Any remaining risks or follow-up actions.
