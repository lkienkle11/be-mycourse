package delivery

import "time"

// PermissionFilterRequest is the query-param DTO for listing permissions.
type PermissionFilterRequest struct {
	Page       int    `form:"page"`
	PerPage    int    `form:"per_page"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order"`
	SearchBy   string `form:"search_by"`
	SearchData string `form:"search_data"`
}

func (f *PermissionFilterRequest) page() int {
	if f.Page < 1 {
		return 1
	}
	return f.Page
}
func (f *PermissionFilterRequest) perPage() int {
	if f.PerPage < 1 {
		return 20
	}
	return f.PerPage
}

// CreatePermissionRequest is the JSON body for POST /rbac/permissions.
type CreatePermissionRequest struct {
	PermissionID   string `json:"permission_id" binding:"required,min=1,max=10"`
	PermissionName string `json:"permission_name" binding:"required,min=1,max=50"`
	Description    string `json:"description" binding:"omitempty,max=512"`
}

// UpdatePermissionRequest is the JSON body for PATCH /rbac/permissions/:id.
type UpdatePermissionRequest struct {
	PermissionName *string `json:"permission_name" binding:"omitempty,min=1,max=50"`
	Description    *string `json:"description"`
}

// CreateRoleRequest is the JSON body for POST /rbac/roles.
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRoleRequest is the JSON body for PATCH /rbac/roles/:id.
type UpdateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// SetRolePermissionsRequest is the JSON body for PUT /rbac/roles/:id/permissions.
type SetRolePermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

// AssignUserRoleRequest is the JSON body for POST /rbac/users/:userId/roles.
type AssignUserRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

// AssignUserPermissionRequest is the JSON body for POST /rbac/users/:userId/direct-permissions.
type AssignUserPermissionRequest struct {
	PermissionID   *string `json:"permission_id" binding:"omitempty,max=10"`
	PermissionName *string `json:"permission_name" binding:"omitempty,max=50"`
}

// PermissionResponse is the JSON response for a single permission.
type PermissionResponse struct {
	PermissionID   string    `json:"permission_id"`
	PermissionName string    `json:"permission_name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RoleResponse is the JSON response for a role (permissions optionally included).
type RoleResponse struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Permissions []PermissionResponse `json:"permissions,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// UserPermissionCodesResponse lists effective permission codes for a user.
type UserPermissionCodesResponse struct {
	PermissionCodes []string `json:"permission_codes"`
}
