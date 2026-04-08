package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	apiV1 "mycourse-io-be/api/v1"
	"mycourse-io-be/middleware"
	"mycourse-io-be/pkg/httperr"
	"mycourse-io-be/pkg/setting"
)

func InitRouter() *gin.Engine {
	router := gin.New()
	router.Use(httperr.Middleware())
	router.Use(httperr.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     setting.AppSetting.CorsAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Refresh-Token", "X-Session-Id"},
		ExposeHeaders:    []string{"X-Token-Expired"},
		AllowCredentials: true,
	}))
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	apiRoot := router.Group("/api")

	v1 := apiRoot.Group("/v1")
	v1.Use(middleware.BeforeInterceptor())

	// Authenticated v1 routes first (stricter subtree). Permission checks (RequirePermission + constants)
	// attach per-route or per-group on routerAuthen when you wire them.
	routerAuthen := v1.Group("")
	routerAuthen.Use(middleware.RateLimitLocal(120, 1), middleware.AuthJWT())
	apiV1.RegisterAuthenRoutes(routerAuthen)

	routerNotAuthen := v1.Group("")
	routerNotAuthen.Use(middleware.RateLimitLocal(60, 1))
	apiV1.RegisterNotAuthenRoutes(routerNotAuthen)

	internalV1 := apiRoot.Group("/internal-v1")
	internalV1.Use(middleware.RateLimitLocal(60, 1), middleware.BeforeInterceptor(), middleware.RequireInternalAPIKey())
	apiV1.RegisterInternalRoutes(internalV1)

	return router
}
