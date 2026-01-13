// Package services provides business logic for the container service.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// MonitoringServiceConfig holds configuration for the monitoring service
type MonitoringServiceConfig struct {
	PrometheusURL      string
	AlertCheckInterval time.Duration
	MetricsRetention   time.Duration
}

// DefaultMonitoringServiceConfig returns default configuration
func DefaultMonitoringServiceConfig() *MonitoringServiceConfig {
	return &MonitoringServiceConfig{
		PrometheusURL:      "http://prometheus:9090",
		AlertCheckInterval: 30 * time.Second,
		MetricsRetention:   7 * 24 * time.Hour,
	}
}

// MonitoringService handles monitoring and alerting operations
type MonitoringService struct {
	config       *MonitoringServiceConfig
	logger       *zap.Logger
	httpClient   *http.Client
	envService   *EnvironmentService
	alertService *AlertService

	// In-memory cache for metrics (in production, use time-series DB)
	metricsCache map[string][]models.EnvironmentMetrics
	metricsMu    sync.RWMutex

	// Stop channel for background workers
	stopCh chan struct{}
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(
	config *MonitoringServiceConfig,
	logger *zap.Logger,
	envService *EnvironmentService,
) *MonitoringService {
	if config == nil {
		config = DefaultMonitoringServiceConfig()
	}

	alertService := NewAlertService(logger)

	return &MonitoringService{
		config:       config,
		logger:       logger.Named("monitoring-service"),
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		envService:   envService,
		alertService: alertService,
		metricsCache: make(map[string][]models.EnvironmentMetrics),
		stopCh:       make(chan struct{}),
	}
}

// Start starts the monitoring service background workers
func (s *MonitoringService) Start(ctx context.Context) {
	go s.collectMetricsLoop(ctx)
	go s.checkAlertsLoop(ctx)
	s.logger.Info("Monitoring service started")
}

// Stop stops the monitoring service
func (s *MonitoringService) Stop() {
	close(s.stopCh)
	s.logger.Info("Monitoring service stopped")
}

// GetMetrics returns current metrics for an environment
func (s *MonitoringService) GetMetrics(ctx context.Context, environmentID string) (*models.EnvironmentMetrics, error) {
	// Try to get from Prometheus first
	metrics, err := s.fetchMetricsFromPrometheus(ctx, environmentID)
	if err != nil {
		s.logger.Debug("Failed to fetch from Prometheus, using simulated data", zap.Error(err))
		// Fall back to simulated metrics for development
		metrics = s.generateSimulatedMetrics(environmentID)
	}

	// Store in cache
	s.metricsMu.Lock()
	s.metricsCache[environmentID] = append(s.metricsCache[environmentID], *metrics)
	// Keep only last 1000 data points per environment
	if len(s.metricsCache[environmentID]) > 1000 {
		s.metricsCache[environmentID] = s.metricsCache[environmentID][1:]
	}
	s.metricsMu.Unlock()

	return metrics, nil
}

// GetMetricsHistory returns historical metrics for an environment
func (s *MonitoringService) GetMetricsHistory(ctx context.Context, req *models.GetMetricsHistoryRequest) (*models.MetricsHistory, error) {
	period := req.Period
	if period == "" {
		period = "1h"
	}
	resolution := req.Resolution
	if resolution == "" {
		resolution = "1m"
	}

	// Parse period
	duration, err := parseDuration(period)
	if err != nil {
		return nil, fmt.Errorf("invalid period: %w", err)
	}

	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// Try to get from Prometheus
	history, err := s.fetchHistoryFromPrometheus(ctx, req.EnvironmentID, startTime, endTime, resolution)
	if err != nil {
		s.logger.Debug("Failed to fetch history from Prometheus, using cached data", zap.Error(err))
		// Fall back to cached data
		history = s.getHistoryFromCache(req.EnvironmentID, startTime, endTime)
	}

	history.Period = period
	history.Resolution = resolution
	return history, nil
}

// fetchMetricsFromPrometheus fetches current metrics from Prometheus
func (s *MonitoringService) fetchMetricsFromPrometheus(ctx context.Context, environmentID string) (*models.EnvironmentMetrics, error) {
	metrics := &models.EnvironmentMetrics{
		EnvironmentID: environmentID,
		Timestamp:     time.Now(),
	}

	// Query CPU usage
	cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=~"%s.*"}[5m])) * 100`, environmentID)
	cpuResult, err := s.queryPrometheus(ctx, cpuQuery)
	if err == nil && len(cpuResult) > 0 {
		metrics.CPUUsage = cpuResult[0]
	}

	// Query memory usage
	memQuery := fmt.Sprintf(`sum(container_memory_usage_bytes{pod=~"%s.*"})`, environmentID)
	memResult, err := s.queryPrometheus(ctx, memQuery)
	if err == nil && len(memResult) > 0 {
		metrics.MemoryUsed = int64(memResult[0])
	}

	// Query memory limit
	memLimitQuery := fmt.Sprintf(`sum(container_spec_memory_limit_bytes{pod=~"%s.*"})`, environmentID)
	memLimitResult, err := s.queryPrometheus(ctx, memLimitQuery)
	if err == nil && len(memLimitResult) > 0 {
		metrics.MemoryLimit = int64(memLimitResult[0])
		if metrics.MemoryLimit > 0 {
			metrics.MemoryUsage = float64(metrics.MemoryUsed) / float64(metrics.MemoryLimit) * 100
		}
	}

	// Query network I/O
	rxQuery := fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{pod=~"%s.*"}[5m]))`, environmentID)
	rxResult, err := s.queryPrometheus(ctx, rxQuery)
	if err == nil && len(rxResult) > 0 {
		metrics.NetworkRxRate = rxResult[0]
	}

	txQuery := fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{pod=~"%s.*"}[5m]))`, environmentID)
	txResult, err := s.queryPrometheus(ctx, txQuery)
	if err == nil && len(txResult) > 0 {
		metrics.NetworkTxRate = txResult[0]
	}

	return metrics, nil
}

// queryPrometheus executes a PromQL query
func (s *MonitoringService) queryPrometheus(ctx context.Context, query string) ([]float64, error) {
	u, err := url.Parse(s.config.PrometheusURL + "/api/v1/query")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result models.PrometheusQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", result.Status)
	}

	var values []float64
	for _, r := range result.Data.Result {
		if len(r.Value) >= 2 {
			if v, ok := r.Value[1].(string); ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					values = append(values, f)
				}
			}
		}
	}

	return values, nil
}

// fetchHistoryFromPrometheus fetches historical metrics from Prometheus
func (s *MonitoringService) fetchHistoryFromPrometheus(ctx context.Context, environmentID string, start, end time.Time, step string) (*models.MetricsHistory, error) {
	history := &models.MetricsHistory{
		EnvironmentID: environmentID,
		StartTime:     start,
		EndTime:       end,
	}

	// Query CPU history
	cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=~"%s.*"}[5m])) * 100`, environmentID)
	cpuData, err := s.queryPrometheusRange(ctx, cpuQuery, start, end, step)
	if err == nil {
		history.CPU = cpuData
	}

	// Query memory history
	memQuery := fmt.Sprintf(`sum(container_memory_usage_bytes{pod=~"%s.*"}) / sum(container_spec_memory_limit_bytes{pod=~"%s.*"}) * 100`, environmentID, environmentID)
	memData, err := s.queryPrometheusRange(ctx, memQuery, start, end, step)
	if err == nil {
		history.Memory = memData
	}

	// Query network history
	rxQuery := fmt.Sprintf(`sum(rate(container_network_receive_bytes_total{pod=~"%s.*"}[5m]))`, environmentID)
	rxData, err := s.queryPrometheusRange(ctx, rxQuery, start, end, step)
	if err == nil {
		history.NetworkRx = rxData
	}

	txQuery := fmt.Sprintf(`sum(rate(container_network_transmit_bytes_total{pod=~"%s.*"}[5m]))`, environmentID)
	txData, err := s.queryPrometheusRange(ctx, txQuery, start, end, step)
	if err == nil {
		history.NetworkTx = txData
	}

	return history, nil
}

// queryPrometheusRange executes a PromQL range query
func (s *MonitoringService) queryPrometheusRange(ctx context.Context, query string, start, end time.Time, step string) ([]models.MetricDataPoint, error) {
	u, err := url.Parse(s.config.PrometheusURL + "/api/v1/query_range")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", step)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result models.PrometheusQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var dataPoints []models.MetricDataPoint
	for _, r := range result.Data.Result {
		for _, v := range r.Values {
			if len(v) >= 2 {
				ts, _ := v[0].(float64)
				val, _ := v[1].(string)
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					dataPoints = append(dataPoints, models.MetricDataPoint{
						Timestamp: time.Unix(int64(ts), 0),
						Value:     f,
					})
				}
			}
		}
	}

	return dataPoints, nil
}

// generateSimulatedMetrics generates simulated metrics for development
func (s *MonitoringService) generateSimulatedMetrics(environmentID string) *models.EnvironmentMetrics {
	// Generate realistic-looking metrics with some randomness
	baseTime := time.Now()

	// Use environment ID hash for consistent base values
	hash := 0
	for _, c := range environmentID {
		hash += int(c)
	}

	baseCPU := 30.0 + float64(hash%40)
	baseMemory := 40.0 + float64(hash%30)
	baseStorage := 20.0 + float64(hash%50)

	return &models.EnvironmentMetrics{
		EnvironmentID:  environmentID,
		Timestamp:      baseTime,
		CPUUsage:       baseCPU + (rand.Float64()-0.5)*20,
		CPUCores:       1.0 + rand.Float64()*0.5,
		CPULimit:       2.0,
		MemoryUsage:    baseMemory + (rand.Float64()-0.5)*15,
		MemoryUsed:     int64((baseMemory / 100) * 2 * 1024 * 1024 * 1024),
		MemoryLimit:    2 * 1024 * 1024 * 1024, // 2GB
		StorageUsage:   baseStorage + (rand.Float64()-0.5)*10,
		StorageUsed:    int64((baseStorage / 100) * 10 * 1024 * 1024 * 1024),
		StorageLimit:   10 * 1024 * 1024 * 1024, // 10GB
		NetworkRxBytes: int64(rand.Float64() * 100 * 1024 * 1024),
		NetworkTxBytes: int64(rand.Float64() * 50 * 1024 * 1024),
		NetworkRxRate:  rand.Float64() * 10 * 1024 * 1024,
		NetworkTxRate:  rand.Float64() * 5 * 1024 * 1024,
	}
}

// getHistoryFromCache returns historical metrics from cache
func (s *MonitoringService) getHistoryFromCache(environmentID string, start, end time.Time) *models.MetricsHistory {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	history := &models.MetricsHistory{
		EnvironmentID: environmentID,
		StartTime:     start,
		EndTime:       end,
	}

	cached, ok := s.metricsCache[environmentID]
	if !ok {
		// Generate simulated history
		return s.generateSimulatedHistory(environmentID, start, end)
	}

	for _, m := range cached {
		if m.Timestamp.After(start) && m.Timestamp.Before(end) {
			history.CPU = append(history.CPU, models.MetricDataPoint{Timestamp: m.Timestamp, Value: m.CPUUsage})
			history.Memory = append(history.Memory, models.MetricDataPoint{Timestamp: m.Timestamp, Value: m.MemoryUsage})
			history.Storage = append(history.Storage, models.MetricDataPoint{Timestamp: m.Timestamp, Value: m.StorageUsage})
			history.NetworkRx = append(history.NetworkRx, models.MetricDataPoint{Timestamp: m.Timestamp, Value: m.NetworkRxRate})
			history.NetworkTx = append(history.NetworkTx, models.MetricDataPoint{Timestamp: m.Timestamp, Value: m.NetworkTxRate})
		}
	}

	return history
}

// generateSimulatedHistory generates simulated historical metrics
func (s *MonitoringService) generateSimulatedHistory(environmentID string, start, end time.Time) *models.MetricsHistory {
	history := &models.MetricsHistory{
		EnvironmentID: environmentID,
		StartTime:     start,
		EndTime:       end,
	}

	// Generate data points every minute
	interval := time.Minute
	current := start

	// Use environment ID hash for consistent base values
	hash := 0
	for _, c := range environmentID {
		hash += int(c)
	}

	baseCPU := 30.0 + float64(hash%40)
	baseMemory := 40.0 + float64(hash%30)
	baseStorage := 20.0 + float64(hash%50)

	for current.Before(end) {
		// Add some time-based variation
		hourFactor := float64(current.Hour()) / 24.0
		variation := (rand.Float64() - 0.5) * 20

		history.CPU = append(history.CPU, models.MetricDataPoint{
			Timestamp: current,
			Value:     baseCPU + hourFactor*20 + variation,
		})
		history.Memory = append(history.Memory, models.MetricDataPoint{
			Timestamp: current,
			Value:     baseMemory + hourFactor*10 + variation*0.5,
		})
		history.Storage = append(history.Storage, models.MetricDataPoint{
			Timestamp: current,
			Value:     baseStorage + float64(current.Sub(start).Hours())*0.1,
		})
		history.NetworkRx = append(history.NetworkRx, models.MetricDataPoint{
			Timestamp: current,
			Value:     rand.Float64() * 10 * 1024 * 1024,
		})
		history.NetworkTx = append(history.NetworkTx, models.MetricDataPoint{
			Timestamp: current,
			Value:     rand.Float64() * 5 * 1024 * 1024,
		})

		current = current.Add(interval)
	}

	return history
}

// collectMetricsLoop periodically collects metrics for all running environments
func (s *MonitoringService) collectMetricsLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.collectAllMetrics(ctx)
		}
	}
}

// collectAllMetrics collects metrics for all running environments
func (s *MonitoringService) collectAllMetrics(ctx context.Context) {
	if s.envService == nil {
		return
	}

	// Get all running environments
	resp, err := s.envService.List(ctx, &models.ListEnvironmentsRequest{
		Phase:    models.EnvironmentPhaseRunning,
		PageSize: 1000,
	})
	if err != nil {
		s.logger.Error("Failed to list environments", zap.Error(err))
		return
	}

	for _, env := range resp.Environments {
		if _, err := s.GetMetrics(ctx, env.ID); err != nil {
			s.logger.Debug("Failed to collect metrics", zap.String("envId", env.ID), zap.Error(err))
		}
	}
}

// checkAlertsLoop periodically checks for alert conditions
func (s *MonitoringService) checkAlertsLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAllAlerts(ctx)
		}
	}
}

// checkAllAlerts checks alert conditions for all environments
func (s *MonitoringService) checkAllAlerts(ctx context.Context) {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	for envID, metrics := range s.metricsCache {
		if len(metrics) == 0 {
			continue
		}
		latestMetrics := metrics[len(metrics)-1]
		s.alertService.CheckAlerts(ctx, envID, &latestMetrics)
	}
}

// GetAlertService returns the alert service
func (s *MonitoringService) GetAlertService() *AlertService {
	return s.alertService
}

// parseDuration parses a duration string like "1h", "24h", "7d"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	value, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'm':
		return time.Duration(value) * time.Minute, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c", unit)
	}
}
