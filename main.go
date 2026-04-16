package main

//go:generate go run ./cmd/syncpermissions
//go:generate go run ./cmd/syncrolepermissions

import (
	"log"
	"os"

	"mycourse-io-be/api"
	"mycourse-io-be/cache_clients"
	"mycourse-io-be/config"
	"mycourse-io-be/internal/jobs"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/envbool"
	"mycourse-io-be/pkg/setting"
	supabasepkg "mycourse-io-be/pkg/supabase"
	"mycourse-io-be/queues"
	"mycourse-io-be/services"
)

func main() {
	if err := setting.Setup(); err != nil {
		log.Fatalf("setup setting failed: %v", err)
	}

	if err := models.Setup(); err != nil {
		log.Fatalf("setup postgres ([database]) failed: %v", err)
	}
	services.SetRBACDB(models.DB)

	if err := supabasepkg.SetupDatabase(); err != nil {
		log.Fatalf("setup supabase postgres (DBURL) failed: %v", err)
	}

	if err := supabasepkg.Setup(); err != nil {
		log.Printf("supabase HTTP client is not initialized: %v", err)
	}

	cache_clients.SetupRedis()

	if os.Getenv("MIGRATE") == "1" {
		if err := models.MigrateDatabase(); err != nil {
			log.Fatalf("migrate database failed: %v", err)
		}
		log.Println("sql migrations applied (see migrations/*.up.sql)")
	}

	config.InitSystem()

	if isAutoSyncPermissionJobEnabled() {
		jobs.StartAutoSyncPermissionJob(models.DB)
	}

	if isAutoSyncRolePermissionJobEnabled() {
		jobs.StartWeeklyRolePermissionSyncJob(models.DB)
	}

	queues.Consume()

	router := api.InitRouter()
	if err := router.Run(":" + setting.ServerSetting.Port); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}

func isAutoSyncPermissionJobEnabled() bool {
	return envbool.Enabled("AUTO_SYNC_PERMISSION_JOB")
}

func isAutoSyncRolePermissionJobEnabled() bool {
	return envbool.Enabled("AUTO_SYNC_ROLE_PERMISSION_JOB")
}
