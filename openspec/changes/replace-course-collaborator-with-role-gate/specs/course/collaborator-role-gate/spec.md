## Purpose

Ensures every course access decision resolves through the generic resource-scoped role gate as an action-level check, so a course collaborator role's permitted actions can be redefined in one place without touching Course code, and no code path ever decides access by comparing a role name directly.

## ADDED Requirements

### Requirement: Course access decisions are action-level, never role-name comparisons
Every course access decision SHALL be based on whether the principal has been granted the specific action being performed, resolved through the role gate. No code path SHALL decide course access by comparing a stored role name value directly.

#### Scenario: An owner-only action is decided by an action check
- **WHEN** a principal attempts an action that only the course OWNER role covers
- **THEN** the decision is based on whether the principal currently has that action granted for the course, not on a direct comparison of a role name

### Requirement: The course owner has the full owner action set on their own course
The registered owner of a course SHALL be granted every action the OWNER role covers for that course, without requiring a separate collaborator role-binding row.

#### Scenario: Owner accesses their own course
- **WHEN** the authenticated principal is the registered owner of a course
- **THEN** the principal is allowed every action the OWNER role covers for that course

### Requirement: A collaborator's allowed actions come from their currently assigned role
A course collaborator's allowed actions SHALL be exactly the actions currently defined for the role assigned to them on that course, and no others.

#### Scenario: Collaborator is limited to their role's current actions
- **WHEN** a principal holds an active collaborator role binding for a course
- **THEN** the principal is allowed exactly the actions currently defined for that role on that course, and is denied any action not in that set

### Requirement: Redefining a collaborator role's actions applies to every existing holder immediately
Changing the set of actions a collaborator role covers SHALL apply to every principal currently holding that role, on their very next access check, without a data migration.

#### Scenario: Role redefinition takes effect without backfill
- **WHEN** the action set for an existing course collaborator role is changed
- **THEN** every principal currently holding that role, on any course, is granted or denied the changed actions on their next access check

### Requirement: Revoking a collaborator's role immediately ends their course access
Revoking a course collaborator's role binding SHALL immediately remove every action that role granted them on that course.

#### Scenario: Revoke a collaborator
- **WHEN** a course collaborator's role binding on a course is revoked
- **THEN** the principal immediately loses every action that role granted them on that course
