package jobsystem

import (
	"github.com/gin-gonic/gin"

	"mycourse-io-be/internal/appdb"
	jobrbac "mycourse-io-be/internal/jobs/rbac"
	"mycourse-io-be/pkg/response"
)

// HTTPCreatePermissionSyncJob starts the in-process permission sync ticker (POST /api/system/create-permission-sync-job).
func HTTPCreatePermissionSyncJob(c *gin.Context) {
	jobrbac.StartPermissionSyncJob(appdb.Conn())
	response.OK(c, "permission_sync_job_started", nil)
}

// HTTPCreateRolePermissionSyncJob starts the in-process role–permission sync ticker.
func HTTPCreateRolePermissionSyncJob(c *gin.Context) {
	jobrbac.StartRolePermissionSyncJob(appdb.Conn())
	response.OK(c, "role_permission_sync_job_started", nil)
}

// HTTPStopPermissionSyncJob stops the permission sync ticker.
func HTTPStopPermissionSyncJob(c *gin.Context) {
	jobrbac.StopPermissionSyncJob()
	response.OK(c, "permission_sync_job_stopped", nil)
}

// HTTPStopRolePermissionSyncJob stops the role–permission sync ticker.
func HTTPStopRolePermissionSyncJob(c *gin.Context) {
	jobrbac.StopRolePermissionSyncJob()
	response.OK(c, "role_permission_sync_job_stopped", nil)
}
