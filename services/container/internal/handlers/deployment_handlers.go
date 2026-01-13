// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"io"
	"net/http"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// DeploymentHandler handles deployment-related HTTP requests
type DeploymentHandler struct {
	service *services.DeploymentService
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(service *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{service: service}
}

// Create handles POST /api/v1/deployments
func (h *DeploymentHandler) Create(c *gin.Context) {
	var req models.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	deployment, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

// Get handles GET /api/v1/deployments/:id
func (h *DeploymentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	deployment, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// List handles GET /api/v1/deployments
func (h *DeploymentHandler) List(c *gin.Context) {
	var req models.ListDeploymentsRequest
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

// Delete handles DELETE /api/v1/deployments/:id
func (h *DeploymentHandler) Delete(c *gin.Context) {
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

// Rollback handles POST /api/v1/deployments/:id/rollback
func (h *DeploymentHandler) Rollback(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	var req models.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deployment, err := h.service.Rollback(c.Request.Context(), userID, id, &req)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// GetLogs handles GET /api/v1/deployments/:id/logs
func (h *DeploymentHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	logs, err := h.service.GetLogs(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// GetHistory handles GET /api/v1/deployments/:id/history
func (h *DeploymentHandler) GetHistory(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	history, err := h.service.GetHistory(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GenerateDockerfile handles POST /api/v1/deployments/dockerfile/generate
func (h *DeploymentHandler) GenerateDockerfile(c *gin.Context) {
	var req models.GenerateDockerfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.GenerateDockerfile(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetDockerfileTemplates handles GET /api/v1/deployments/dockerfile/templates
func (h *DeploymentHandler) GetDockerfileTemplates(c *gin.Context) {
	templates := h.service.GetDockerfileTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// StreamLogs handles GET /api/v1/deployments/:id/logs/stream (SSE)
func (h *DeploymentHandler) StreamLogs(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	logChan, err := h.service.StreamLogs(c.Request.Context(), userID, id)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		select {
		case log, ok := <-logChan:
			if !ok {
				return false
			}
			c.SSEvent("log", log)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// RegisterDeploymentRoutes registers deployment-related routes
func RegisterDeploymentRoutes(router *gin.RouterGroup, deploymentService *services.DeploymentService) {
	handler := NewDeploymentHandler(deploymentService)

	// Deployment CRUD
	router.POST("/deployments", handler.Create)
	router.GET("/deployments", handler.List)
	router.GET("/deployments/:id", handler.Get)
	router.DELETE("/deployments/:id", handler.Delete)

	// Deployment operations
	router.POST("/deployments/:id/rollback", handler.Rollback)

	// Logs and history
	router.GET("/deployments/:id/logs", handler.GetLogs)
	router.GET("/deployments/:id/logs/stream", handler.StreamLogs)
	router.GET("/deployments/:id/history", handler.GetHistory)

	// Dockerfile generation
	router.POST("/deployments/dockerfile/generate", handler.GenerateDockerfile)
	router.GET("/deployments/dockerfile/templates", handler.GetDockerfileTemplates)
}
