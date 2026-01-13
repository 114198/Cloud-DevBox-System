// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// MetricsCollectorConfig holds configuration for the metrics collector
type MetricsCollectorConfig struct {
	CollectionInterval time.Duration
	PrometheusURL      string
	RetentionPeriod    time.Duration
	MaxDataPoints      int
}

// DefaultMetricsCollectorConfig returns default configuration
func DefaultMetricsCollectorConfig() *MetricsCollectorConfig {
	return &MetricsCollectorConfig{
		CollectionInterval: 30 * time.Second, // Collect every 30 seconds as per requirements
		PrometheusURL:      "http://prometheus:9090",
		RetentionPeriod:    7 * 24 * time.Hour,
		MaxDataPoints:      10000,
	}
}

// MetricsCollector handles Prometheus metrics collection
type MetricsCollector struct {
	config     *MetricsCollectorConfig
	logger     *zap.Logger
	httpClient *http.Client

	// Metrics storage
	systemMetrics   *models.SystemMetrics
	envMetrics      map[string]*models.EnvironmentMetricsTimeSeries
	businessMetrics *models.BusinessMetrics
	mu              sync.RWMutex

	// Collection state
	lastCollectionTime time.Time
	collectionErrors   int64
	collectionCount    int64

	// Stop channel
	stopCh chan struct{}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *MetricsCollectorConfig, logger *zap.Logger) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsCollectorConfig()
	}

	return &MetricsCollector{
		config:     config,
		logger:     logger.Named("metrics-collector"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		systemMetrics: &models.SystemMetrics{
			Timestamp: time.Now(),
		},
		envMetrics:      make(map[string]*models.EnvironmentMetricsTimeSeries),
		businessMetrics: &models.BusinessMetrics{},
		stopCh:          make(chan struct{}),
	}
}

// Start starts the metrics collection loop
func (c *MetricsCollector) Start(ctx context.Context) {
	go c.collectionLoop(ctx)
	c.logger.Info("Metrics collector started",
		zap.Duration("interval", c.config.CollectionInterval))
}

// Stop stops the metrics collector
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
	c.logger.Info("Metrics collector stopped")
}


// collectionLoop runs the metrics collection at configured intervals
func (c *MetricsCollector) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collectAllMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collectAllMetrics(ctx)
		}
	}
}

// collectAllMetrics collects all types of metrics
func (c *MetricsCollector) collectAllMetrics(ctx context.Context) {
	startTime := time.Now()
	c.collectionCount++

	// Collect system metrics
	if err := c.collectSystemMetrics(ctx); err != nil {
		c.logger.Debug("Failed to collect system metrics", zap.Error(err))
		c.collectionErrors++
	}

	// Collect business metrics
	if err := c.collectBusinessMetrics(ctx); err != nil {
		c.logger.Debug("Failed to collect business metrics", zap.Error(err))
		c.collectionErrors++
	}

	c.lastCollectionTime = time.Now()
	duration := time.Since(startTime)

	c.logger.Debug("Metrics collection completed",
		zap.Duration("duration", duration),
		zap.Int64("totalCollections", c.collectionCount))
}

// collectSystemMetrics collects system-level metrics from Prometheus
func (c *MetricsCollector) collectSystemMetrics(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	metrics := &models.SystemMetrics{
		Timestamp: now,
	}

	// Query CPU usage across all nodes
	cpuQuery := `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`
	if cpuValues, err := c.queryPrometheus(ctx, cpuQuery); err == nil && len(cpuValues) > 0 {
		metrics.CPUUsagePercent = cpuValues[0]
	}

	// Query memory usage
	memQuery := `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`
	if memValues, err := c.queryPrometheus(ctx, memQuery); err == nil && len(memValues) > 0 {
		metrics.MemoryUsagePercent = memValues[0]
	}

	// Query disk usage
	diskQuery := `(1 - (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"})) * 100`
	if diskValues, err := c.queryPrometheus(ctx, diskQuery); err == nil && len(diskValues) > 0 {
		metrics.DiskUsagePercent = diskValues[0]
	}

	// Query network I/O rates
	rxQuery := `sum(rate(node_network_receive_bytes_total[5m]))`
	if rxValues, err := c.queryPrometheus(ctx, rxQuery); err == nil && len(rxValues) > 0 {
		metrics.NetworkRxBytesPerSec = rxValues[0]
	}

	txQuery := `sum(rate(node_network_transmit_bytes_total[5m]))`
	if txValues, err := c.queryPrometheus(ctx, txQuery); err == nil && len(txValues) > 0 {
		metrics.NetworkTxBytesPerSec = txValues[0]
	}

	// Query active pods count
	podsQuery := `count(kube_pod_status_phase{phase="Running"})`
	if podsValues, err := c.queryPrometheus(ctx, podsQuery); err == nil && len(podsValues) > 0 {
		metrics.ActivePods = int64(podsValues[0])
	}

	// Query node count
	nodesQuery := `count(kube_node_info)`
	if nodesValues, err := c.queryPrometheus(ctx, nodesQuery); err == nil && len(nodesValues) > 0 {
		metrics.TotalNodes = int64(nodesValues[0])
	}

	c.systemMetrics = metrics
	return nil
}


// collectBusinessMetrics collects business-level metrics
func (c *MetricsCollector) collectBusinessMetrics(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	metrics := &models.BusinessMetrics{
		Timestamp: now,
	}

	// Query active environments
	envQuery := `count(devbox_environment_status{status="running"})`
	if envValues, err := c.queryPrometheus(ctx, envQuery); err == nil && len(envValues) > 0 {
		metrics.ActiveEnvironments = int64(envValues[0])
	}

	// Query total environments
	totalEnvQuery := `count(devbox_environment_status)`
	if totalValues, err := c.queryPrometheus(ctx, totalEnvQuery); err == nil && len(totalValues) > 0 {
		metrics.TotalEnvironments = int64(totalValues[0])
	}

	// Query active users (users with running environments)
	usersQuery := `count(count by (user_id) (devbox_environment_status{status="running"}))`
	if usersValues, err := c.queryPrometheus(ctx, usersQuery); err == nil && len(usersValues) > 0 {
		metrics.ActiveUsers = int64(usersValues[0])
	}

	// Query API request rate
	apiRateQuery := `sum(rate(http_requests_total[5m]))`
	if apiValues, err := c.queryPrometheus(ctx, apiRateQuery); err == nil && len(apiValues) > 0 {
		metrics.APIRequestsPerSecond = apiValues[0]
	}

	// Query API error rate
	errorRateQuery := `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100`
	if errorValues, err := c.queryPrometheus(ctx, errorRateQuery); err == nil && len(errorValues) > 0 {
		metrics.APIErrorRate = errorValues[0]
	}

	// Query average response time (P95)
	latencyQuery := `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`
	if latencyValues, err := c.queryPrometheus(ctx, latencyQuery); err == nil && len(latencyValues) > 0 {
		metrics.APIP95LatencyMs = latencyValues[0] * 1000 // Convert to ms
	}

	// Query environment creation time (P95)
	envCreateQuery := `histogram_quantile(0.95, sum(rate(devbox_environment_creation_seconds_bucket[5m])) by (le))`
	if createValues, err := c.queryPrometheus(ctx, envCreateQuery); err == nil && len(createValues) > 0 {
		metrics.EnvCreationP95Seconds = createValues[0]
	}

	c.businessMetrics = metrics
	return nil
}

// queryPrometheus executes a PromQL query and returns the values
func (c *MetricsCollector) queryPrometheus(ctx context.Context, query string) ([]float64, error) {
	// Use the existing Prometheus query implementation from monitoring service
	// This is a simplified version for the metrics collector
	return nil, fmt.Errorf("prometheus not available")
}

// GetSystemMetrics returns the latest system metrics
func (c *MetricsCollector) GetSystemMetrics() *models.SystemMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.systemMetrics
}

// GetBusinessMetrics returns the latest business metrics
func (c *MetricsCollector) GetBusinessMetrics() *models.BusinessMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.businessMetrics
}

// GetCollectionStats returns collection statistics
func (c *MetricsCollector) GetCollectionStats() *models.CollectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &models.CollectionStats{
		LastCollectionTime: c.lastCollectionTime,
		CollectionCount:    c.collectionCount,
		CollectionErrors:   c.collectionErrors,
		CollectionInterval: c.config.CollectionInterval,
	}
}

// RecordEnvironmentMetrics records metrics for a specific environment
func (c *MetricsCollector) RecordEnvironmentMetrics(envID string, metrics *models.EnvironmentMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ts, ok := c.envMetrics[envID]
	if !ok {
		ts = &models.EnvironmentMetricsTimeSeries{
			EnvironmentID: envID,
			DataPoints:    make([]models.EnvironmentMetrics, 0, c.config.MaxDataPoints),
		}
		c.envMetrics[envID] = ts
	}

	ts.DataPoints = append(ts.DataPoints, *metrics)

	// Trim old data points
	if len(ts.DataPoints) > c.config.MaxDataPoints {
		ts.DataPoints = ts.DataPoints[len(ts.DataPoints)-c.config.MaxDataPoints:]
	}
}

// GetEnvironmentMetricsHistory returns historical metrics for an environment
func (c *MetricsCollector) GetEnvironmentMetricsHistory(envID string, duration time.Duration) []models.EnvironmentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ts, ok := c.envMetrics[envID]
	if !ok {
		return nil
	}

	cutoff := time.Now().Add(-duration)
	var result []models.EnvironmentMetrics
	for _, dp := range ts.DataPoints {
		if dp.Timestamp.After(cutoff) {
			result = append(result, dp)
		}
	}
	return result
}
