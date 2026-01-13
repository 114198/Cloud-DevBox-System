// Package models provides data models for the core service.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// API Key Models
// ============================================================================

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID      uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	Name        string     `json:"name" gorm:"type:varchar(255);not null"`
	Description string     `json:"description,omitempty" gorm:"type:text"`
	KeyHash     string     `json:"-" gorm:"type:varchar(255);not null"` // Hashed API key
	KeyPrefix   string     `json:"key_prefix" gorm:"type:varchar(10);not null"` // First 8 chars for identification
	Scopes      []string   `json:"scopes" gorm:"type:text[];default:'{}'"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" gorm:"type:timestamp with time zone"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" gorm:"type:timestamp with time zone"`
	LastUsedIP  string     `json:"last_used_ip,omitempty" gorm:"type:varchar(45)"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for APIKey
func (APIKey) TableName() string {
	return "api_keys"
}

// APIKeyScope defines available API key scopes
type APIKeyScope string

const (
	ScopeEnvironmentsRead   APIKeyScope = "environments:read"
	ScopeEnvironmentsWrite  APIKeyScope = "environments:write"
	ScopeTemplatesRead      APIKeyScope = "templates:read"
	ScopeTemplatesWrite     APIKeyScope = "templates:write"
	ScopeProjectsRead       APIKeyScope = "projects:read"
	ScopeProjectsWrite      APIKeyScope = "projects:write"
	ScopeDeploymentsRead    APIKeyScope = "deployments:read"
	ScopeDeploymentsWrite   APIKeyScope = "deployments:write"
	ScopeCollaborationRead  APIKeyScope = "collaboration:read"
	ScopeCollaborationWrite APIKeyScope = "collaboration:write"
	ScopeWebhooksRead       APIKeyScope = "webhooks:read"
	ScopeWebhooksWrite      APIKeyScope = "webhooks:write"
	ScopeAdminRead          APIKeyScope = "admin:read"
	ScopeAdminWrite         APIKeyScope = "admin:write"
)

// AllScopes returns all available API key scopes
func AllScopes() []APIKeyScope {
	return []APIKeyScope{
		ScopeEnvironmentsRead, ScopeEnvironmentsWrite,
		ScopeTemplatesRead, ScopeTemplatesWrite,
		ScopeProjectsRead, ScopeProjectsWrite,
		ScopeDeploymentsRead, ScopeDeploymentsWrite,
		ScopeCollaborationRead, ScopeCollaborationWrite,
		ScopeWebhooksRead, ScopeWebhooksWrite,
		ScopeAdminRead, ScopeAdminWrite,
	}
}

// ============================================================================
// Webhook Models
// ============================================================================

// Webhook represents a webhook subscription
type Webhook struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID      uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	Description string          `json:"description,omitempty" gorm:"type:text"`
	URL         string          `json:"url" gorm:"type:text;not null"`
	Secret      string          `json:"-" gorm:"type:varchar(255)"` // For HMAC signature
	Events      []string        `json:"events" gorm:"type:text[];not null"`
	IsActive    bool            `json:"is_active" gorm:"default:true"`
	Headers     map[string]string `json:"headers,omitempty" gorm:"type:jsonb;default:'{}'"`
	RetryConfig *WebhookRetryConfig `json:"retry_config,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Webhook
func (Webhook) TableName() string {
	return "webhooks"
}

// WebhookRetryConfig defines retry configuration for webhooks
type WebhookRetryConfig struct {
	MaxRetries     int   `json:"max_retries"`
	RetryDelayMs   int64 `json:"retry_delay_ms"`
	BackoffFactor  float64 `json:"backoff_factor"`
	MaxDelayMs     int64 `json:"max_delay_ms"`
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *WebhookRetryConfig {
	return &WebhookRetryConfig{
		MaxRetries:    3,
		RetryDelayMs:  1000,
		BackoffFactor: 2.0,
		MaxDelayMs:    60000,
	}
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	WebhookID     uuid.UUID  `json:"webhook_id" gorm:"type:uuid;not null"`
	EventType     string     `json:"event_type" gorm:"type:varchar(100);not null"`
	EventID       string     `json:"event_id" gorm:"type:varchar(100);not null"`
	Payload       string     `json:"payload" gorm:"type:text;not null"`
	ResponseCode  int        `json:"response_code,omitempty"`
	ResponseBody  string     `json:"response_body,omitempty" gorm:"type:text"`
	Status        string     `json:"status" gorm:"type:varchar(50);not null"` // pending, success, failed
	AttemptCount  int        `json:"attempt_count" gorm:"default:0"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty" gorm:"type:timestamp with time zone"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty" gorm:"type:timestamp with time zone"`
	ErrorMessage  string     `json:"error_message,omitempty" gorm:"type:text"`
	DurationMs    int64      `json:"duration_ms,omitempty"`
	CreatedAt     time.Time  `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for WebhookDelivery
func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}

// WebhookEvent defines available webhook events
type WebhookEvent string

const (
	// Environment events
	EventEnvironmentCreated   WebhookEvent = "environment.created"
	EventEnvironmentStarted   WebhookEvent = "environment.started"
	EventEnvironmentStopped   WebhookEvent = "environment.stopped"
	EventEnvironmentDeleted   WebhookEvent = "environment.deleted"
	EventEnvironmentFailed    WebhookEvent = "environment.failed"
	EventEnvironmentSuspended WebhookEvent = "environment.suspended"

	// Deployment events
	EventDeploymentStarted   WebhookEvent = "deployment.started"
	EventDeploymentSucceeded WebhookEvent = "deployment.succeeded"
	EventDeploymentFailed    WebhookEvent = "deployment.failed"
	EventDeploymentRolledBack WebhookEvent = "deployment.rolled_back"

	// Collaboration events
	EventCollaboratorJoined WebhookEvent = "collaboration.joined"
	EventCollaboratorLeft   WebhookEvent = "collaboration.left"

	// Git events
	EventGitPushReceived WebhookEvent = "git.push_received"
	EventGitSyncCompleted WebhookEvent = "git.sync_completed"
	EventGitSyncFailed    WebhookEvent = "git.sync_failed"

	// Alert events
	EventAlertTriggered  WebhookEvent = "alert.triggered"
	EventAlertResolved   WebhookEvent = "alert.resolved"
)

// AllWebhookEvents returns all available webhook events
func AllWebhookEvents() []WebhookEvent {
	return []WebhookEvent{
		EventEnvironmentCreated, EventEnvironmentStarted, EventEnvironmentStopped,
		EventEnvironmentDeleted, EventEnvironmentFailed, EventEnvironmentSuspended,
		EventDeploymentStarted, EventDeploymentSucceeded, EventDeploymentFailed, EventDeploymentRolledBack,
		EventCollaboratorJoined, EventCollaboratorLeft,
		EventGitPushReceived, EventGitSyncCompleted, EventGitSyncFailed,
		EventAlertTriggered, EventAlertResolved,
	}
}

// ============================================================================
// API Response Models
// ============================================================================

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPagination creates a new Pagination instance
func NewPagination(page, pageSize int, totalItems int64) Pagination {
	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// APIError represents an API error response
type APIError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	RequestID  string            `json:"request_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// NewAPIError creates a new APIError
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details map[string]string) *APIError {
	e.Details = details
	return e
}

// WithRequestID adds request ID to the error
func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// Common API error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeInvalidAPIKey       = "INVALID_API_KEY"
	ErrCodeExpiredAPIKey       = "EXPIRED_API_KEY"
	ErrCodeInsufficientScope   = "INSUFFICIENT_SCOPE"
)
