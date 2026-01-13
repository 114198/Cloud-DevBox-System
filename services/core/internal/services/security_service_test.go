// Package services provides business logic services for the core service.
package services

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSecurityTestDB creates an in-memory SQLite database for security testing
func setupSecurityTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate security models
	err = db.AutoMigrate(
		&models.User{},
		&models.IPBlacklist{},
		&models.AccessAttempt{},
		&models.SecurityEvent{},
		&models.EncryptionKey{},
		&models.EncryptedData{},
		&models.AuditLog{},
	)
	require.NoError(t, err)

	return db
}

// generateTestEncryptionKey generates a random 32-byte key for testing
func generateTestEncryptionKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// =============================================================================
// IP Blacklist Tests
// =============================================================================

func TestSecurityService_BlockIP(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	t.Run("block valid IP", func(t *testing.T) {
		err := securityService.BlockIP(ctx, "192.168.1.100", "Test block", "admin", nil)
		assert.NoError(t, err)

		blocked, err := securityService.IsIPBlocked(ctx, "192.168.1.100")
		assert.NoError(t, err)
		assert.True(t, blocked)
	})

	t.Run("block IP with duration", func(t *testing.T) {
		duration := 1 * time.Hour
		err := securityService.BlockIP(ctx, "192.168.1.101", "Temporary block", "system", &duration)
		assert.NoError(t, err)

		blocked, err := securityService.IsIPBlocked(ctx, "192.168.1.101")
		assert.NoError(t, err)
		assert.True(t, blocked)
	})

	t.Run("block invalid IP", func(t *testing.T) {
		err := securityService.BlockIP(ctx, "invalid-ip", "Test block", "admin", nil)
		assert.Error(t, err)
	})

	t.Run("unblock IP", func(t *testing.T) {
		// First block
		err := securityService.BlockIP(ctx, "192.168.1.102", "Test block", "admin", nil)
		require.NoError(t, err)

		// Then unblock
		err = securityService.UnblockIP(ctx, "192.168.1.102")
		assert.NoError(t, err)

		blocked, err := securityService.IsIPBlocked(ctx, "192.168.1.102")
		assert.NoError(t, err)
		assert.False(t, blocked)
	})

	t.Run("unblock non-existent IP", func(t *testing.T) {
		err := securityService.UnblockIP(ctx, "10.0.0.1")
		assert.Error(t, err)
		assert.Equal(t, ErrIPNotFound, err)
	})
}

func TestSecurityService_IsIPBlocked(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	t.Run("non-blocked IP returns false", func(t *testing.T) {
		blocked, err := securityService.IsIPBlocked(ctx, "10.0.0.100")
		assert.NoError(t, err)
		assert.False(t, blocked)
	})

	t.Run("blocked IP returns true", func(t *testing.T) {
		err := securityService.BlockIP(ctx, "10.0.0.101", "Test", "admin", nil)
		require.NoError(t, err)

		blocked, err := securityService.IsIPBlocked(ctx, "10.0.0.101")
		assert.NoError(t, err)
		assert.True(t, blocked)
	})

	t.Run("invalid IP returns error", func(t *testing.T) {
		_, err := securityService.IsIPBlocked(ctx, "not-an-ip")
		assert.Error(t, err)
	})
}

func TestSecurityService_GetBlacklistedIPs(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	// Block some IPs
	for i := 0; i < 5; i++ {
		ip := "192.168.2." + string(rune('0'+i))
		err := securityService.BlockIP(ctx, "192.168.2."+string(rune('0'+i)), "Test "+ip, "admin", nil)
		require.NoError(t, err)
	}

	t.Run("get paginated blacklist", func(t *testing.T) {
		blacklist, total, err := securityService.GetBlacklistedIPs(ctx, 1, 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, blacklist, 3)
	})

	t.Run("get second page", func(t *testing.T) {
		blacklist, total, err := securityService.GetBlacklistedIPs(ctx, 2, 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, blacklist, 2)
	})
}

// =============================================================================
// Property Test: Anomalous Access Auto-Block
// Feature: cloud-devbox, Property 16: 异常访问自动阻止
// Validates: Requirements 9.3
// =============================================================================

func TestSecurityService_Property16_AnomalousAccessAutoBlock(t *testing.T) {
	// Property 16: For any IP address that exceeds the failed login threshold,
	// the system should automatically block that IP address.
	
	db := setupSecurityTestDB(t)
	config := &SecurityConfig{
		MaxFailedLoginAttempts:  5,
		FailedLoginWindow:       15 * time.Minute,
		MaxAPIRequestsPerMinute: 100,
		BlockDuration:           24 * time.Hour,
		AuditLogRetentionDays:   365,
		MasterKeyID:             "test-key",
	}
	securityService := NewSecurityService(db, nil, config)
	ctx := context.Background()

	// Test with multiple random IPs
	testIPs := []string{
		"192.168.10.1",
		"10.0.0.50",
		"172.16.0.100",
		"8.8.8.8",
		"1.1.1.1",
	}

	for _, testIP := range testIPs {
		t.Run("auto-block after threshold exceeded for "+testIP, func(t *testing.T) {
			// Verify IP is not blocked initially
			blocked, err := securityService.IsIPBlocked(ctx, testIP)
			require.NoError(t, err)
			assert.False(t, blocked, "IP should not be blocked initially")

			// Record failed attempts up to threshold
			for i := 0; i < config.MaxFailedLoginAttempts; i++ {
				attempt := &models.AccessAttempt{
					IPAddress:     testIP,
					AttemptType:   "login",
					Success:       false,
					FailureReason: "invalid credentials",
					Endpoint:      "/api/v1/auth/login",
				}
				err := securityService.RecordAccessAttempt(ctx, attempt)
				require.NoError(t, err)
			}

			// Give the async goroutine time to process
			time.Sleep(100 * time.Millisecond)

			// Property assertion: IP should now be blocked
			blocked, err = securityService.IsIPBlocked(ctx, testIP)
			require.NoError(t, err)
			assert.True(t, blocked, "IP should be automatically blocked after exceeding threshold")

			// Verify a security event was created
			filter := services.SecurityEventFilter{
				EventType: models.EventBruteForce,
				IPAddress: testIP,
				Page:      1,
				PageSize:  10,
			}
			events, count, err := securityService.GetSecurityEvents(ctx, filter)
			require.NoError(t, err)
			assert.Greater(t, count, int64(0), "Security event should be created")
			if len(events) > 0 {
				assert.Equal(t, models.EventBruteForce, events[0].EventType)
				assert.Equal(t, models.SeverityHigh, events[0].Severity)
			}

			// Clean up for next test
			securityService.UnblockIP(ctx, testIP)
		})
	}
}

func TestSecurityService_Property16_SuccessfulAttemptsNotBlocked(t *testing.T) {
	// Property: Successful login attempts should not trigger blocking
	
	db := setupSecurityTestDB(t)
	config := &SecurityConfig{
		MaxFailedLoginAttempts:  5,
		FailedLoginWindow:       15 * time.Minute,
		MaxAPIRequestsPerMinute: 100,
		BlockDuration:           24 * time.Hour,
		AuditLogRetentionDays:   365,
		MasterKeyID:             "test-key",
	}
	securityService := NewSecurityService(db, nil, config)
	ctx := context.Background()

	testIP := "192.168.20.1"

	// Record many successful attempts
	for i := 0; i < 20; i++ {
		attempt := &models.AccessAttempt{
			IPAddress:   testIP,
			AttemptType: "login",
			Success:     true,
			Endpoint:    "/api/v1/auth/login",
		}
		err := securityService.RecordAccessAttempt(ctx, attempt)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Property assertion: IP should NOT be blocked
	blocked, err := securityService.IsIPBlocked(ctx, testIP)
	require.NoError(t, err)
	assert.False(t, blocked, "IP should not be blocked for successful attempts")
}

func TestSecurityService_Property16_MixedAttemptsThreshold(t *testing.T) {
	// Property: Only failed attempts count toward the threshold
	
	db := setupSecurityTestDB(t)
	config := &SecurityConfig{
		MaxFailedLoginAttempts:  5,
		FailedLoginWindow:       15 * time.Minute,
		MaxAPIRequestsPerMinute: 100,
		BlockDuration:           24 * time.Hour,
		AuditLogRetentionDays:   365,
		MasterKeyID:             "test-key",
	}
	securityService := NewSecurityService(db, nil, config)
	ctx := context.Background()

	testIP := "192.168.30.1"

	// Record mixed attempts (4 failed, 10 successful)
	for i := 0; i < 4; i++ {
		attempt := &models.AccessAttempt{
			IPAddress:     testIP,
			AttemptType:   "login",
			Success:       false,
			FailureReason: "invalid credentials",
			Endpoint:      "/api/v1/auth/login",
		}
		err := securityService.RecordAccessAttempt(ctx, attempt)
		require.NoError(t, err)
	}

	for i := 0; i < 10; i++ {
		attempt := &models.AccessAttempt{
			IPAddress:   testIP,
			AttemptType: "login",
			Success:     true,
			Endpoint:    "/api/v1/auth/login",
		}
		err := securityService.RecordAccessAttempt(ctx, attempt)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Property assertion: IP should NOT be blocked (only 4 failed, threshold is 5)
	blocked, err := securityService.IsIPBlocked(ctx, testIP)
	require.NoError(t, err)
	assert.False(t, blocked, "IP should not be blocked when failed attempts are below threshold")
}

// =============================================================================
// Encryption Tests
// =============================================================================

func TestSecurityService_Encryption(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	// Set encryption key
	key := generateTestEncryptionKey()
	err := securityService.SetEncryptionKey(key)
	require.NoError(t, err)

	t.Run("encrypt and decrypt data", func(t *testing.T) {
		plaintext := []byte("sensitive data to encrypt")

		ciphertext, nonce, err := securityService.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEmpty(t, nonce)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := securityService.Decrypt(ciphertext, nonce)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt and decrypt string", func(t *testing.T) {
		plaintext := "secret password"

		ciphertext, nonce, err := securityService.EncryptString(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEmpty(t, nonce)

		decrypted, err := securityService.DecryptString(ciphertext, nonce)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("store and retrieve encrypted data", func(t *testing.T) {
		resourceID := uuid.New()
		data := []byte("encrypted resource data")

		encData, err := securityService.StoreEncryptedData(ctx, "test_resource", resourceID, data)
		require.NoError(t, err)
		assert.NotNil(t, encData)

		retrieved, err := securityService.RetrieveEncryptedData(ctx, "test_resource", resourceID)
		require.NoError(t, err)
		assert.Equal(t, data, retrieved)
	})

	t.Run("invalid key size", func(t *testing.T) {
		err := securityService.SetEncryptionKey([]byte("short"))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKeySize, err)
	})

	t.Run("decrypt with wrong nonce fails", func(t *testing.T) {
		plaintext := []byte("test data")
		ciphertext, _, err := securityService.Encrypt(plaintext)
		require.NoError(t, err)

		wrongNonce := make([]byte, 12)
		rand.Read(wrongNonce)

		_, err = securityService.Decrypt(ciphertext, wrongNonce)
		assert.Error(t, err)
		assert.Equal(t, ErrDecryptionFailed, err)
	})
}

// =============================================================================
// Audit Log Tests
// =============================================================================

func TestSecurityService_AuditLog(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	t.Run("create audit log", func(t *testing.T) {
		userID := uuid.New()
		resourceID := uuid.New()
		ipAddress := "192.168.1.1"

		err := securityService.LogAction(
			ctx,
			&userID,
			"create",
			"environment",
			&resourceID,
			models.JSONMap{"name": "test-env"},
			ipAddress,
			"Mozilla/5.0",
		)
		require.NoError(t, err)

		// Verify log was created
		filter := AuditLogFilter{
			UserID:   &userID,
			Page:     1,
			PageSize: 10,
		}
		logs, count, err := securityService.GetAuditLogs(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		assert.Len(t, logs, 1)
		assert.Equal(t, "create", logs[0].Action)
		assert.Equal(t, "environment", logs[0].ResourceType)
	})

	t.Run("filter audit logs by action", func(t *testing.T) {
		userID := uuid.New()

		// Create logs with different actions
		for _, action := range []string{"create", "update", "delete"} {
			err := securityService.LogAction(ctx, &userID, action, "resource", nil, nil, "", "")
			require.NoError(t, err)
		}

		filter := AuditLogFilter{
			Action:   "create",
			Page:     1,
			PageSize: 10,
		}
		logs, _, err := securityService.GetAuditLogs(ctx, filter)
		require.NoError(t, err)
		for _, log := range logs {
			assert.Equal(t, "create", log.Action)
		}
	})

	t.Run("filter audit logs by date range", func(t *testing.T) {
		userID := uuid.New()

		err := securityService.LogAction(ctx, &userID, "test", "resource", nil, nil, "", "")
		require.NoError(t, err)

		filter := AuditLogFilter{
			StartDate: time.Now().Add(-1 * time.Hour),
			EndDate:   time.Now().Add(1 * time.Hour),
			Page:      1,
			PageSize:  10,
		}
		logs, count, err := securityService.GetAuditLogs(ctx, filter)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
		assert.NotEmpty(t, logs)
	})
}

// =============================================================================
// Security Events Tests
// =============================================================================

func TestSecurityService_SecurityEvents(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	t.Run("create security event", func(t *testing.T) {
		ipAddress := "192.168.1.50"
		event := &models.SecurityEvent{
			EventType:   models.EventBruteForce,
			Severity:    models.SeverityHigh,
			IPAddress:   &ipAddress,
			Description: "Multiple failed login attempts detected",
			Details: models.JSONMap{
				"attempts": 10,
			},
		}

		err := securityService.CreateSecurityEvent(ctx, event)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, event.ID)
	})

	t.Run("get security events with filters", func(t *testing.T) {
		filter := SecurityEventFilter{
			EventType: models.EventBruteForce,
			Severity:  models.SeverityHigh,
			Page:      1,
			PageSize:  10,
		}

		events, count, err := securityService.GetSecurityEvents(ctx, filter)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
		for _, event := range events {
			assert.Equal(t, models.EventBruteForce, event.EventType)
			assert.Equal(t, models.SeverityHigh, event.Severity)
		}
	})

	t.Run("resolve security event", func(t *testing.T) {
		ipAddress := "192.168.1.51"
		event := &models.SecurityEvent{
			EventType:   models.EventSuspiciousActivity,
			Severity:    models.SeverityMedium,
			IPAddress:   &ipAddress,
			Description: "Suspicious activity detected",
		}
		err := securityService.CreateSecurityEvent(ctx, event)
		require.NoError(t, err)

		resolverID := uuid.New()
		err = securityService.ResolveSecurityEvent(ctx, event.ID, resolverID)
		require.NoError(t, err)

		// Verify event is resolved
		resolved := true
		filter := SecurityEventFilter{
			Resolved: &resolved,
			Page:     1,
			PageSize: 100,
		}
		events, _, err := securityService.GetSecurityEvents(ctx, filter)
		require.NoError(t, err)

		found := false
		for _, e := range events {
			if e.ID == event.ID {
				found = true
				assert.True(t, e.Resolved)
				assert.NotNil(t, e.ResolvedAt)
				assert.Equal(t, resolverID, *e.ResolvedBy)
			}
		}
		assert.True(t, found, "Resolved event should be found")
	})
}

// =============================================================================
// Key Rotation Tests
// =============================================================================

func TestSecurityService_KeyRotation(t *testing.T) {
	db := setupSecurityTestDB(t)
	securityService := NewSecurityService(db, nil, nil)
	ctx := context.Background()

	// Set initial key
	initialKey := generateTestEncryptionKey()
	err := securityService.SetEncryptionKey(initialKey)
	require.NoError(t, err)

	t.Run("rotate encryption key", func(t *testing.T) {
		// Store some data with initial key
		resourceID := uuid.New()
		data := []byte("data encrypted with initial key")
		_, err := securityService.StoreEncryptedData(ctx, "test", resourceID, data)
		require.NoError(t, err)

		// Rotate to new key
		newKey := generateTestEncryptionKey()
		err = securityService.RotateEncryptionKey(ctx, newKey)
		require.NoError(t, err)

		// Verify new key is active
		activeKey, err := securityService.GetActiveEncryptionKey(ctx)
		require.NoError(t, err)
		assert.Equal(t, models.KeyStatusActive, activeKey.Status)
	})

	t.Run("rotate with invalid key size fails", func(t *testing.T) {
		err := securityService.RotateEncryptionKey(ctx, []byte("short"))
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidKeySize, err)
	})
}
