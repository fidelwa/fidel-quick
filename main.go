package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/theluisbolivar/fidel-quick/api"
	"github.com/theluisbolivar/fidel-quick/internal/config"
	"github.com/theluisbolivar/fidel-quick/internal/flow"
	"github.com/theluisbolivar/fidel-quick/internal/landing"
	"github.com/theluisbolivar/fidel-quick/internal/loyalty"
	"github.com/theluisbolivar/fidel-quick/internal/modules/cashback"
	"github.com/theluisbolivar/fidel-quick/internal/modules/earnburn"
	"github.com/theluisbolivar/fidel-quick/internal/platform/cache"
	"github.com/theluisbolivar/fidel-quick/internal/platform/db"
	"github.com/theluisbolivar/fidel-quick/internal/platform/logger"
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
	cbService := cashback.NewService(cbRepo, cbCache, ebCache, log)
	cbAPI := cashback.NewAPIHandler(cbService)
	cbModule := cashback.NewModule(cbService, cbAPI)
	registry.Register(cbModule)

	// WhatsApp client
	waClient := whatsapp.NewClient(cfg.WhatsAppAPIToken, cfg.WhatsAppPhoneNumberID)

	// Flow engine (needs a MessageSender adapter for the WhatsApp client)
	flowStore := flow.NewStateStore(redisClient)
	flowEngine := flow.NewEngine(registry, flowStore, &waAdapter{client: waClient}, log)

	// WhatsApp webhook handler
	webhookHandler := whatsapp.NewWebhookHandler(
		cfg.WhatsAppVerifyToken,
		waClient,
		sessionMgr,
		businessResolver,
		roleResolver,
		resolverRepo,
		flowEngine,
		log,
	)

	// Landing page
	landingHandler := landing.NewHandler(resolverRepo, log, cfg.WhatsAppDisplayPhone)

	// Router (API + landing + webhook)
	r := api.SetupRouter(cfg.BearerToken, landingHandler, webhookHandler, registry, log, cfg.IsDevelopment())

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

// stripEndpoint removes protocol prefix for MinIO client.
func stripEndpoint(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}
