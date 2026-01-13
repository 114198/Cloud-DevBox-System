// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// PerformanceHandler handles performance monitoring HTTP requests
type PerformanceHandler struct {
	metricsCollector    *services.MetricsCollector
	alertManagerService *services.AlertManagerService
	autoScalingService  *services.AutoScalingService
	grafanaService      *services.GrafanaService
}

// NewPerformanceHandler creates a new performance handler
func NewPerformanceHandler(
	metricsCollector *services.MetricsCollector,
	alertManagerService *services.AlertManagerService,
	autoScalingService *services.AutoScalingService,
	grafanaService *services.GrafanaService,
) *PerformanceHandler {
	return &PerformanceHandler{
		metricsCollector:    metricsCollector,
		alertManagerService: alertManagerService,
		autoScalingService:  autoScalingService,
		grafanaService:      grafanaService,
	}
}

// GetSystemMetrics returns current system metrics
// @Summary Get system metrics
// @Description Get current system-level metrics
// @Tags performance
// @Produce json
// @Success 200 {object} models.SystemMetrics
// @Router /performance/system [get]
func (h *PerformanceHandler) GetSystemMetrics(c *gin.Context) {
	if h.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "metrics collector not available"})
		return
	}

	metrics := h.metricsCollector.GetSystemMetrics()
	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetBusinessMetrics returns current business metrics
// @Summary Get business metrics
// @Description Get current business-level metrics
// @Tags performance
// @Produce json
// @Success 200 {object} models.BusinessMetrics
// @Router /performance/business [get]
func (h *PerformanceHandler) GetBusinessMetrics(c *gin.Context) {
	if h.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "metrics collector not available"})
		return
	}

	metrics := h.metricsCollector.GetBusinessMetrics()
	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

// GetCollectionStats returns metrics collection statistics
// @Summary Get collection statistics
// @Description Get metrics collection statistics
// @Tags performance
// @Produce json
// @Success 200 {object} models.CollectionStats
// @Router /performance/stats [get]
func (h *PerformanceHandler) GetCollectionStats(c *gin.Context) {
	if h.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "metrics collector not available"})
		return
	}

	stats := h.metricsCollector.GetCollectionStats()
	c.JSON(http.StatusOK, gin.H{"data": stats})
}


// GetActiveAlerts returns active performance alerts
// @Summary Get active alerts
// @Description Get active performance alerts from AlertManager
// @Tags performance
// @Produce json
// @Success 200 {array} models.PerformanceAlert
// @Router /performance/alerts [get]
func (h *PerformanceHandler) GetActiveAlerts(c *gin.Context) {
	if h.alertManagerService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "alertmanager service not available"})
		return
	}

	alerts := h.alertManagerService.GetActiveAlerts()
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

// GetAlertHistory returns alert history
// @Summary Get alert history
// @Description Get performance alert history
// @Tags performance
// @Produce json
// @Param limit query int false "Limit number of alerts"
// @Success 200 {array} models.PerformanceAlert
// @Router /performance/alerts/history [get]
func (h *PerformanceHandler) GetAlertHistory(c *gin.Context) {
	if h.alertManagerService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "alertmanager service not available"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if _, err := c.GetQuery("limit"); err {
			// Use default
		}
	}

	alerts := h.alertManagerService.GetAlertHistory(limit)
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

// GetScalingEvents returns auto-scaling events
// @Summary Get scaling events
// @Description Get auto-scaling events history
// @Tags performance
// @Produce json
// @Param limit query int false "Limit number of events"
// @Success 200 {array} models.ScalingEvent
// @Router /performance/scaling/events [get]
func (h *PerformanceHandler) GetScalingEvents(c *gin.Context) {
	if h.autoScalingService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "autoscaling service not available"})
		return
	}

	limit := 100
	events := h.autoScalingService.GetScalingEvents(limit)
	c.JSON(http.StatusOK, gin.H{"data": events})
}

// GetSystemDashboard returns the system monitoring dashboard configuration
// @Summary Get system dashboard
// @Description Get Grafana system monitoring dashboard configuration
// @Tags performance
// @Produce json
// @Success 200 {object} models.GrafanaDashboard
// @Router /performance/dashboards/system [get]
func (h *PerformanceHandler) GetSystemDashboard(c *gin.Context) {
	if h.grafanaService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "grafana service not available"})
		return
	}

	dashboard := h.grafanaService.GetSystemDashboardConfig()
	c.JSON(http.StatusOK, gin.H{"data": dashboard})
}

// GetBusinessDashboard returns the business metrics dashboard configuration
// @Summary Get business dashboard
// @Description Get Grafana business metrics dashboard configuration
// @Tags performance
// @Produce json
// @Success 200 {object} models.GrafanaDashboard
// @Router /performance/dashboards/business [get]
func (h *PerformanceHandler) GetBusinessDashboard(c *gin.Context) {
	if h.grafanaService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "grafana service not available"})
		return
	}

	dashboard := h.grafanaService.GetBusinessDashboardConfig()
	c.JSON(http.StatusOK, gin.H{"data": dashboard})
}

// CreateDashboards creates both system and business dashboards in Grafana
// @Summary Create dashboards
// @Description Create system and business dashboards in Grafana
// @Tags performance
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /performance/dashboards [post]
func (h *PerformanceHandler) CreateDashboards(c *gin.Context) {
	if h.grafanaService == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "grafana service not available"})
		return
	}

	systemDashboard, err := h.grafanaService.CreateSystemMonitoringDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	businessDashboard, err := h.grafanaService.CreateBusinessMetricsDashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"systemDashboard":   systemDashboard,
			"businessDashboard": businessDashboard,
		},
	})
}

// RegisterPerformanceRoutes registers performance monitoring routes
func RegisterPerformanceRoutes(
	router *gin.RouterGroup,
	metricsCollector *services.MetricsCollector,
	alertManagerService *services.AlertManagerService,
	autoScalingService *services.AutoScalingService,
	grafanaService *services.GrafanaService,
) {
	handler := NewPerformanceHandler(metricsCollector, alertManagerService, autoScalingService, grafanaService)

	perf := router.Group("/performance")
	{
		// Metrics
		perf.GET("/system", handler.GetSystemMetrics)
		perf.GET("/business", handler.GetBusinessMetrics)
		perf.GET("/stats", handler.GetCollectionStats)

		// Alerts
		perf.GET("/alerts", handler.GetActiveAlerts)
		perf.GET("/alerts/history", handler.GetAlertHistory)

		// Scaling
		perf.GET("/scaling/events", handler.GetScalingEvents)

		// Dashboards
		perf.GET("/dashboards/system", handler.GetSystemDashboard)
		perf.GET("/dashboards/business", handler.GetBusinessDashboard)
		perf.POST("/dashboards", handler.CreateDashboards)
	}
}
