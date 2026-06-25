INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.permission_id = 'P67'
ON CONFLICT DO NOTHING;
