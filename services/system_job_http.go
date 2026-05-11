package services

import (
	"github.com/gin-gonic/gin"

	jobsystem "mycourse-io-be/internal/jobs/system"
)

// System job HTTP adapters: api/ must not import internal/jobs (Rule 6); services delegates here.

// SystemHTTPCreatePermissionSyncJob starts the in-process permission sync ticker.
func SystemHTTPCreatePermissionSyncJob(c *gin.Context) {
	jobsystem.HTTPCreatePermissionSyncJob(c)
}

// SystemHTTPCreateRolePermissionSyncJob starts the in-process role–permission sync ticker.
func SystemHTTPCreateRolePermissionSyncJob(c *gin.Context) {
	jobsystem.HTTPCreateRolePermissionSyncJob(c)
}

// SystemHTTPStopPermissionSyncJob stops the permission sync ticker.
func SystemHTTPStopPermissionSyncJob(c *gin.Context) {
	jobsystem.HTTPStopPermissionSyncJob(c)
}

// SystemHTTPStopRolePermissionSyncJob stops the role–permission sync ticker.
func SystemHTTPStopRolePermissionSyncJob(c *gin.Context) {
	jobsystem.HTTPStopRolePermissionSyncJob(c)
}
