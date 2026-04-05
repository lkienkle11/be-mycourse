-- Removes seeded RBAC rows. Also clears user role assignments (user_roles / user_permissions).

DELETE FROM user_roles;
DELETE FROM user_permissions;
DELETE FROM role_permissions;
DELETE FROM roles;
DELETE FROM permissions;
