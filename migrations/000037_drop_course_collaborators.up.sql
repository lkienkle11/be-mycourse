-- course_collaborators is fully replaced by the resource-scoped role gate
-- (authorization_role_bindings/authorization_role_actions): the owner is synthesized from
-- courses.owner_user_id by CoursePolicyProvider, and every collaborator's access/display role
-- now comes from an active EDITOR authorization_role_bindings row (backfilled by migration
-- 000036). No code path reads or writes this table anymore (internal/course/**, and
-- internal/appcli's legacy importer, both re-pointed onto the role gate).
DROP TABLE IF EXISTS course_collaborators;
