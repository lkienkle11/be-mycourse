package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	authdelivery "mycourse-io-be/internal/auth/delivery"
	mediadelivery "mycourse-io-be/internal/media/delivery"
	rbacdelivery "mycourse-io-be/internal/rbac/delivery"
	sysdelivery "mycourse-io-be/internal/system/delivery"
	taxdelivery "mycourse-io-be/internal/taxonomy/delivery"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/middleware"
	"mycourse-io-be/internal/shared/response"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/pkg/httperr"
)

func ginDefaultCORS() cors.Config {
	return cors.Config{
		AllowOrigins: setting.AppSetting.CorsAllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Authorization",
			"X-API-Key", "X-Refresh-Token", "X-Session-Id",
			middleware.HeaderCSRFToken,
		},
		ExposeHeaders: []string{
			"X-Token-Expired",
			constants.HeaderRegisterRetryAfter,
			constants.HeaderRegisterRetryAfterExtended,
			middleware.HeaderCSRFToken,
		},
		AllowCredentials: true,
	}
}

// InitRouter builds and returns the configured Gin engine with all routes registered.
// svc and h must be obtained from Wire().
func InitRouter(svc *Services, h *Handlers) *gin.Engine {
	router := gin.New()
	router.MaxMultipartMemory = constants.MediaMultipartParseMemoryBytes
	router.Use(middleware.RequestLogger())
	router.Use(httperr.Middleware())
	router.Use(httperr.Recovery())
	router.Use(cors.New(ginDefaultCORS()))
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	apiRoot := router.Group("/api")
	mountAPITree(apiRoot, svc, h)
	return router
}

func mountAPITree(apiRoot *gin.RouterGroup, svc *Services, h *Handlers) {
	// --- /api/system (privileged, rate-limited by IP) ---
	systemGroup := apiRoot.Group("/system")
	systemGroup.Use(middleware.BeforeInterceptor())
	systemGroup.Use(middleware.RateLimitSystemIP(10, 3))
	sysdelivery.RegisterRoutes(systemGroup, svc.System)

	// --- /api/v1 no-filter (webhooks bypass JWT + rate-limit) ---
	v1NoFilter := apiRoot.Group("/v1")
	mediadelivery.RegisterWebhookRoutes(v1NoFilter, h.Media)

	// --- /api/v1 base (before-interceptor applied) ---
	v1 := apiRoot.Group("/v1")
	v1.Use(middleware.BeforeInterceptor())
	// Temporarily disabled for rollout safety; keep logic in codebase for quick re-enable.
	// v1.Use(middleware.EnsureCSRFCookie())

	// Unauthenticated v1 routes
	notAuthen := v1.Group("")
	// Temporarily disabled for rollout safety; keep logic in codebase for quick re-enable.
	// notAuthen.Use(middleware.RateLimitLocal(60, 1), middleware.RequireCSRF())
	notAuthen.Use(middleware.RateLimitLocal(60, 1))
	notAuthen.GET("/health", func(c *gin.Context) { response.Health(c) })
	authdelivery.RegisterRoutes(nil, notAuthen, h.Auth, svc.RBAC)

	// Authenticated v1 routes
	authen := v1.Group("")
	// Temporarily disabled for rollout safety; keep logic in codebase for quick re-enable.
	// authen.Use(middleware.RateLimitLocal(120, 1), middleware.AuthJWT(), middleware.RequireCSRF())
	authen.Use(middleware.RateLimitLocal(120, 1), middleware.AuthJWT())
	authdelivery.RegisterRoutes(authen, nil, h.Auth, svc.RBAC)
	taxdelivery.RegisterRoutes(authen, h.Taxonomy, svc.RBAC)
	mediadelivery.RegisterRoutes(authen, h.Media, svc.RBAC)

	// --- /api/internal-v1 (internal API key required) ---
	internalV1 := apiRoot.Group("/internal-v1")
	internalV1.Use(
		middleware.RateLimitLocal(60, 1),
		middleware.BeforeInterceptor(),
		middleware.RequireInternalAPIKey(),
	)
	rbacdelivery.RegisterRoutes(internalV1, h.RBAC)
}
