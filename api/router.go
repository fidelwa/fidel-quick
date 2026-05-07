package api

import (
	"context"
	"database/sql"
	_ "embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/theluisbolivar/fidel-quick/api/middleware"
	"github.com/theluisbolivar/fidel-quick/internal/admin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/landing"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/onboarding"
	"github.com/theluisbolivar/fidel-quick/internal/platform/whatsapp"
	"github.com/theluisbolivar/fidel-quick/internal/sisfi"
)

//go:embed openapi.yaml
var openapiSpec []byte

// SetupRouter creates and configures the Gin router with all routes.
func SetupRouter(
	bearerToken string,
	jwtSecret string,
	landingHandler *landing.Handler,
	webhookHandler *whatsapp.WebhookHandler,
	registry *loyalty.Registry,
	adminAPI *admin.APIHandler,
	onboardingAPI *onboarding.APIHandler,
	sisfiAPI *sisfi.APIHandler,
	database *sql.DB,
	redisClient *redis.Client,
	log *slog.Logger,
	isDev bool,
) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Health probes (públicas) — Cloud Run liveness/readiness.
	// /healthz: liveness (proceso vivo, sin tocar dependencias).
	// /readyz: readiness (Postgres + Redis responden en <1s).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", readyzHandler(database, redisClient))

	// Landing page (public)
	r.GET("/unirse/:slug", landingHandler.Join)

	// WhatsApp webhook (public — Meta needs direct access)
	r.GET("/webhook", webhookHandler.Verify)
	r.POST("/webhook", webhookHandler.Receive)

	// OpenAPI spec & Swagger UI (public)
	r.GET("/api/docs/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", openapiSpec)
	})
	r.GET("/api/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})

	// Sisfi catalog (public — no auth required)
	sisfiPublic := r.Group("/api/v1")
	sisfiPublic.Use(apperror.ErrorHandler(log))
	sisfiAPI.RegisterPublicRoutes(sisfiPublic)

	// Auth endpoints (public — no auth required)
	auth := r.Group("/api/v1/auth")
	auth.Use(apperror.ErrorHandler(log))
	adminAPI.RegisterRoutes(auth)

	// Onboarding endpoints (public — no auth required)
	onboarding := r.Group("/api/v1/onboarding")
	onboarding.Use(apperror.ErrorHandler(log))
	adminAPI.RegisterOnboardingRoutes(onboarding)

	// REST API (JWT or bearer token auth + error middleware)
	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWTOrBearer(jwtSecret, bearerToken))
	v1.Use(apperror.ErrorHandler(log))

	// Authenticated admin routes (link/unlink Google, me)
	adminAPI.RegisterAuthenticatedRoutes(v1)

	// Onboarding routes (JWT-authenticated)
	onboardingAPI.RegisterRoutes(v1)

	// Sisfi routes (JWT-authenticated)
	sisfiAPI.RegisterRoutes(v1)

	// Module REST routes (each module registers its own)
	registry.RegisterAllRoutes(v1)

	return r
}

// readyzHandler verifica conectividad con Postgres y Redis con timeout 1s.
// Devuelve 200 si ambos OK, 503 con detalle del fallo.
func readyzHandler(database *sql.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := database.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "not_ready",
				"postgres": err.Error(),
			})
			return
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"redis":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Fidel Quick API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({url: "/api/docs/openapi.yaml", dom_id: "#swagger-ui"});
  </script>
</body>
</html>`
