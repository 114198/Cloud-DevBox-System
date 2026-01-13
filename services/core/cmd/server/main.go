// Cloud DevBox Core Service
//
// Main business logic service handling users, environments, templates, and projects.
package main

import (
	"log"
	"os"

	"github.com/cloud-devbox/services/core/internal/config"
	"github.com/cloud-devbox/services/core/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting Core Service on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
