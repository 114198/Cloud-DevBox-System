// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"
	"strings"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// EnvironmentHandler handles environment-related HTTP requests
type EnvironmentHandler struct {
	service *services.EnvironmentService
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(service *services.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{service: service}
}

// Create handles POST /api/v1/environments
func (h *EnvironmentHandler) Create(c *gin.Context) {
	var req models.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	env, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		// Check for rate limit error and return 429
		if rateLimitErr, ok := err.(*services.RateLimitError); ok {
			c.Header("Retry-After", rateLimitErr.RetryAfter.String())
			c.Header("X-RateLimit-Limit", string(rune(rateLimitErr.Limit)))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        rateLimitErr.Message,
				"limit":        rateLimitErr.Limit,
				"window":       rateLimitErr.Window.String(),
				"retryAfter":   rateLimitErr.RetryAfter.String(),
				"currentCount": rateLimitErr.CurrentCount,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, env)
}

// Get handles GET /api/v1/environments/:id
func (h *EnvironmentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// List handles GET /api/v1/environments
func (h *EnvironmentHandler) List(c *gin.Context) {
	var req models.ListEnvironmentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Filter by current user if not specified
	if req.UserID == "" {
		req.UserID = getUserID(c)
	}

	resp, err := h.service.List(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Update handles PUT /api/v1/environments/:id
func (h *EnvironmentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	var req models.UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.service.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// Delete handles DELETE /api/v1/environments/:id
func (h *EnvironmentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	err := h.service.Delete(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Start handles POST /api/v1/environments/:id/start
func (h *EnvironmentHandler) Start(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Start(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Check for invalid state transition
		if contains(err.Error(), "cannot start environment") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// Stop handles POST /api/v1/environments/:id/stop
func (h *EnvironmentHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Stop(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Check for invalid state transition
		if contains(err.Error(), "cannot stop environment") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// Suspend handles POST /api/v1/environments/:id/suspend
func (h *EnvironmentHandler) Suspend(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Suspend(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Check for invalid state transition
		if contains(err.Error(), "cannot suspend environment") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// Archive handles POST /api/v1/environments/:id/archive
func (h *EnvironmentHandler) Archive(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Archive(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		// Check for invalid state transition
		if contains(err.Error(), "cannot archive environment") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Restart handles POST /api/v1/environments/:id/restart
func (h *EnvironmentHandler) Restart(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	env, err := h.service.Restart(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// BatchOperation handles POST /api/v1/environments/batch
func (h *EnvironmentHandler) BatchOperation(c *gin.Context) {
	var req models.BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	resp, err := h.service.BatchOperation(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetStats handles GET /api/v1/environments/stats
func (h *EnvironmentHandler) GetStats(c *gin.Context) {
	userID := getUserID(c)

	stats, err := h.service.GetStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRateLimitStatus handles GET /api/v1/environments/rate-limit
func (h *EnvironmentHandler) GetRateLimitStatus(c *gin.Context) {
	userID := getUserID(c)

	status := h.service.GetRateLimitStatus(userID)
	c.JSON(http.StatusOK, gin.H{
		"allowed":      status.Allowed,
		"currentCount": status.CurrentCount,
		"limit":        status.Limit,
		"window":       status.Window.String(),
		"remaining":    status.Limit - status.CurrentCount,
	})
}

// RecordActivity handles POST /api/v1/environments/:id/activity
func (h *EnvironmentHandler) RecordActivity(c *gin.Context) {
	id := c.Param("id")

	err := h.service.UpdateActivity(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity recorded"})
}
