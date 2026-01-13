// Cloud DevBox Container Service
//
// Kubernetes-based container management service.
package main

import (
	"log"
	"os"

	"github.com/cloud-devbox/services/container/internal/config"
	"github.com/cloud-devbox/services/container/internal/server"
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
		port = "8083"
	}

	log.Printf("Starting Container Service on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
