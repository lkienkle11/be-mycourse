package dto

type CreatePermissionRequest struct {
	Code        string `json:"code" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"omitempty,max=512"`
}

type UpdatePermissionRequest struct {
	Code        *string `json:"code"`
	Description *string `json:"description"`
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParentID    *uint  `json:"parent_id,omitempty"`
}

type UpdateRoleRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	ParentID     *uint   `json:"parent_id,omitempty"`
	RemoveParent bool    `json:"remove_parent,omitempty"`
}

type SetRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required"`
}

type AssignUserRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}
