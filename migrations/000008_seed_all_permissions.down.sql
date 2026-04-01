-- Remove only the permissions inserted by this migration.
-- Existing data from earlier seeds (000002) is intentionally left untouched.

DELETE FROM permissions
WHERE code IN (
    'course_create',
    'course_delete',
    'course_read',
    'course_update',
    'course_write',
    'profile.course_write',
    'profile.read',
    'rbac.manage'
);
