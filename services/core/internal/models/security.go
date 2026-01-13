// Package models provides data models for the core service.
package models

import (
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IPBlacklist represents a blocked IP address
type IPBlacklist struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	IPAddress string     `gorm:"type:inet;not null;uniqueIndex" json:"ip_address"`
	Reason    string     `gorm:"not null" json:"reason"`
	BlockedBy string     `gorm:"default:system" json:"blocked_by"` // 'system' or 'admin'
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName returns the table name for IPBlacklist
func (IPBlacklist) TableName() string {
	return "ip_blacklist"
}

// BeforeCreate sets default values before creating
func (b *IPBlacklist) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.BlockedBy == "" {
		b.BlockedBy = "system"
	}
	return nil
}

// IsExpired checks if the blacklist entry has expired
func (b *IPBlacklist) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false // Permanent block
	}
	return time.Now().After(*b.ExpiresAt)
}

// AccessAttempt represents a login or access attempt
type AccessAttempt struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	IPAddress     string     `gorm:"type:inet;not null" json:"ip_address"`
	UserID        *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	AttemptType   string     `gorm:"not null" json:"attempt_type"` // 'login', 'api', 'ssh', 'resource'
	Success       bool       `gorm:"default:false" json:"success"`
	FailureReason string     `json:"failure_reason,omitempty"`
	UserAgent     string     `json:"user_agent,omitempty"`
	Endpoint      string     `json:"endpoint,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName returns the table name for AccessAttempt
func (AccessAttempt) TableName() string {
	return "access_attempts"
}

// BeforeCreate sets default values before creating
func (a *AccessAttempt) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// SecurityEventSeverity represents the severity level of a security event
type SecurityEventSeverity string

const (
	SeverityLow      SecurityEventSeverity = "low"
	SeverityMedium   SecurityEventSeverity = "medium"
	SeverityHigh     SecurityEventSeverity = "high"
	SeverityCritical SecurityEventSeverity = "critical"
)

// SecurityEventType represents the type of security event
type SecurityEventType string

const (
	EventBruteForce         SecurityEventType = "brute_force"
	EventSuspiciousActivity SecurityEventType = "suspicious_activity"
	EventRateLimitExceeded  SecurityEventType = "rate_limit_exceeded"
	EventUnauthorizedAccess SecurityEventType = "unauthorized_access"
	EventIPBlocked          SecurityEventType = "ip_blocked"
	EventAnomalyDetected    SecurityEventType = "anomaly_detected"
	EventDataBreach         SecurityEventType = "data_breach"
)

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID          uuid.UUID             `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	EventType   SecurityEventType     `gorm:"not null" json:"event_type"`
	Severity    SecurityEventSeverity `gorm:"default:medium" json:"severity"`
	IPAddress   *string               `gorm:"type:inet" json:"ip_address,omitempty"`
	UserID      *uuid.UUID            `gorm:"type:uuid" json:"user_id,omitempty"`
	Description string                `gorm:"not null" json:"description"`
	Details     JSONMap               `gorm:"type:jsonb;default:'{}'" json:"details"`
	Resolved    bool                  `gorm:"default:false" json:"resolved"`
	ResolvedAt  *time.Time            `json:"resolved_at,omitempty"`
	ResolvedBy  *uuid.UUID            `gorm:"type:uuid" json:"resolved_by,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
}

// TableName returns the table name for SecurityEvent
func (SecurityEvent) TableName() string {
	return "security_events"
}

// BeforeCreate sets default values before creating
func (e *SecurityEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Severity == "" {
		e.Severity = SeverityMedium
	}
	return nil
}

// EncryptionKeyStatus represents the status of an encryption key
type EncryptionKeyStatus string

const (
	KeyStatusActive   EncryptionKeyStatus = "active"
	KeyStatusRotating EncryptionKeyStatus = "rotating"
	KeyStatusRetired  EncryptionKeyStatus = "retired"
)

// EncryptionKey represents metadata for an encryption key (actual key in Vault)
type EncryptionKey struct {
	ID        uuid.UUID           `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	KeyID     string              `gorm:"uniqueIndex;not null" json:"key_id"` // Reference to key in Vault
	KeyType   string              `gorm:"not null" json:"key_type"`           // 'data', 'config', 'backup'
	Algorithm string              `gorm:"default:AES-256-GCM" json:"algorithm"`
	Status    EncryptionKeyStatus `gorm:"default:active" json:"status"`
	Version   int                 `gorm:"default:1" json:"version"`
	CreatedAt time.Time           `json:"created_at"`
	RotatedAt *time.Time          `json:"rotated_at,omitempty"`
	ExpiresAt *time.Time          `json:"expires_at,omitempty"`
}

// TableName returns the table name for EncryptionKey
func (EncryptionKey) TableName() string {
	return "encryption_keys"
}

// BeforeCreate sets default values before creating
func (k *EncryptionKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	if k.Algorithm == "" {
		k.Algorithm = "AES-256-GCM"
	}
	if k.Status == "" {
		k.Status = KeyStatusActive
	}
	if k.Version == 0 {
		k.Version = 1
	}
	return nil
}

// EncryptedData represents encrypted data stored in the database
type EncryptedData struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ResourceType  string    `gorm:"not null" json:"resource_type"` // 'environment_secret', 'user_credential', etc.
	ResourceID    uuid.UUID `gorm:"type:uuid;not null" json:"resource_id"`
	KeyID         uuid.UUID `gorm:"type:uuid;not null" json:"key_id"`
	EncryptedData []byte    `gorm:"type:bytea;not null" json:"-"`
	Nonce         []byte    `gorm:"type:bytea;not null" json:"-"` // IV for AES-GCM
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName returns the table name for EncryptedData
func (EncryptedData) TableName() string {
	return "encrypted_data"
}

// BeforeCreate sets default values before creating
func (d *EncryptedData) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// AuditLog represents an audit log entry (extends existing audit_logs table)
type AuditLog struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID         *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	OrganizationID *uuid.UUID `gorm:"type:uuid" json:"organization_id,omitempty"`
	Action         string     `gorm:"not null" json:"action"`
	ResourceType   string     `gorm:"not null" json:"resource_type"`
	ResourceID     *uuid.UUID `gorm:"type:uuid" json:"resource_id,omitempty"`
	Details        JSONMap    `gorm:"type:jsonb;default:'{}'" json:"details"`
	IPAddress      *string    `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent      string     `json:"user_agent,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// TableName returns the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate sets default values before creating
func (l *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// IsValidIP checks if the given string is a valid IP address
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// AuditAction represents common audit actions
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionRead   AuditAction = "read"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"
	AuditActionGrant  AuditAction = "grant"
	AuditActionRevoke AuditAction = "revoke"
)

// AuditResourceType represents common resource types for auditing
type AuditResourceType string

const (
	ResourceUser        AuditResourceType = "user"
	ResourceEnvironment AuditResourceType = "environment"
	ResourceTemplate    AuditResourceType = "template"
	ResourceProject     AuditResourceType = "project"
	ResourceAPIKey      AuditResourceType = "api_key"
	ResourcePermission  AuditResourceType = "permission"
	ResourceSecret      AuditResourceType = "secret"
)
