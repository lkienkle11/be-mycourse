-- Baseline permissions and admin role (idempotent)

INSERT INTO permissions (code, description, created_at, updated_at)
VALUES
    ('rbac.manage', 'Manage roles, permissions, and user-role assignments', NOW(), NOW()),
    ('profile.read', 'Read own profile', NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO roles (name, description, created_at, updated_at)
VALUES ('admin', 'Full RBAC administration', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.code = 'rbac.manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;
