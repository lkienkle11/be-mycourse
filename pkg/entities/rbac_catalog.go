package entities

type PermissionCatalogEntry struct {
	PermissionID   string
	PermissionName string
}

type RolePermissionPair struct {
	RoleName string
	PermID   string
}
