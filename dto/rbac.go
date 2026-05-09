package dto

import "time"

// PermissionFilter is the query-param DTO for GET /internal-v1/rbac/permissions.
//
// Sortable fields : permission_id, permission_name, description, created_at
// Searchable fields: permission_id, permission_name, description
type PermissionFilter struct {
	BaseFilter // required embed — provides page, per_page, sort_by, sort_order, search_by, search_data
}

type CreatePermissionRequest struct {
	PermissionID   string `json:"permission_id" binding:"required,min=1,max=10"`
	PermissionName string `json:"permission_name" binding:"required,min=1,max=50"`
	Description    string `json:"description" binding:"omitempty,max=512"`
}

type UpdatePermissionRequest struct {
	PermissionID   *string `json:"permission_id" binding:"omitempty,min=1,max=10"`
	PermissionName *string `json:"permission_name" binding:"omitempty,min=1,max=50"`
	Description    *string `json:"description"`
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
	PermissionIDs []string `json:"permission_ids"`
}

type AssignUserRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

type AssignUserPermissionRequest struct {
	PermissionID   *string `json:"permission_id" binding:"omitempty,max=10"`
	PermissionName *string `json:"permission_name" binding:"omitempty,max=50"`
}

// RBACPermissionResponse is a permission row returned by internal RBAC JSON APIs.
type RBACPermissionResponse struct {
	PermissionID   string    `json:"permission_id"`
	PermissionName string    `json:"permission_name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RBACRoleResponse is a role row (optionally with nested permissions) for internal RBAC JSON APIs.
type RBACRoleResponse struct {
	ID          uint                     `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Permissions []RBACPermissionResponse `json:"permissions,omitempty"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// UserRBACPermissionCodesResponse lists effective permission name strings for a user (internal API).
type UserRBACPermissionCodesResponse struct {
	PermissionCodes []string `json:"permission_codes"`
}
