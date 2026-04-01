package dto

// PermissionFilter is the query-param DTO for GET /internal-v1/rbac/permissions.
//
// Sortable fields : id, code, description, created_at
// Searchable fields: code, description
type PermissionFilter struct {
	BaseFilter // required embed — provides page, per_page, sort_by, sort_order, search_by, search_data
}

type CreatePermissionRequest struct {
	Code        string  `json:"code" binding:"required,min=1,max=128"`
	CodeCheck   *string `json:"code_check"`
	Description string  `json:"description" binding:"omitempty,max=512"`
}

type UpdatePermissionRequest struct {
	Code        *string `json:"code"`
	CodeCheck   *string `json:"code_check"`
	Description *string `json:"description"`
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type SetRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" binding:"required"`
}

type AssignUserRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

type AssignUserPermissionRequest struct {
	PermissionID *uint  `json:"permission_id"`
	Code         string `json:"permission_code"`
}
