package config

import "os"

type Config struct {
	Port                  string
	Env                   string
	DatabaseURL           string
	RedisURL              string
	S3Endpoint            string
	S3Bucket              string
	S3Region              string
	AWSAccessKeyID        string
	AWSSecretAccessKey    string
	AnthropicAPIKey       string
	GeminiAPIKey          string
	GoogleClientID        string
	WhatsAppVerifyToken   string
	WhatsAppAPIToken      string
	WhatsAppPhoneNumberID string
	WhatsAppDisplayPhone  string
	WhatsAppAppSecret     string
	PlatformURL           string
	BearerToken           string
	JWTSecret             string
	// Password reset email (FID-16). EmailProvider selects the delivery
	// backend ("stdout" logs the link — the dev default). EmailFrom is the
	// From: address. AppURL is the public admin base used to build the
	// reset link (<AppURL>/reset-password?token=...).
	EmailProvider string
	EmailFrom     string
	AppURL        string
}

func Load() *Config {
	return &Config{
		Port:                  getEnv("PORT", "8080"),
		Env:                   getEnv("ENV", "development"),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379"),
		S3Endpoint:            getEnv("S3_ENDPOINT", ""),
		S3Bucket:              getEnv("S3_BUCKET", "loyalty-invoices"),
		S3Region:              getEnv("S3_REGION", "us-east-1"),
		AWSAccessKeyID:        getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:    getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AnthropicAPIKey:       getEnv("ANTHROPIC_API_KEY", ""),
		GeminiAPIKey:          getEnv("GEMINI_API_KEY", ""),
		GoogleClientID:        getEnv("GOOGLE_CLIENT_ID", ""),
		WhatsAppVerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", ""),
		WhatsAppAPIToken:      getEnv("WHATSAPP_API_TOKEN", ""),
		WhatsAppPhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
		WhatsAppDisplayPhone:  getEnv("WHATSAPP_DISPLAY_PHONE", ""),
		WhatsAppAppSecret:     getEnv("WHATSAPP_APP_SECRET", ""),
		PlatformURL:           getEnv("PLATFORM_URL", ""),
		BearerToken:           getEnv("BEARER_TOKEN", ""),
		JWTSecret:             getEnv("JWT_SECRET", "change-me-in-production"),
		EmailProvider:         getEnv("EMAIL_PROVIDER", "stdout"),
		EmailFrom:             getEnv("EMAIL_FROM", "no-reply@fidel.app"),
		AppURL:                getEnv("APP_URL", "http://localhost:5173"),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
