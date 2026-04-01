-- Full upsert of every permission entry defined in constants/permissions.go.
-- Mirrors what cmd/syncpermissions does but as a migration so the DB is
-- bootstrapped even without running the sync command separately.
-- code  = perm struct-tag value  (stable API key)
-- code_check = field string value (runtime check key used in RequirePermission)

INSERT INTO permissions (code, code_check, description, created_at, updated_at)
VALUES
    ('course_create',        'course_create',              '', NOW(), NOW()),
    ('course_delete',        'course_delete',              '', NOW(), NOW()),
    ('course_read',          'course_read',                '', NOW(), NOW()),
    ('course_update',        'course_update',              '', NOW(), NOW()),
    ('course_write',         'course_write',               '', NOW(), NOW()),
    ('profile.course_write', 'profile_read_course_write',  '', NOW(), NOW()),
    ('profile.read',         'profile_read',               '', NOW(), NOW()),
    ('rbac.manage',          'rbac_manage',                '', NOW(), NOW())
ON CONFLICT (code) DO UPDATE
    SET code_check = EXCLUDED.code_check,
        updated_at = NOW();
