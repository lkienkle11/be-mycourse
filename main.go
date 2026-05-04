package main

//go:generate go run ./cmd/syncpermissions
//go:generate go run ./cmd/syncrolepermissions

import (
	"log"
	"os"

	"mycourse-io-be/api"
	"mycourse-io-be/config"
	"mycourse-io-be/internal/appcli"
	"mycourse-io-be/internal/appdb"
	"mycourse-io-be/internal/jobs"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/cache_clients"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	supabasepkg "mycourse-io-be/pkg/supabase"
	"mycourse-io-be/queues"
	"mycourse-io-be/services/rbac"
)

func mustCoreSettingsAndDB() {
	if err := setting.Setup(); err != nil {
		log.Fatalf("setup setting failed: %v", err)
	}
	if err := models.Setup(); err != nil {
		log.Fatalf("setup postgres ([database]) failed: %v", err)
	}
	appdb.Set(models.DB)
	rbac.SetRBACDB(models.DB)
	if appcli.MaybeRunRegisterNewSystemUser(models.DB) {
		os.Exit(0)
	}
}

func mustSupabaseRedisAndMedia() {
	if err := supabasepkg.SetupDatabase(); err != nil {
		log.Fatalf("setup supabase postgres (DBURL) failed: %v", err)
	}
	if err := supabasepkg.Setup(); err != nil {
		log.Printf("supabase HTTP client is not initialized: %v", err)
	}
	cache_clients.SetupRedis()
	if err := pkgmedia.Setup(); err != nil {
		log.Fatalf("setup media sdk clients failed: %v", err)
	}
}

func maybeMigrateFromEnv() {
	if os.Getenv("MIGRATE") != "1" {
		return
	}
	if err := models.MigrateDatabase(); err != nil {
		log.Fatalf("migrate database failed: %v", err)
	}
	log.Println("sql migrations applied (see migrations/*.up.sql)")
}

func mustBackgroundConsumers() {
	config.InitSystem()
	jobs.StartMediaPendingCleanupJob(models.DB)
	queues.Consume()
}

func mustBootstrapRuntime() {
	mustCoreSettingsAndDB()
	mustSupabaseRedisAndMedia()
	maybeMigrateFromEnv()
	mustBackgroundConsumers()
}

func main() {
	mustBootstrapRuntime()

	router := api.InitRouter()
	if err := router.Run(":" + setting.ServerSetting.Port); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
