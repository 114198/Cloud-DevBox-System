// Package config provides configuration management for the core service.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the service configuration
type Config struct {
	// Server settings
	Port    string
	Env     string
	BaseURL string

	// Database settings
	DatabaseURL string

	// Redis settings
	RedisURL string

	// JWT settings
	JWTPrivateKey   string
	JWTPublicKey    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// OAuth settings
	GitHubClientID     string
	GitHubClientSecret string
	GitLabClientID     string
	GitLabClientSecret string
	GiteeClientID      string
	GiteeClientSecret  string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	accessTTL, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_TTL_MINUTES", "15"))
	refreshTTL, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_TTL_DAYS", "7"))

	return &Config{
		// Server settings
		Port:    getEnv("PORT", "8081"),
		Env:     getEnv("ENV", "development"),
		BaseURL: getEnv("BASE_URL", "http://localhost:8081"),

		// Database settings
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/devbox?sslmode=disable"),

		// Redis settings
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// JWT settings
		JWTPrivateKey:   getEnv("JWT_PRIVATE_KEY", ""),
		JWTPublicKey:    getEnv("JWT_PUBLIC_KEY", ""),
		AccessTokenTTL:  time.Duration(accessTTL) * time.Minute,
		RefreshTokenTTL: time.Duration(refreshTTL) * 24 * time.Hour,

		// OAuth settings
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitLabClientID:     getEnv("GITLAB_CLIENT_ID", ""),
		GitLabClientSecret: getEnv("GITLAB_CLIENT_SECRET", ""),
		GiteeClientID:      getEnv("GITEE_CLIENT_ID", ""),
		GiteeClientSecret:  getEnv("GITEE_CLIENT_SECRET", ""),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
