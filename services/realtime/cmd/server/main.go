// Cloud DevBox Realtime Service
//
// WebSocket-based real-time collaboration service.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloud-devbox/services/realtime/internal/config"
	"github.com/cloud-devbox/services/realtime/internal/server"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background services
	if err := srv.Start(ctx); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Shutting down...")
		cancel()
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	logger.Info("Starting Realtime Service", zap.String("port", port))
	if err := srv.Run(":" + port); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
