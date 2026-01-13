// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// PreviewHandler handles preview-related HTTP requests
type PreviewHandler struct {
	previewService   *services.PreviewService
	hotReloadService *services.HotReloadService
}

// NewPreviewHandler creates a new preview handler
func NewPreviewHandler(previewService *services.PreviewService, hotReloadService *services.HotReloadService) *PreviewHandler {
	return &PreviewHandler{
		previewService:   previewService,
		hotReloadService: hotReloadService,
	}
}

// getUserID extracts user ID from context (set by auth middleware)
func (h *PreviewHandler) getUserID(c *gin.Context) string {
	userID := c.GetString("userId")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		userID = "anonymous"
	}
	return userID
}

// CreateDomain creates a new domain
// @Summary Create a new domain
// @Tags Preview
// @Accept json
// @Produce json
// @Param request body models.CreateDomainRequest true "Create domain request"
// @Success 201 {object} models.Domain
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/preview/domains [post]
func (h *PreviewHandler) CreateDomain(c *gin.Context) {
	var req models.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	userID := h.getUserID(c)
	domain, err := h.previewService.CreateDomain(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, domain)
}

// GetDomain retrieves a domain by ID
// @Summary Get domain by ID
// @Tags Preview
// @Produce json
// @Param id path string true "Domain ID"
// @Success 200 {object} models.Domain
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/domains/{id} [get]
func (h *PreviewHandler) GetDomain(c *gin.Context) {
	domainID := c.Param("id")
	userID := h.getUserID(c)

	domain, err := h.previewService.GetDomain(c.Request.Context(), userID, domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// ListDomains lists domains with filtering
// @Summary List domains
// @Tags Preview
// @Produce json
// @Param environmentId query string false "Environment ID"
// @Param type query string false "Domain type"
// @Param status query string false "Domain status"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} models.ListDomainsResponse
// @Router /api/v1/preview/domains [get]
func (h *PreviewHandler) ListDomains(c *gin.Context) {
	var req models.ListDomainsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Set user ID from context
	req.UserID = h.getUserID(c)

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.previewService.ListDomains(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateDomain updates a domain
// @Summary Update domain
// @Tags Preview
// @Accept json
// @Produce json
// @Param id path string true "Domain ID"
// @Param request body models.UpdateDomainRequest true "Update domain request"
// @Success 200 {object} models.Domain
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/domains/{id} [put]
func (h *PreviewHandler) UpdateDomain(c *gin.Context) {
	domainID := c.Param("id")
	userID := h.getUserID(c)

	var req models.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	domain, err := h.previewService.UpdateDomain(c.Request.Context(), userID, domainID, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// DeleteDomain deletes a domain
// @Summary Delete domain
// @Tags Preview
// @Param id path string true "Domain ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/domains/{id} [delete]
func (h *PreviewHandler) DeleteDomain(c *gin.Context) {
	domainID := c.Param("id")
	userID := h.getUserID(c)

	if err := h.previewService.DeleteDomain(c.Request.Context(), userID, domainID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// VerifyDomain verifies a custom domain's DNS configuration
// @Summary Verify custom domain
// @Tags Preview
// @Produce json
// @Param id path string true "Domain ID"
// @Success 200 {object} models.DomainValidationResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/domains/{id}/verify [post]
func (h *PreviewHandler) VerifyDomain(c *gin.Context) {
	domainID := c.Param("id")
	userID := h.getUserID(c)

	result, err := h.previewService.VerifyCustomDomain(c.Request.Context(), userID, domainID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDNSInstructions returns DNS configuration instructions
// @Summary Get DNS instructions for custom domain
// @Tags Preview
// @Produce json
// @Param id path string true "Domain ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/domains/{id}/dns-instructions [get]
func (h *PreviewHandler) GetDNSInstructions(c *gin.Context) {
	domainID := c.Param("id")
	userID := h.getUserID(c)

	instructions, err := h.previewService.GetDNSInstructions(c.Request.Context(), userID, domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, instructions)
}


// CreateShareLink creates a new share link
// @Summary Create a share link
// @Tags Preview
// @Accept json
// @Produce json
// @Param request body models.CreateShareLinkRequest true "Create share link request"
// @Success 201 {object} models.ShareLink
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/preview/share-links [post]
func (h *PreviewHandler) CreateShareLink(c *gin.Context) {
	var req models.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	userID := h.getUserID(c)
	shareLink, err := h.previewService.CreateShareLink(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, shareLink)
}

// GetShareLink retrieves a share link by ID
// @Summary Get share link by ID
// @Tags Preview
// @Produce json
// @Param id path string true "Share link ID"
// @Success 200 {object} models.ShareLink
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/share-links/{id} [get]
func (h *PreviewHandler) GetShareLink(c *gin.Context) {
	shareLinkID := c.Param("id")
	userID := h.getUserID(c)

	shareLink, err := h.previewService.GetShareLink(c.Request.Context(), userID, shareLinkID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shareLink)
}

// ListShareLinks lists share links with filtering
// @Summary List share links
// @Tags Preview
// @Produce json
// @Param domainId query string false "Domain ID"
// @Param environmentId query string false "Environment ID"
// @Param activeOnly query bool false "Active only"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} models.ListShareLinksResponse
// @Router /api/v1/preview/share-links [get]
func (h *PreviewHandler) ListShareLinks(c *gin.Context) {
	var req models.ListShareLinksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Set user ID from context
	req.UserID = h.getUserID(c)

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	response, err := h.previewService.ListShareLinks(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ValidateShareLink validates a share link for public access
// @Summary Validate share link
// @Tags Preview
// @Accept json
// @Produce json
// @Param request body models.ValidateShareLinkRequest true "Validate share link request"
// @Success 200 {object} models.ShareLink
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/preview/share-links/validate [post]
func (h *PreviewHandler) ValidateShareLink(c *gin.Context) {
	var req models.ValidateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	shareLink, err := h.previewService.ValidateShareLink(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, shareLink)
}

// RevokeShareLink revokes a share link
// @Summary Revoke share link
// @Tags Preview
// @Param id path string true "Share link ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/share-links/{id}/revoke [post]
func (h *PreviewHandler) RevokeShareLink(c *gin.Context) {
	shareLinkID := c.Param("id")
	userID := h.getUserID(c)

	if err := h.previewService.RevokeShareLink(c.Request.Context(), userID, shareLinkID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share link revoked"})
}

// DeleteShareLink deletes a share link
// @Summary Delete share link
// @Tags Preview
// @Param id path string true "Share link ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/preview/share-links/{id} [delete]
func (h *PreviewHandler) DeleteShareLink(c *gin.Context) {
	shareLinkID := c.Param("id")
	userID := h.getUserID(c)

	if err := h.previewService.DeleteShareLink(c.Request.Context(), userID, shareLinkID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetPreviewStats returns preview statistics
// @Summary Get preview statistics
// @Tags Preview
// @Produce json
// @Success 200 {object} models.PreviewStats
// @Router /api/v1/preview/stats [get]
func (h *PreviewHandler) GetPreviewStats(c *gin.Context) {
	userID := h.getUserID(c)

	stats, err := h.previewService.GetPreviewStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetHotReloadConfig retrieves hot reload configuration
// @Summary Get hot reload configuration
// @Tags Preview
// @Produce json
// @Param environmentId path string true "Environment ID"
// @Success 200 {object} models.HotReloadConfig
// @Router /api/v1/preview/environments/{environmentId}/hotreload [get]
func (h *PreviewHandler) GetHotReloadConfig(c *gin.Context) {
	environmentID := c.Param("environmentId")

	config, err := h.hotReloadService.GetConfig(c.Request.Context(), environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SetHotReloadConfig sets hot reload configuration
// @Summary Set hot reload configuration
// @Tags Preview
// @Accept json
// @Produce json
// @Param environmentId path string true "Environment ID"
// @Param request body models.HotReloadConfig true "Hot reload configuration"
// @Success 200 {object} models.HotReloadConfig
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/preview/environments/{environmentId}/hotreload [put]
func (h *PreviewHandler) SetHotReloadConfig(c *gin.Context) {
	environmentID := c.Param("environmentId")

	var config models.HotReloadConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.hotReloadService.SetConfig(c.Request.Context(), environmentID, &config); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// NotifyFileChange notifies clients of a file change
// @Summary Notify file change
// @Tags Preview
// @Accept json
// @Produce json
// @Param request body models.FileChangeEvent true "File change event"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/preview/file-change [post]
func (h *PreviewHandler) NotifyFileChange(c *gin.Context) {
	var event models.FileChangeEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.hotReloadService.NotifyFileChange(c.Request.Context(), &event); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File change notified"})
}

// GetActiveSessions returns active preview sessions
// @Summary Get active preview sessions
// @Tags Preview
// @Produce json
// @Param environmentId path string true "Environment ID"
// @Success 200 {array} models.PreviewSession
// @Router /api/v1/preview/environments/{environmentId}/sessions [get]
func (h *PreviewHandler) GetActiveSessions(c *gin.Context) {
	environmentID := c.Param("environmentId")

	sessions, err := h.hotReloadService.GetActiveSessions(c.Request.Context(), environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// RegisterPreviewRoutes registers preview-related routes
func RegisterPreviewRoutes(router *gin.RouterGroup, previewService *services.PreviewService, hotReloadService *services.HotReloadService) {
	handler := NewPreviewHandler(previewService, hotReloadService)

	// Domain routes
	router.POST("/preview/domains", handler.CreateDomain)
	router.GET("/preview/domains", handler.ListDomains)
	router.GET("/preview/domains/:id", handler.GetDomain)
	router.PUT("/preview/domains/:id", handler.UpdateDomain)
	router.DELETE("/preview/domains/:id", handler.DeleteDomain)
	router.POST("/preview/domains/:id/verify", handler.VerifyDomain)
	router.GET("/preview/domains/:id/dns-instructions", handler.GetDNSInstructions)

	// Share link routes
	router.POST("/preview/share-links", handler.CreateShareLink)
	router.GET("/preview/share-links", handler.ListShareLinks)
	router.GET("/preview/share-links/:id", handler.GetShareLink)
	router.DELETE("/preview/share-links/:id", handler.DeleteShareLink)
	router.POST("/preview/share-links/:id/revoke", handler.RevokeShareLink)
	router.POST("/preview/share-links/validate", handler.ValidateShareLink)

	// Stats
	router.GET("/preview/stats", handler.GetPreviewStats)

	// Hot reload routes
	router.GET("/preview/environments/:environmentId/hotreload", handler.GetHotReloadConfig)
	router.PUT("/preview/environments/:environmentId/hotreload", handler.SetHotReloadConfig)
	router.POST("/preview/file-change", handler.NotifyFileChange)
	router.GET("/preview/environments/:environmentId/sessions", handler.GetActiveSessions)
}
