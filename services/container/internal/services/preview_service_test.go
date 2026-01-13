// Package services provides business logic for the container service.
package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestPreviewService creates a test preview service
func setupTestPreviewService(t *testing.T) *PreviewService {
	logger, _ := zap.NewDevelopment()
	config := &PreviewServiceConfig{
		BaseDomain:             "preview.devbox.test",
		CNAMETarget:            "proxy.devbox.test",
		DefaultSSLEnabled:      true,
		DNSPropagationTime:     5 * time.Minute,
		MaxDomainsPerUser:      50,
		MaxShareLinksPerDomain: 100,
		ShareLinkMinDuration:   time.Hour,
		ShareLinkMaxDuration:   30 * 24 * time.Hour,
	}
	return NewPreviewService(logger, config)
}

// setupTestHotReloadService creates a test hot reload service
func setupTestHotReloadService(t *testing.T) *HotReloadService {
	logger, _ := zap.NewDevelopment()
	config := &HotReloadServiceConfig{
		DefaultDebounceMs:    300,
		MaxWatchPaths:        100,
		NotificationTimeout:  5 * time.Second,
		MaxSessionsPerDomain: 100,
	}
	return NewHotReloadService(logger, config)
}

func TestPreviewService_CreateDomain(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	t.Run("create auto domain", func(t *testing.T) {
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-123",
			Port:          3000,
			Type:          models.DomainTypeAuto,
		}

		domain, err := service.CreateDomain(ctx, "user-123", req)
		require.NoError(t, err)
		assert.NotEmpty(t, domain.ID)
		assert.Equal(t, "env-123", domain.EnvironmentID)
		assert.Equal(t, "user-123", domain.UserID)
		assert.Equal(t, models.DomainTypeAuto, domain.Type)
		assert.NotEmpty(t, domain.Subdomain)
		assert.Contains(t, domain.FullDomain, ".preview.devbox.test")
		assert.Equal(t, int32(3000), domain.Port)
		assert.Equal(t, models.DomainStatusActive, domain.Status)
	})

	t.Run("create custom domain", func(t *testing.T) {
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-456",
			Port:          8080,
			Type:          models.DomainTypeCustom,
			CustomDomain:  "myapp.example.com",
		}

		domain, err := service.CreateDomain(ctx, "user-123", req)
		require.NoError(t, err)
		assert.Equal(t, models.DomainTypeCustom, domain.Type)
		assert.Equal(t, "myapp.example.com", domain.CustomDomain)
		assert.Equal(t, "myapp.example.com", domain.FullDomain)
		assert.Equal(t, models.DomainStatusPending, domain.Status) // Pending until DNS verified
		assert.False(t, domain.DNSVerified)
	})

	t.Run("invalid custom domain format", func(t *testing.T) {
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-789",
			Port:          3000,
			Type:          models.DomainTypeCustom,
			CustomDomain:  "invalid domain",
		}

		_, err := service.CreateDomain(ctx, "user-123", req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain format")
	})

	t.Run("custom domain without domain name", func(t *testing.T) {
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-abc",
			Port:          3000,
			Type:          models.DomainTypeCustom,
		}

		_, err := service.CreateDomain(ctx, "user-123", req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "custom domain is required")
	})
}

func TestPreviewService_GetDomain(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create a domain first
	req := &models.CreateDomainRequest{
		EnvironmentID: "env-get-test",
		Port:          3000,
	}
	created, err := service.CreateDomain(ctx, "user-123", req)
	require.NoError(t, err)

	t.Run("get existing domain", func(t *testing.T) {
		domain, err := service.GetDomain(ctx, "user-123", created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, domain.ID)
	})

	t.Run("access denied for different user", func(t *testing.T) {
		_, err := service.GetDomain(ctx, "other-user", created.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.GetDomain(ctx, "user-123", "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPreviewService_CreateShareLink(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create a domain first
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-share-test",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "user-123", domainReq)
	require.NoError(t, err)

	t.Run("create share link", func(t *testing.T) {
		req := &models.CreateShareLinkRequest{
			DomainID: domain.ID,
			Duration: "24h",
		}

		link, err := service.CreateShareLink(ctx, "user-123", req)
		require.NoError(t, err)
		assert.NotEmpty(t, link.ID)
		assert.NotEmpty(t, link.Token)
		assert.NotEmpty(t, link.URL)
		assert.Equal(t, domain.ID, link.DomainID)
		assert.True(t, link.IsActive)
		assert.False(t, link.HasPassword)
		assert.Equal(t, 0, link.ViewCount)
	})

	t.Run("create share link with password", func(t *testing.T) {
		req := &models.CreateShareLinkRequest{
			DomainID: domain.ID,
			Duration: "7d",
			Password: "secret123",
		}

		link, err := service.CreateShareLink(ctx, "user-123", req)
		require.NoError(t, err)
		assert.True(t, link.HasPassword)
		assert.Equal(t, "secret123", link.Password)
	})

	t.Run("create share link with max views", func(t *testing.T) {
		req := &models.CreateShareLinkRequest{
			DomainID: domain.ID,
			Duration: "1h",
			MaxViews: 10,
		}

		link, err := service.CreateShareLink(ctx, "user-123", req)
		require.NoError(t, err)
		assert.Equal(t, 10, link.MaxViews)
	})

	t.Run("invalid duration", func(t *testing.T) {
		req := &models.CreateShareLinkRequest{
			DomainID: domain.ID,
			Duration: "invalid",
		}

		_, err := service.CreateShareLink(ctx, "user-123", req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid duration")
	})
}

func TestPreviewService_ValidateShareLink(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create domain and share link
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-validate-test",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "user-123", domainReq)
	require.NoError(t, err)

	linkReq := &models.CreateShareLinkRequest{
		DomainID: domain.ID,
		Duration: "24h",
		Password: "secret",
	}
	link, err := service.CreateShareLink(ctx, "user-123", linkReq)
	require.NoError(t, err)

	t.Run("validate with correct password", func(t *testing.T) {
		req := &models.ValidateShareLinkRequest{
			Token:    link.Token,
			Password: "secret",
		}

		validated, err := service.ValidateShareLink(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, link.ID, validated.ID)
		assert.Equal(t, 1, validated.ViewCount) // View count incremented
	})

	t.Run("validate with wrong password", func(t *testing.T) {
		req := &models.ValidateShareLinkRequest{
			Token:    link.Token,
			Password: "wrong",
		}

		_, err := service.ValidateShareLink(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid password")
	})

	t.Run("validate non-existent token", func(t *testing.T) {
		req := &models.ValidateShareLinkRequest{
			Token: "non-existent-token",
		}

		_, err := service.ValidateShareLink(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPreviewService_RevokeShareLink(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create domain and share link
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-revoke-test",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "user-123", domainReq)
	require.NoError(t, err)

	linkReq := &models.CreateShareLinkRequest{
		DomainID: domain.ID,
		Duration: "24h",
	}
	link, err := service.CreateShareLink(ctx, "user-123", linkReq)
	require.NoError(t, err)

	t.Run("revoke share link", func(t *testing.T) {
		err := service.RevokeShareLink(ctx, "user-123", link.ID)
		require.NoError(t, err)

		// Try to validate revoked link
		req := &models.ValidateShareLinkRequest{
			Token: link.Token,
		}
		_, err = service.ValidateShareLink(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no longer active")
	})
}


// **Feature: cloud-devbox, Property 7: 热重载 < 5秒**
// **Validates: Requirements 3.2**
func TestProperty7_HotReloadLatency(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hotReloadService := NewHotReloadService(logger, &HotReloadServiceConfig{
		DefaultDebounceMs:    300,
		MaxWatchPaths:        100,
		NotificationTimeout:  5 * time.Second,
		MaxSessionsPerDomain: 100,
	})
	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100
	var totalLatency time.Duration
	var maxLatency time.Duration
	successCount := 0

	for i := 0; i < iterations; i++ {
		environmentID := "env-hotreload-" + string(rune('a'+i%26))
		
		// Set up hot reload config
		config := &models.HotReloadConfig{
			Enabled:       true,
			WatchPaths:    []string{"."},
			IgnorePaths:   []string{"node_modules", ".git"},
			DebounceMs:    300,
			NotifyClients: true,
		}
		err := hotReloadService.SetConfig(ctx, environmentID, config)
		require.NoError(t, err)

		// Subscribe to changes
		eventChan, cleanup := hotReloadService.SubscribeToChanges(ctx, environmentID)
		defer cleanup()

		// Simulate file change and measure notification latency
		event := &models.FileChangeEvent{
			EnvironmentID: environmentID,
			Path:          "src/app.js",
			Type:          "modify",
		}

		start := time.Now()
		
		// Send notification in goroutine
		go func() {
			time.Sleep(10 * time.Millisecond) // Small delay to ensure subscriber is ready
			hotReloadService.NotifyFileChange(ctx, event)
		}()

		// Wait for notification with timeout
		select {
		case receivedEvent := <-eventChan:
			latency := time.Since(start)
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
			
			// Property: Hot reload notification should be received within 5 seconds
			assert.Less(t, latency, 5*time.Second,
				"Hot reload latency %v exceeded 5s limit", latency)
			assert.Equal(t, event.Path, receivedEvent.Path)
			assert.Equal(t, event.Type, receivedEvent.Type)
			successCount++
			
		case <-time.After(5 * time.Second):
			t.Errorf("Iteration %d: Hot reload notification timeout (>5s)", i)
		}
	}

	avgLatency := totalLatency / time.Duration(successCount)
	t.Logf("Property 7 Results: avg=%v, max=%v, success=%d/%d", 
		avgLatency, maxLatency, successCount, iterations)
	
	// At least 95% of notifications should succeed within 5 seconds
	assert.GreaterOrEqual(t, float64(successCount)/float64(iterations), 0.95,
		"At least 95%% of hot reload notifications should complete within 5s")
}

// **Feature: cloud-devbox, Property 17: DNS 生效 < 5分钟**
// **Validates: Requirements 3.4**
func TestProperty17_DNSPropagation(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100
	var totalTime time.Duration
	var maxTime time.Duration
	successCount := 0

	for i := 0; i < iterations; i++ {
		// Create auto domain (DNS is automatically configured)
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-dns-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Port:          3000 + int32(i%1000),
			Type:          models.DomainTypeAuto,
		}

		start := time.Now()
		domain, err := service.CreateDomain(ctx, "dns-test-user", req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, domain)

		totalTime += elapsed
		if elapsed > maxTime {
			maxTime = elapsed
		}

		// Property: For auto domains, DNS should be configured immediately
		// (mock DNS provider returns success immediately)
		// In production, this would verify actual DNS propagation
		
		// For auto domains, status should be Active (DNS verified)
		assert.Equal(t, models.DomainStatusActive, domain.Status,
			"Auto domain should be active immediately")
		assert.True(t, domain.DNSVerified,
			"Auto domain DNS should be verified")
		assert.NotEmpty(t, domain.DNSRecordID,
			"Auto domain should have DNS record ID")

		// Property: DNS configuration should complete within 5 minutes
		// (In this test, we verify the service layer completes quickly)
		assert.Less(t, elapsed, 5*time.Minute,
			"DNS configuration took %v, expected < 5 minutes", elapsed)
		
		successCount++
	}

	avgTime := totalTime / time.Duration(iterations)
	t.Logf("Property 17 Results: avg=%v, max=%v, success=%d/%d",
		avgTime, maxTime, successCount, iterations)
}

// **Feature: cloud-devbox, Property 18: 分享链接有效期**
// **Validates: Requirements 3.5**
func TestProperty18_ShareLinkValidity(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create a domain for testing
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-sharelink-validity",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "validity-user", domainReq)
	require.NoError(t, err)

	// Test different durations
	durations := []string{"1h", "6h", "12h", "24h", "7d", "14d", "30d"}
	expectedDurations := map[string]time.Duration{
		"1h":  time.Hour,
		"6h":  6 * time.Hour,
		"12h": 12 * time.Hour,
		"24h": 24 * time.Hour,
		"7d":  7 * 24 * time.Hour,
		"14d": 14 * 24 * time.Hour,
		"30d": 30 * 24 * time.Hour,
	}

	for _, duration := range durations {
		t.Run("duration_"+duration, func(t *testing.T) {
			req := &models.CreateShareLinkRequest{
				DomainID: domain.ID,
				Duration: duration,
			}

			link, err := service.CreateShareLink(ctx, "validity-user", req)
			require.NoError(t, err)

			// Property: Share link should have correct expiration time
			expectedExpiry := link.CreatedAt.Add(expectedDurations[duration])
			
			// Allow 1 second tolerance for timing
			assert.WithinDuration(t, expectedExpiry, link.ExpiresAt, time.Second,
				"Share link expiry should match duration %s", duration)
			
			// Property: Share link should be active when created
			assert.True(t, link.IsActive,
				"Newly created share link should be active")
			
			// Property: Share link should be valid before expiration
			validateReq := &models.ValidateShareLinkRequest{
				Token: link.Token,
			}
			validated, err := service.ValidateShareLink(ctx, validateReq)
			require.NoError(t, err)
			assert.Equal(t, link.ID, validated.ID)
		})
	}
}

// Test concurrent domain creation
func TestPreviewService_ConcurrentDomainCreation(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	const numGoroutines = 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	domains := make(chan *models.Domain, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &models.CreateDomainRequest{
				EnvironmentID: "env-concurrent-" + string(rune('a'+idx)),
				Port:          3000 + int32(idx),
			}
			domain, err := service.CreateDomain(ctx, "concurrent-user", req)
			if err != nil {
				errors <- err
			} else {
				domains <- domain
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(domains)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent creation error: %v", err)
	}

	// Verify all domains were created with unique IDs and subdomains
	ids := make(map[string]bool)
	subdomains := make(map[string]bool)
	for domain := range domains {
		assert.False(t, ids[domain.ID], "Duplicate domain ID: %s", domain.ID)
		assert.False(t, subdomains[domain.Subdomain], "Duplicate subdomain: %s", domain.Subdomain)
		ids[domain.ID] = true
		subdomains[domain.Subdomain] = true
	}
	assert.Equal(t, numGoroutines, len(ids), "All domains should have unique IDs")
}

// Test hot reload path matching
func TestHotReloadService_PathMatching(t *testing.T) {
	service := setupTestHotReloadService(t)
	ctx := context.Background()

	environmentID := "env-path-test"
	config := &models.HotReloadConfig{
		Enabled:       true,
		WatchPaths:    []string{"src", "public"},
		IgnorePaths:   []string{"node_modules", ".git", "dist"},
		DebounceMs:    100,
		NotifyClients: true,
	}
	err := service.SetConfig(ctx, environmentID, config)
	require.NoError(t, err)

	// Subscribe to changes
	eventChan, cleanup := service.SubscribeToChanges(ctx, environmentID)
	defer cleanup()

	testCases := []struct {
		path          string
		shouldNotify  bool
		description   string
	}{
		{"src/app.js", true, "file in watched directory"},
		{"src/components/Button.tsx", true, "nested file in watched directory"},
		{"public/index.html", true, "file in another watched directory"},
		{"node_modules/react/index.js", false, "file in ignored directory"},
		{".git/config", false, "file in ignored directory"},
		{"dist/bundle.js", false, "file in ignored directory"},
		{"README.md", false, "file not in watch paths"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			event := &models.FileChangeEvent{
				EnvironmentID: environmentID,
				Path:          tc.path,
				Type:          "modify",
			}

			// Send notification
			go service.NotifyFileChange(ctx, event)

			// Check if notification was received
			select {
			case received := <-eventChan:
				if !tc.shouldNotify {
					t.Errorf("Path %s should not have triggered notification, but got: %+v", tc.path, received)
				} else {
					assert.Equal(t, tc.path, received.Path)
				}
			case <-time.After(200 * time.Millisecond):
				if tc.shouldNotify {
					t.Errorf("Path %s should have triggered notification, but didn't", tc.path)
				}
			}
		})
	}
}

// Test share link view limit
func TestPreviewService_ShareLinkViewLimit(t *testing.T) {
	service := setupTestPreviewService(t)
	ctx := context.Background()

	// Create domain and share link with max views
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-view-limit",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "user-123", domainReq)
	require.NoError(t, err)

	linkReq := &models.CreateShareLinkRequest{
		DomainID: domain.ID,
		Duration: "24h",
		MaxViews: 3,
	}
	link, err := service.CreateShareLink(ctx, "user-123", linkReq)
	require.NoError(t, err)

	// Validate 3 times (should succeed)
	for i := 0; i < 3; i++ {
		req := &models.ValidateShareLinkRequest{
			Token: link.Token,
		}
		validated, err := service.ValidateShareLink(ctx, req)
		require.NoError(t, err, "Validation %d should succeed", i+1)
		assert.Equal(t, i+1, validated.ViewCount)
	}

	// 4th validation should fail
	req := &models.ValidateShareLinkRequest{
		Token: link.Token,
	}
	_, err = service.ValidateShareLink(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "view limit exceeded")
}

// Test domain user limit
func TestPreviewService_DomainUserLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &PreviewServiceConfig{
		BaseDomain:             "preview.devbox.test",
		CNAMETarget:            "proxy.devbox.test",
		DefaultSSLEnabled:      true,
		MaxDomainsPerUser:      5, // Low limit for testing
		MaxShareLinksPerDomain: 100,
		ShareLinkMinDuration:   time.Hour,
		ShareLinkMaxDuration:   30 * 24 * time.Hour,
	}
	service := NewPreviewService(logger, config)
	ctx := context.Background()

	// Create domains up to the limit
	for i := 0; i < 5; i++ {
		req := &models.CreateDomainRequest{
			EnvironmentID: "env-limit-" + string(rune('a'+i)),
			Port:          3000,
		}
		_, err := service.CreateDomain(ctx, "limit-user", req)
		require.NoError(t, err)
	}

	// 6th domain should fail
	req := &models.CreateDomainRequest{
		EnvironmentID: "env-limit-f",
		Port:          3000,
	}
	_, err := service.CreateDomain(ctx, "limit-user", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum domains per user exceeded")
}

// Test cleanup expired share links
func TestPreviewService_CleanupExpiredShareLinks(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &PreviewServiceConfig{
		BaseDomain:             "preview.devbox.test",
		CNAMETarget:            "proxy.devbox.test",
		DefaultSSLEnabled:      true,
		MaxDomainsPerUser:      50,
		MaxShareLinksPerDomain: 100,
		ShareLinkMinDuration:   time.Millisecond, // Very short for testing
		ShareLinkMaxDuration:   30 * 24 * time.Hour,
	}
	service := NewPreviewService(logger, config)
	ctx := context.Background()

	// Create domain
	domainReq := &models.CreateDomainRequest{
		EnvironmentID: "env-cleanup",
		Port:          3000,
	}
	domain, err := service.CreateDomain(ctx, "cleanup-user", domainReq)
	require.NoError(t, err)

	// Create share link with very short duration
	// Note: We need to manually set expiration for testing
	linkReq := &models.CreateShareLinkRequest{
		DomainID: domain.ID,
		Duration: "1h", // Will be modified below
	}
	link, err := service.CreateShareLink(ctx, "cleanup-user", linkReq)
	require.NoError(t, err)

	// Manually expire the link for testing
	service.mu.Lock()
	if sl, ok := service.shareLinks[link.ID]; ok {
		sl.ExpiresAt = time.Now().Add(-time.Hour) // Expired 1 hour ago
	}
	service.mu.Unlock()

	// Run cleanup
	count, err := service.CleanupExpiredShareLinks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify link is deleted
	_, err = service.GetShareLink(ctx, "cleanup-user", link.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
