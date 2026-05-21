-- Role modify permissions (P38–P40): scoped admin actions per role tier.

INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P38', 'sysadmin:modify', '', NOW(), NOW()),
    ('P39', 'admin:modify', '', NOW(), NOW()),
    ('P40', 'instructor:modify', '', NOW(), NOW())
ON CONFLICT (permission_id) DO UPDATE
SET permission_name = EXCLUDED.permission_name,
    updated_at = NOW();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P38', 'P39', 'P40')
WHERE r.name = 'sysadmin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id IN ('P39', 'P40')
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
INNER JOIN permissions p ON p.permission_id = 'P40'
WHERE r.name = 'instructor'
ON CONFLICT DO NOTHING;
