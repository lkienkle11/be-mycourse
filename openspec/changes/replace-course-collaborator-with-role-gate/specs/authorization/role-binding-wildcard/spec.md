## Purpose

Lets a resource-scoped role binding cover every resource instance of a given resource type at once, instead of requiring one binding row per instance, so "this principal holds role R on every resource of type T" is a single, first-class statement.

## ADDED Requirements

### Requirement: Wildcard binding grants a role's actions on every resource of its type
A role binding whose resource identifier is the reserved wildcard value SHALL grant the principal every action the bound role currently covers, for every existing and future resource instance of the binding's resource type.

#### Scenario: Wildcard binding covers an existing resource
- **WHEN** a principal holds an active role binding with resource type "course" and the wildcard resource identifier
- **THEN** the principal is granted every action the bound role covers on any specific course, without a binding row for that course

#### Scenario: Wildcard binding covers a resource created after the binding
- **WHEN** a principal holds an active wildcard role binding for resource type "course", and a new course is created after the binding was made
- **THEN** the principal is granted every action the bound role covers on the newly created course, with no additional binding required

### Requirement: Wildcard scope does not cross resource types
A wildcard role binding SHALL only affect access to resources whose type matches the binding's own resource type.

#### Scenario: Wildcard on one resource type grants nothing on another
- **WHEN** a principal holds a wildcard role binding for resource type "course"
- **THEN** the principal gains no access to resources of any other resource type from that binding

### Requirement: Wildcard and resource-specific bindings combine without conflict
When a principal holds both a wildcard role binding and one or more resource-specific role bindings for the same resource type, the decision SHALL reflect the union of actions granted by all of them.

#### Scenario: Specific binding adds no less than the wildcard already grants
- **WHEN** a principal holds a wildcard role binding for resource type "course" and also holds a resource-specific role binding for one particular course
- **THEN** the principal's allowed actions on that course are at least the union of what each binding independently grants

### Requirement: Revoking a wildcard binding immediately revokes its blanket access
Revoking an active wildcard role binding SHALL remove every action it granted, across every resource of its type, on the next access decision.

#### Scenario: Revoke wildcard binding
- **WHEN** an active wildcard role binding is revoked
- **THEN** the principal immediately loses every action that was only granted through that wildcard binding, on every resource of that type, with no per-resource cleanup step required
