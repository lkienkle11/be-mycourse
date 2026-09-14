## Purpose

Lets an authorized issuer assign and revoke a resource-scoped role binding for a principal, so any resource type can manage "who holds which role on which resource" without adding resource-type-specific code inside `internal/authorization`.

## ADDED Requirements

### Requirement: A role binding can be assigned to a principal for a resource or for every resource of a type
The system SHALL allow an authorized issuer to assign a named role to a principal, scoped to one specific resource or to every resource of a type (via the wildcard resource identifier).

#### Scenario: Assign a role binding for one resource
- **WHEN** an authorized issuer assigns role R to a principal for one specific resource
- **THEN** the principal immediately gains every action role R currently covers for that resource's type, on that resource only

#### Scenario: Assign a role binding for every resource of a type
- **WHEN** an authorized issuer assigns role R to a principal using the wildcard resource identifier for a resource type
- **THEN** the principal immediately gains every action role R currently covers, on every resource of that type

### Requirement: Assignment and revocation accept multiple principals in one call
The system SHALL accept a batch of principal IDs in a single assign or revoke call for the same role and resource, rather than requiring the caller to issue one call per principal.

#### Scenario: Batch-assign a role to multiple principals in one call
- **WHEN** an authorized issuer assigns role R for one resource to a list of principal IDs in a single call
- **THEN** every valid principal in the list receives an active binding from that one call, without the caller issuing a separate call per principal

#### Scenario: Batch-revoke bindings for multiple principals in one call
- **WHEN** an authorized issuer revokes role R for one resource from a list of principal IDs in a single call
- **THEN** every matching active binding for the listed principals is revoked from that one call

### Requirement: Only an issuer authorized to manage grants on the resource may assign or revoke a binding
The system SHALL reject an assignment or revocation attempt from an issuer who is not authorized to manage grants for the target resource (or, for a wildcard assignment, for that resource type).

#### Scenario: Unauthorized issuer is rejected
- **WHEN** a principal without grant-management authority over a resource attempts to assign or revoke a role binding on it
- **THEN** the system rejects the request and makes no change to any role binding

### Requirement: A role binding can be revoked
The system SHALL allow an authorized issuer to revoke an active role binding, immediately ending the access it granted.

#### Scenario: Revoke an active binding
- **WHEN** an authorized issuer revokes a principal's active role binding
- **THEN** the principal immediately loses every action that binding granted, on its next access decision

#### Scenario: Revoking an already-revoked binding is a no-op
- **WHEN** an authorized issuer revokes a role binding that is already revoked
- **THEN** the system leaves the binding's revocation state unchanged and does not error in a way that blocks the caller from treating the resource as no longer bound

### Requirement: Every assignment records who granted it
The system SHALL record the identity of the issuer who assigned a role binding, alongside the binding itself.

#### Scenario: Assignment is attributed to its issuer
- **WHEN** an authorized issuer assigns a role binding to a principal
- **THEN** the resulting binding records that issuer as the grantor, retrievable for later audit
