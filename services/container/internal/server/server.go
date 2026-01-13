// Package server provides the HTTP server for the container service.
package server

import (
	"context"

	"github.com/cloud-devbox/services/container/internal/config"
	"github.com/cloud-devbox/services/container/internal/handlers"
	"github.com/cloud-devbox/services/container/internal/k8s/client"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	router              *gin.Engine
	config              *config.Config
	environmentHandler  *handlers.EnvironmentHandler
	monitoringHandler   *handlers.MonitoringHandler
	monitoringService   *services.MonitoringService
	logger              *zap.Logger
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	router := gin.Default()

	// Initialize logger
	logger, _ := zap.NewProduction()

	// Initialize Kubernetes client
	kubeConfig := &client.Config{
		KubeConfig: cfg.KubeConfig,
		Namespace:  cfg.Namespace,
	}
	kubeClient, err := client.NewClient(kubeConfig)
	if err != nil {
		logger.Warn("Failed to create Kubernetes client, running in mock mode", zap.Error(err))
		kubeClient = nil
	}

	// Initialize environment service
	envServiceConfig := &services.EnvironmentServiceConfig{
		Namespace:          cfg.Namespace,
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  20,
		CreationRateWindow: 3600000000000, // 1 hour in nanoseconds
	}
	envService := services.NewEnvironmentService(kubeClient, logger, envServiceConfig)

	// Initialize monitoring service
	monitoringConfig := services.DefaultMonitoringServiceConfig()
	monitoringService := services.NewMonitoringService(monitoringConfig, logger, envService)

	// Initialize handlers
	envHandler := handlers.NewEnvironmentHandler(envService)
	monitoringHandler := handlers.NewMonitoringHandler(monitoringService)

	s := &Server{
		router:              router,
		config:              cfg,
		environmentHandler:  envHandler,
		monitoringHandler:   monitoringHandler,
		monitoringService:   monitoringService,
		logger:              logger,
	}

	s.setupRoutes()

	// Start monitoring service background workers
	monitoringService.Start(context.Background())

	return s, nil
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Environment routes (new)
		environments := v1.Group("/environments")
		{
			environments.POST("", s.environmentHandler.Create)
			environments.GET("", s.environmentHandler.List)
			environments.GET("/stats", s.environmentHandler.GetStats)
			environments.POST("/batch", s.environmentHandler.BatchOperation)
			environments.GET("/:id", s.environmentHandler.Get)
			environments.PUT("/:id", s.environmentHandler.Update)
			environments.DELETE("/:id", s.environmentHandler.Delete)
			environments.POST("/:id/start", s.environmentHandler.Start)
			environments.POST("/:id/stop", s.environmentHandler.Stop)
			environments.POST("/:id/restart", s.environmentHandler.Restart)

			// Monitoring routes for environments
			environments.GET("/:id/metrics", s.monitoringHandler.GetMetrics)
			environments.GET("/:id/metrics/history", s.monitoringHandler.GetMetricsHistory)
		}

		// Alert routes
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", s.monitoringHandler.ListAlerts)
			alerts.GET("/stats", s.monitoringHandler.GetAlertStats)
			alerts.GET("/:id", s.monitoringHandler.GetAlert)
			alerts.POST("/:id/acknowledge", s.monitoringHandler.AcknowledgeAlert)

			// Alert rules
			alerts.GET("/rules", s.monitoringHandler.ListAlertRules)
			alerts.POST("/rules", s.monitoringHandler.CreateAlertRule)
			alerts.PUT("/rules/:id", s.monitoringHandler.UpdateAlertRule)
			alerts.DELETE("/rules/:id", s.monitoringHandler.DeleteAlertRule)
		}

		// Legacy container routes (for backward compatibility)
		containers := v1.Group("/containers")
		{
			containers.POST("", handlers.CreateContainer)
			containers.GET("/:id", handlers.GetContainer)
			containers.DELETE("/:id", handlers.DeleteContainer)
			containers.POST("/:id/start", handlers.StartContainer)
			containers.POST("/:id/stop", handlers.StopContainer)
			containers.POST("/:id/restart", handlers.RestartContainer)
			containers.GET("/:id/logs", handlers.GetContainerLogs)
			containers.GET("/:id/metrics", handlers.GetContainerMetrics)
		}
	}
}
