// Package config provides configuration management for the realtime service.
package config

import (
	"os"
)

// Config holds the service configuration
type Config struct {
	RedisURL string
	Port     string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		Port:     getEnv("PORT", "8082"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
