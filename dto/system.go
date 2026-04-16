package dto

// SystemLoginRequest is JSON for POST /api/system/login.
type SystemLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
