// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler handles API key related requests
type APIKeyHandler struct {
	service *services.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler
func NewAPIKeyHandler(service *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// CreateAPIKey creates a new API key
// @Summary Create API key
// @Description Create a new API key for programmatic access
// @Tags API Keys
// @Accept json
// @Produce json
// @Param request body services.CreateAPIKeyRequest true "API key creation request"
// @Success 201 {object} services.CreateAPIKeyResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req services.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	resp, err := h.service.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// ListAPIKeys lists all API keys for the current user
// @Summary List API keys
// @Description List all API keys for the authenticated user
// @Tags API Keys
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.APIError
// @Router /api/v1/api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.service.List(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetAPIKey gets an API key by ID
// @Summary Get API key
// @Description Get an API key by ID
// @Tags API Keys
// @Produce json
// @Param id path string true "API key ID"
// @Success 200 {object} models.APIKey
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/api-keys/{id} [get]
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid API key ID"))
		return
	}

	key, err := h.service.Get(userID, keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, key)
}

// UpdateAPIKey updates an API key
// @Summary Update API key
// @Description Update an existing API key
// @Tags API Keys
// @Accept json
// @Produce json
// @Param id path string true "API key ID"
// @Param request body object true "Update request"
// @Success 200 {object} models.APIKey
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/api-keys/{id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid API key ID"))
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Scopes      []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	key, err := h.service.Update(userID, keyID, req.Name, req.Description, req.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, key)
}

// DeleteAPIKey deletes an API key
// @Summary Delete API key
// @Description Delete an API key
// @Tags API Keys
// @Param id path string true "API key ID"
// @Success 204 "No Content"
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/api-keys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid API key ID"))
		return
	}

	if err := h.service.Delete(userID, keyID); err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeAPIKey revokes an API key
// @Summary Revoke API key
// @Description Revoke an API key (deactivate without deleting)
// @Tags API Keys
// @Param id path string true "API key ID"
// @Success 200 {object} object{message=string}
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/api-keys/{id}/revoke [post]
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid API key ID"))
		return
	}

	if err := h.service.Revoke(userID, keyID); err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}

// GetAvailableScopes returns all available API key scopes
// @Summary Get available scopes
// @Description Get all available API key scopes
// @Tags API Keys
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/api-keys/scopes [get]
func (h *APIKeyHandler) GetAvailableScopes(c *gin.Context) {
	scopes := h.service.GetAvailableScopes()
	c.JSON(http.StatusOK, scopes)
}

// ============================================================================
// Webhook Handlers
// ============================================================================

// WebhookHandler handles webhook related requests
type WebhookHandler struct {
	service *services.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(service *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

// CreateWebhook creates a new webhook
// @Summary Create webhook
// @Description Create a new webhook subscription
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param request body services.CreateWebhookRequest true "Webhook creation request"
// @Success 201 {object} models.Webhook
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/webhooks [post]
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req services.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	webhook, err := h.service.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

// ListWebhooks lists all webhooks for the current user
// @Summary List webhooks
// @Description List all webhooks for the authenticated user
// @Tags Webhooks
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.APIError
// @Router /api/v1/webhooks [get]
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.service.List(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetWebhook gets a webhook by ID
// @Summary Get webhook
// @Description Get a webhook by ID
// @Tags Webhooks
// @Produce json
// @Param id path string true "Webhook ID"
// @Success 200 {object} models.Webhook
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id} [get]
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	webhook, err := h.service.Get(userID, webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// UpdateWebhook updates a webhook
// @Summary Update webhook
// @Description Update an existing webhook
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Param request body services.CreateWebhookRequest true "Update request"
// @Success 200 {object} models.Webhook
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id} [put]
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	var req services.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	webhook, err := h.service.Update(userID, webhookID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, webhook)
}

// DeleteWebhook deletes a webhook
// @Summary Delete webhook
// @Description Delete a webhook
// @Tags Webhooks
// @Param id path string true "Webhook ID"
// @Success 204 "No Content"
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id} [delete]
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	if err := h.service.Delete(userID, webhookID); err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

// SetWebhookActive activates or deactivates a webhook
// @Summary Set webhook active status
// @Description Activate or deactivate a webhook
// @Tags Webhooks
// @Accept json
// @Param id path string true "Webhook ID"
// @Param request body object{active=bool} true "Active status"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id}/active [put]
func (h *WebhookHandler) SetWebhookActive(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	if err := h.service.SetActive(userID, webhookID, req.Active); err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	status := "deactivated"
	if req.Active {
		status = "activated"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook " + status + " successfully"})
}

// RegenerateWebhookSecret regenerates the webhook secret
// @Summary Regenerate webhook secret
// @Description Regenerate the HMAC secret for a webhook
// @Tags Webhooks
// @Param id path string true "Webhook ID"
// @Success 200 {object} object{secret=string}
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id}/secret [post]
func (h *WebhookHandler) RegenerateWebhookSecret(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	secret, err := h.service.RegenerateSecret(userID, webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

// GetWebhookDeliveries gets webhook delivery history
// @Summary Get webhook deliveries
// @Description Get delivery history for a webhook
// @Tags Webhooks
// @Produce json
// @Param id path string true "Webhook ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id}/deliveries [get]
func (h *WebhookHandler) GetWebhookDeliveries(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.service.GetDeliveries(userID, webhookID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RetryWebhookDelivery retries a failed webhook delivery
// @Summary Retry webhook delivery
// @Description Retry a failed webhook delivery
// @Tags Webhooks
// @Param id path string true "Webhook ID"
// @Param delivery_id path string true "Delivery ID"
// @Success 200 {object} object{message=string}
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id}/deliveries/{delivery_id}/retry [post]
func (h *WebhookHandler) RetryWebhookDelivery(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	deliveryID, err := uuid.Parse(c.Param("delivery_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid delivery ID"))
		return
	}

	if err := h.service.RetryDelivery(userID, webhookID, deliveryID); err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delivery retry initiated"})
}

// GetAvailableEvents returns all available webhook events
// @Summary Get available events
// @Description Get all available webhook events
// @Tags Webhooks
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/webhooks/events [get]
func (h *WebhookHandler) GetAvailableEvents(c *gin.Context) {
	events := h.service.GetAvailableEvents()
	c.JSON(http.StatusOK, events)
}

// TestWebhook sends a test event to a webhook
// @Summary Test webhook
// @Description Send a test event to verify webhook configuration
// @Tags Webhooks
// @Param id path string true "Webhook ID"
// @Success 200 {object} object{message=string,delivery_id=string}
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/webhooks/{id}/test [post]
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid webhook ID"))
		return
	}

	// Verify webhook exists and belongs to user
	_, err = h.service.Get(userID, webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	// Send test event
	testData := map[string]interface{}{
		"test":    true,
		"message": "This is a test webhook delivery",
	}

	if err := h.service.TriggerEvent(c.Request.Context(), models.WebhookEvent("test.ping"), testData); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test webhook sent successfully"})
}