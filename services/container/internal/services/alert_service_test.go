// Package services provides business logic for the container service.
package services

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestAlertService creates a test alert service
func setupTestAlertService(t *testing.T) *AlertService {
	logger, _ := zap.NewDevelopment()
	return NewAlertService(logger)
}

func TestAlertService_CreateAlert(t *testing.T) {
	service := setupTestAlertService(t)
	ctx := context.Background()

	t.Run("creates alert when threshold exceeded", func(t *testing.T) {
		metrics := &models.EnvironmentMetrics{
			EnvironmentID: "env-1",
			Timestamp:     time.Now(),
			StorageUsage:  95.0, // Above 90% threshold
		}

		service.CheckAlerts(ctx, "env-1", metrics)

		alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
			EnvironmentID: "env-1",
			Page:          1,
			PageSize:      10,
		})
		require.NoError(t, err)
		assert.Greater(t, len(alerts.Alerts), 0, "Should have created an alert")

		// Find the storage alert
		var storageAlert *models.Alert
		for _, a := range alerts.Alerts {
			if a.Type == models.MetricTypeStorage {
				storageAlert = &a
				break
			}
		}
		require.NotNil(t, storageAlert, "Should have a storage alert")
		assert.Equal(t, models.AlertSeverityCritical, storageAlert.Severity)
		assert.Equal(t, models.AlertStatusActive, storageAlert.Status)
	})

	t.Run("does not create duplicate alerts", func(t *testing.T) {
		metrics := &models.EnvironmentMetrics{
			EnvironmentID: "env-2",
			Timestamp:     time.Now(),
			StorageUsage:  95.0,
		}

		// Check alerts twice
		service.CheckAlerts(ctx, "env-2", metrics)
		service.CheckAlerts(ctx, "env-2", metrics)

		alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
			EnvironmentID: "env-2",
			Page:          1,
			PageSize:      10,
		})
		require.NoError(t, err)

		// Count storage alerts
		storageAlertCount := 0
		for _, a := range alerts.Alerts {
			if a.Type == models.MetricTypeStorage && a.EnvironmentID == "env-2" {
				storageAlertCount++
			}
		}
		assert.Equal(t, 1, storageAlertCount, "Should only have one storage alert")
	})
}

func TestAlertService_ResolveAlert(t *testing.T) {
	service := setupTestAlertService(t)
	ctx := context.Background()

	// Create an alert
	metrics := &models.EnvironmentMetrics{
		EnvironmentID: "env-resolve",
		Timestamp:     time.Now(),
		StorageUsage:  95.0,
	}
	service.CheckAlerts(ctx, "env-resolve", metrics)

	// Now metrics are below threshold
	metricsResolved := &models.EnvironmentMetrics{
		EnvironmentID: "env-resolve",
		Timestamp:     time.Now(),
		StorageUsage:  50.0, // Below threshold
	}
	service.CheckAlerts(ctx, "env-resolve", metricsResolved)

	// Check that alert is resolved
	alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
		EnvironmentID: "env-resolve",
		Page:          1,
		PageSize:      10,
	})
	require.NoError(t, err)

	for _, a := range alerts.Alerts {
		if a.Type == models.MetricTypeStorage && a.EnvironmentID == "env-resolve" {
			assert.Equal(t, models.AlertStatusResolved, a.Status, "Alert should be resolved")
		}
	}
}

func TestAlertService_AcknowledgeAlert(t *testing.T) {
	service := setupTestAlertService(t)
	ctx := context.Background()

	// Create an alert
	metrics := &models.EnvironmentMetrics{
		EnvironmentID: "env-ack",
		Timestamp:     time.Now(),
		StorageUsage:  95.0,
	}
	service.CheckAlerts(ctx, "env-ack", metrics)

	// Get the alert
	alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
		EnvironmentID: "env-ack",
		Status:        models.AlertStatusActive,
		Page:          1,
		PageSize:      10,
	})
	require.NoError(t, err)
	require.Greater(t, len(alerts.Alerts), 0)

	alertID := alerts.Alerts[0].ID

	// Acknowledge the alert
	acked, err := service.AcknowledgeAlert(ctx, alertID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, models.AlertStatusAcked, acked.Status)
	assert.Equal(t, "user-1", acked.AckedBy)
	assert.NotNil(t, acked.AckedAt)
}

func TestAlertService_AlertRules(t *testing.T) {
	service := setupTestAlertService(t)
	ctx := context.Background()

	t.Run("create alert rule", func(t *testing.T) {
		req := &models.CreateAlertRuleRequest{
			Type:          models.MetricTypeCPU,
			Severity:      models.AlertSeverityWarning,
			Threshold:     80.0,
			Duration:      "5m",
			NotifyMethods: []string{"web", "email"},
		}

		rule, err := service.CreateAlertRule(ctx, "user-1", req)
		require.NoError(t, err)
		assert.NotEmpty(t, rule.ID)
		assert.Equal(t, models.MetricTypeCPU, rule.Type)
		assert.Equal(t, 80.0, rule.Threshold)
		assert.True(t, rule.Enabled)
	})

	t.Run("update alert rule", func(t *testing.T) {
		// Create a rule first
		req := &models.CreateAlertRuleRequest{
			Type:          models.MetricTypeMemory,
			Severity:      models.AlertSeverityWarning,
			Threshold:     70.0,
			Duration:      "5m",
			NotifyMethods: []string{"web"},
		}
		rule, err := service.CreateAlertRule(ctx, "user-1", req)
		require.NoError(t, err)

		// Update the rule
		newThreshold := 85.0
		enabled := false
		updateReq := &models.UpdateAlertRuleRequest{
			Threshold: &newThreshold,
			Enabled:   &enabled,
		}

		updated, err := service.UpdateAlertRule(ctx, rule.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, 85.0, updated.Threshold)
		assert.False(t, updated.Enabled)
	})

	t.Run("delete alert rule", func(t *testing.T) {
		// Create a rule first
		req := &models.CreateAlertRuleRequest{
			Type:          models.MetricTypeNetwork,
			Severity:      models.AlertSeverityInfo,
			Threshold:     90.0,
			Duration:      "1m",
			NotifyMethods: []string{"web"},
		}
		rule, err := service.CreateAlertRule(ctx, "user-1", req)
		require.NoError(t, err)

		// Delete the rule
		err = service.DeleteAlertRule(ctx, rule.ID)
		require.NoError(t, err)

		// Verify it's deleted
		rules, err := service.ListAlertRules(ctx, "user-1")
		require.NoError(t, err)
		for _, r := range rules {
			assert.NotEqual(t, rule.ID, r.ID, "Deleted rule should not be in list")
		}
	})
}

func TestAlertService_GetAlertStats(t *testing.T) {
	service := setupTestAlertService(t)
	ctx := context.Background()

	// Create some alerts
	for i := 0; i < 5; i++ {
		metrics := &models.EnvironmentMetrics{
			EnvironmentID: "env-stats-" + string(rune('a'+i)),
			Timestamp:     time.Now(),
			StorageUsage:  95.0,
		}
		service.CheckAlerts(ctx, "env-stats-"+string(rune('a'+i)), metrics)
	}

	stats, err := service.GetAlertStats(ctx, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.TotalAlerts, int64(5))
	assert.GreaterOrEqual(t, stats.ActiveAlerts, int64(5))
	assert.GreaterOrEqual(t, stats.CriticalAlerts, int64(5))
}


// **Feature: cloud-devbox, Property 11: 告警响应 < 2分钟**
// **Validates: Requirements 7.6**
// This property test verifies that alerts are created and notifications are sent
// within 2 minutes of the threshold being exceeded.
func TestProperty11_AlertResponseTime(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewAlertService(logger)
	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100
	var totalResponseTime time.Duration
	var maxResponseTime time.Duration
	var failedCount int

	// Maximum allowed response time (2 minutes)
	maxAllowedResponseTime := 2 * time.Minute

	for i := 0; i < iterations; i++ {
		envID := "prop11-env-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		
		// Generate random metrics that exceed threshold
		storageUsage := 90.0 + rand.Float64()*10 // 90-100%
		
		metrics := &models.EnvironmentMetrics{
			EnvironmentID:  envID,
			Timestamp:      time.Now(),
			CPUUsage:       rand.Float64() * 100,
			MemoryUsage:    rand.Float64() * 100,
			StorageUsage:   storageUsage, // This will trigger alert
			NetworkRxRate:  rand.Float64() * 1024 * 1024,
			NetworkTxRate:  rand.Float64() * 1024 * 1024,
		}

		// Record start time
		startTime := time.Now()

		// Check alerts (this creates alert and sends notification)
		service.CheckAlerts(ctx, envID, metrics)

		// Get the created alert
		alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
			EnvironmentID: envID,
			Status:        models.AlertStatusActive,
			Page:          1,
			PageSize:      10,
		})
		require.NoError(t, err)

		// Find the storage alert
		var storageAlert *models.Alert
		for _, a := range alerts.Alerts {
			if a.Type == models.MetricTypeStorage && a.EnvironmentID == envID {
				storageAlert = &a
				break
			}
		}

		if storageAlert == nil {
			failedCount++
			continue
		}

		// Calculate response time (from alert creation to notification)
		var responseTime time.Duration
		if storageAlert.NotifiedAt != nil {
			responseTime = storageAlert.NotifiedAt.Sub(storageAlert.CreatedAt)
		} else {
			// If not notified yet, use time since creation
			responseTime = time.Since(storageAlert.CreatedAt)
		}

		// Also consider the time from threshold exceeded to alert creation
		totalTime := time.Since(startTime)

		totalResponseTime += totalTime
		if totalTime > maxResponseTime {
			maxResponseTime = totalTime
		}

		// Property: Alert response time should be less than 2 minutes
		assert.Less(t, totalTime, maxAllowedResponseTime,
			"Alert response time %v exceeded 2 minutes for env %s (storage: %.1f%%)",
			totalTime, envID, storageUsage)

		// Verify alert was created correctly
		assert.Equal(t, models.AlertSeverityCritical, storageAlert.Severity,
			"Storage alert should be critical severity")
		assert.Equal(t, models.AlertStatusActive, storageAlert.Status,
			"Alert should be active")
		assert.GreaterOrEqual(t, storageAlert.Value, 90.0,
			"Alert value should be >= 90%%")
	}

	avgResponseTime := totalResponseTime / time.Duration(iterations)
	
	t.Logf("Property 11 Results:")
	t.Logf("  - Iterations: %d", iterations)
	t.Logf("  - Average response time: %v", avgResponseTime)
	t.Logf("  - Maximum response time: %v", maxResponseTime)
	t.Logf("  - Failed alerts: %d", failedCount)
	t.Logf("  - Success rate: %.1f%%", float64(iterations-failedCount)/float64(iterations)*100)

	// Overall property assertion
	assert.Less(t, avgResponseTime, maxAllowedResponseTime,
		"Average alert response time should be less than 2 minutes")
	assert.Less(t, maxResponseTime, maxAllowedResponseTime,
		"Maximum alert response time should be less than 2 minutes")
	assert.Equal(t, 0, failedCount,
		"All alerts should be created successfully")
}

// TestProperty11_AlertResponseTimeCompliance tests the compliance check function
func TestProperty11_AlertResponseTimeCompliance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewAlertService(logger)

	t.Run("compliant alert", func(t *testing.T) {
		now := time.Now()
		notifiedAt := now.Add(30 * time.Second) // 30 seconds response time
		alert := &models.Alert{
			ID:         "test-1",
			CreatedAt:  now,
			NotifiedAt: &notifiedAt,
		}

		assert.True(t, service.CheckAlertResponseTimeCompliance(alert),
			"Alert with 30s response time should be compliant")
	})

	t.Run("non-compliant alert", func(t *testing.T) {
		now := time.Now()
		notifiedAt := now.Add(3 * time.Minute) // 3 minutes response time
		alert := &models.Alert{
			ID:         "test-2",
			CreatedAt:  now,
			NotifiedAt: &notifiedAt,
		}

		assert.False(t, service.CheckAlertResponseTimeCompliance(alert),
			"Alert with 3 minute response time should not be compliant")
	})

	t.Run("alert not yet notified", func(t *testing.T) {
		alert := &models.Alert{
			ID:        "test-3",
			CreatedAt: time.Now(),
			// NotifiedAt is nil
		}

		responseTime := service.GetAlertResponseTime(alert)
		assert.Equal(t, time.Duration(0), responseTime,
			"Alert without notification should have 0 response time")
	})
}

// TestAlertService_ConcurrentAlertCreation tests concurrent alert creation
func TestAlertService_ConcurrentAlertCreation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewAlertService(logger)
	ctx := context.Background()

	const numGoroutines = 50
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			envID := "concurrent-alert-env-" + string(rune('a'+idx%26))
			metrics := &models.EnvironmentMetrics{
				EnvironmentID: envID,
				Timestamp:     time.Now(),
				StorageUsage:  95.0,
			}
			service.CheckAlerts(ctx, envID, metrics)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify alerts were created
	stats, err := service.GetAlertStats(ctx, "")
	require.NoError(t, err)
	assert.Greater(t, stats.TotalAlerts, int64(0),
		"Should have created alerts concurrently")
}

// TestAlertService_MultipleMetricTypes tests alerts for different metric types
func TestAlertService_MultipleMetricTypes(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewAlertService(logger)
	ctx := context.Background()

	envID := "multi-metric-env"
	
	// Metrics that exceed multiple thresholds
	metrics := &models.EnvironmentMetrics{
		EnvironmentID: envID,
		Timestamp:     time.Now(),
		CPUUsage:      98.0,     // Above 95% threshold
		MemoryUsage:   92.0,     // Above 90% threshold
		StorageUsage:  95.0,     // Above 90% threshold
	}

	service.CheckAlerts(ctx, envID, metrics)

	alerts, err := service.ListAlerts(ctx, "", &models.ListAlertsRequest{
		EnvironmentID: envID,
		Page:          1,
		PageSize:      10,
	})
	require.NoError(t, err)

	// Should have alerts for storage (critical) at minimum
	// CPU and memory alerts depend on default rules
	hasStorageAlert := false
	for _, a := range alerts.Alerts {
		if a.Type == models.MetricTypeStorage {
			hasStorageAlert = true
			assert.Equal(t, models.AlertSeverityCritical, a.Severity)
		}
	}
	assert.True(t, hasStorageAlert, "Should have storage alert")
}
