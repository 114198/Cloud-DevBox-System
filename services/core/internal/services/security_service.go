// Package services provides business logic services for the core service.
package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrIPBlocked           = errors.New("IP address is blocked")
	ErrIPNotFound          = errors.New("IP address not found in blacklist")
	ErrEncryptionKeyNotFound = errors.New("encryption key not found")
	ErrDecryptionFailed    = errors.New("decryption failed")
	ErrInvalidKeySize      = errors.New("invalid encryption key size")
)

// SecurityConfig holds configuration for the security service
type SecurityConfig struct {
	// Anomaly detection thresholds
	MaxFailedLoginAttempts    int           // Max failed logins before blocking
	FailedLoginWindow         time.Duration // Time window for counting failed logins
	MaxAPIRequestsPerMinute   int           // Max API requests per minute per IP
	BlockDuration             time.Duration // How long to block an IP
	
	// Audit log retention
	AuditLogRetentionDays     int           // Days to retain audit logs (365 for 1 year)
	
	// Encryption settings
	MasterKeyID               string        // Vault key ID for master encryption key
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MaxFailedLoginAttempts:  5,
		FailedLoginWindow:       15 * time.Minute,
		MaxAPIRequestsPerMinute: 100,
		BlockDuration:           24 * time.Hour,
		AuditLogRetentionDays:   365,
		MasterKeyID:             "devbox-master-key",
	}
}

// SecurityService handles security operations including anomaly detection,
// audit logging, and data encryption
type SecurityService struct {
	db          *gorm.DB
	redis       *redis.Client
	config      *SecurityConfig
	encryptionKey []byte // In production, this would come from Vault
	mu          sync.RWMutex
}

// NewSecurityService creates a new SecurityService instance
func NewSecurityService(db *gorm.DB, redisClient *redis.Client, config *SecurityConfig) *SecurityService {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	
	return &SecurityService{
		db:     db,
		redis:  redisClient,
		config: config,
	}
}

// SetEncryptionKey sets the encryption key (in production, this would be fetched from Vault)
func (s *SecurityService) SetEncryptionKey(key []byte) error {
	if len(key) != 32 { // AES-256 requires 32-byte key
		return ErrInvalidKeySize
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.encryptionKey = key
	return nil
}

// =============================================================================
// IP Blacklist Management
// =============================================================================

// IsIPBlocked checks if an IP address is blocked
func (s *SecurityService) IsIPBlocked(ctx context.Context, ipAddress string) (bool, error) {
	if !models.IsValidIP(ipAddress) {
		return false, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	// Check Redis cache first
	if s.redis != nil {
		cacheKey := fmt.Sprintf("ip_blocked:%s", ipAddress)
		blocked, err := s.redis.Get(ctx, cacheKey).Bool()
		if err == nil {
			return blocked, nil
		}
	}

	// Check database
	var blacklist models.IPBlacklist
	err := s.db.Where("ip_address = ?", ipAddress).First(&blacklist).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Cache negative result
			if s.redis != nil {
				cacheKey := fmt.Sprintf("ip_blocked:%s", ipAddress)
				s.redis.Set(ctx, cacheKey, false, 5*time.Minute)
			}
			return false, nil
		}
		return false, fmt.Errorf("failed to check IP blacklist: %w", err)
	}

	// Check if expired
	if blacklist.IsExpired() {
		// Remove expired entry
		s.db.Delete(&blacklist)
		return false, nil
	}

	// Cache positive result
	if s.redis != nil {
		cacheKey := fmt.Sprintf("ip_blocked:%s", ipAddress)
		ttl := time.Hour
		if blacklist.ExpiresAt != nil {
			ttl = time.Until(*blacklist.ExpiresAt)
		}
		s.redis.Set(ctx, cacheKey, true, ttl)
	}

	return true, nil
}

// BlockIP adds an IP address to the blacklist
func (s *SecurityService) BlockIP(ctx context.Context, ipAddress, reason, blockedBy string, duration *time.Duration) error {
	if !models.IsValidIP(ipAddress) {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	var expiresAt *time.Time
	if duration != nil {
		t := time.Now().Add(*duration)
		expiresAt = &t
	}

	blacklist := &models.IPBlacklist{
		IPAddress: ipAddress,
		Reason:    reason,
		BlockedBy: blockedBy,
		ExpiresAt: expiresAt,
	}

	// Upsert - update if exists, create if not
	err := s.db.Where("ip_address = ?", ipAddress).
		Assign(models.IPBlacklist{
			Reason:    reason,
			BlockedBy: blockedBy,
			ExpiresAt: expiresAt,
		}).
		FirstOrCreate(blacklist).Error
	if err != nil {
		return fmt.Errorf("failed to block IP: %w", err)
	}

	// Update cache
	if s.redis != nil {
		cacheKey := fmt.Sprintf("ip_blocked:%s", ipAddress)
		ttl := time.Hour * 24 * 365 // Default to 1 year for permanent blocks
		if duration != nil {
			ttl = *duration
		}
		s.redis.Set(ctx, cacheKey, true, ttl)
	}

	// Create security event
	s.CreateSecurityEvent(ctx, &models.SecurityEvent{
		EventType:   models.EventIPBlocked,
		Severity:    models.SeverityHigh,
		IPAddress:   &ipAddress,
		Description: fmt.Sprintf("IP address %s blocked: %s", ipAddress, reason),
		Details: models.JSONMap{
			"blocked_by": blockedBy,
			"duration":   duration,
		},
	})

	return nil
}

// UnblockIP removes an IP address from the blacklist
func (s *SecurityService) UnblockIP(ctx context.Context, ipAddress string) error {
	if !models.IsValidIP(ipAddress) {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	result := s.db.Where("ip_address = ?", ipAddress).Delete(&models.IPBlacklist{})
	if result.Error != nil {
		return fmt.Errorf("failed to unblock IP: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrIPNotFound
	}

	// Clear cache
	if s.redis != nil {
		cacheKey := fmt.Sprintf("ip_blocked:%s", ipAddress)
		s.redis.Del(ctx, cacheKey)
	}

	return nil
}

// GetBlacklistedIPs returns all blacklisted IPs with pagination
func (s *SecurityService) GetBlacklistedIPs(ctx context.Context, page, pageSize int) ([]models.IPBlacklist, int64, error) {
	var blacklist []models.IPBlacklist
	var total int64

	offset := (page - 1) * pageSize

	// Count total
	if err := s.db.Model(&models.IPBlacklist{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count blacklist: %w", err)
	}

	// Get paginated results
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&blacklist).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get blacklist: %w", err)
	}

	return blacklist, total, nil
}

// =============================================================================
// Anomaly Detection and Behavior Analysis
// =============================================================================

// RecordAccessAttempt records an access attempt for analysis
func (s *SecurityService) RecordAccessAttempt(ctx context.Context, attempt *models.AccessAttempt) error {
	if err := s.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to record access attempt: %w", err)
	}

	// Check for anomalies if this was a failed attempt
	if !attempt.Success {
		go s.analyzeAccessPattern(ctx, attempt.IPAddress, attempt.AttemptType)
	}

	return nil
}

// analyzeAccessPattern analyzes access patterns for anomalies
func (s *SecurityService) analyzeAccessPattern(ctx context.Context, ipAddress, attemptType string) {
	// Count failed attempts in the window
	var failedCount int64
	windowStart := time.Now().Add(-s.config.FailedLoginWindow)

	err := s.db.Model(&models.AccessAttempt{}).
		Where("ip_address = ? AND success = false AND created_at > ?", ipAddress, windowStart).
		Count(&failedCount).Error
	if err != nil {
		return
	}

	// Check if threshold exceeded
	if int(failedCount) >= s.config.MaxFailedLoginAttempts {
		// Block the IP
		duration := s.config.BlockDuration
		reason := fmt.Sprintf("Exceeded %d failed %s attempts in %v", 
			s.config.MaxFailedLoginAttempts, attemptType, s.config.FailedLoginWindow)
		
		s.BlockIP(ctx, ipAddress, reason, "system", &duration)

		// Create security event
		s.CreateSecurityEvent(ctx, &models.SecurityEvent{
			EventType:   models.EventBruteForce,
			Severity:    models.SeverityHigh,
			IPAddress:   &ipAddress,
			Description: reason,
			Details: models.JSONMap{
				"failed_attempts": failedCount,
				"window":          s.config.FailedLoginWindow.String(),
				"attempt_type":    attemptType,
			},
		})
	}
}

// CheckRateLimit checks if an IP has exceeded the rate limit
func (s *SecurityService) CheckRateLimit(ctx context.Context, ipAddress string) (bool, error) {
	if s.redis == nil {
		return true, nil // Allow if Redis not available
	}

	key := fmt.Sprintf("rate_limit:%s", ipAddress)
	
	// Increment counter
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return true, fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Set expiry on first request
	if count == 1 {
		s.redis.Expire(ctx, key, time.Minute)
	}

	// Check if exceeded
	if int(count) > s.config.MaxAPIRequestsPerMinute {
		// Create security event on first exceed
		if int(count) == s.config.MaxAPIRequestsPerMinute+1 {
			s.CreateSecurityEvent(ctx, &models.SecurityEvent{
				EventType:   models.EventRateLimitExceeded,
				Severity:    models.SeverityMedium,
				IPAddress:   &ipAddress,
				Description: fmt.Sprintf("Rate limit exceeded: %d requests/minute", count),
				Details: models.JSONMap{
					"request_count": count,
					"limit":         s.config.MaxAPIRequestsPerMinute,
				},
			})
		}
		return false, nil
	}

	return true, nil
}

// DetectAnomalousActivity checks for anomalous activity patterns
func (s *SecurityService) DetectAnomalousActivity(ctx context.Context, userID uuid.UUID, ipAddress string, activity string) error {
	// Check for unusual login location (simplified - in production would use GeoIP)
	// Check for unusual time of access
	// Check for unusual activity patterns

	// Get user's recent activity
	var recentAttempts []models.AccessAttempt
	err := s.db.Where("user_id = ? AND created_at > ?", userID, time.Now().Add(-24*time.Hour)).
		Order("created_at DESC").
		Limit(100).
		Find(&recentAttempts).Error
	if err != nil {
		return fmt.Errorf("failed to get recent attempts: %w", err)
	}

	// Check for multiple IPs in short time (potential account compromise)
	uniqueIPs := make(map[string]bool)
	for _, attempt := range recentAttempts {
		uniqueIPs[attempt.IPAddress] = true
	}

	if len(uniqueIPs) > 5 {
		s.CreateSecurityEvent(ctx, &models.SecurityEvent{
			EventType:   models.EventAnomalyDetected,
			Severity:    models.SeverityMedium,
			IPAddress:   &ipAddress,
			UserID:      &userID,
			Description: "Multiple IP addresses detected for user in 24 hours",
			Details: models.JSONMap{
				"unique_ips": len(uniqueIPs),
				"activity":   activity,
			},
		})
	}

	return nil
}

// =============================================================================
// Security Events
// =============================================================================

// CreateSecurityEvent creates a new security event
func (s *SecurityService) CreateSecurityEvent(ctx context.Context, event *models.SecurityEvent) error {
	if err := s.db.Create(event).Error; err != nil {
		return fmt.Errorf("failed to create security event: %w", err)
	}
	return nil
}

// GetSecurityEvents retrieves security events with filtering
func (s *SecurityService) GetSecurityEvents(ctx context.Context, filter SecurityEventFilter) ([]models.SecurityEvent, int64, error) {
	var events []models.SecurityEvent
	var total int64

	query := s.db.Model(&models.SecurityEvent{})

	// Apply filters
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.IPAddress != "" {
		query = query.Where("ip_address = ?", filter.IPAddress)
	}
	if filter.Resolved != nil {
		query = query.Where("resolved = ?", *filter.Resolved)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count security events: %w", err)
	}

	// Get paginated results
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get security events: %w", err)
	}

	return events, total, nil
}

// SecurityEventFilter defines filters for querying security events
type SecurityEventFilter struct {
	EventType  models.SecurityEventType
	Severity   models.SecurityEventSeverity
	UserID     *uuid.UUID
	IPAddress  string
	Resolved   *bool
	StartDate  time.Time
	EndDate    time.Time
	Page       int
	PageSize   int
}

// ResolveSecurityEvent marks a security event as resolved
func (s *SecurityService) ResolveSecurityEvent(ctx context.Context, eventID, resolvedByUserID uuid.UUID) error {
	now := time.Now()
	result := s.db.Model(&models.SecurityEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"resolved":    true,
			"resolved_at": now,
			"resolved_by": resolvedByUserID,
		})
	
	if result.Error != nil {
		return fmt.Errorf("failed to resolve security event: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("security event not found")
	}

	return nil
}


// =============================================================================
// Audit Logging
// =============================================================================

// CreateAuditLog creates a new audit log entry
func (s *SecurityService) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	if err := s.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// LogAction is a convenience method to create an audit log entry
func (s *SecurityService) LogAction(ctx context.Context, userID *uuid.UUID, action string, resourceType string, resourceID *uuid.UUID, details models.JSONMap, ipAddress, userAgent string) error {
	var ip *string
	if ipAddress != "" {
		ip = &ipAddress
	}

	log := &models.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ip,
		UserAgent:    userAgent,
	}

	return s.CreateAuditLog(ctx, log)
}

// GetAuditLogs retrieves audit logs with filtering
func (s *SecurityService) GetAuditLogs(ctx context.Context, filter AuditLogFilter) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := s.db.Model(&models.AuditLog{})

	// Apply filters
	if filter.UserID != nil {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.IPAddress != "" {
		query = query.Where("ip_address = ?", filter.IPAddress)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, total, nil
}

// AuditLogFilter defines filters for querying audit logs
type AuditLogFilter struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	IPAddress    string
	StartDate    time.Time
	EndDate      time.Time
	Page         int
	PageSize     int
}

// CleanupOldAuditLogs removes audit logs older than the retention period
func (s *SecurityService) CleanupOldAuditLogs(ctx context.Context) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -s.config.AuditLogRetentionDays)
	
	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup old audit logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// =============================================================================
// Data Encryption (AES-256-GCM)
// =============================================================================

// Encrypt encrypts data using AES-256-GCM
func (s *SecurityService) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	s.mu.RLock()
	key := s.encryptionKey
	s.mu.RUnlock()

	if len(key) == 0 {
		return nil, nil, errors.New("encryption key not set")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts data using AES-256-GCM
func (s *SecurityService) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	s.mu.RLock()
	key := s.encryptionKey
	s.mu.RUnlock()

	if len(key) == 0 {
		return nil, errors.New("encryption key not set")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext and nonce
func (s *SecurityService) EncryptString(plaintext string) (ciphertext, nonce string, err error) {
	ct, n, err := s.Encrypt([]byte(plaintext))
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(ct), base64.StdEncoding.EncodeToString(n), nil
}

// DecryptString decrypts base64-encoded ciphertext and nonce
func (s *SecurityService) DecryptString(ciphertext, nonce string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	n, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	plaintext, err := s.Decrypt(ct, n)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// StoreEncryptedData stores encrypted data in the database
func (s *SecurityService) StoreEncryptedData(ctx context.Context, resourceType string, resourceID uuid.UUID, data []byte) (*models.EncryptedData, error) {
	// Get active encryption key
	var encKey models.EncryptionKey
	err := s.db.Where("status = ?", models.KeyStatusActive).First(&encKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create a new key if none exists
			encKey = models.EncryptionKey{
				KeyID:     s.config.MasterKeyID,
				KeyType:   "data",
				Algorithm: "AES-256-GCM",
				Status:    models.KeyStatusActive,
				Version:   1,
			}
			if err := s.db.Create(&encKey).Error; err != nil {
				return nil, fmt.Errorf("failed to create encryption key: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}
	}

	// Encrypt the data
	ciphertext, nonce, err := s.Encrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Store encrypted data
	encData := &models.EncryptedData{
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		KeyID:         encKey.ID,
		EncryptedData: ciphertext,
		Nonce:         nonce,
	}

	// Upsert - update if exists, create if not
	err = s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Assign(models.EncryptedData{
			KeyID:         encKey.ID,
			EncryptedData: ciphertext,
			Nonce:         nonce,
		}).
		FirstOrCreate(encData).Error
	if err != nil {
		return nil, fmt.Errorf("failed to store encrypted data: %w", err)
	}

	return encData, nil
}

// RetrieveEncryptedData retrieves and decrypts data from the database
func (s *SecurityService) RetrieveEncryptedData(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]byte, error) {
	var encData models.EncryptedData
	err := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).First(&encData).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEncryptionKeyNotFound
		}
		return nil, fmt.Errorf("failed to get encrypted data: %w", err)
	}

	// Decrypt the data
	plaintext, err := s.Decrypt(encData.EncryptedData, encData.Nonce)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// DeleteEncryptedData removes encrypted data from the database
func (s *SecurityService) DeleteEncryptedData(ctx context.Context, resourceType string, resourceID uuid.UUID) error {
	result := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).Delete(&models.EncryptedData{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete encrypted data: %w", result.Error)
	}
	return nil
}

// =============================================================================
// Key Management (Vault Integration Placeholder)
// =============================================================================

// RotateEncryptionKey rotates the encryption key
func (s *SecurityService) RotateEncryptionKey(ctx context.Context, newKey []byte) error {
	if len(newKey) != 32 {
		return ErrInvalidKeySize
	}

	// Mark current key as rotating
	err := s.db.Model(&models.EncryptionKey{}).
		Where("status = ?", models.KeyStatusActive).
		Update("status", models.KeyStatusRotating).Error
	if err != nil {
		return fmt.Errorf("failed to mark key as rotating: %w", err)
	}

	// Create new key
	newEncKey := &models.EncryptionKey{
		KeyID:     fmt.Sprintf("%s-v%d", s.config.MasterKeyID, time.Now().Unix()),
		KeyType:   "data",
		Algorithm: "AES-256-GCM",
		Status:    models.KeyStatusActive,
		Version:   1,
	}

	// Get the latest version
	var latestKey models.EncryptionKey
	if err := s.db.Order("version DESC").First(&latestKey).Error; err == nil {
		newEncKey.Version = latestKey.Version + 1
	}

	if err := s.db.Create(newEncKey).Error; err != nil {
		return fmt.Errorf("failed to create new encryption key: %w", err)
	}

	// Update the encryption key in memory
	s.mu.Lock()
	s.encryptionKey = newKey
	s.mu.Unlock()

	// Mark old keys as retired
	err = s.db.Model(&models.EncryptionKey{}).
		Where("status = ?", models.KeyStatusRotating).
		Updates(map[string]interface{}{
			"status":     models.KeyStatusRetired,
			"rotated_at": time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to retire old keys: %w", err)
	}

	return nil
}

// GetActiveEncryptionKey returns the active encryption key metadata
func (s *SecurityService) GetActiveEncryptionKey(ctx context.Context) (*models.EncryptionKey, error) {
	var key models.EncryptionKey
	err := s.db.Where("status = ?", models.KeyStatusActive).First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEncryptionKeyNotFound
		}
		return nil, fmt.Errorf("failed to get active encryption key: %w", err)
	}
	return &key, nil
}
