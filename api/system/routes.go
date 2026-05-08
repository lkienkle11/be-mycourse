package system

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/internal/appdb"
	"mycourse-io-be/internal/rbacsync"
	"mycourse-io-be/internal/systemauth"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/errcode"
	pkgerrors "mycourse-io-be/pkg/errors"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services"
)

// RegisterRoutes wires /api/system/* (caller mounts the group at /api/system).
// Uses internal/appdb (set from main after models.Setup); api/ does not import models or database drivers.
func RegisterRoutes(g *gin.RouterGroup) {
	db := appdb.Conn()
	if g == nil || db == nil {
		return
	}
	g.POST("/login", systemLogin)

	authd := g.Group("")
	authd.Use(middleware.RequireSystemAccessToken(db))
	authd.POST("/permission-sync-now", permissionSyncNow)
	authd.POST("/role-permission-sync-now", rolePermissionSyncNow)
	authd.POST("/create-permission-sync-job", startPermissionSyncScheduler)
	authd.POST("/create-role-permission-sync-job", startRolePermissionSyncScheduler)
	authd.POST("/delete-permission-sync-job", stopPermissionSyncScheduler)
	authd.POST("/delete-role-permission-sync-job", stopRolePermissionSyncScheduler)
}

func systemLogin(c *gin.Context) {
	var req dto.SystemLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}
	tok, err := services.SystemLogin(appdb.Conn(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, pkgerrors.ErrSystemLoginFailed):
			response.Fail(c, http.StatusUnauthorized, errcode.InvalidCredentials, errcode.DefaultMessage(errcode.InvalidCredentials), nil)
		case errors.Is(err, pkgerrors.ErrSystemSecretsNotReady):
			response.Fail(c, http.StatusServiceUnavailable, errcode.InternalError, "system token secrets are not configured", nil)
		default:
			response.Fail(c, http.StatusInternalServerError, errcode.InternalError, errcode.DefaultMessage(errcode.InternalError), nil)
		}
		return
	}
	response.OK(c, "system_login_ok", gin.H{
		"access_token": tok,
		"expires_in":   int(systemauth.SystemAccessTokenTTL.Seconds()),
	})
}

func permissionSyncNow(c *gin.Context) {
	n, err := rbacsync.SyncPermissionsFromConstants(appdb.Conn())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "permission_sync_completed", gin.H{"synced": n})
}

func rolePermissionSyncNow(c *gin.Context) {
	n, err := rbacsync.SyncRolePermissionsFromConstants(appdb.Conn())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "role_permission_sync_completed", gin.H{"rows": n})
}

func startPermissionSyncScheduler(c *gin.Context) {
	services.StartPermissionSyncScheduler(appdb.Conn())
	response.OK(c, "permission_sync_job_started", nil)
}

func startRolePermissionSyncScheduler(c *gin.Context) {
	services.StartRolePermissionSyncScheduler(appdb.Conn())
	response.OK(c, "role_permission_sync_job_started", nil)
}

func stopPermissionSyncScheduler(c *gin.Context) {
	services.StopPermissionSyncScheduler()
	response.OK(c, "permission_sync_job_stopped", nil)
}

func stopRolePermissionSyncScheduler(c *gin.Context) {
	services.StopRolePermissionSyncScheduler()
	response.OK(c, "role_permission_sync_job_stopped", nil)
}
