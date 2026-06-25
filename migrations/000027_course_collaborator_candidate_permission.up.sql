INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P67', 'course_collaborator_candidate:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('sysadmin', 'admin', 'instructor')
  AND p.permission_id = 'P67'
ON CONFLICT DO NOTHING;
