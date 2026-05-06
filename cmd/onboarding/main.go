package main

import (
	"log/slog"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/theluisbolivar/fidel-quick/internal/admin"
	"github.com/theluisbolivar/fidel-quick/internal/apperror"
	"github.com/theluisbolivar/fidel-quick/internal/config"
	"github.com/theluisbolivar/fidel-quick/internal/platform/db"
	"github.com/theluisbolivar/fidel-quick/internal/platform/logger"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, using environment variables")
	}

	cfg := config.Load()
	log := logger.Setup(cfg.Env)

	// Database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Admin service (handles onboarding + auth)
	adminRepo := admin.NewPostgresRepository(database)
	var googleVerifier admin.GoogleVerifier
	if cfg.GoogleClientID != "" {
		googleVerifier = admin.NewGoogleVerifier(cfg.GoogleClientID)
	}
	adminService := admin.NewService(adminRepo, cfg.JWTSecret, googleVerifier)
	adminAPI := admin.NewAPIHandler(adminService)

	// Router
	if !cfg.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Public routes — no auth required
	api := r.Group("/api/v1")
	api.Use(apperror.ErrorHandler(log))

	// Auth
	auth := api.Group("/auth")
	adminAPI.RegisterRoutes(auth)

	// Onboarding
	onboarding := api.Group("/onboarding")
	adminAPI.RegisterOnboardingRoutes(onboarding)

	port := getEnv("ONBOARDING_PORT", "8081")
	log.Info("onboarding server starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
