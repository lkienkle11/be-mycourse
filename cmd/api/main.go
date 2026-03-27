package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mycourse-io-be/internal/config"
	"mycourse-io-be/internal/repository"
	"mycourse-io-be/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.App.Env)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	db, err := repository.NewPostgres(cfg.Database.DSN)
	if err != nil {
		log.Fatal("failed to connect database", zap.Error(err))
	}
	_ = db

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"env":    cfg.App.Env,
		})
	})

	addr := ":" + cfg.App.Port
	log.Info("starting API server", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		log.Fatal("server stopped", zap.Error(err))
	}
}
