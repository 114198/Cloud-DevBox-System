// Package services provides business logic for the core service.
package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookService handles webhook operations
type WebhookService struct {
	db         *gorm.DB
	httpClient *http.Client
	mu         sync.RWMutex
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewWebhookService creates a new WebhookService
func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	Name        string                    `json:"name" binding:"required,min=1,max=255"`
	Description string                    `json:"description,omitempty"`
	URL         string                    `json:"url" binding:"required,url"`
	Events      []string                  `json:"events" binding:"required,min=1"`
	Headers     map[string]string         `json:"headers,omitempty"`
	RetryConfig *models.WebhookRetryConfig `json:"retry_config,omitempty"`
}

// Create creates a new webhook for a user
func (s *WebhookService) Create(userID uuid.UUID, req *CreateWebhookRequest) (*models.Webhook, error) {
	// Validate events
	if err := s.validateEvents(req.Events); err != nil {
		return nil, err
	}

	// Generate a secret for HMAC signature
	secret, err := s.generateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate webhook secret: %w", err)
	}

	retryConfig := req.RetryConfig
	if retryConfig == nil {
		retryConfig = models.DefaultRetryConfig()
	}

	webhook := &models.Webhook{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		Secret:      secret,
		Events:      req.Events,
		IsActive:    true,
		Headers:     req.Headers,
		RetryConfig: retryConfig,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(webhook).Error; err != nil {
		if strings.Contains(err.Error(), "unique_user_webhook_name") {
			return nil, errors.New("webhook with this name already exists")
		}
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, nil
}

// List returns all webhooks for a user
func (s *WebhookService) List(userID uuid.UUID, page, pageSize int) (*models.PaginatedResponse, error) {
	var webhooks []models.Webhook
	var total int64

	query := s.db.Model(&models.Webhook{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count webhooks: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&webhooks).Error; err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	return &models.PaginatedResponse{
		Data:       webhooks,
		Pagination: models.NewPagination(page, pageSize, total),
	}, nil
}

// Get returns a webhook by ID
func (s *WebhookService) Get(userID, webhookID uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	if err := s.db.Where("id = ? AND user_id = ?", webhookID, userID).First(&webhook).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("webhook not found")
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return &webhook, nil
}

// Update updates a webhook
func (s *WebhookService) Update(userID, webhookID uuid.UUID, req *CreateWebhookRequest) (*models.Webhook, error) {
	webhook, err := s.Get(userID, webhookID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		webhook.Name = req.Name
	}
	if req.Description != "" {
		webhook.Description = req.Description
	}
	if req.URL != "" {
		webhook.URL = req.URL
	}
	if len(req.Events) > 0 {
		if err := s.validateEvents(req.Events); err != nil {
			return nil, err
		}
		webhook.Events = req.Events
	}
	if req.Headers != nil {
		webhook.Headers = req.Headers
	}
	if req.RetryConfig != nil {
		webhook.RetryConfig = req.RetryConfig
	}
	webhook.UpdatedAt = time.Now()

	if err := s.db.Save(webhook).Error; err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return webhook, nil
}

// Delete deletes a webhook
func (s *WebhookService) Delete(userID, webhookID uuid.UUID) error {
	result := s.db.Where("id = ? AND user_id = ?", webhookID, userID).Delete(&models.Webhook{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete webhook: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("webhook not found")
	}
	return nil
}

// SetActive activates or deactivates a webhook
func (s *WebhookService) SetActive(userID, webhookID uuid.UUID, active bool) error {
	result := s.db.Model(&models.Webhook{}).
		Where("id = ? AND user_id = ?", webhookID, userID).
		Update("is_active", active)
	if result.Error != nil {
		return fmt.Errorf("failed to update webhook status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("webhook not found")
	}
	return nil
}

// RegenerateSecret regenerates the webhook secret
func (s *WebhookService) RegenerateSecret(userID, webhookID uuid.UUID) (string, error) {
	webhook, err := s.Get(userID, webhookID)
	if err != nil {
		return "", err
	}

	secret, err := s.generateSecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate new secret: %w", err)
	}

	webhook.Secret = secret
	webhook.UpdatedAt = time.Now()

	if err := s.db.Save(webhook).Error; err != nil {
		return "", fmt.Errorf("failed to save new secret: %w", err)
	}

	return secret, nil
}

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	ID        string      `json:"id"`
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// TriggerEvent triggers webhooks for a specific event
func (s *WebhookService) TriggerEvent(ctx context.Context, event models.WebhookEvent, data interface{}) error {
	// Find all active webhooks subscribed to this event
	var webhooks []models.Webhook
	if err := s.db.Where("is_active = ? AND ? = ANY(events)", true, string(event)).Find(&webhooks).Error; err != nil {
		return fmt.Errorf("failed to find webhooks: %w", err)
	}

	if len(webhooks) == 0 {
		return nil
	}

	// Create payload
	payload := WebhookPayload{
		ID:        uuid.New().String(),
		Event:     string(event),
		Timestamp: time.Now(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create delivery records and send webhooks
	for _, webhook := range webhooks {
		delivery := &models.WebhookDelivery{
			ID:           uuid.New(),
			WebhookID:    webhook.ID,
			EventType:    string(event),
			EventID:      payload.ID,
			Payload:      string(payloadBytes),
			Status:       "pending",
			AttemptCount: 0,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.db.Create(delivery).Error; err != nil {
			continue // Log error but continue with other webhooks
		}

		// Send webhook asynchronously
		go s.deliverWebhook(ctx, &webhook, delivery, payloadBytes)
	}

	return nil
}

// deliverWebhook sends a webhook and handles retries
func (s *WebhookService) deliverWebhook(ctx context.Context, webhook *models.Webhook, delivery *models.WebhookDelivery, payload []byte) {
	retryConfig := webhook.RetryConfig
	if retryConfig == nil {
		retryConfig = models.DefaultRetryConfig()
	}

	var lastErr error
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(retryConfig.RetryDelayMs) * 
				pow(retryConfig.BackoffFactor, float64(attempt-1))) * time.Millisecond
			if delay > time.Duration(retryConfig.MaxDelayMs)*time.Millisecond {
				delay = time.Duration(retryConfig.MaxDelayMs) * time.Millisecond
			}
			time.Sleep(delay)
		}

		delivery.AttemptCount = attempt + 1
		delivery.Status = "retrying"
		s.db.Save(delivery)

		startTime := time.Now()
		statusCode, responseBody, err := s.sendWebhookRequest(webhook, payload)
		duration := time.Since(startTime).Milliseconds()

		delivery.DurationMs = duration
		delivery.ResponseCode = statusCode
		delivery.ResponseBody = responseBody

		if err != nil {
			lastErr = err
			delivery.ErrorMessage = err.Error()
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			// Success
			now := time.Now()
			delivery.Status = "success"
			delivery.DeliveredAt = &now
			delivery.ErrorMessage = ""
			s.db.Save(delivery)
			return
		}

		lastErr = fmt.Errorf("webhook returned status %d", statusCode)
		delivery.ErrorMessage = lastErr.Error()
	}

	// All retries failed
	delivery.Status = "failed"
	delivery.ErrorMessage = lastErr.Error()
	s.db.Save(delivery)
}

// sendWebhookRequest sends a single webhook request
func (s *WebhookService) sendWebhookRequest(webhook *models.Webhook, payload []byte) (int, string, error) {
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CloudDevBox-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", webhook.ID.String())

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature
	if webhook.Secret != "" {
		signature := s.computeSignature(payload, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", "sha256="+signature)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*10)) // Limit to 10KB
	return resp.StatusCode, string(body), nil
}

// computeSignature computes HMAC-SHA256 signature
func (s *WebhookService) computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// GetDeliveries returns webhook deliveries
func (s *WebhookService) GetDeliveries(userID, webhookID uuid.UUID, page, pageSize int) (*models.PaginatedResponse, error) {
	// Verify webhook belongs to user
	if _, err := s.Get(userID, webhookID); err != nil {
		return nil, err
	}

	var deliveries []models.WebhookDelivery
	var total int64

	query := s.db.Model(&models.WebhookDelivery{}).Where("webhook_id = ?", webhookID)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count deliveries: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}

	return &models.PaginatedResponse{
		Data:       deliveries,
		Pagination: models.NewPagination(page, pageSize, total),
	}, nil
}

// RetryDelivery retries a failed webhook delivery
func (s *WebhookService) RetryDelivery(userID, webhookID, deliveryID uuid.UUID) error {
	webhook, err := s.Get(userID, webhookID)
	if err != nil {
		return err
	}

	var delivery models.WebhookDelivery
	if err := s.db.Where("id = ? AND webhook_id = ?", deliveryID, webhookID).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("delivery not found")
		}
		return fmt.Errorf("failed to get delivery: %w", err)
	}

	// Reset delivery status
	delivery.Status = "pending"
	delivery.AttemptCount = 0
	delivery.ErrorMessage = ""
	s.db.Save(&delivery)

	// Retry delivery
	go s.deliverWebhook(context.Background(), webhook, &delivery, []byte(delivery.Payload))

	return nil
}

// GetAvailableEvents returns all available webhook events
func (s *WebhookService) GetAvailableEvents() []models.WebhookEvent {
	return models.AllWebhookEvents()
}

// generateSecret generates a random webhook secret
func (s *WebhookService) generateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(bytes), nil
}

// validateEvents validates that all events are valid
func (s *WebhookService) validateEvents(events []string) error {
	validEvents := make(map[string]bool)
	for _, event := range models.AllWebhookEvents() {
		validEvents[string(event)] = true
	}

	for _, event := range events {
		// Allow wildcard events
		if event == "*" {
			continue
		}
		if !validEvents[event] {
			return fmt.Errorf("invalid event: %s", event)
		}
	}
	return nil
}

// StartRetryWorker starts the background worker for retrying failed webhooks
func (s *WebhookService) StartRetryWorker(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.processRetries(ctx)
			}
		}
	}()
}

// processRetries processes pending retries
func (s *WebhookService) processRetries(ctx context.Context) {
	var deliveries []models.WebhookDelivery
	now := time.Now()

	if err := s.db.Where("status = ? AND next_retry_at <= ?", "retrying", now).
		Limit(100).Find(&deliveries).Error; err != nil {
		return
	}

	for _, delivery := range deliveries {
		var webhook models.Webhook
		if err := s.db.First(&webhook, delivery.WebhookID).Error; err != nil {
			continue
		}

		go s.deliverWebhook(ctx, &webhook, &delivery, []byte(delivery.Payload))
	}
}

// Stop stops the webhook service
func (s *WebhookService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// pow calculates x^y for float64
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}
