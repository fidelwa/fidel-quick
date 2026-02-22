package api

import (
	_ "embed"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theluisbolivar/fidel-quick/api/middleware"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/landing"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/platform/whatsapp"
)

//go:embed openapi.yaml
var openapiSpec []byte

// SetupRouter creates and configures the Gin router with all routes.
func SetupRouter(
	bearerToken string,
	landingHandler *landing.Handler,
	webhookHandler *whatsapp.WebhookHandler,
	registry *loyalty.Registry,
	log *slog.Logger,
	isDev bool,
) *gin.Engine {
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

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

	// REST API (bearer token auth + error middleware)
	v1 := r.Group("/api/v1")
	v1.Use(middleware.BearerAuth(bearerToken))
	v1.Use(apperror.ErrorHandler(log))

	// Module REST routes (each module registers its own)
	registry.RegisterAllRoutes(v1)

	return r
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
