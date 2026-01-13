package devbox

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// WebhooksService handles webhook operations
type WebhooksService struct {
	client *Client
}

// Webhook represents a webhook subscription
type Webhook struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	URL         string             `json:"url"`
	Events      []string           `json:"events"`
	IsActive    bool               `json:"is_active"`
	Headers     map[string]string  `json:"headers,omitempty"`
	RetryConfig *WebhookRetryConfig `json:"retry_config,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// WebhookRetryConfig defines retry configuration
type WebhookRetryConfig struct {
	MaxRetries    int     `json:"max_retries"`
	RetryDelayMs  int64   `json:"retry_delay_ms"`
	BackoffFactor float64 `json:"backoff_factor"`
	MaxDelayMs    int64   `json:"max_delay_ms"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID           string     `json:"id"`
	WebhookID    string     `json:"webhook_id"`
	EventType    string     `json:"event_type"`
	EventID      string     `json:"event_id"`
	Payload      string     `json:"payload"`
	ResponseCode int        `json:"response_code,omitempty"`
	ResponseBody string     `json:"response_body,omitempty"`
	Status       string     `json:"status"`
	AttemptCount int        `json:"attempt_count"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	DurationMs   int64      `json:"duration_ms,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateWebhookRequest contains parameters for creating a webhook
type CreateWebhookRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	URL         string             `json:"url"`
	Events      []string           `json:"events"`
	Headers     map[string]string  `json:"headers,omitempty"`
	RetryConfig *WebhookRetryConfig `json:"retry_config,omitempty"`
}

// WebhookListResponse contains a paginated list of webhooks
type WebhookListResponse struct {
	Data       []Webhook  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// WebhookDeliveryListResponse contains a paginated list of webhook deliveries
type WebhookDeliveryListResponse struct {
	Data       []WebhookDelivery `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// List returns a list of webhooks
func (s *WebhooksService) List(ctx context.Context, opts *ListOptions) (*WebhookListResponse, error) {
	path := "/webhooks" + opts.toQuery()
	var resp WebhookListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns a webhook by ID
func (s *WebhooksService) Get(ctx context.Context, id string) (*Webhook, error) {
	var webhook Webhook
	if err := s.client.request(ctx, "GET", "/webhooks/"+id, nil, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// Create creates a new webhook
func (s *WebhooksService) Create(ctx context.Context, req *CreateWebhookRequest) (*Webhook, error) {
	var webhook Webhook
	if err := s.client.request(ctx, "POST", "/webhooks", req, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// Update updates a webhook
func (s *WebhooksService) Update(ctx context.Context, id string, req *CreateWebhookRequest) (*Webhook, error) {
	var webhook Webhook
	if err := s.client.request(ctx, "PUT", "/webhooks/"+id, req, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// Delete deletes a webhook
func (s *WebhooksService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", "/webhooks/"+id, nil, nil)
}

// SetActive activates or deactivates a webhook
func (s *WebhooksService) SetActive(ctx context.Context, id string, active bool) error {
	req := map[string]bool{"active": active}
	return s.client.request(ctx, "PUT", "/webhooks/"+id+"/active", req, nil)
}

// RegenerateSecret regenerates the webhook secret
func (s *WebhooksService) RegenerateSecret(ctx context.Context, id string) (string, error) {
	var resp struct {
		Secret string `json:"secret"`
	}
	if err := s.client.request(ctx, "POST", "/webhooks/"+id+"/secret", nil, &resp); err != nil {
		return "", err
	}
	return resp.Secret, nil
}

// Test sends a test event to a webhook
func (s *WebhooksService) Test(ctx context.Context, id string) error {
	return s.client.request(ctx, "POST", "/webhooks/"+id+"/test", nil, nil)
}

// ListDeliveries returns webhook delivery history
func (s *WebhooksService) ListDeliveries(ctx context.Context, webhookID string, opts *ListOptions) (*WebhookDeliveryListResponse, error) {
	path := fmt.Sprintf("/webhooks/%s/deliveries%s", webhookID, opts.toQuery())
	var resp WebhookDeliveryListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RetryDelivery retries a failed webhook delivery
func (s *WebhooksService) RetryDelivery(ctx context.Context, webhookID, deliveryID string) error {
	path := fmt.Sprintf("/webhooks/%s/deliveries/%s/retry", webhookID, deliveryID)
	return s.client.request(ctx, "POST", path, nil, nil)
}

// GetAvailableEvents returns all available webhook events
func (s *WebhooksService) GetAvailableEvents(ctx context.Context) ([]string, error) {
	var events []string
	if err := s.client.request(ctx, "GET", "/webhooks/events", nil, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// APIKeysService handles API key operations
type APIKeysService struct {
	client *Client
}

// APIKey represents an API key
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyPrefix   string     `json:"key_prefix"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	LastUsedIP  string     `json:"last_used_ip,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateAPIKeyRequest contains parameters for creating an API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse contains the created API key with the plain key
type CreateAPIKeyResponse struct {
	APIKey   *APIKey `json:"api_key"`
	PlainKey string  `json:"plain_key"`
}

// UpdateAPIKeyRequest contains parameters for updating an API key
type UpdateAPIKeyRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
}

// APIKeyListResponse contains a paginated list of API keys
type APIKeyListResponse struct {
	Data       []APIKey   `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// List returns a list of API keys
func (s *APIKeysService) List(ctx context.Context, opts *ListOptions) (*APIKeyListResponse, error) {
	path := "/api-keys" + opts.toQuery()
	var resp APIKeyListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns an API key by ID
func (s *APIKeysService) Get(ctx context.Context, id string) (*APIKey, error) {
	var key APIKey
	if err := s.client.request(ctx, "GET", "/api-keys/"+id, nil, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Create creates a new API key
func (s *APIKeysService) Create(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	var resp CreateAPIKeyResponse
	if err := s.client.request(ctx, "POST", "/api-keys", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates an API key
func (s *APIKeysService) Update(ctx context.Context, id string, req *UpdateAPIKeyRequest) (*APIKey, error) {
	var key APIKey
	if err := s.client.request(ctx, "PUT", "/api-keys/"+id, req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// Delete deletes an API key
func (s *APIKeysService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", "/api-keys/"+id, nil, nil)
}

// Revoke revokes an API key
func (s *APIKeysService) Revoke(ctx context.Context, id string) error {
	return s.client.request(ctx, "POST", "/api-keys/"+id+"/revoke", nil, nil)
}

// GetAvailableScopes returns all available API key scopes
func (s *APIKeysService) GetAvailableScopes(ctx context.Context) ([]string, error) {
	var scopes []string
	if err := s.client.request(ctx, "GET", "/api-keys/scopes", nil, &scopes); err != nil {
		return nil, err
	}
	return scopes, nil
}
