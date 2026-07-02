package api

import (
	"context"
	"database/sql"
	_ "embed"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/theluisbolivar/fidel-quick/api/middleware"
	"github.com/theluisbolivar/fidel-quick/internal/admin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/featureflags"
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
	flagsAPI *featureflags.APIHandler,
	database *sql.DB,
	redisClient *redis.Client,
	adminSPA fs.FS,
	log *slog.Logger,
	isDev bool,
) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Admin SPA — sólo cuando el binario fue compilado con `-tags prod`
	// y trae el bundle embebido. En dev `adminSPA` es nil y el SPA se
	// sirve por Vite en :5173.
	if adminSPA != nil {
		registerAdminSPA(r, adminSPA)
	}

	// Health probes (públicas) — Cloud Run liveness/readiness.
	// /healthz: liveness (proceso vivo, sin tocar dependencias).
	// /readyz: readiness (Postgres + Redis responden en <1s).
	r.GET("/healthz", healthzHandler())
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

	// Feature flags admin routes (JWT-authenticated)
	flagsAPI.RegisterRoutes(v1)

	// Module REST routes (each module registers its own)
	registry.RegisterAllRoutes(v1)

	return r
}

// registerAdminSPA monta el bundle React en /admin/* con SPA fallback:
// rutas client-side (ej. /admin/registro) que no matchean un asset
// del bundle se sirven con index.html para que React Router las maneje.
func registerAdminSPA(r *gin.Engine, spa fs.FS) {
	httpFS := http.FS(spa)
	fileServer := http.StripPrefix("/admin/", http.FileServer(httpFS))

	// Pre-leer index.html una sola vez para servir el fallback SPA
	// directamente (sin pasar por http.FileServer, que tiene un
	// comportamiento built-in de redirigir cualquier path que termine
	// en "/index.html" a "./", lo cual rompía los deep-links del SPA:
	// GET /admin/registro → fallback intenta servir /admin/index.html
	// → FileServer redirige 301 a "./" → el browser termina en /admin/
	// y el SPA no puede llegar a la ruta original.
	indexBytes, indexErr := readAllFS(spa, "index.html")

	// Bare / redirige al SPA — UX para cuando alguien comparte la URL
	// raíz del servicio. /healthz, /readyz, /webhook tienen handlers
	// propios y no caen acá.
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})
	r.GET("/admin", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/admin/")
	})
	r.GET("/admin/*filepath", func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("filepath"), "/")
		if path == "" || path == "index.html" {
			serveSPAIndex(c, indexBytes, indexErr)
			return
		}
		// Asset real (assets/*.js, /vite.svg, etc.) → FileServer normal.
		if f, err := spa.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback para deep-links sin asset (ej. /admin/registro,
		// /admin/onboarding) — index.html servido inline, no redirect.
		serveSPAIndex(c, indexBytes, indexErr)
	})
}

func readAllFS(fsys fs.FS, name string) ([]byte, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func serveSPAIndex(c *gin.Context, indexBytes []byte, indexErr error) {
	if indexErr != nil {
		c.String(http.StatusInternalServerError, "admin bundle missing index.html")
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
}

// healthzHandler es el liveness probe: responde 200 {"status":"ok"} sin tocar
// ninguna dependencia. Extraído a función nombrada para que los tests ejerciten
// el handler real registrado en SetupRouter (no una copia inline).
func healthzHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// pingFunc verifica una dependencia dado un contexto (respeta su deadline).
type pingFunc func(ctx context.Context) error

// readyzHandler verifica conectividad con Postgres y Redis con timeout 1s.
// Devuelve 200 si ambos OK, 503 con detalle del fallo.
func readyzHandler(database *sql.DB, redisClient *redis.Client) gin.HandlerFunc {
	return readyzHandlerFor(
		database.PingContext,
		func(ctx context.Context) error { return redisClient.Ping(ctx).Err() },
	)
}

// readyzHandlerFor construye el handler a partir de pingers inyectables.
// Producción usa Postgres/Redis reales; los tests inyectan fakes.
func readyzHandlerFor(pingPostgres, pingRedis pingFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := pingPostgres(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "not_ready",
				"postgres": err.Error(),
			})
			return
		}
		if err := pingRedis(ctx); err != nil {
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
