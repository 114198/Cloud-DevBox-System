// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AutoScalingServiceConfig holds configuration for auto-scaling
type AutoScalingServiceConfig struct {
	Enabled           bool
	CheckInterval     time.Duration
	DefaultMinReplicas int32
	DefaultMaxReplicas int32
	CPUThreshold      float64
	MemoryThreshold   float64
	ScaleUpCooldown   time.Duration
	ScaleDownCooldown time.Duration
}

// DefaultAutoScalingServiceConfig returns default configuration
func DefaultAutoScalingServiceConfig() *AutoScalingServiceConfig {
	return &AutoScalingServiceConfig{
		Enabled:           true,
		CheckInterval:     30 * time.Second,
		DefaultMinReplicas: 1,
		DefaultMaxReplicas: 10,
		CPUThreshold:      80.0,
		MemoryThreshold:   80.0,
		ScaleUpCooldown:   3 * time.Minute,
		ScaleDownCooldown: 5 * time.Minute,
	}
}

// AutoScalingService handles automatic scaling based on metrics
type AutoScalingService struct {
	config           *AutoScalingServiceConfig
	logger           *zap.Logger
	metricsCollector *MetricsCollector

	// Scaling state
	serviceConfigs   map[string]*models.AutoScalingConfig
	lastScaleUp      map[string]time.Time
	lastScaleDown    map[string]time.Time
	scalingEvents    []models.ScalingEvent
	mu               sync.RWMutex

	// Stop channel
	stopCh chan struct{}
}

// NewAutoScalingService creates a new auto-scaling service
func NewAutoScalingService(
	config *AutoScalingServiceConfig,
	logger *zap.Logger,
	metricsCollector *MetricsCollector,
) *AutoScalingService {
	if config == nil {
		config = DefaultAutoScalingServiceConfig()
	}

	return &AutoScalingService{
		config:           config,
		logger:           logger.Named("autoscaling-service"),
		metricsCollector: metricsCollector,
		serviceConfigs:   make(map[string]*models.AutoScalingConfig),
		lastScaleUp:      make(map[string]time.Time),
		lastScaleDown:    make(map[string]time.Time),
		scalingEvents:    make([]models.ScalingEvent, 0),
		stopCh:           make(chan struct{}),
	}
}

// Start starts the auto-scaling service
func (s *AutoScalingService) Start(ctx context.Context) {
	if !s.config.Enabled {
		s.logger.Info("Auto-scaling is disabled")
		return
	}

	go s.scalingLoop(ctx)
	s.logger.Info("Auto-scaling service started",
		zap.Duration("checkInterval", s.config.CheckInterval))
}

// Stop stops the auto-scaling service
func (s *AutoScalingService) Stop() {
	close(s.stopCh)
	s.logger.Info("Auto-scaling service stopped")
}


// scalingLoop runs the auto-scaling check loop
func (s *AutoScalingService) scalingLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndScale(ctx)
		}
	}
}

// checkAndScale checks metrics and triggers scaling if needed
func (s *AutoScalingService) checkAndScale(ctx context.Context) {
	if s.metricsCollector == nil {
		return
	}

	systemMetrics := s.metricsCollector.GetSystemMetrics()
	if systemMetrics == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for serviceName, config := range s.serviceConfigs {
		if !config.Enabled {
			continue
		}

		// Check CPU threshold
		if systemMetrics.CPUUsagePercent > config.CPUThreshold {
			s.tryScaleUp(ctx, serviceName, "cpu", systemMetrics.CPUUsagePercent, config.CPUThreshold)
		} else if systemMetrics.CPUUsagePercent < config.CPUThreshold*0.5 {
			s.tryScaleDown(ctx, serviceName, "cpu", systemMetrics.CPUUsagePercent, config.CPUThreshold)
		}

		// Check memory threshold
		if systemMetrics.MemoryUsagePercent > config.MemoryThreshold {
			s.tryScaleUp(ctx, serviceName, "memory", systemMetrics.MemoryUsagePercent, config.MemoryThreshold)
		} else if systemMetrics.MemoryUsagePercent < config.MemoryThreshold*0.5 {
			s.tryScaleDown(ctx, serviceName, "memory", systemMetrics.MemoryUsagePercent, config.MemoryThreshold)
		}
	}
}

// tryScaleUp attempts to scale up a service
func (s *AutoScalingService) tryScaleUp(ctx context.Context, serviceName, metricName string, value, threshold float64) {
	config := s.serviceConfigs[serviceName]
	if config == nil {
		return
	}

	// Check cooldown
	if lastScale, ok := s.lastScaleUp[serviceName]; ok {
		if time.Since(lastScale) < config.ScaleUpCooldown {
			return
		}
	}

	// Get current replicas (simulated)
	currentReplicas := s.getCurrentReplicas(serviceName)
	if currentReplicas >= config.MaxReplicas {
		return
	}

	newReplicas := currentReplicas + 1
	if newReplicas > config.MaxReplicas {
		newReplicas = config.MaxReplicas
	}

	// Execute scale up
	event := s.executeScaling(ctx, serviceName, "scale_up", currentReplicas, newReplicas, metricName, value, threshold)
	s.scalingEvents = append(s.scalingEvents, event)
	s.lastScaleUp[serviceName] = time.Now()

	s.logger.Info("Scaled up service",
		zap.String("service", serviceName),
		zap.Int32("from", currentReplicas),
		zap.Int32("to", newReplicas),
		zap.String("reason", metricName),
		zap.Float64("value", value))
}

// tryScaleDown attempts to scale down a service
func (s *AutoScalingService) tryScaleDown(ctx context.Context, serviceName, metricName string, value, threshold float64) {
	config := s.serviceConfigs[serviceName]
	if config == nil {
		return
	}

	// Check cooldown
	if lastScale, ok := s.lastScaleDown[serviceName]; ok {
		if time.Since(lastScale) < config.ScaleDownCooldown {
			return
		}
	}

	// Get current replicas (simulated)
	currentReplicas := s.getCurrentReplicas(serviceName)
	if currentReplicas <= config.MinReplicas {
		return
	}

	newReplicas := currentReplicas - 1
	if newReplicas < config.MinReplicas {
		newReplicas = config.MinReplicas
	}

	// Execute scale down
	event := s.executeScaling(ctx, serviceName, "scale_down", currentReplicas, newReplicas, metricName, value, threshold)
	s.scalingEvents = append(s.scalingEvents, event)
	s.lastScaleDown[serviceName] = time.Now()

	s.logger.Info("Scaled down service",
		zap.String("service", serviceName),
		zap.Int32("from", currentReplicas),
		zap.Int32("to", newReplicas),
		zap.String("reason", metricName),
		zap.Float64("value", value))
}


// executeScaling executes the scaling operation
func (s *AutoScalingService) executeScaling(ctx context.Context, serviceName, scaleType string, from, to int32, metricName string, value, threshold float64) models.ScalingEvent {
	event := models.ScalingEvent{
		ID:           uuid.New().String(),
		Type:         scaleType,
		ServiceName:  serviceName,
		FromReplicas: from,
		ToReplicas:   to,
		Reason:       fmt.Sprintf("%s threshold exceeded", metricName),
		MetricName:   metricName,
		MetricValue:  value,
		Threshold:    threshold,
		Timestamp:    time.Now(),
		Success:      true,
	}

	// In production, this would call Kubernetes API to scale the deployment
	// For now, we simulate the scaling operation
	s.logger.Info("Executing scaling operation",
		zap.String("service", serviceName),
		zap.String("type", scaleType),
		zap.Int32("from", from),
		zap.Int32("to", to))

	return event
}

// getCurrentReplicas returns the current replica count for a service
func (s *AutoScalingService) getCurrentReplicas(serviceName string) int32 {
	// In production, this would query Kubernetes API
	// For now, return a default value
	return 2
}

// RegisterService registers a service for auto-scaling
func (s *AutoScalingService) RegisterService(serviceName string, config *models.AutoScalingConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		config = &models.AutoScalingConfig{
			Enabled:           true,
			MinReplicas:       s.config.DefaultMinReplicas,
			MaxReplicas:       s.config.DefaultMaxReplicas,
			CPUThreshold:      s.config.CPUThreshold,
			MemoryThreshold:   s.config.MemoryThreshold,
			ScaleUpCooldown:   s.config.ScaleUpCooldown,
			ScaleDownCooldown: s.config.ScaleDownCooldown,
		}
	}

	s.serviceConfigs[serviceName] = config
	s.logger.Info("Registered service for auto-scaling",
		zap.String("service", serviceName),
		zap.Int32("minReplicas", config.MinReplicas),
		zap.Int32("maxReplicas", config.MaxReplicas))
}

// UnregisterService removes a service from auto-scaling
func (s *AutoScalingService) UnregisterService(serviceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.serviceConfigs, serviceName)
	delete(s.lastScaleUp, serviceName)
	delete(s.lastScaleDown, serviceName)

	s.logger.Info("Unregistered service from auto-scaling",
		zap.String("service", serviceName))
}

// GetScalingEvents returns recent scaling events
func (s *AutoScalingService) GetScalingEvents(limit int) []models.ScalingEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.scalingEvents) {
		limit = len(s.scalingEvents)
	}

	start := len(s.scalingEvents) - limit
	if start < 0 {
		start = 0
	}

	result := make([]models.ScalingEvent, limit)
	copy(result, s.scalingEvents[start:])
	return result
}

// GetServiceConfig returns the auto-scaling config for a service
func (s *AutoScalingService) GetServiceConfig(serviceName string) *models.AutoScalingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serviceConfigs[serviceName]
}

// UpdateServiceConfig updates the auto-scaling config for a service
func (s *AutoScalingService) UpdateServiceConfig(serviceName string, config *models.AutoScalingConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.serviceConfigs[serviceName]; !exists {
		return fmt.Errorf("service not registered: %s", serviceName)
	}

	s.serviceConfigs[serviceName] = config
	return nil
}

// TriggerScaleFromAlert triggers scaling based on an alert
func (s *AutoScalingService) TriggerScaleFromAlert(alert *models.PerformanceAlert) {
	if alert.Severity != models.AlertSeverityCritical {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Scale up all registered services on critical alerts
	for serviceName, config := range s.serviceConfigs {
		if !config.Enabled {
			continue
		}

		currentReplicas := s.getCurrentReplicas(serviceName)
		if currentReplicas < config.MaxReplicas {
			newReplicas := currentReplicas + 1
			event := s.executeScaling(context.Background(), serviceName, "scale_up", currentReplicas, newReplicas, alert.MetricName, alert.MetricValue, alert.Threshold)
			s.scalingEvents = append(s.scalingEvents, event)
			s.lastScaleUp[serviceName] = time.Now()
		}
	}
}
