// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthCheck handles health check requests
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "0.1.0",
	})
}

// RegisterEnvironmentRoutes registers environment-related routes
func RegisterEnvironmentRoutes(router *gin.RouterGroup, envService *services.EnvironmentService) {
	handler := NewEnvironmentHandler(envService)

	// Environment CRUD
	router.POST("/environments", handler.Create)
	router.GET("/environments", handler.List)
	router.GET("/environments/stats", handler.GetStats)
	router.GET("/environments/rate-limit", handler.GetRateLimitStatus)
	router.GET("/environments/:id", handler.Get)
	router.PUT("/environments/:id", handler.Update)
	router.DELETE("/environments/:id", handler.Delete)

	// Environment operations
	router.POST("/environments/:id/start", handler.Start)
	router.POST("/environments/:id/stop", handler.Stop)
	router.POST("/environments/:id/restart", handler.Restart)
	router.POST("/environments/:id/suspend", handler.Suspend)
	router.POST("/environments/:id/archive", handler.Archive)
	router.POST("/environments/:id/activity", handler.RecordActivity)

	// Batch operations
	router.POST("/environments/batch", handler.BatchOperation)
}

// RegisterLegacyContainerRoutes maps legacy /containers endpoints to environments.
func RegisterLegacyContainerRoutes(router *gin.RouterGroup, envService *services.EnvironmentService) {
	handler := NewEnvironmentHandler(envService)

	router.POST("", handler.Create)
	router.GET("/:id", handler.Get)
	router.DELETE("/:id", handler.Delete)
	router.POST("/:id/start", handler.Start)
	router.POST("/:id/stop", handler.Stop)
	router.POST("/:id/restart", handler.Restart)
	router.GET("/:id/logs", handler.GetLogs)
	router.GET("/:id/metrics", handler.GetMetrics)
}
