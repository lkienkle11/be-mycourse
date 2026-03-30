package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"mycourse-io-be/api/exceptions"
	apiV1 "mycourse-io-be/api/v1"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/httperr"
)

func InitRouter() *gin.Engine {
	router := gin.New()
	router.Use(httperr.Middleware())
	router.Use(httperr.Recovery())
	router.Use(cors.Default())
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	apiRoot := router.Group("/api")
	apiRoot.Use(middleware.RateLimitLocal())

	v1 := apiRoot.Group("/v1")
	v1.Use(
		middleware.BeforeInterceptor(),
		middleware.AuthJWTUnlessPublic(exceptions.PublicEndpoints()),
	)
	apiV1.RegisterPublicRoutes(v1)
	apiV1.RegisterRoutes(v1)

	internalV1 := apiRoot.Group("/internal-v1")
	internalV1.Use(middleware.BeforeInterceptor(), middleware.RequireInternalAPIKey())
	apiV1.RegisterInternalRoutes(internalV1)

	return router
}
