package system

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mycourse-io-be/dto"
	"mycourse-io-be/internal/appdb"
	jobsystem "mycourse-io-be/internal/jobs/system"
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
	authd.POST("/create-permission-sync-job", jobsystem.HTTPCreatePermissionSyncJob)
	authd.POST("/create-role-permission-sync-job", jobsystem.HTTPCreateRolePermissionSyncJob)
	authd.POST("/delete-permission-sync-job", jobsystem.HTTPStopPermissionSyncJob)
	authd.POST("/delete-role-permission-sync-job", jobsystem.HTTPStopRolePermissionSyncJob)
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
	response.OK(c, "system_login_ok", dto.SystemLoginResponse{
		AccessToken: tok,
		ExpiresIn:   int(systemauth.SystemAccessTokenTTL.Seconds()),
	})
}

func permissionSyncNow(c *gin.Context) {
	n, err := rbacsync.SyncPermissionsFromConstants(appdb.Conn())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "permission_sync_completed", dto.PermissionSyncNowResponse{Synced: n})
}

func rolePermissionSyncNow(c *gin.Context) {
	n, err := rbacsync.SyncRolePermissionsFromConstants(appdb.Conn())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "role_permission_sync_completed", dto.RolePermissionSyncNowResponse{Rows: n})
}
