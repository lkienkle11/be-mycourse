package main

import (
	"log"
	"os"

	"mycourse-io-be/api"
	"mycourse-io-be/cache_clients"
	"mycourse-io-be/config"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/setting"
	"mycourse-io-be/queues"
)

func main() {
	if err := setting.Setup(); err != nil {
		log.Fatalf("setup setting failed: %v", err)
	}

	if err := models.Setup(); err != nil {
		log.Fatalf("setup database failed: %v", err)
	}

	cache_clients.SetupRedis()

	if os.Getenv("MIGRATE") == "1" {
		if err := models.MigrateDatabase(); err != nil {
			log.Fatalf("migrate database failed: %v", err)
		}
	}

	config.InitSystem()
	queues.Consume()

	router := api.InitRouter()
	if err := router.Run(":" + setting.ServerSetting.Port); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}
