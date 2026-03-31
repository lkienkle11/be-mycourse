-- So seeded `admin` can call GET /me/permissions (perm profile.read → CodeProfileRead.CourseRead value).

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.code = 'profile.read'
ON CONFLICT (role_id, permission_id) DO NOTHING;
