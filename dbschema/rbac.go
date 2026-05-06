package dbschema

import "mycourse-io-be/constants"

// RBAC exposes table names for authorization / role models and raw SQL.
// Prefer RBAC.Permissions() (and siblings); literals live in constants/dbschema_name.go.
var RBAC rbacNS

type rbacNS struct{}

func (rbacNS) Permissions() string { return constants.TableRBACPermissions }

func (rbacNS) Roles() string { return constants.TableRBACRoles }

func (rbacNS) RolePermissions() string { return constants.TableRBACRolePermissions }

func (rbacNS) UserRoles() string { return constants.TableRBACUserRoles }

func (rbacNS) UserPermissions() string { return constants.TableRBACUserPermissions }
