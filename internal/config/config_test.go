package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("PORT")
	os.Unsetenv("ENV")
	os.Unsetenv("REDIS_URL")

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "redis://localhost:6379", cfg.RedisURL)
	assert.Equal(t, "loyalty-invoices", cfg.S3Bucket)
	assert.Equal(t, "us-east-1", cfg.S3Region)
	assert.Equal(t, "change-me-in-production", cfg.JWTSecret)
}

func TestLoad_OverrideEnv(t *testing.T) {
	t.Setenv("PORT", "3000")
	t.Setenv("ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg := Load()

	assert.Equal(t, "3000", cfg.Port)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
}

func TestIsDevelopment(t *testing.T) {
	cfg := &Config{Env: "development"}
	assert.True(t, cfg.IsDevelopment())

	cfg.Env = "production"
	assert.False(t, cfg.IsDevelopment())

	cfg.Env = "staging"
	assert.False(t, cfg.IsDevelopment())
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")

	assert.Equal(t, "hello", getEnv("TEST_VAR", "default"))
	assert.Equal(t, "default", getEnv("NONEXISTENT_VAR", "default"))
}
