DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'admin')
  AND permission_id IN (SELECT id FROM permissions WHERE code = 'rbac.manage');

DELETE FROM roles WHERE name = 'admin';

DELETE FROM permissions WHERE code IN ('rbac.manage', 'profile.read');
