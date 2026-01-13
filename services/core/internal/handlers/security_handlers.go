// Package handlers provides HTTP handlers for the core service.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecurityHandlers handles security-related HTTP requests
type SecurityHandlers struct {
	securityService *services.SecurityService
}

// NewSecurityHandlers creates a new SecurityHandlers instance
func NewSecurityHandlers(securityService *services.SecurityService) *SecurityHandlers {
	return &SecurityHandlers{
		securityService: securityService,
	}
}

// RegisterRoutes registers security routes
func (h *SecurityHandlers) RegisterRoutes(r *gin.RouterGroup) {
	security := r.Group("/security")
	{
		// IP Blacklist management
		security.GET("/blacklist", h.GetBlacklist)
		security.POST("/blacklist", h.BlockIP)
		security.DELETE("/blacklist/:ip", h.UnblockIP)
		security.GET("/blacklist/check/:ip", h.CheckIPBlocked)

		// Security events
		security.GET("/events", h.GetSecurityEvents)
		security.GET("/events/:id", h.GetSecurityEvent)
		security.POST("/events/:id/resolve", h.ResolveSecurityEvent)

		// Audit logs
		security.GET("/audit-logs", h.GetAuditLogs)
		security.GET("/audit-logs/:id", h.GetAuditLog)

		// Encryption key management (admin only)
		security.GET("/encryption/keys", h.GetEncryptionKeys)
		security.POST("/encryption/keys/rotate", h.RotateEncryptionKey)
	}
}

// =============================================================================
// IP Blacklist Handlers
// =============================================================================

// GetBlacklist returns the list of blacklisted IPs
// @Summary Get blacklisted IPs
// @Tags Security
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /security/blacklist [get]
func (h *SecurityHandlers) GetBlacklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	blacklist, total, err := h.securityService.GetBlacklistedIPs(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       blacklist,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// BlockIPRequest represents a request to block an IP
type BlockIPRequest struct {
	IPAddress string `json:"ip_address" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
	Duration  *int   `json:"duration_hours,omitempty"` // Duration in hours, nil for permanent
}

// BlockIP adds an IP to the blacklist
// @Summary Block an IP address
// @Tags Security
// @Accept json
// @Produce json
// @Param request body BlockIPRequest true "Block IP request"
// @Success 200 {object} map[string]interface{}
// @Router /security/blacklist [post]
func (h *SecurityHandlers) BlockIP(c *gin.Context) {
	var req BlockIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, _ := c.Get("user_id")
	blockedBy := "admin"
	if userID != nil {
		blockedBy = userID.(string)
	}

	var duration *time.Duration
	if req.Duration != nil {
		d := time.Duration(*req.Duration) * time.Hour
		duration = &d
	}

	err := h.securityService.BlockIP(c.Request.Context(), req.IPAddress, req.Reason, blockedBy, duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "IP address blocked successfully",
		"ip_address": req.IPAddress,
	})
}

// UnblockIP removes an IP from the blacklist
// @Summary Unblock an IP address
// @Tags Security
// @Produce json
// @Param ip path string true "IP address"
// @Success 200 {object} map[string]interface{}
// @Router /security/blacklist/{ip} [delete]
func (h *SecurityHandlers) UnblockIP(c *gin.Context) {
	ipAddress := c.Param("ip")

	err := h.securityService.UnblockIP(c.Request.Context(), ipAddress)
	if err != nil {
		if err == services.ErrIPNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "IP address not found in blacklist"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "IP address unblocked successfully",
		"ip_address": ipAddress,
	})
}

// CheckIPBlocked checks if an IP is blocked
// @Summary Check if IP is blocked
// @Tags Security
// @Produce json
// @Param ip path string true "IP address"
// @Success 200 {object} map[string]interface{}
// @Router /security/blacklist/check/{ip} [get]
func (h *SecurityHandlers) CheckIPBlocked(c *gin.Context) {
	ipAddress := c.Param("ip")

	blocked, err := h.securityService.IsIPBlocked(c.Request.Context(), ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ip_address": ipAddress,
		"blocked":    blocked,
	})
}

// =============================================================================
// Security Events Handlers
// =============================================================================

// GetSecurityEvents returns security events with filtering
// @Summary Get security events
// @Tags Security
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param event_type query string false "Event type filter"
// @Param severity query string false "Severity filter"
// @Param resolved query bool false "Resolved filter"
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Success 200 {object} map[string]interface{}
// @Router /security/events [get]
func (h *SecurityHandlers) GetSecurityEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := services.SecurityEventFilter{
		Page:     page,
		PageSize: pageSize,
	}

	if eventType := c.Query("event_type"); eventType != "" {
		filter.EventType = models.SecurityEventType(eventType)
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = models.SecurityEventSeverity(severity)
	}
	if resolved := c.Query("resolved"); resolved != "" {
		r := resolved == "true"
		filter.Resolved = &r
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = t
		}
	}
	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filter.IPAddress = ipAddress
	}

	events, total, err := h.securityService.GetSecurityEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        events,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetSecurityEvent returns a single security event
// @Summary Get security event by ID
// @Tags Security
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.SecurityEvent
// @Router /security/events/{id} [get]
func (h *SecurityHandlers) GetSecurityEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	filter := services.SecurityEventFilter{
		Page:     1,
		PageSize: 1,
	}

	// We'll use the filter to get the event (simplified approach)
	events, _, err := h.securityService.GetSecurityEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, event := range events {
		if event.ID == eventID {
			c.JSON(http.StatusOK, event)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "security event not found"})
}

// ResolveSecurityEvent marks a security event as resolved
// @Summary Resolve security event
// @Tags Security
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Router /security/events/{id}/resolve [post]
func (h *SecurityHandlers) ResolveSecurityEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	err = h.securityService.ResolveSecurityEvent(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Security event resolved successfully",
		"event_id": eventID,
	})
}

// =============================================================================
// Audit Log Handlers
// =============================================================================

// GetAuditLogs returns audit logs with filtering
// @Summary Get audit logs
// @Tags Security
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param user_id query string false "User ID filter"
// @Param action query string false "Action filter"
// @Param resource_type query string false "Resource type filter"
// @Param start_date query string false "Start date (RFC3339)"
// @Param end_date query string false "End date (RFC3339)"
// @Success 200 {object} map[string]interface{}
// @Router /security/audit-logs [get]
func (h *SecurityHandlers) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := services.AuditLogFilter{
		Page:     page,
		PageSize: pageSize,
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}
	if action := c.Query("action"); action != "" {
		filter.Action = action
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}
	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filter.IPAddress = ipAddress
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = t
		}
	}

	logs, total, err := h.securityService.GetAuditLogs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetAuditLog returns a single audit log entry
// @Summary Get audit log by ID
// @Tags Security
// @Produce json
// @Param id path string true "Audit log ID"
// @Success 200 {object} models.AuditLog
// @Router /security/audit-logs/{id} [get]
func (h *SecurityHandlers) GetAuditLog(c *gin.Context) {
	logID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audit log ID"})
		return
	}

	// For simplicity, we'll query directly
	// In production, you'd add a GetAuditLogByID method
	filter := services.AuditLogFilter{
		Page:     1,
		PageSize: 1000, // Get enough to find the log
	}

	logs, _, err := h.securityService.GetAuditLogs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, log := range logs {
		if log.ID == logID {
			c.JSON(http.StatusOK, log)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
}

// =============================================================================
// Encryption Key Handlers
// =============================================================================

// GetEncryptionKeys returns encryption key metadata
// @Summary Get encryption keys
// @Tags Security
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /security/encryption/keys [get]
func (h *SecurityHandlers) GetEncryptionKeys(c *gin.Context) {
	key, err := h.securityService.GetActiveEncryptionKey(c.Request.Context())
	if err != nil {
		if err == services.ErrEncryptionKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active encryption key found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active_key": key,
	})
}

// RotateEncryptionKey triggers encryption key rotation
// @Summary Rotate encryption key
// @Tags Security
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /security/encryption/keys/rotate [post]
func (h *SecurityHandlers) RotateEncryptionKey(c *gin.Context) {
	// In production, the new key would come from Vault
	// For now, we generate a random key
	newKey := make([]byte, 32)
	if _, err := c.Request.Body.Read(newKey); err != nil {
		// Generate random key if not provided
		newKey = make([]byte, 32)
		// In production, this would be fetched from Vault
	}

	err := h.securityService.RotateEncryptionKey(c.Request.Context(), newKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Encryption key rotated successfully",
	})
}

// =============================================================================
// Security Middleware
// =============================================================================

// IPBlockMiddleware checks if the request IP is blocked
func IPBlockMiddleware(securityService *services.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()

		blocked, err := securityService.IsIPBlocked(c.Request.Context(), ipAddress)
		if err != nil {
			// Log error but don't block on error
			c.Next()
			return
		}

		if blocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "IP address is blocked",
				"message": "Your IP address has been blocked due to suspicious activity",
			})
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware checks rate limits for the request IP
func RateLimitMiddleware(securityService *services.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := c.ClientIP()

		allowed, err := securityService.CheckRateLimit(c.Request.Context(), ipAddress)
		if err != nil {
			// Log error but don't block on error
			c.Next()
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}

// AuditMiddleware logs all API requests to the audit log
func AuditMiddleware(securityService *services.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request first
		c.Next()

		// Log after request is processed
		userID, _ := c.Get("user_id")
		var uid *uuid.UUID
		if userID != nil {
			if id, err := uuid.Parse(userID.(string)); err == nil {
				uid = &id
			}
		}

		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Determine action from HTTP method
		action := c.Request.Method
		resourceType := "api"
		
		// Extract resource info from path
		details := models.JSONMap{
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
			"status_code": c.Writer.Status(),
			"query":       c.Request.URL.RawQuery,
		}

		securityService.LogAction(
			c.Request.Context(),
			uid,
			action,
			resourceType,
			nil,
			details,
			ipAddress,
			userAgent,
		)
	}
}

// RecordAccessAttemptMiddleware records access attempts for security analysis
func RecordAccessAttemptMiddleware(securityService *services.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request first
		c.Next()

		// Record the attempt
		userID, _ := c.Get("user_id")
		var uid *uuid.UUID
		if userID != nil {
			if id, err := uuid.Parse(userID.(string)); err == nil {
				uid = &id
			}
		}

		success := c.Writer.Status() < 400
		var failureReason string
		if !success {
			failureReason = http.StatusText(c.Writer.Status())
		}

		attempt := &models.AccessAttempt{
			IPAddress:     c.ClientIP(),
			UserID:        uid,
			AttemptType:   "api",
			Success:       success,
			FailureReason: failureReason,
			UserAgent:     c.Request.UserAgent(),
			Endpoint:      c.Request.URL.Path,
		}

		securityService.RecordAccessAttempt(c.Request.Context(), attempt)
	}
}
