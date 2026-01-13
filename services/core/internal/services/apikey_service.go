// Package services provides business logic for the core service.
package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyService handles API key operations
type APIKeyService struct {
	db *gorm.DB
}

// NewAPIKeyService creates a new APIKeyService
func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string    `json:"name" binding:"required,min=1,max=255"`
	Description string    `json:"description,omitempty"`
	Scopes      []string  `json:"scopes" binding:"required,min=1"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey    *models.APIKey `json:"api_key"`
	PlainKey  string         `json:"plain_key"` // Only returned once at creation
}

// Create creates a new API key for a user
func (s *APIKeyService) Create(userID uuid.UUID, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Validate scopes
	if err := s.validateScopes(req.Scopes); err != nil {
		return nil, err
	}

	// Generate a random API key
	plainKey, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	keyHash := s.hashAPIKey(plainKey)
	keyPrefix := plainKey[:8]

	apiKey := &models.APIKey{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(apiKey).Error; err != nil {
		if strings.Contains(err.Error(), "unique_user_key_name") {
			return nil, errors.New("API key with this name already exists")
		}
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &CreateAPIKeyResponse{
		APIKey:   apiKey,
		PlainKey: plainKey,
	}, nil
}

// List returns all API keys for a user
func (s *APIKeyService) List(userID uuid.UUID, page, pageSize int) (*models.PaginatedResponse, error) {
	var keys []models.APIKey
	var total int64

	query := s.db.Model(&models.APIKey{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count API keys: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return &models.PaginatedResponse{
		Data:       keys,
		Pagination: models.NewPagination(page, pageSize, total),
	}, nil
}

// Get returns an API key by ID
func (s *APIKeyService) Get(userID, keyID uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	if err := s.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return &key, nil
}

// Update updates an API key
func (s *APIKeyService) Update(userID, keyID uuid.UUID, name, description string, scopes []string) (*models.APIKey, error) {
	key, err := s.Get(userID, keyID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		key.Name = name
	}
	if description != "" {
		key.Description = description
	}
	if len(scopes) > 0 {
		if err := s.validateScopes(scopes); err != nil {
			return nil, err
		}
		key.Scopes = scopes
	}
	key.UpdatedAt = time.Now()

	if err := s.db.Save(key).Error; err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return key, nil
}

// Delete deletes an API key
func (s *APIKeyService) Delete(userID, keyID uuid.UUID) error {
	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.APIKey{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	return nil
}

// Revoke deactivates an API key
func (s *APIKeyService) Revoke(userID, keyID uuid.UUID) error {
	result := s.db.Model(&models.APIKey{}).
		Where("id = ? AND user_id = ?", keyID, userID).
		Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	return nil
}

// ValidateKey validates an API key and returns the associated user ID and scopes
func (s *APIKeyService) ValidateKey(plainKey string) (*models.APIKey, error) {
	if len(plainKey) < 8 {
		return nil, errors.New("invalid API key format")
	}

	keyPrefix := plainKey[:8]
	keyHash := s.hashAPIKey(plainKey)

	var key models.APIKey
	if err := s.db.Where("key_prefix = ? AND key_hash = ? AND is_active = ?", keyPrefix, keyHash, true).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid API key")
		}
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check expiration
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("API key has expired")
	}

	// Update last used timestamp
	s.db.Model(&key).Updates(map[string]interface{}{
		"last_used_at": time.Now(),
	})

	return &key, nil
}

// ValidateKeyWithIP validates an API key and records the IP address
func (s *APIKeyService) ValidateKeyWithIP(plainKey, ipAddress string) (*models.APIKey, error) {
	key, err := s.ValidateKey(plainKey)
	if err != nil {
		return nil, err
	}

	// Update last used IP
	s.db.Model(key).Updates(map[string]interface{}{
		"last_used_ip": ipAddress,
	})

	return key, nil
}

// HasScope checks if an API key has a specific scope
func (s *APIKeyService) HasScope(key *models.APIKey, requiredScope string) bool {
	for _, scope := range key.Scopes {
		if scope == requiredScope {
			return true
		}
		// Check for wildcard scopes (e.g., "environments:*" matches "environments:read")
		if strings.HasSuffix(scope, ":*") {
			prefix := strings.TrimSuffix(scope, "*")
			if strings.HasPrefix(requiredScope, prefix) {
				return true
			}
		}
	}
	return false
}

// generateAPIKey generates a random API key
func (s *APIKeyService) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "devbox_" + hex.EncodeToString(bytes), nil
}

// hashAPIKey hashes an API key using SHA-256
func (s *APIKeyService) hashAPIKey(plainKey string) string {
	hash := sha256.Sum256([]byte(plainKey))
	return hex.EncodeToString(hash[:])
}

// validateScopes validates that all scopes are valid
func (s *APIKeyService) validateScopes(scopes []string) error {
	validScopes := make(map[string]bool)
	for _, scope := range models.AllScopes() {
		validScopes[string(scope)] = true
	}

	for _, scope := range scopes {
		// Allow wildcard scopes
		if strings.HasSuffix(scope, ":*") {
			continue
		}
		if !validScopes[scope] {
			return fmt.Errorf("invalid scope: %s", scope)
		}
	}
	return nil
}

// GetAvailableScopes returns all available API key scopes
func (s *APIKeyService) GetAvailableScopes() []models.APIKeyScope {
	return models.AllScopes()
}
