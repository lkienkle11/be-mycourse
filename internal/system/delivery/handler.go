// Package delivery contains the HTTP handlers for the SYSTEM bounded context.
package delivery

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/system/application"
	"mycourse-io-be/internal/system/jobs"
)

// --- DTOs --------------------------------------------------------------------

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

func (h *Handler) permissionSyncNow(c *gin.Context) {
	runSyncNow(c, h.svc.SyncPermissions, "permission_sync_completed", func(n int) any {
		return permissionSyncResponse{Synced: n}
	})
}

func (h *Handler) rolePermissionSyncNow(c *gin.Context) {
	runSyncNow(c, h.svc.SyncRolePermissions, "role_permission_sync_completed", func(n int) any {
		return rolePermissionSyncResponse{Rows: n}
	})
}

func runSyncNow(c *gin.Context, syncFn func(context.Context) (int, error), okMsg string, resp func(int) any) {
	n, err := syncFn(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, apperrors.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, okMsg, resp(n))
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
