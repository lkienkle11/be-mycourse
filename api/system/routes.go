package system

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mycourse-io-be/dto"
	"mycourse-io-be/internal/jobs"
	"mycourse-io-be/internal/rbacsync"
	"mycourse-io-be/internal/systemauth"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/errcode"
	"mycourse-io-be/pkg/response"
	"mycourse-io-be/services"
)

// RegisterRoutes wires /api/system/* (caller mounts the group at /api/system).
func RegisterRoutes(g *gin.RouterGroup, db *gorm.DB) {
	if g == nil || db == nil {
		return
	}

	g.POST("/login", func(c *gin.Context) { systemLogin(c, db) })

	authd := g.Group("")
	authd.Use(middleware.RequireSystemAccessToken(db))
	authd.POST("/permission-sync-now", func(c *gin.Context) { permissionSyncNow(c, db) })
	authd.POST("/role-permission-sync-now", func(c *gin.Context) { rolePermissionSyncNow(c, db) })
	authd.POST("/create-permission-sync-job", func(c *gin.Context) { createPermissionSyncJob(c, db) })
	authd.POST("/create-role-permission-sync-job", func(c *gin.Context) { createRolePermissionSyncJob(c, db) })
	authd.POST("/delete-permission-sync-job", deletePermissionSyncJob)
	authd.POST("/delete-role-permission-sync-job", deleteRolePermissionSyncJob)
}

func systemLogin(c *gin.Context, db *gorm.DB) {
	var req dto.SystemLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, errcode.ValidationFailed, err.Error(), nil)
		return
	}
	tok, err := services.SystemLogin(db, req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSystemLoginFailed):
			response.Fail(c, http.StatusUnauthorized, errcode.InvalidCredentials, errcode.DefaultMessage(errcode.InvalidCredentials), nil)
		case errors.Is(err, services.ErrSystemSecretsNotReady):
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

func permissionSyncNow(c *gin.Context, db *gorm.DB) {
	n, err := rbacsync.SyncPermissionsFromConstants(db)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "permission_sync_completed", gin.H{"synced": n})
}

func rolePermissionSyncNow(c *gin.Context, db *gorm.DB) {
	n, err := rbacsync.SyncRolePermissionsFromConstants(db)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, errcode.InternalError, err.Error(), nil)
		return
	}
	response.OK(c, "role_permission_sync_completed", gin.H{"rows": n})
}

func createPermissionSyncJob(c *gin.Context, db *gorm.DB) {
	jobs.StartPermissionSyncJob(db)
	response.OK(c, "permission_sync_job_started", nil)
}

func createRolePermissionSyncJob(c *gin.Context, db *gorm.DB) {
	jobs.StartRolePermissionSyncJob(db)
	response.OK(c, "role_permission_sync_job_started", nil)
}

func deletePermissionSyncJob(c *gin.Context) {
	jobs.StopPermissionSyncJob()
	response.OK(c, "permission_sync_job_stopped", nil)
}

func deleteRolePermissionSyncJob(c *gin.Context) {
	jobs.StopRolePermissionSyncJob()
	response.OK(c, "role_permission_sync_job_stopped", nil)
}
