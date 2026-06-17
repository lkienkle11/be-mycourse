INSERT INTO permissions (permission_id, permission_name, description, created_at, updated_at)
VALUES
    ('P59', 'course_review:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P60', 'course_review:approve', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P61', 'course_review:reject', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P62', 'course_catalog:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P63', 'course_catalog:trash', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P64', 'course_trash:read', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P65', 'course_trash:restore', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT),
    ('P66', 'course_trash:delete', '', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
ON CONFLICT (permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.permission_id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('sysadmin', 'admin')
  AND p.permission_id BETWEEN 'P59' AND 'P66'
ON CONFLICT DO NOTHING;
