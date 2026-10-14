## Purpose

Defines the format, generation, preservation, and conflict-resolution rules for the `slug` field on tree nodes stored inside a course topic's `child_topics` or a course skill's `children` JSONB column, so a manual node slug survives writes that do not touch that node, and every node slug remains a safe, unique-within-its-tree identifier.

## ADDED Requirements

### Requirement: Node slug format
A tree node's slug SHALL contain only lowercase ASCII letters `a-z`, digits `0-9`, and hyphens `-`; it SHALL NOT start or end with a hyphen; consecutive hyphens ARE allowed. The system SHALL validate every node slug — whether manually supplied or generated — against the pattern `^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$` before it is persisted.

#### Scenario: Valid manual node slug accepted
- **WHEN** a caller supplies a tree node with slug `advanced-topics`
- **THEN** the node's slug passes format validation

#### Scenario: Invalid manual node slug rejected
- **WHEN** a caller supplies a tree node with a slug containing uppercase letters, non-ASCII characters, or a leading/trailing hyphen
- **THEN** the request is rejected with a `400` validation error and the tree is not persisted

### Requirement: Node creation accepts an optional manual slug
When a tree node is submitted without a corresponding existing node (a brand-new node, identified by an `id` not present in the entity's currently stored tree), the system SHALL accept an optional `slug` for that node. A missing field, `null`, an empty string, or a whitespace-only string (after trimming) SHALL all be treated as "no manual slug supplied for this node" and SHALL NOT produce a validation error.

#### Scenario: New node slug omitted
- **WHEN** a caller adds a new tree node with a `name` but no `slug`
- **THEN** the system auto-generates the new node's slug from its `name`, reusing the same name-to-slug transliteration and ASCII-fallback rule as taxonomy root entities, prefixed `node-{randomSuffix}` when the name yields no usable ASCII slug

### Requirement: Existing node slug is preserved when omitted
When a write includes a tree node whose `id` matches a node already present in the entity's currently stored tree, and that write's node object omits the `slug` field, the system SHALL preserve that node's existing slug unchanged — including when the write changes that node's `name`, when the write changes unrelated fields on the entity (status, other nodes), or when the write is a metadata-only update that reloads the tree without the caller ever supplying it in the request body.

#### Scenario: Renaming an existing node does not change its slug
- **WHEN** a caller updates an existing tree node's `name` but omits `slug` for that node
- **THEN** the node's stored slug is unchanged after the write

#### Scenario: Metadata-only update does not touch node slugs
- **WHEN** a caller updates a course topic's `status` without including `child_topics` in the request body at all
- **THEN** every existing node's slug in the stored tree is unchanged after the write

### Requirement: Existing node slug explicitly blanked is rejected
When a write includes a tree node whose `id` matches an existing node, and that node's `slug` field is present but set to `null`, an empty string, or a whitespace-only string, the system SHALL reject the entire write with a `400` validation error identifying which node caused it, rather than silently regenerating or clearing that node's slug.

#### Scenario: Explicit empty slug on an existing node is rejected
- **WHEN** a caller submits an existing node with `slug: ""` or `slug: null`
- **THEN** the request is rejected with a `400` error identifying the offending node, and no part of the write is applied

### Requirement: Node slug uniqueness is scoped to its own tree
Two nodes SHALL NOT share the same slug within the same entity's tree (across all depths of that one `child_topics`/`children` structure), regardless of which parent they are nested under. Node slug uniqueness is NOT enforced across different entities' trees, and is NOT limited to only direct siblings.

#### Scenario: Two nodes in the same tree collide
- **WHEN** a write would result in two nodes anywhere in the same course topic's `child_topics` tree having the same slug
- **THEN** the system does not persist that state unresolved (see "In-tree node slug conflict is auto-resolved")

#### Scenario: Same slug in two different entities' trees is allowed
- **WHEN** a course topic's tree and a different course topic's tree each contain a node with slug `basics`
- **THEN** neither write is rejected on that basis; node slug uniqueness only applies within one entity's own tree

### Requirement: In-tree node slug conflict is auto-resolved
When resolving a tree for write would otherwise produce two nodes with the same slug, the system SHALL automatically resolve the conflict by appending a random suffix to one of the colliding slugs (retrying with escalating suffix length until unique within that tree) rather than rejecting the write, consistent with taxonomy root's update-time auto-resolve behavior.

#### Scenario: Two new nodes derive the same slug from similar names
- **WHEN** two new sibling-or-cousin nodes in the same write both derive the same base slug from their names
- **THEN** the system persists one of them with an escalating random suffix appended instead of rejecting the request

### Requirement: A node slug with an appended suffix never exceeds the storage limit
When appending a random suffix to a node's base slug, the system SHALL ensure the resulting combined slug never exceeds the maximum node slug length already enforced by this module's tree validation, truncating the base portion as needed.

#### Scenario: Long node name conflicts and needs a suffix
- **WHEN** a new node's auto-generated base slug is near the maximum node slug length and collides with another node's slug in the same tree
- **THEN** the resolved slug truncates the base portion as needed so the final value including the appended suffix still fits within the maximum length

### Requirement: Every response reflects the final stored node slugs
Any API response for an operation that creates or updates a course topic or course skill SHALL include, for every tree node in the response, the actual final slug as stored in the database — not the slug value as submitted in the request, and not a value recomputed independently of what was actually persisted.

#### Scenario: Response reflects auto-resolved node slug
- **WHEN** a write triggers in-tree node slug auto-resolution due to a collision
- **THEN** the response body's tree includes the newly resolved slug for that node, not the value the caller submitted or omitted
