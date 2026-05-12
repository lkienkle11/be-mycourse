package infra

// Raw RBAC SQL templates with %s placeholders for table names (filled in services/rbac init via dbschema).
const (
	RbacSQLPermissionCodesForUserTmpl = `
SELECT DISTINCT cc FROM (
  SELECT p.permission_name AS cc
  FROM %s ur
  INNER JOIN %s rp ON rp.role_id = ur.role_id
  INNER JOIN %s p ON p.permission_id = rp.permission_id
  WHERE ur.user_id = :user_id
  UNION
  SELECT p.permission_name
  FROM %s up
  INNER JOIN %s p ON p.permission_id = up.permission_id
  WHERE up.user_id = :user_id
) AS _
`
	RbacSQLDeleteRolePermissionsByPermissionIDTmpl = `
DELETE FROM %s WHERE permission_id = :permission_id
`
	RbacSQLDeleteRolePermissionsByRoleIDTmpl = `
DELETE FROM %s WHERE role_id = :role_id
`
	RbacSQLDeleteUserPermissionsByPermissionIDTmpl = `
DELETE FROM %s WHERE permission_id = :permission_id
`
)
