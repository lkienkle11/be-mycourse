package services

import (
	"gorm.io/gorm"

	"mycourse-io-be/internal/jobs"
)

func StartPermissionSyncScheduler(db *gorm.DB) {
	jobs.StartPermissionSyncJob(db)
}

func StartRolePermissionSyncScheduler(db *gorm.DB) {
	jobs.StartRolePermissionSyncJob(db)
}

func StopPermissionSyncScheduler() {
	jobs.StopPermissionSyncJob()
}

func StopRolePermissionSyncScheduler() {
	jobs.StopRolePermissionSyncJob()
}
