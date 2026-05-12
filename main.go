package main

//go:generate go run ./cmd/syncpermissions
//go:generate go run ./cmd/syncrolepermissions

import (
	"log"
	"os"

	"go.uber.org/zap"

	"mycourse-io-be/internal/appcli"
	mediainfra "mycourse-io-be/internal/media/infra"
	mediajobs "mycourse-io-be/internal/media/jobs"
	"mycourse-io-be/internal/server"
	"mycourse-io-be/internal/shared/cache"
	shareddb "mycourse-io-be/internal/shared/db"
	"mycourse-io-be/internal/shared/logger"
	"mycourse-io-be/internal/shared/setting"
	supabasepkg "mycourse-io-be/pkg/supabase"
)

func mustCoreSettingsAndDB() {
	if err := shareddb.Setup(); err != nil {
		zap.L().Fatal("setup postgres ([database]) failed", zap.Error(err))
	}
	if appcli.MaybeRunRegisterNewSystemUser(shareddb.Conn()) {
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
	cache.SetupRedis()
	if _, err := mediainfra.NewCloudClientsFromSetting(); err != nil {
		zap.L().Fatal("setup media sdk clients failed", zap.Error(err))
	}
}

func maybeMigrateFromEnv() {
	if os.Getenv("MIGRATE") != "1" {
		return
	}
	if err := shareddb.MigrateDatabase(); err != nil {
		zap.L().Fatal("migrate database failed", zap.Error(err))
	}
	zap.L().Info("sql migrations applied (see migrations/*.up.sql)")
}

func mustBootstrapRuntime() {
	mustCoreSettingsAndDB()
	mustSupabaseRedisAndMedia()
	maybeMigrateFromEnv()
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

	svcs, handlers, err := server.Wire(shareddb.Conn(), cache.Redis)
	if err != nil {
		zap.L().Fatal("dependency wiring failed", zap.Error(err))
	}

	// Start background jobs that require wired services.
	mediajobs.StartMediaPendingCleanupJob(mediainfra.NewGormPendingCleanupRepository(shareddb.Conn()))

	router := server.InitRouter(svcs, handlers)
	if err := router.Run(":" + setting.ServerSetting.Port); err != nil {
		zap.L().Fatal("server run failed", zap.Error(err))
	}
}
