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
