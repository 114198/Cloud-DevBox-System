// Package models provides data models for the core service.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Email          string          `gorm:"uniqueIndex;not null" json:"email"`
	Username       string          `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash   string          `gorm:"column:password_hash" json:"-"`
	DisplayName    string          `json:"display_name"`
	Avatar         string          `json:"avatar,omitempty"`
	Role           string          `gorm:"default:developer" json:"role"`
	OrganizationID *uuid.UUID      `gorm:"type:uuid" json:"organization_id,omitempty"`
	Preferences    JSONMap         `gorm:"type:jsonb;default:'{}'" json:"preferences"`
	Quotas         JSONMap         `gorm:"type:jsonb;not null" json:"quotas"`
	EmailVerified  bool            `gorm:"default:false" json:"email_verified"`
	MFAEnabled     bool            `gorm:"default:false" json:"mfa_enabled"`
	MFASecret      string          `json:"-"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastLoginAt    *time.Time      `json:"last_login_at,omitempty"`
	OAuthAccounts  []OAuthAccount  `gorm:"foreignKey:UserID" json:"oauth_accounts,omitempty"`
	RecoveryCodes  []RecoveryCode  `gorm:"foreignKey:UserID" json:"-"`
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate sets default values before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Quotas == nil {
		u.Quotas = JSONMap{
			"maxEnvironments": 10,
			"maxCPU":          8,
			"maxMemory":       16,
			"maxStorage":      100,
			"maxRunningTime":  720,
		}
	}
	if u.Role == "" {
		u.Role = "developer"
	}
	return nil
}

// OAuthAccount represents an OAuth provider account linked to a user
type OAuthAccount struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Provider          string     `gorm:"not null" json:"provider"`
	ProviderAccountID string     `gorm:"column:provider_account_id;not null" json:"provider_account_id"`
	AccessToken       string     `json:"-"`
	RefreshToken      string     `json:"-"`
	TokenExpiresAt    *time.Time `json:"token_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName returns the table name for OAuthAccount
func (OAuthAccount) TableName() string {
	return "oauth_accounts"
}

// RecoveryCode represents a backup recovery code for MFA
type RecoveryCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Code      string    `gorm:"not null" json:"-"`
	Used      bool      `gorm:"default:false" json:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name for RecoveryCode
func (RecoveryCode) TableName() string {
	return "recovery_codes"
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Token     string     `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	Revoked   bool       `gorm:"default:false" json:"revoked"`
	CreatedAt time.Time  `json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}

// TableName returns the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
