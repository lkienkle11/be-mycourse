package delivery

import "mycourse-io-be/internal/rbac/domain" //nolint:depguard // delivery maps domain.Permission/Role entities to DTOs; pure data transformation

func toPermissionResponse(p domain.Permission) PermissionResponse {
	return PermissionResponse{
		PermissionID:   p.PermissionID,
		PermissionName: p.PermissionName,
		Description:    p.Description,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toPermissionResponses(ps []domain.Permission) []PermissionResponse {
	out := make([]PermissionResponse, len(ps))
	for i, p := range ps {
		out[i] = toPermissionResponse(p)
	}
	return out
}

func toPermissionResponsePtr(p *domain.Permission) *PermissionResponse {
	if p == nil {
		return nil
	}
	r := toPermissionResponse(*p)
	return &r
}

func toRoleResponse(r domain.Role) RoleResponse {
	resp := RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if len(r.Permissions) > 0 {
		resp.Permissions = toPermissionResponses(r.Permissions)
	}
	return resp
}

func toRoleResponses(roles []domain.Role) []RoleResponse {
	out := make([]RoleResponse, len(roles))
	for i, r := range roles {
		out[i] = toRoleResponse(r)
	}
	return out
}

func toRoleResponsePtr(r *domain.Role) *RoleResponse {
	if r == nil {
		return nil
	}
	v := toRoleResponse(*r)
	return &v
}
