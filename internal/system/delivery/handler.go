// Package delivery contains the HTTP handlers for the SYSTEM bounded context.
package delivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/system/application"
	"mycourse-io-be/internal/system/jobs" //nolint:depguard // system delivery controls background job lifecycle (start/stop schedulers); control-plane responsibility
)

// --- DTOs --------------------------------------------------------------------

type systemLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type systemLoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type permissionSyncResponse struct {
	Synced int `json:"synced"`
}

type rolePermissionSyncResponse struct {
	Rows int `json:"rows"`
}

// --- Handler -----------------------------------------------------------------

// Handler holds HTTP handler methods for the SYSTEM domain.
type Handler struct {
	svc *application.SystemService
}

// NewHandler constructs a SYSTEM delivery Handler.
func NewHandler(svc *application.SystemService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) systemLogin(c *gin.Context) {
	var req systemLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, apperrors.ValidationFailed, err.Error(), nil)
		return
	}
	tok, err := h.svc.SystemLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrSystemLoginFailed):
			response.Fail(c, http.StatusUnauthorized, apperrors.InvalidCredentials, apperrors.DefaultMessage(apperrors.InvalidCredentials), nil)
		case errors.Is(err, apperrors.ErrSystemSecretsNotReady):
			response.Fail(c, http.StatusServiceUnavailable, apperrors.InternalError, "system token secrets are not configured", nil)
		default:
			response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, apperrors.DefaultMessage(apperrors.InternalError), nil)
		}
		return
	}
	response.OK(c, "system_login_ok", systemLoginResponse{
		AccessToken: tok,
		ExpiresIn:   application.SystemAccessTokenTTL(),
	})
}

func (h *Handler) permissionSyncNow(c *gin.Context) {
	n, err := h.svc.SyncPermissions(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "permission_sync_completed", permissionSyncResponse{Synced: n})
}

func (h *Handler) rolePermissionSyncNow(c *gin.Context) {
	n, err := h.svc.SyncRolePermissions(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "role_permission_sync_completed", rolePermissionSyncResponse{Rows: n})
}

func (h *Handler) createPermissionSyncJob(c *gin.Context) {
	jobs.StartPermissionSyncJob(h.svc)
	response.OK(c, "permission_sync_job_started", nil)
}

func (h *Handler) createRolePermissionSyncJob(c *gin.Context) {
	jobs.StartRolePermissionSyncJob(h.svc)
	response.OK(c, "role_permission_sync_job_started", nil)
}

func (h *Handler) stopPermissionSyncJob(c *gin.Context) {
	jobs.StopPermissionSyncJob()
	response.OK(c, "permission_sync_job_stopped", nil)
}

func (h *Handler) stopRolePermissionSyncJob(c *gin.Context) {
	jobs.StopRolePermissionSyncJob()
	response.OK(c, "role_permission_sync_job_stopped", nil)
}
