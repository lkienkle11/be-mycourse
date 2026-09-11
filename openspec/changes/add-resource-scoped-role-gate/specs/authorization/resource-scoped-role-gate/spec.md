## Purpose

Lets any resource type registered with the authorization engine attach a named, resource-scoped role to a principal, so the role's current action set is honored by the same allow/deny decision path as a directly stored grant, without rewriting every existing holder's grants when the role's action set changes later.

## ADDED Requirements

### Requirement: Resource-scoped role grants the role's current action set
A principal holding an active resource-scoped role binding for a given resource instance SHALL be treated as having an ALLOW grant for every action currently associated with that role name for that resource type, subject to the same global boundary-permission check, provider evaluation, and explicit-DENY-wins ordering already applied to directly stored grants.

#### Scenario: Role holder is allowed an action the role currently covers
- **WHEN** a principal holds an active role binding for role name "collaborator" on a resource, role "collaborator" is currently associated with the requested action for that resource type, and the principal holds the action's global boundary permission
- **THEN** the authorization decision for that principal/action/resource is ALLOW

#### Scenario: Role holder is denied an action the role does not cover
- **WHEN** a principal holds an active role binding for role name "collaborator" on a resource, role "collaborator" is not currently associated with the requested action for that resource type, and no direct grant exists for that action
- **THEN** the authorization decision for that principal/action/resource is DENY

### Requirement: Redefining a role's action set applies to all existing holders immediately
Adding or removing an action's association with a role name SHALL change the effective access of every principal already holding that role name on every resource instance, without any change to the role-binding data itself.

#### Scenario: Associating an action with a role grants it to existing holders without new bindings
- **WHEN** an action becomes associated with role name "collaborator" for a resource type, and a principal already holds an active "collaborator" role binding on a resource of that type from before the association
- **THEN** the next authorization decision for that principal/action/resource is ALLOW, with no new or modified role-binding data for that principal

#### Scenario: Removing an action's association with a role revokes it from existing holders without touching bindings
- **WHEN** an action's association with role name "collaborator" is removed for a resource type, and a principal holds an active "collaborator" role binding on a resource of that type
- **THEN** the next authorization decision for that principal/action/resource on the removed action is DENY, assuming no other direct grant covers it, with no change to the role-binding data

### Requirement: Revoking a role binding takes effect immediately
A role binding marked revoked SHALL stop contributing to authorization decisions on the next evaluation, with no caching or delay.

#### Scenario: Revoked role binding no longer grants its role's actions
- **WHEN** a principal's role binding for a resource is revoked
- **THEN** subsequent authorization decisions for that principal on that resource no longer treat the role's actions as granted, unless a separate direct grant covers them

### Requirement: An explicit DENY grant overrides a role-expanded ALLOW
When a principal's role-expanded access and a directly stored explicit DENY grant apply to the same principal, action, and resource, the explicit DENY SHALL win, exactly as it does between two directly stored grants.

#### Scenario: Per-principal exception subtracts from a role's normally-granted action
- **WHEN** a principal holds an active role binding whose role currently covers an action, and a directly stored explicit DENY grant exists for that same principal, action, and resource
- **THEN** the authorization decision for that principal/action/resource is DENY

### Requirement: A role can only ever be associated with an already-registered action
The system SHALL reject an attempt to associate a role name with an action that is not already registered for that resource type in the action catalog, enforced as a referential integrity constraint rather than application-level validation alone.

#### Scenario: Associating a role with an unregistered action is rejected
- **WHEN** an attempt is made to associate a role name with an action name and resource type that has no corresponding entry in the action catalog
- **THEN** the association is rejected and not persisted

### Requirement: No role-management API is exposed by this capability
This capability SHALL NOT expose any service, HTTP route, or authorization check for creating, updating, or revoking a role binding or a role's action association. Population of role bindings and role-action associations is out of scope for this capability and is expected to be added by a later, separate capability.

#### Scenario: No production code path can create or modify a role binding or a role's action association
- **WHEN** the system is deployed with only this capability
- **THEN** no HTTP route, application service method, or `GrantService` operation exists that creates, updates, or deletes a role binding or a role's action association
