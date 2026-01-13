// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// MonitoringHandler handles monitoring-related HTTP requests
type MonitoringHandler struct {
	monitoringService *services.MonitoringService
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(monitoringService *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
	}
}

// GetMetrics returns current metrics for an environment
// @Summary Get current metrics
// @Description Get current resource metrics for an environment
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Environment ID"
// @Success 200 {object} models.EnvironmentMetrics
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /environments/{id}/metrics [get]
func (h *MonitoringHandler) GetMetrics(c *gin.Context) {
	environmentID := c.Param("id")
	if environmentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "environment ID is required"})
		return
	}

	metrics, err := h.monitoringService.GetMetrics(c.Request.Context(), environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}


// GetMetricsHistory returns historical metrics for an environment
// @Summary Get metrics history
// @Description Get historical resource metrics for an environment
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path string true "Environment ID"
// @Param period query string false "Time period (1h, 6h, 24h, 7d, 30d)"
// @Param resolution query string false "Data resolution (1m, 5m, 15m, 1h)"
// @Success 200 {object} models.MetricsHistory
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /environments/{id}/metrics/history [get]
func (h *MonitoringHandler) GetMetricsHistory(c *gin.Context) {
	environmentID := c.Param("id")
	if environmentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "environment ID is required"})
		return
	}

	req := &models.GetMetricsHistoryRequest{
		EnvironmentID: environmentID,
		Period:        c.DefaultQuery("period", "1h"),
		Resolution:    c.DefaultQuery("resolution", "1m"),
	}

	history, err := h.monitoringService.GetMetricsHistory(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": history})
}

// ListAlerts returns alerts for the user
// @Summary List alerts
// @Description List alerts with optional filtering
// @Tags alerts
// @Accept json
// @Produce json
// @Param environmentId query string false "Filter by environment ID"
// @Param status query string false "Filter by status (active, resolved, acknowledged)"
// @Param severity query string false "Filter by severity (info, warning, critical)"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} models.ListAlertsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts [get]
func (h *MonitoringHandler) ListAlerts(c *gin.Context) {
	userID := c.GetString("userId")

	var req models.ListAlertsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	alertService := h.monitoringService.GetAlertService()
	response, err := alertService.ListAlerts(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}


// GetAlert returns a specific alert
// @Summary Get alert
// @Description Get a specific alert by ID
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} models.Alert
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts/{id} [get]
func (h *MonitoringHandler) GetAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "alert ID is required"})
		return
	}

	alertService := h.monitoringService.GetAlertService()
	alert, err := alertService.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

// AcknowledgeAlert acknowledges an alert
// @Summary Acknowledge alert
// @Description Acknowledge an active alert
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} models.Alert
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts/{id}/acknowledge [post]
func (h *MonitoringHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "alert ID is required"})
		return
	}

	userID := c.GetString("userId")

	alertService := h.monitoringService.GetAlertService()
	alert, err := alertService.AcknowledgeAlert(c.Request.Context(), alertID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

// GetAlertStats returns alert statistics
// @Summary Get alert statistics
// @Description Get alert statistics for the user
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {object} models.AlertStats
// @Failure 500 {object} ErrorResponse
// @Router /alerts/stats [get]
func (h *MonitoringHandler) GetAlertStats(c *gin.Context) {
	userID := c.GetString("userId")

	alertService := h.monitoringService.GetAlertService()
	stats, err := alertService.GetAlertStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}


// ListAlertRules returns alert rules for the user
// @Summary List alert rules
// @Description List alert rules for the user
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {array} models.AlertRule
// @Failure 500 {object} ErrorResponse
// @Router /alerts/rules [get]
func (h *MonitoringHandler) ListAlertRules(c *gin.Context) {
	userID := c.GetString("userId")

	alertService := h.monitoringService.GetAlertService()
	rules, err := alertService.ListAlertRules(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// CreateAlertRule creates a new alert rule
// @Summary Create alert rule
// @Description Create a new alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param request body models.CreateAlertRuleRequest true "Alert rule request"
// @Success 201 {object} models.AlertRule
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts/rules [post]
func (h *MonitoringHandler) CreateAlertRule(c *gin.Context) {
	userID := c.GetString("userId")

	var req models.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	alertService := h.monitoringService.GetAlertService()
	rule, err := alertService.CreateAlertRule(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

// UpdateAlertRule updates an alert rule
// @Summary Update alert rule
// @Description Update an existing alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param request body models.UpdateAlertRuleRequest true "Update request"
// @Success 200 {object} models.AlertRule
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts/rules/{id} [put]
func (h *MonitoringHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "rule ID is required"})
		return
	}

	var req models.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	alertService := h.monitoringService.GetAlertService()
	rule, err := alertService.UpdateAlertRule(c.Request.Context(), ruleID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// DeleteAlertRule deletes an alert rule
// @Summary Delete alert rule
// @Description Delete an alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /alerts/rules/{id} [delete]
func (h *MonitoringHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "rule ID is required"})
		return
	}

	alertService := h.monitoringService.GetAlertService()
	if err := alertService.DeleteAlertRule(c.Request.Context(), ruleID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterMonitoringRoutes registers monitoring-related routes
func RegisterMonitoringRoutes(router *gin.RouterGroup, monitoringService *services.MonitoringService) {
	handler := NewMonitoringHandler(monitoringService)

	// Environment metrics
	router.GET("/environments/:id/metrics", handler.GetMetrics)
	router.GET("/environments/:id/metrics/history", handler.GetMetricsHistory)

	// Alerts
	alerts := router.Group("/alerts")
	{
		alerts.GET("", handler.ListAlerts)
		alerts.GET("/stats", handler.GetAlertStats)
		alerts.GET("/:id", handler.GetAlert)
		alerts.POST("/:id/acknowledge", handler.AcknowledgeAlert)

		// Alert rules
		alerts.GET("/rules", handler.ListAlertRules)
		alerts.POST("/rules", handler.CreateAlertRule)
		alerts.PUT("/rules/:id", handler.UpdateAlertRule)
		alerts.DELETE("/rules/:id", handler.DeleteAlertRule)
	}
}
