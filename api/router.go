package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	apisystem "mycourse-io-be/api/system"
	apiV1 "mycourse-io-be/api/v1"
	"mycourse-io-be/constants"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/setting"
)

func ginDefaultCORS() cors.Config {
	return cors.Config{
		AllowOrigins:     setting.AppSetting.CorsAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Refresh-Token", "X-Session-Id"},
		ExposeHeaders:    []string{"X-Token-Expired", constants.HeaderRegisterRetryAfter, constants.HeaderRegisterRetryAfterExtended},
		AllowCredentials: true,
	}
}

func mountAPIV1Tree(apiRoot *gin.RouterGroup) {
	system := apiRoot.Group("/system")
	system.Use(middleware.BeforeInterceptor())
	system.Use(middleware.RateLimitSystemIP(10, 3))
	apisystem.RegisterRoutes(system)

	v1NoFilter := apiRoot.Group("/v1")
	apiV1.RegisterNoFilterRoutes(v1NoFilter)

	v1 := apiRoot.Group("/v1")
	v1.Use(middleware.BeforeInterceptor())

	routerAuthen := v1.Group("")
	routerAuthen.Use(middleware.RateLimitLocal(120, 1), middleware.AuthJWT())
	apiV1.RegisterAuthenRoutes(routerAuthen)

	routerNotAuthen := v1.Group("")
	routerNotAuthen.Use(middleware.RateLimitLocal(60, 1))
	apiV1.RegisterNotAuthenRoutes(routerNotAuthen)

	internalV1 := apiRoot.Group("/internal-v1")
	internalV1.Use(middleware.RateLimitLocal(60, 1), middleware.BeforeInterceptor(), middleware.RequireInternalAPIKey())
	apiV1.RegisterInternalRoutes(internalV1)
}

func InitRouter() *gin.Engine {
	router := gin.New()
	// Multipart: keep only this much of each part in memory; larger bodies spill to temp files.
	// Per-request bodies up to the aggregate multi-file cap (see constants.MaxMediaMultipartTotalBytes)
	// may be accepted when parts are streamed from disk; reverse proxies must still allow ≥ 2G (docs/deploy.md).
	router.MaxMultipartMemory = constants.MediaMultipartParseMemoryBytes // must match handler ParseMultipartForm budget (see constants/error_msg.go)
	router.Use(middleware.RequestLogger())
	router.Use(httperr.Middleware())
	router.Use(httperr.Recovery())
	router.Use(cors.New(ginDefaultCORS()))
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	apiRoot := router.Group("/api")
	mountAPIV1Tree(apiRoot)

	return router
}
