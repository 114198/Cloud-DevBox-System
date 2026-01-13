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

// Container handlers (placeholders)
func CreateContainer(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func GetContainer(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func DeleteContainer(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func StartContainer(c *gin.Context)   { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func StopContainer(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func RestartContainer(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func GetContainerLogs(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func GetContainerMetrics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
