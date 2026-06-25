DELETE FROM role_permissions
WHERE permission_id = 'P67'
  AND role_id IN (SELECT id FROM roles WHERE name = 'admin');
