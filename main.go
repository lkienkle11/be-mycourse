package main

//go:generate go run ./cmd/syncpermissions
//go:generate go run ./cmd/syncrolepermissions

import (
	"log"
	"os"

	"go.uber.org/zap"

	"mycourse-io-be/api"
	"mycourse-io-be/config"
	"mycourse-io-be/internal/appcli"
	"mycourse-io-be/internal/appdb"
	jobmedia "mycourse-io-be/internal/jobs/media"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/cache_clients"
	"mycourse-io-be/pkg/logger"
	pkgmedia "mycourse-io-be/pkg/media"
	"mycourse-io-be/pkg/setting"
	supabasepkg "mycourse-io-be/pkg/supabase"
	"mycourse-io-be/queues"
	"mycourse-io-be/services/rbac"
)

func mustCoreSettingsAndDB() {
	if err := models.Setup(); err != nil {
		zap.L().Fatal("setup postgres ([database]) failed", zap.Error(err))
	}
	appdb.Set(models.DB)
	rbac.SetRBACDB(models.DB)
	if appcli.MaybeRunRegisterNewSystemUser(models.DB) {
		os.Exit(0)
	}
}

func mustSupabaseRedisAndMedia() {
	if err := supabasepkg.SetupDatabase(); err != nil {
		zap.L().Fatal("setup supabase postgres (DBURL) failed", zap.Error(err))
	}
	if err := supabasepkg.Setup(); err != nil {
		zap.L().Warn("supabase HTTP client is not initialized", zap.Error(err))
	}
	cache_clients.SetupRedis()
	if err := pkgmedia.Setup(); err != nil {
		zap.L().Fatal("setup media sdk clients failed", zap.Error(err))
	}
}

func maybeMigrateFromEnv() {
	if os.Getenv("MIGRATE") != "1" {
		return
	}
	if err := models.MigrateDatabase(); err != nil {
		zap.L().Fatal("migrate database failed", zap.Error(err))
	}
	zap.L().Info("sql migrations applied (see migrations/*.up.sql)")
}

func mustBackgroundConsumers() {
	config.InitSystem()
	jobmedia.StartMediaPendingCleanupJob(models.DB)
	queues.Consume()
}

func mustBootstrapRuntime() {
	mustCoreSettingsAndDB()
	mustSupabaseRedisAndMedia()
	maybeMigrateFromEnv()
	mustBackgroundConsumers()
}

func main() {
	if err := setting.Setup(); err != nil {
		log.Fatalf("setup setting failed: %v", err)
	}
	if _, err := logger.InitFromSettings(); err != nil {
		log.Fatalf("logger init failed: %v", err)
	}
	defer logger.Sync()

	mustBootstrapRuntime()

	router := api.InitRouter()
	if err := router.Run(":" + setting.ServerSetting.Port); err != nil {
		zap.L().Fatal("server run failed", zap.Error(err))
	}
}
