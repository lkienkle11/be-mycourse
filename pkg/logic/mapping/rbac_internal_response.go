package mapping

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
)

// ToRBACPermissionResponse maps a persistence model to the internal RBAC API DTO.
func ToRBACPermissionResponse(p models.Permission) dto.RBACPermissionResponse {
	return dto.RBACPermissionResponse{
		PermissionID:   p.PermissionID,
		PermissionName: p.PermissionName,
		Description:    p.Description,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToRBACPermissionResponses maps a slice of permission models.
func ToRBACPermissionResponses(ps []models.Permission) []dto.RBACPermissionResponse {
	out := make([]dto.RBACPermissionResponse, len(ps))
	for i := range ps {
		out[i] = ToRBACPermissionResponse(ps[i])
	}
	return out
}

// ToRBACPermissionPtrResponse maps a permission pointer (nil-safe).
func ToRBACPermissionPtrResponse(p *models.Permission) *dto.RBACPermissionResponse {
	if p == nil {
		return nil
	}
	v := ToRBACPermissionResponse(*p)
	return &v
}

// ToRBACRoleResponse maps a role model (including preloaded Permissions when present).
func ToRBACRoleResponse(r models.Role) dto.RBACRoleResponse {
	perms := make([]dto.RBACPermissionResponse, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, ToRBACPermissionResponse(p))
	}
	return dto.RBACRoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// ToRBACRoleResponses maps a slice of role models.
func ToRBACRoleResponses(rs []models.Role) []dto.RBACRoleResponse {
	out := make([]dto.RBACRoleResponse, len(rs))
	for i := range rs {
		out[i] = ToRBACRoleResponse(rs[i])
	}
	return out
}

// ToRBACRolePtrResponse maps a role pointer (nil-safe).
func ToRBACRolePtrResponse(r *models.Role) *dto.RBACRoleResponse {
	if r == nil {
		return nil
	}
	v := ToRBACRoleResponse(*r)
	return &v
}

// ToUserRBACPermissionCodesResponse maps sorted permission name strings to the internal RBAC API DTO.
func ToUserRBACPermissionCodesResponse(sortedCodes []string) dto.UserRBACPermissionCodesResponse {
	return dto.UserRBACPermissionCodesResponse{PermissionCodes: sortedCodes}
}
