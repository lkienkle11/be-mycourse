-- One-time backfill (dev data only, per the openspec change's non-goal on a production
-- migration strategy): insert one authorization_role_bindings row per active EDITOR
-- course_collaborators row, so Course's cutover onto the role gate (see
-- internal/course/infra/repo_access.go) does not lose access for any existing collaborator.
--
-- OWNER-role course_collaborators rows are intentionally excluded: the course owner's access
-- is synthesized directly from courses.owner_user_id by CoursePolicyProvider, never from a
-- stored role binding (see design.md's "course owner is a synthesized grant" decision) — an
-- OWNER-role course_collaborators row is a display-only artifact, not an access grant.
INSERT INTO authorization_role_bindings (id, principal_user_id, role_name, resource_type, resource_id, granted_by_user_id, created_at, revoked_at)
SELECT gen_random_uuid(), cc.user_id, 'EDITOR', 'course', cc.course_id, NULL, cc.created_at, NULL
FROM course_collaborators cc
WHERE cc.deleted_at IS NULL AND cc.role <> 'OWNER';
