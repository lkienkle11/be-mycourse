## 1. Required session and repository preparation

- [x] 1.1 Before implementation work, validate the externally bootstrapped `AI_SESSION_ID_FILE` and `AI_SESSION_ID`, confirm the file is outside every repository and contains the same single non-empty ID, and verify those exact values remain available to every later command or subagent
- [x] 1.2 Read every existing `be-mycourse/.context/*.md` file before planning or editing, then resolve exactly one writable `.context/session-<AI_SESSION_ID>.md` for this coding session; verify no second matching context exists and record the resolved path in that file
- [x] 1.3 Read the complete current `AGENTS.md`, its imported guidance, the applicable `.ai/skills` (including the no-N+1 batch rule), all OpenSpec artifacts for `add-me-roles-field`, and the complete project documentation set; verify the current session context records which rules, skills, and documents apply
- [x] 1.4 Inspect `git status`, staged and unstaged diffs, and non-ignored untracked files before editing; record the pre-existing dirty Authorization/Course work separately from this change and verify no unrelated user-owned change is overwritten, reverted, formatted, staged, or committed
- [x] 1.5 If `.codegraph/` exists, use CodeGraph before text search to trace `/me`, RBAC role reads, cache producers, and role mutation callers; otherwise record that CodeGraph is unavailable and use GitNexus plus focused `rtk rg`/file reads, verifying the reusable paths include `RBACService.ListRolesForUser` and `AuthService.InvalidateUserMeCache`

## 2. GitNexus pre-edit analysis and implementation lock

- [x] 2.1 Check GitNexus freshness before relying on graph results and, when stale, force-refresh it while preserving embeddings when `.gitnexus/meta.json` reports a non-zero embedding count; verify the refreshed index represents the current checkout and do not change or bypass GitNexus/tooling rules
- [x] 2.2 Run upstream GitNexus impact analysis before editing every indexed function, method, class, or constructor in the final implementation set, including at minimum `NewAuthService`, `GetMe`, `warmMeCache`, `buildMeProfile`, `toMeResponse`, RBAC `NewHandler`, `assignUserRole`, and `removeUserRole`; record direct callers, affected processes, risk, and all depth-1 dependents in the current session context
- [x] 2.3 If any impact result is HIGH or CRITICAL, warn the user with the concrete callers/processes before making that edit; verify the warning was delivered before editing, no high-risk result is silently ignored, and every depth-1 dependent is included in the test/update plan
- [x] 2.4 Lock the minimal design before coding: an auth-owned role-name reader port, one call to the existing RBAC role query, server-layer mapping to names, and direct reuse of `AuthService.InvalidateUserMeCache`; verify the architecture/import graph has no cycle, no duplicated role query or Redis invalidation logic, and no per-role/N+1 query loop

## 3. `/me` projection and deterministic ranking

- [x] 3.1 Add display-only `Roles []string` fields to `domain.MeProfile` and `MeResponse` with `json:"roles"` on the HTTP DTO and English comments forbidding authorization use; verify focused compile/tests confirm the field is additive and is never omitted
- [x] 3.2 Add one auth-application ranking helper for `sysadmin`, `admin`, `instructor`, then `learner`, with stable placement of unknown names after ranked names; verify table-driven tests cover mixed ranked roles, multiple unknown roles preserving input order, empty non-nil input, nil input, and a single role
- [x] 3.3 Normalize every freshly built no-role profile to a non-nil empty role slice before caching; verify unit tests distinguish a legitimate `[]` result from a legacy missing field and JSON serializes it as `"roles":[]`

## 4. RBAC read port, auth service, and composition wiring

- [x] 4.1 Add the smallest auth-owned `RoleNameReader`-style interface and inject it into `AuthService`/`NewAuthService` without importing RBAC domain types into auth; verify all constructor callers and test fixtures identified by GitNexus compile after the signature change
- [x] 4.2 Add one server adapter that calls the existing `RBACService.ListRolesForUser` exactly once and maps only role names to `[]string`, then wire it in `wireCore`; verify tests cover empty and multi-role results and confirm nested RBAC permissions are neither loaded into nor exposed through `/me`
- [x] 4.3 Update every `/me` cache producer, including `AuthService.GetMe` cache-miss handling and `warmMeCache` used by login/OAuth, to load, rank, and pass role names into `buildMeProfile`; verify tests cover multi-role ordering, no roles, role-reader failure without writing a partial cache entry, login warm-up, and OAuth warm-up
- [x] 4.4 Map `MeProfile.Roles` in `toMeResponse` and defensively normalize nil to a non-nil empty slice; verify handler/JSON tests assert `"roles":[]` for no roles and ordered raw lowercase names for multiple roles while all existing response fields remain unchanged

## 5. Cache compatibility and role-mutation invalidation

- [x] 5.1 Treat a successfully decoded legacy cached `MeProfile` with `Roles == nil` as a stale-shape cache miss, while accepting a decoded non-nil empty role slice as a valid no-role cache hit; verify cache tests cover missing key, explicit empty array, malformed JSON, wrong user ID, and a populated role array without adding an arbitrary mock/cache dependency
- [x] 5.2 Add the smallest `MeCacheInvalidator` interface to RBAC delivery, inject the existing `AuthService` instance from the server composition root, and invalidate only after successful `assignUserRole` and `removeUserRole`; verify handler tests assert one invalidation for the correct user after success and zero invalidations after parse, validation, not-found, or persistence failure
- [x] 5.3 Audit every caller of `AssignRoleToUser`, `RemoveRoleFromUser`, `EnsureLearnerRole`, and transaction-bound role assignment; verify each path either performs the new RBAC-handler invalidation, reuses the existing instructor invalidator, clears stale `/me` before/after assignment, or assigns before cache warm-up, with no uncovered cache-staleness path
- [x] 5.4 Add an integration-level regression for a warm `/me` cache followed by role assignment and removal, verifying the next `GET /api/v1/me` observes each new binding state without waiting for TTL and without changing the existing one-minute cache policy

## 6. Authorization boundary and scope regression checks

- [x] 6.1 Prove `roles` remains display-only by testing or tracing a user that has an `admin` role name but lacks a required permission and confirming the existing permission gate still denies the action; verify no middleware, `RequirePermission`, `HasPermission`, principal, JWT, or authorization-engine path reads the new projection field
- [x] 6.2 Run a focused `rtk rg` audit over changed and consuming Go files for `.Roles`/`roles` usage, then inspect every match; verify uses are limited to RBAC role retrieval, `/me` projection/ranking/cache/tests, and documentation rather than access-control decisions
- [x] 6.3 Verify the implementation adds no migration, role-management endpoint, frontend change, backend label/i18n mapping, dynamic priority configuration, policy dump, resource-scoped grant data, or tooling/configuration bypass

## 7. Documentation, generated artifacts, and security review

- [x] 7.1 Update every affected current-state document, including `docs/modules/auth.md`, `docs/modules/rbac.md`, `docs/data-flow.md`, `docs/api-overview.md`, `docs/curl_api.md`, `docs/return_types.md`, and any other document found to describe `/me`, role bindings, or cache invalidation; verify all descriptions agree that raw ordered role names are display-only and permissions remain authoritative
- [x] 7.2 Update `docs/api_swagger.yaml` so `MeProfile.roles` is a required/non-null array with a display-only description and representative examples, then run `rtk ruby scripts/generate-apidog-postman.rb`; verify `docs/api-dog-import.json` is regenerated from Swagger rather than hand-edited and contains the matching `/me` contract
- [x] 7.3 Re-read the complete relevant documentation set after code and generated-file changes, replace outdated/conflicting sections instead of appending update notes, and verify documentation matches the final source and cache behavior exactly
- [x] 7.4 Use Git-aware inclusion checks to inspect only changed committable files for credentials or environment-derived values; verify examples contain placeholders only, no dev-account value is copied from the environment, and no ignored/external session file is scanned or reported as repository content

## 8. Verification, GitNexus sync, and handoff

- [x] 8.1 Run focused tests for all changed auth, RBAC, and server packages (at minimum `rtk go test ./internal/auth/... ./internal/rbac/... ./internal/server/...`) and verify every new ranking, cache, invalidation, wiring, and authorization-boundary case passes
- [ ] 8.2 Start the local backend with the documented environment and smoke-test `GET /api/v1/me` using only the `DEV_TEST_ACCOUNT_EMAIL`/`DEV_TEST_ACCOUNT_PASSWORD` variable names without printing their values; verify `roles` is present, ordered, non-null, and existing fields/permission behavior do not regress
- [x] 8.3 Force-sync GitNexus after the final code/documentation state, preserving existing embeddings when present, and verify the index refresh succeeds without overwriting manual `AGENTS.md` content or unrelated user changes
- [x] 8.4 Run `rtk make test-all`, then `rtk make check-all`, and fix every failure without suppressions, exclusions, skipped hooks, rule changes, or unrelated cross-project edits; verify formatting, default/CGO tests, vet, golangci-lint, layout, architecture, duplicate detection, and CGO/non-CGO builds all complete successfully
- [x] 8.5 Run `rtk git diff --check`, `rtk openspec validate add-me-roles-field --strict --no-interactive`, and `rtk openspec instructions apply --change add-me-roles-field --json`; verify the checks exit successfully and the OpenSpec apply parser reports every task in this checklist
- [x] 8.6 Run `gitnexus_detect_changes(scope="all")`, compare its changed symbols/processes with the recorded pre-existing dirty worktree and this change's intended scope, and verify all depth-1 dependents were updated, no HIGH/CRITICAL result was ignored, and no unrelated file is attributed to this implementation
- [x] 8.7 Perform the required post-implementation review of all changed code, docs, generated artifacts, staged/unstaged changes, and non-ignored untracked files; verify architecture/reuse/deduplication, authorization boundaries, secrets, CI readiness, and current-task scope, without staging or committing unless the user separately requests it
- [x] 8.8 Re-read the complete project `.context` folder, the exact current session context, relevant project documents, and GitNexus results, then replace the current session context with an accurate final summary of decisions, files, commands, outcomes, failures/fixes, risks, and next steps; verify the Session ID still matches `AI_SESSION_ID_FILE` and exactly one project context file exists for it

## 9. Post-review hardening

- [x] 9.1 Invalidate the affected user's cached `/me` projection only after successful direct-permission assignment/removal, and add handler regressions for success plus parse, validation, not-found, and persistence failures
- [x] 9.2 Replace the unused authorization service bundle return with the minimal startup action-catalog synchronization result while no `PolicyProvider` or runtime consumer exists
- [x] 9.3 Remove the unconsumed context-carried GORM transaction branch and its isolated helper test; keep `ReplaceMany` atomic through its repository-owned transaction and verify focused authorization tests
- [x] 9.4 Correct migration and authorization documentation so `000034` is described as four-table DDL with an empty provider/action catalog, with no Course seed or collaborator backfill
- [x] 9.5 Make `/me` cache freshness and JWT authorization revocation latency explicit and distinct in RBAC/auth/data-flow documentation
- [x] 9.6 Run focused tests, force-sync GitNexus, run all mandatory backend quality gates, inspect the final diff, and update this task list plus the canonical session context with actual outcomes
