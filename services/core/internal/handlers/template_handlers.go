// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	templateService        *services.TemplateService
	templateVersionService *services.TemplateVersionService
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateService *services.TemplateService, templateVersionService *services.TemplateVersionService) *TemplateHandler {
	return &TemplateHandler{
		templateService:        templateService,
		templateVersionService: templateVersionService,
	}
}

// ListTemplatesRequest represents the request for listing templates
type ListTemplatesRequest struct {
	Category string   `form:"category"`
	Tags     []string `form:"tags"`
	Search   string   `form:"search"`
	IsPublic *bool    `form:"is_public"`
	Page     int      `form:"page,default=1"`
	PageSize int      `form:"page_size,default=20"`
}

// ListTemplates handles GET /api/v1/templates
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	var req ListTemplatesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := services.TemplateFilter{
		Category: req.Category,
		Tags:     req.Tags,
		Search:   req.Search,
		IsPublic: req.IsPublic,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	result, err := h.templateService.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        result.Templates,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// GetTemplate handles GET /api/v1/templates/:id
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	template, err := h.templateService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// CreateTemplate handles POST /api/v1/templates
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var input services.CreateTemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	template, err := h.templateService.Create(c.Request.Context(), userID.(uuid.UUID), input)
	if err != nil {
		switch err {
		case services.ErrTemplateNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Template name already exists"})
		case services.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// UpdateTemplate handles PUT /api/v1/templates/:id
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var input services.UpdateTemplateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID and role from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("userRole")
	isAdmin := userRole == string(services.RoleAdmin)

	template, err := h.templateService.Update(c.Request.Context(), id, userID.(uuid.UUID), isAdmin, input)
	if err != nil {
		switch err {
		case services.ErrTemplateNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this template"})
		case services.ErrInvalidCategory:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteTemplate handles DELETE /api/v1/templates/:id
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	// Get user ID and role from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("userRole")
	isAdmin := userRole == string(services.RoleAdmin)

	err = h.templateService.Delete(c.Request.Context(), id, userID.(uuid.UUID), isAdmin)
	if err != nil {
		switch err {
		case services.ErrTemplateNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this template"})
		case services.ErrCannotDeleteSystemTpl:
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system template"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

// GetCategories handles GET /api/v1/templates/categories
func (h *TemplateHandler) GetCategories(c *gin.Context) {
	categories, err := h.templateService.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// GetTags handles GET /api/v1/templates/tags
func (h *TemplateHandler) GetTags(c *gin.Context) {
	tags, err := h.templateService.GetTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tags})
}

// ListVersions handles GET /api/v1/templates/:id/versions
func (h *TemplateHandler) ListVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	// Verify template exists
	_, err = h.templateService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == services.ErrTemplateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template"})
		return
	}

	versions, err := h.templateVersionService.ListVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": versions})
}

// GetVersion handles GET /api/v1/templates/:id/versions/:versionId
func (h *TemplateHandler) GetVersion(c *gin.Context) {
	versionIDStr := c.Param("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID"})
		return
	}

	version, err := h.templateVersionService.GetVersion(c.Request.Context(), versionID)
	if err != nil {
		if err == services.ErrVersionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": version})
}

// CreateVersion handles POST /api/v1/templates/:id/versions
func (h *TemplateHandler) CreateVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var input services.CreateVersionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID and role from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("userRole")
	isAdmin := userRole == string(services.RoleAdmin)

	version, err := h.templateVersionService.CreateVersion(c.Request.Context(), id, userID.(uuid.UUID), isAdmin, input)
	if err != nil {
		switch err {
		case services.ErrTemplateNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to create version"})
		case services.ErrInvalidVersionBump:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version bump type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": version})
}

// RollbackVersion handles POST /api/v1/templates/:id/versions/:versionId/rollback
func (h *TemplateHandler) RollbackVersion(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	versionIDStr := c.Param("versionId")
	versionID, err := uuid.Parse(versionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version ID"})
		return
	}

	// Get user ID and role from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("userRole")
	isAdmin := userRole == string(services.RoleAdmin)

	template, err := h.templateVersionService.RollbackToVersion(c.Request.Context(), id, versionID, userID.(uuid.UUID), isAdmin)
	if err != nil {
		switch err {
		case services.ErrTemplateNotFound, services.ErrVersionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Template or version not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to rollback"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback version"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// GetValidCategories handles GET /api/v1/templates/valid-categories
func (h *TemplateHandler) GetValidCategories(c *gin.Context) {
	categories := models.ValidCategories()
	c.JSON(http.StatusOK, gin.H{"data": categories})
}
