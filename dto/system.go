package dto

// SystemLoginRequest is JSON for POST /api/system/login.
type SystemLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SystemLoginResponse is the JSON data for a successful system login.
type SystemLoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// PermissionSyncNowResponse is the data for POST /api/system/permission-sync-now.
type PermissionSyncNowResponse struct {
	Synced int `json:"synced"`
}

// RolePermissionSyncNowResponse is the data for POST /api/system/role-permission-sync-now.
type RolePermissionSyncNowResponse struct {
	Rows int `json:"rows"`
}
