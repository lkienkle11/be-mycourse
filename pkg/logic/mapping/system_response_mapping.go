package mapping

import "mycourse-io-be/dto"

// ToSystemLoginResponse maps a system access token to the public system-login payload.
func ToSystemLoginResponse(accessToken string, expiresIn int) dto.SystemLoginResponse {
	return dto.SystemLoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	}
}

// ToPermissionSyncNowResponse wraps the permission sync row count.
func ToPermissionSyncNowResponse(synced int) dto.PermissionSyncNowResponse {
	return dto.PermissionSyncNowResponse{Synced: synced}
}

// ToRolePermissionSyncNowResponse wraps the role-permission sync row count.
func ToRolePermissionSyncNowResponse(rows int) dto.RolePermissionSyncNowResponse {
	return dto.RolePermissionSyncNowResponse{Rows: rows}
}
