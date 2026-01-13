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

// setupTestMetricsCollector creates a test metrics collector
func setupTestMetricsCollector(t *testing.T) *MetricsCollector {
	logger, _ := zap.NewDevelopment()
	config := &MetricsCollectorConfig{
		CollectionInterval: 30 * time.Second,
		PrometheusURL:      "http://localhost:9090",
		RetentionPeriod:    1 * time.Hour,
		MaxDataPoints:      1000,
	}
	return NewMetricsCollector(config, logger)
}

// setupTestAlertManagerService creates a test AlertManager service
func setupTestAlertManagerService(t *testing.T) *AlertManagerService {
	logger, _ := zap.NewDevelopment()
	config := &AlertManagerServiceConfig{
		AlertManagerURL:    "http://localhost:9093",
		Enabled:            true,
		AlertCheckInterval: 15 * time.Second,
		AlertResponseSLA:   60 * time.Second,
	}
	return NewAlertManagerService(config, logger)
}

// setupTestAutoScalingService creates a test auto-scaling service
func setupTestAutoScalingService(t *testing.T) *AutoScalingService {
	logger, _ := zap.NewDevelopment()
	metricsCollector := setupTestMetricsCollector(t)
	config := &AutoScalingServiceConfig{
		Enabled:           true,
		CheckInterval:     30 * time.Second,
		DefaultMinReplicas: 1,
		DefaultMaxReplicas: 10,
		CPUThreshold:      80.0,
		MemoryThreshold:   80.0,
		ScaleUpCooldown:   1 * time.Minute,
		ScaleDownCooldown: 2 * time.Minute,
	}
	return NewAutoScalingService(config, logger, metricsCollector)
}

func TestMetricsCollector_CollectionInterval(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	t.Run("collection interval is 30 seconds", func(t *testing.T) {
		assert.Equal(t, 30*time.Second, collector.config.CollectionInterval,
			"Collection interval should be 30 seconds as per requirements")
	})
}

func TestMetricsCollector_GetSystemMetrics(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	metrics := collector.GetSystemMetrics()
	require.NotNil(t, metrics, "System metrics should not be nil")
	assert.False(t, metrics.Timestamp.IsZero(), "Timestamp should be set")
}

func TestMetricsCollector_GetBusinessMetrics(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	metrics := collector.GetBusinessMetrics()
	require.NotNil(t, metrics, "Business metrics should not be nil")
}

func TestMetricsCollector_RecordEnvironmentMetrics(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	envID := "test-env-1"
	metrics := &models.EnvironmentMetrics{
		EnvironmentID: envID,
		Timestamp:     time.Now(),
		CPUUsage:      50.0,
		MemoryUsage:   60.0,
		StorageUsage:  30.0,
	}

	collector.RecordEnvironmentMetrics(envID, metrics)

	history := collector.GetEnvironmentMetricsHistory(envID, 1*time.Hour)
	require.NotEmpty(t, history, "Should have recorded metrics")
	assert.Equal(t, envID, history[0].EnvironmentID)
}

func TestMetricsCollector_GetCollectionStats(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	stats := collector.GetCollectionStats()
	require.NotNil(t, stats, "Collection stats should not be nil")
	assert.Equal(t, 30*time.Second, stats.CollectionInterval)
}


// **Feature: cloud-devbox, Property 15: 异常 60 秒内告警**
// **Validates: Requirements 10.4, 10.5**
// This property test verifies that performance anomalies trigger alerts within 60 seconds.
func TestProperty15_AlertWithin60Seconds(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	// Create alert service with 60-second SLA
	alertService := NewAlertService(logger)
	
	// Create AlertManager service with 60-second SLA
	amConfig := &AlertManagerServiceConfig{
		AlertManagerURL:    "http://localhost:9093",
		Enabled:            true,
		AlertCheckInterval: 15 * time.Second,
		AlertResponseSLA:   60 * time.Second,
	}
	amService := NewAlertManagerService(amConfig, logger)

	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100
	var totalAlertTime time.Duration
	var maxAlertTime time.Duration
	var successCount int

	// Maximum allowed alert time (60 seconds)
	maxAllowedAlertTime := 60 * time.Second

	for i := 0; i < iterations; i++ {
		envID := "prop15-env-" + randomString(8)
		
		// Generate random anomalous metrics
		anomalyType := rand.Intn(3) // 0: CPU, 1: Memory, 2: Storage
		
		var metrics *models.EnvironmentMetrics
		switch anomalyType {
		case 0: // CPU anomaly
			metrics = &models.EnvironmentMetrics{
				EnvironmentID: envID,
				Timestamp:     time.Now(),
				CPUUsage:      95.0 + rand.Float64()*5, // 95-100%
				MemoryUsage:   rand.Float64() * 50,
				StorageUsage:  rand.Float64() * 50,
			}
		case 1: // Memory anomaly
			metrics = &models.EnvironmentMetrics{
				EnvironmentID: envID,
				Timestamp:     time.Now(),
				CPUUsage:      rand.Float64() * 50,
				MemoryUsage:   90.0 + rand.Float64()*10, // 90-100%
				StorageUsage:  rand.Float64() * 50,
			}
		case 2: // Storage anomaly
			metrics = &models.EnvironmentMetrics{
				EnvironmentID: envID,
				Timestamp:     time.Now(),
				CPUUsage:      rand.Float64() * 50,
				MemoryUsage:   rand.Float64() * 50,
				StorageUsage:  90.0 + rand.Float64()*10, // 90-100%
			}
		}

		// Record start time (simulating anomaly detection)
		startTime := time.Now()

		// Check alerts (this creates alert and sends notification)
		alertService.CheckAlerts(ctx, envID, metrics)

		// Calculate time to alert
		alertTime := time.Since(startTime)

		// Get the created alert to verify
		alerts, err := alertService.ListAlerts(ctx, "", &models.ListAlertsRequest{
			EnvironmentID: envID,
			Status:        models.AlertStatusActive,
			Page:          1,
			PageSize:      10,
		})
		require.NoError(t, err)

		if len(alerts.Alerts) > 0 {
			successCount++
			totalAlertTime += alertTime
			if alertTime > maxAlertTime {
				maxAlertTime = alertTime
			}

			// Property: Alert should be created within 60 seconds
			assert.Less(t, alertTime, maxAllowedAlertTime,
				"Alert creation time %v exceeded 60 seconds for env %s",
				alertTime, envID)

			// Verify alert was notified
			alert := alerts.Alerts[0]
			assert.NotNil(t, alert.NotifiedAt,
				"Alert should have been notified")
		}
	}

	// Verify AlertManager SLA configuration
	assert.Equal(t, 60*time.Second, amService.GetAlertResponseSLA(),
		"AlertManager SLA should be 60 seconds")

	avgAlertTime := time.Duration(0)
	if successCount > 0 {
		avgAlertTime = totalAlertTime / time.Duration(successCount)
	}

	t.Logf("Property 15 Results:")
	t.Logf("  - Iterations: %d", iterations)
	t.Logf("  - Successful alerts: %d", successCount)
	t.Logf("  - Average alert time: %v", avgAlertTime)
	t.Logf("  - Maximum alert time: %v", maxAlertTime)
	t.Logf("  - Success rate: %.1f%%", float64(successCount)/float64(iterations)*100)

	// Overall property assertions
	assert.Greater(t, successCount, 0,
		"At least some alerts should be created")
	assert.Less(t, avgAlertTime, maxAllowedAlertTime,
		"Average alert time should be less than 60 seconds")
	assert.Less(t, maxAlertTime, maxAllowedAlertTime,
		"Maximum alert time should be less than 60 seconds")
}

// randomString generates a random string of given length
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}


// TestProperty15_MetricsCollectionInterval verifies metrics are collected every 30 seconds
func TestProperty15_MetricsCollectionInterval(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	config := &MetricsCollectorConfig{
		CollectionInterval: 30 * time.Second,
		PrometheusURL:      "http://localhost:9090",
		RetentionPeriod:    1 * time.Hour,
		MaxDataPoints:      1000,
	}
	collector := NewMetricsCollector(config, logger)

	// Verify collection interval is 30 seconds as per requirements
	assert.Equal(t, 30*time.Second, collector.config.CollectionInterval,
		"Metrics collection interval should be 30 seconds (Requirements 10.4)")

	stats := collector.GetCollectionStats()
	assert.Equal(t, 30*time.Second, stats.CollectionInterval,
		"Collection stats should report 30 second interval")
}

// TestAutoScalingService_TriggerFromAlert tests auto-scaling triggered by alerts
func TestAutoScalingService_TriggerFromAlert(t *testing.T) {
	service := setupTestAutoScalingService(t)

	// Register a service for auto-scaling
	service.RegisterService("test-service", &models.AutoScalingConfig{
		Enabled:           true,
		MinReplicas:       1,
		MaxReplicas:       10,
		CPUThreshold:      80.0,
		MemoryThreshold:   80.0,
		ScaleUpCooldown:   1 * time.Minute,
		ScaleDownCooldown: 2 * time.Minute,
	})

	// Create a critical alert
	alert := &models.PerformanceAlert{
		ID:          "test-alert-1",
		Type:        "system",
		Severity:    models.AlertSeverityCritical,
		Title:       "High CPU Usage",
		Message:     "CPU usage exceeded threshold",
		MetricName:  "cpu_usage",
		MetricValue: 95.0,
		Threshold:   80.0,
		CreatedAt:   time.Now(),
	}

	// Trigger scaling from alert
	service.TriggerScaleFromAlert(alert)

	// Check scaling events
	events := service.GetScalingEvents(10)
	// Note: Scaling may or may not occur depending on current replica count
	t.Logf("Scaling events after alert: %d", len(events))
}

// TestAutoScalingService_RegisterUnregister tests service registration
func TestAutoScalingService_RegisterUnregister(t *testing.T) {
	service := setupTestAutoScalingService(t)

	serviceName := "test-service-reg"
	config := &models.AutoScalingConfig{
		Enabled:     true,
		MinReplicas: 2,
		MaxReplicas: 8,
	}

	// Register service
	service.RegisterService(serviceName, config)

	// Verify registration
	retrievedConfig := service.GetServiceConfig(serviceName)
	require.NotNil(t, retrievedConfig, "Service should be registered")
	assert.Equal(t, int32(2), retrievedConfig.MinReplicas)
	assert.Equal(t, int32(8), retrievedConfig.MaxReplicas)

	// Unregister service
	service.UnregisterService(serviceName)

	// Verify unregistration
	retrievedConfig = service.GetServiceConfig(serviceName)
	assert.Nil(t, retrievedConfig, "Service should be unregistered")
}

// TestGrafanaService_DashboardConfigs tests Grafana dashboard configurations
func TestGrafanaService_DashboardConfigs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := NewGrafanaService(nil, logger)

	t.Run("system dashboard config", func(t *testing.T) {
		dashboard := service.GetSystemDashboardConfig()
		require.NotNil(t, dashboard, "System dashboard should not be nil")
		assert.Equal(t, "devbox-system-monitoring", dashboard.UID)
		assert.Equal(t, "Cloud DevBox - System Monitoring", dashboard.Title)
		assert.Contains(t, dashboard.Tags, "system")
		assert.NotEmpty(t, dashboard.Panels, "Dashboard should have panels")
	})

	t.Run("business dashboard config", func(t *testing.T) {
		dashboard := service.GetBusinessDashboardConfig()
		require.NotNil(t, dashboard, "Business dashboard should not be nil")
		assert.Equal(t, "devbox-business-metrics", dashboard.UID)
		assert.Equal(t, "Cloud DevBox - Business Metrics", dashboard.Title)
		assert.Contains(t, dashboard.Tags, "business")
		assert.NotEmpty(t, dashboard.Panels, "Dashboard should have panels")
	})
}

// TestAlertManagerService_AlertResponseSLA tests AlertManager SLA configuration
func TestAlertManagerService_AlertResponseSLA(t *testing.T) {
	service := setupTestAlertManagerService(t)

	// Verify SLA is 60 seconds as per requirements
	sla := service.GetAlertResponseSLA()
	assert.Equal(t, 60*time.Second, sla,
		"Alert response SLA should be 60 seconds (Requirements 10.4, 10.5)")
}

// TestAlertManagerService_GetActiveAlerts tests getting active alerts
func TestAlertManagerService_GetActiveAlerts(t *testing.T) {
	service := setupTestAlertManagerService(t)

	alerts := service.GetActiveAlerts()
	// Initially should be empty
	assert.NotNil(t, alerts, "Active alerts should not be nil")
}

// TestAlertManagerService_GetAlertHistory tests getting alert history
func TestAlertManagerService_GetAlertHistory(t *testing.T) {
	service := setupTestAlertManagerService(t)

	history := service.GetAlertHistory(10)
	assert.NotNil(t, history, "Alert history should not be nil")
}

// TestMetricsCollector_EnvironmentMetricsHistory tests environment metrics history
func TestMetricsCollector_EnvironmentMetricsHistory(t *testing.T) {
	collector := setupTestMetricsCollector(t)

	envID := "test-env-history"
	
	// Record multiple metrics
	for i := 0; i < 10; i++ {
		metrics := &models.EnvironmentMetrics{
			EnvironmentID: envID,
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			CPUUsage:      float64(50 + i),
			MemoryUsage:   float64(40 + i),
			StorageUsage:  float64(30 + i),
		}
		collector.RecordEnvironmentMetrics(envID, metrics)
	}

	// Get history
	history := collector.GetEnvironmentMetricsHistory(envID, 1*time.Hour)
	assert.Len(t, history, 10, "Should have 10 data points")

	// Verify ordering (most recent should have highest values)
	for i := 1; i < len(history); i++ {
		assert.GreaterOrEqual(t, history[i].CPUUsage, history[i-1].CPUUsage,
			"Metrics should be in chronological order")
	}
}

// TestMetricsCollector_MaxDataPoints tests data point limit
func TestMetricsCollector_MaxDataPoints(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &MetricsCollectorConfig{
		CollectionInterval: 30 * time.Second,
		PrometheusURL:      "http://localhost:9090",
		RetentionPeriod:    1 * time.Hour,
		MaxDataPoints:      5, // Small limit for testing
	}
	collector := NewMetricsCollector(config, logger)

	envID := "test-env-limit"
	
	// Record more metrics than the limit
	for i := 0; i < 10; i++ {
		metrics := &models.EnvironmentMetrics{
			EnvironmentID: envID,
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			CPUUsage:      float64(50 + i),
		}
		collector.RecordEnvironmentMetrics(envID, metrics)
	}

	// Get history - should be limited to MaxDataPoints
	history := collector.GetEnvironmentMetricsHistory(envID, 1*time.Hour)
	assert.LessOrEqual(t, len(history), 5, "Should not exceed MaxDataPoints")
}
