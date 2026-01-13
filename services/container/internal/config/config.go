// Package config provides configuration management for the container service.
package config

import (
	"os"
)

// Config holds the service configuration
type Config struct {
	KubeConfig string
	Namespace  string
	RedisURL   string
	Port       string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		KubeConfig: getEnv("KUBECONFIG", ""),
		Namespace:  getEnv("DEVBOX_NAMESPACE", "devbox"),
		RedisURL:   getEnv("REDIS_URL", "redis://localhost:6379"),
		Port:       getEnv("PORT", "8083"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
