package dbschema

// rbacNames holds RBAC table names (one place to edit).
const (
	rbacPermissions     = "permissions"
	rbacRoles             = "roles"
	rbacRolePermissions   = "role_permissions"
	rbacUserRoles         = "user_roles"
	rbacRoleClosure       = "role_closure"
)

// RBAC exposes table names for authorization / role models and raw SQL.
// Prefer RBAC.Permissions() (and siblings) so all RBAC names stay in this file.
var RBAC rbacNS

type rbacNS struct{}

func (rbacNS) Permissions() string      { return rbacPermissions }
func (rbacNS) Roles() string            { return rbacRoles }
func (rbacNS) RolePermissions() string  { return rbacRolePermissions }
func (rbacNS) UserRoles() string        { return rbacUserRoles }
func (rbacNS) RoleClosure() string      { return rbacRoleClosure }
