package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	apiV1 "mycourse-io-be/api/v1"
	"mycourse-io-be/middleware"
)

func InitRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	public := router.Group("/api/v1")
	apiV1.RegisterPublicRoutes(public)

	mainV1 := router.Group("/api/v1")
	mainV1.Use(middleware.BeforeInterceptor())
	apiV1.RegisterRoutes(mainV1)

	internalV1 := router.Group("/api/internal-v1")
	internalV1.Use(middleware.BeforeInterceptor())
	apiV1.RegisterInternalRoutes(internalV1)

	return router
}
