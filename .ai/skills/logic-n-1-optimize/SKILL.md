# Backend Skill: Batch API Design for Multi-Item Operations

## Purpose

When implementing backend APIs that add, update, delete, assign, unassign, or otherwise modify multiple resources, the agent must design the API to support batch operations instead of requiring the client to call the same API repeatedly.

This rule applies to any programming language, framework, or backend architecture.

## Core Rule

Do not design or implement workflows where the frontend, client, or caller must repeatedly call a single-item API in a loop to perform a multi-item operation.

Instead, create a batch-capable API endpoint that accepts multiple items in one request and processes them safely, consistently, and efficiently.

## Examples

### Bad Pattern

The client calls the API repeatedly:

```http
POST /users/123/roles/admin
POST /users/123/roles/editor
POST /users/123/roles/viewer
```

Or:

```http
DELETE /objects/1
DELETE /objects/2
DELETE /objects/3
DELETE /objects/4
```

This is not allowed for multi-item operations.

## Required Pattern

Design batch APIs such as:

```http
POST /users/123/roles/batch
```

```json
{
  "roleIds": ["admin", "editor", "viewer"]
}
```

Or:

```http
DELETE /objects/batch
```

```json
{
  "ids": ["1", "2", "3", "4"]
}
```

## Common Batch Use Cases

Use batch API design for operations such as:

* Adding multiple roles to a user
* Removing multiple roles from a user
* Assigning multiple users to a role
* Removing multiple users from a role
* Creating multiple objects
* Updating multiple objects
* Deleting multiple objects
* Attaching or detaching multiple resources
* Reordering multiple records
* Updating many-to-many relationships
* Applying bulk status changes
* Any operation where the user action naturally affects more than one item

## API Design Requirements

A batch endpoint should:

1. Accept a list or array of items in the request body.
2. Validate all input before making changes whenever possible.
3. Use a database transaction when the operation must be atomic.
4. Avoid partial writes unless partial success is explicitly required.
5. Return a clear result for the whole operation.
6. Return item-level errors when partial success is supported.
7. Be idempotent where possible.
8. Avoid N+1 database queries.
9. Avoid calling internal APIs repeatedly when direct service/repository/database batch logic is available.
10. Respect authorization for every affected resource.

## Atomic Batch Operations

Use atomic behavior when all changes must succeed or fail together.

Example:

```json
{
  "userId": "user_123",
  "roleIds": ["role_admin", "role_editor"]
}
```

Expected behavior:

* If all role IDs are valid, assign all roles.
* If one role ID is invalid, assign none.
* Return a validation error.

Example response:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_ROLE_ID",
    "message": "One or more role IDs are invalid.",
    "invalidIds": ["role_unknown"]
  }
}
```

## Partial Success Batch Operations

Use partial success only when the product requirement explicitly allows some items to succeed and others to fail.

Example response:

```json
{
  "success": true,
  "summary": {
    "total": 4,
    "succeeded": 3,
    "failed": 1
  },
  "results": [
    {
      "id": "1",
      "status": "success"
    },
    {
      "id": "2",
      "status": "success"
    },
    {
      "id": "3",
      "status": "failed",
      "error": {
        "code": "NOT_FOUND",
        "message": "Object not found."
      }
    },
    {
      "id": "4",
      "status": "success"
    }
  ]
}
```

## Naming Guidelines

Prefer clear endpoint names that express batch behavior.

Good examples:

```http
POST /users/{userId}/roles/batch
DELETE /users/{userId}/roles/batch
POST /roles/{roleId}/users/batch
PATCH /objects/batch
DELETE /objects/batch
POST /objects/bulk-create
PATCH /objects/bulk-update
POST /objects/bulk-delete
```

Avoid ambiguous endpoints that hide multi-item behavior.

## Service Layer Requirement

The backend service layer must also support batch processing directly.

Do not implement a batch endpoint by simply looping over an existing single-item controller/API method.

Bad:

```pseudo
for item in items:
    callSingleItemApi(item)
```

Better:

```pseudo
validateAll(items)
authorizeAll(items)
beginTransaction()
batchUpdate(items)
commitTransaction()
```

## Database Requirement

When possible, use database-level batch operations.

Examples:

```sql
INSERT INTO user_roles (user_id, role_id)
VALUES
  (?, ?),
  (?, ?),
  (?, ?)
ON CONFLICT DO NOTHING;
```

```sql
DELETE FROM user_roles
WHERE user_id = ?
AND role_id IN (?, ?, ?);
```

```sql
UPDATE objects
SET status = ?
WHERE id IN (?, ?, ?);
```

Avoid issuing one database query per item unless there is a strong reason.

## Validation Rules

Before performing the batch operation, check:

* The request list is not empty.
* The request list does not exceed the configured maximum batch size.
* IDs are unique when duplicates are not meaningful.
* All referenced resources exist.
* The current user has permission for every affected resource.
* The operation does not violate business rules.
* The operation is safe to retry when possible.

## Batch Size Limit

Every batch API should define a maximum allowed item count.

Example:

```json
{
  "maxBatchSize": 100
}
```

If the request exceeds the limit, return a clear error:

```json
{
  "success": false,
  "error": {
    "code": "BATCH_SIZE_EXCEEDED",
    "message": "Maximum batch size is 100 items."
  }
}
```

## Idempotency

Batch APIs should be idempotent where practical.

Examples:

* Adding an already-assigned role should not create duplicates.
* Removing a role that is already absent may be treated as success if the final state is correct.
* Deleting already-deleted items should follow the project’s existing deletion semantics.

For critical operations, support an idempotency key if needed:

```http
Idempotency-Key: 7f3d3e9a-1234-4567-9000-abc123
```

## Authorization

Authorization must be checked for every affected item.

Never assume permission for all items just because the user has permission for one item.

For example, when deleting multiple objects, the backend must verify that the requester can delete each object.

## Response Design

A batch API response should clearly communicate:

* Whether the operation succeeded.
* How many items were affected.
* Which items failed, if partial success is supported.
* Any validation or permission errors.
* The final state when useful.

Example:

```json
{
  "success": true,
  "affectedCount": 3,
  "data": {
    "userId": "user_123",
    "roleIds": ["role_admin", "role_editor", "role_viewer"]
  }
}
```

## Documentation Requirement

When documenting APIs, explicitly mention when an endpoint supports batch operations.

Documentation should include:

* Endpoint path
* Request body
* Maximum batch size
* Atomic or partial-success behavior
* Example request
* Example success response
* Example error response
* Authorization rules

## Testing Requirement

Batch APIs must include tests for:

* Successful batch operation
* Empty list
* Duplicate IDs
* Invalid IDs
* Unauthorized item
* Batch size exceeded
* Partial failure, if supported
* Transaction rollback, if atomic
* Idempotent retry behavior, when applicable

## Final Instruction

Whenever the requested feature involves adding, updating, deleting, assigning, or unassigning multiple items, the agent must propose and implement a batch API design.

The agent must not instruct the client to call the same single-item API repeatedly.

The correct default is:

```text
One user action affecting many resources = one batch API request.
```
