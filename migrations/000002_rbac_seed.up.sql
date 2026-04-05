-- Catalog and roles aligned with constants/permissions.go and constants/roles.go.
-- learner / teacher / creator / admin permission matrix. Idempotent.

INSERT INTO permissions (code, code_check, description, created_at, updated_at)
VALUES
    ('course.create', 'course:create', '', NOW(), NOW()),
    ('course.delete', 'course:delete', '', NOW(), NOW()),
    ('course.read', 'course:read', '', NOW(), NOW()),
    ('course.update', 'course:update', '', NOW(), NOW()),
    ('course.write', 'course:write', '', NOW(), NOW()),
    ('profile.course.write', 'profile:course:write', '', NOW(), NOW()),
    ('profile.course.read', 'profile:course:read', 'Read own profile', NOW(), NOW()),
    ('rbac.manage', 'rbac:manage', 'Manage roles, permissions, and user-role assignments', NOW(), NOW())
ON CONFLICT (code) DO UPDATE
    SET code_check = EXCLUDED.code_check,
        description = EXCLUDED.description,
        updated_at = NOW();

INSERT INTO roles (name, description, created_at, updated_at)
VALUES
    ('admin', 'Full RBAC administration', NOW(), NOW()),
    ('creator', 'Course author — create and manage courses', NOW(), NOW()),
    ('teacher', 'Instructor — teach and update course content', NOW(), NOW()),
    ('learner', 'Student — consume enrolled content', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'creator'
  AND p.code IN (
    'profile.course.read',
    'profile.course.write',
    'course.read',
    'course.write',
    'course.update',
    'course.create',
    'course.delete'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'teacher'
  AND p.code IN (
    'profile.course.read',
    'profile.course.write',
    'course.read',
    'course.write',
    'course.update'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'learner'
  AND p.code IN ('profile.course.read', 'course.read')
ON CONFLICT (role_id, permission_id) DO NOTHING;
