package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/theluisbolivar/fidel-quick/api"
	"github.com/theluisbolivar/fidel-quick/internal/admin"
	"github.com/theluisbolivar/fidel-quick/internal/config"
	"github.com/theluisbolivar/fidel-quick/internal/flow"
	"github.com/theluisbolivar/fidel-quick/internal/landing"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/modules/cashback"
	"github.com/theluisbolivar/fidel-quick/internal/modules/earnburn"
	"github.com/theluisbolivar/fidel-quick/internal/modules/pushcard"
	"github.com/theluisbolivar/fidel-quick/internal/onboarding"
	sisfiPkg "github.com/theluisbolivar/fidel-quick/internal/sisfi"
	"github.com/theluisbolivar/fidel-quick/internal/platform/ai"
	"github.com/theluisbolivar/fidel-quick/internal/platform/cache"
	"github.com/theluisbolivar/fidel-quick/internal/platform/db"
	"github.com/theluisbolivar/fidel-quick/internal/platform/logger"
	"github.com/theluisbolivar/fidel-quick/internal/platform/storage"
	"github.com/theluisbolivar/fidel-quick/internal/platform/whatsapp"
	"github.com/theluisbolivar/fidel-quick/internal/resolver"
	"github.com/theluisbolivar/fidel-quick/internal/session"
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
	log.Info("connected to PostgreSQL")

	// Redis
	redisClient, err := cache.Connect(cfg.RedisURL)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	log.Info("connected to Redis")

	// Session manager
	sessionMgr := session.NewManager(redisClient)

	// Resolver repository (shared by business resolver, role resolver, landing, webhook)
	resolverRepo := resolver.NewPostgresRepository(database)

	// Resolvers
	businessResolver := resolver.NewBusinessResolver(resolverRepo)
	roleResolver := resolver.NewRoleResolver(resolverRepo)

	// Module registry
	registry := loyalty.NewRegistry()

	// Earn-Burn module
	ebRepo := earnburn.NewPostgresRepository(database)
	ebCache := earnburn.NewRedisCache(redisClient)
	ebService := earnburn.NewService(ebRepo, ebCache, log)
	ebAPI := earnburn.NewAPIHandler(ebService)
	ebModule := earnburn.NewModule(ebService, ebAPI)
	registry.Register(ebModule)

	// Cashback module
	cbRepo := cashback.NewPostgresRepository(database)
	cbCache := cashback.NewRedisCache(redisClient)
	cbService := cashback.NewService(cbRepo, cbCache, log)
	cbAPI := cashback.NewAPIHandler(cbService)
	cbModule := cashback.NewModule(cbService, cbAPI)
	registry.Register(cbModule)

	// Pushcard module
	pcRepo := pushcard.NewPostgresRepository(database)
	pcCache := pushcard.NewRedisCache(redisClient)
	pcService := pushcard.NewService(pcRepo, pcCache, log)
	pcAPI := pushcard.NewAPIHandler(pcService)
	pcModule := pushcard.NewModule(pcService, pcAPI)
	registry.Register(pcModule)

	// WhatsApp client
	waClient := whatsapp.NewClient(cfg.WhatsAppAPIToken, cfg.WhatsAppPhoneNumberID)

	// S3/MinIO storage
	s3Client, err := storage.NewS3Client(
		stripEndpoint(cfg.S3Endpoint),
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.S3Bucket,
		cfg.S3Region,
		!strings.HasPrefix(cfg.S3Endpoint, "http://"),
	)
	if err != nil {
		log.Error("failed to connect to S3/MinIO", "error", err)
		os.Exit(1)
	}
	log.Info("connected to S3/MinIO", "endpoint", cfg.S3Endpoint)

	// Invoice photo processor (Gemini AI + S3 + WhatsApp download)
	var photoProcessor flow.PhotoProcessor
	if cfg.GeminiAPIKey != "" {
		geminiClient := ai.NewGeminiClient(cfg.GeminiAPIKey)
		invoiceProcessor := ai.NewInvoiceProcessor(waClient, geminiClient, s3Client, log)
		photoProcessor = &photoAdapter{processor: invoiceProcessor}
		log.Info("invoice processor enabled (Gemini + S3)")
	} else {
		log.Warn("GEMINI_API_KEY not set — photo processing disabled")
	}

	// Flow engine (needs a MessageSender adapter for the WhatsApp client)
	flowStore := flow.NewStateStore(redisClient)
	flowEngine := flow.NewEngine(registry, flowStore, &waAdapter{client: waClient}, photoProcessor, log)

	// WhatsApp webhook handler
	webhookHandler := whatsapp.NewWebhookHandler(
		cfg.WhatsAppVerifyToken,
		cfg.WhatsAppAppSecret,
		waClient,
		sessionMgr,
		businessResolver,
		roleResolver,
		resolverRepo,
		flowEngine,
		log,
	)

	// Admin auth (Google verifier uses JWKS — nil verifier disables Google flows)
	adminRepo := admin.NewPostgresRepository(database)
	var googleVerifier admin.GoogleVerifier
	if cfg.GoogleClientID != "" {
		googleVerifier = admin.NewGoogleVerifier(cfg.GoogleClientID)
		log.Info("google auth enabled", "client_id_suffix", lastN(cfg.GoogleClientID, 12))
	} else {
		log.Warn("GOOGLE_CLIENT_ID not set — Google login/signup disabled")
	}
	adminService := admin.NewService(adminRepo, cfg.JWTSecret, googleVerifier)
	adminAPI := admin.NewAPIHandler(adminService)

	// Onboarding
	onboardingRepo := onboarding.NewRepository(database)
	onboardingAPI := onboarding.NewAPIHandler(onboardingRepo)

	// Sisfi (loyalty system catalog)
	sisfiRepo := sisfiPkg.NewRepository(database)
	sisfiAPI := sisfiPkg.NewAPIHandler(sisfiRepo)

	// Landing page
	landingHandler := landing.NewHandler(resolverRepo, log, cfg.WhatsAppDisplayPhone)

	// Router (API + landing + webhook)
	r := api.SetupRouter(cfg.BearerToken, cfg.JWTSecret, landingHandler, webhookHandler, registry, adminAPI, onboardingAPI, sisfiAPI, database, redisClient, adminFS(), log, cfg.IsDevelopment())

	log.Info("server starting", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// waAdapter wraps *whatsapp.Client to satisfy flow.MessageSender.
// This avoids a circular dependency between the flow and whatsapp packages.
type waAdapter struct {
	client *whatsapp.Client
}

func (a *waAdapter) SendText(ctx context.Context, to, text string) error {
	return a.client.SendText(ctx, to, text)
}

func (a *waAdapter) SendInteractiveList(ctx context.Context, to, header, body string, options []flow.ListOption) error {
	waOpts := make([]whatsapp.ListOption, len(options))
	for i, o := range options {
		waOpts[i] = whatsapp.ListOption{ID: o.ID, Title: o.Title, Description: o.Description}
	}
	return a.client.SendInteractiveList(ctx, to, header, body, waOpts)
}

// photoAdapter wraps *ai.InvoiceProcessor to satisfy flow.PhotoProcessor.
type photoAdapter struct {
	processor *ai.InvoiceProcessor
}

func (a *photoAdapter) ProcessPhoto(ctx context.Context, imageURL string) (*flow.PhotoProcessResult, error) {
	result, err := a.processor.ProcessPhoto(ctx, imageURL)
	if err != nil {
		return nil, err
	}
	return &flow.PhotoProcessResult{
		StorageURL: result.StorageURL,
		Amount:     result.Amount,
		Currency:   result.Currency,
		Invoice:    result.Invoice,
	}, nil
}

// stripEndpoint removes protocol prefix for MinIO client.
func stripEndpoint(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
