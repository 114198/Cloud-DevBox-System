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

// AlertServiceConfig holds configuration for the alert service
type AlertServiceConfig struct {
	DefaultStorageThreshold float64
	DefaultMemoryThreshold  float64
	DefaultCPUThreshold     float64
	AlertResponseTimeout    time.Duration
}

// DefaultAlertServiceConfig returns default configuration
func DefaultAlertServiceConfig() *AlertServiceConfig {
	return &AlertServiceConfig{
		DefaultStorageThreshold: 90.0,
		DefaultMemoryThreshold:  90.0,
		DefaultCPUThreshold:     95.0,
		AlertResponseTimeout:    2 * time.Minute,
	}
}


// AlertService handles alert management and notifications
type AlertService struct {
	config *AlertServiceConfig
	logger *zap.Logger

	// In-memory storage (in production, use database)
	alerts      map[string]*models.Alert
	alertRules  map[string]*models.AlertRule
	notifications map[string]*models.Notification
	mu          sync.RWMutex

	// Notification channels
	notificationCh chan *models.Notification

	// Alert state tracking for deduplication
	activeAlertKeys map[string]string // key -> alertID
}

// NewAlertService creates a new alert service
func NewAlertService(logger *zap.Logger) *AlertService {
	config := DefaultAlertServiceConfig()
	
	s := &AlertService{
		config:          config,
		logger:          logger.Named("alert-service"),
		alerts:          make(map[string]*models.Alert),
		alertRules:      make(map[string]*models.AlertRule),
		notifications:   make(map[string]*models.Notification),
		notificationCh:  make(chan *models.Notification, 100),
		activeAlertKeys: make(map[string]string),
	}

	// Start notification worker
	go s.notificationWorker()

	// Create default alert rules
	s.createDefaultRules()

	return s
}

// createDefaultRules creates default alert rules
func (s *AlertService) createDefaultRules() {
	defaultRules := []*models.AlertRule{
		{
			ID:            uuid.New().String(),
			Type:          models.MetricTypeStorage,
			Severity:      models.AlertSeverityCritical,
			Threshold:     s.config.DefaultStorageThreshold,
			Duration:      0, // Immediate
			Enabled:       true,
			NotifyMethods: []string{"web", "email"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            uuid.New().String(),
			Type:          models.MetricTypeMemory,
			Severity:      models.AlertSeverityWarning,
			Threshold:     s.config.DefaultMemoryThreshold,
			Duration:      5 * time.Minute,
			Enabled:       true,
			NotifyMethods: []string{"web"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            uuid.New().String(),
			Type:          models.MetricTypeCPU,
			Severity:      models.AlertSeverityWarning,
			Threshold:     s.config.DefaultCPUThreshold,
			Duration:      10 * time.Minute,
			Enabled:       true,
			NotifyMethods: []string{"web"},
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	for _, rule := range defaultRules {
		s.alertRules[rule.ID] = rule
	}
}


// CheckAlerts checks alert conditions for an environment
func (s *AlertService) CheckAlerts(ctx context.Context, environmentID string, metrics *models.EnvironmentMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.alertRules {
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to this environment
		if rule.EnvironmentID != "" && rule.EnvironmentID != environmentID {
			continue
		}

		var currentValue float64
		switch rule.Type {
		case models.MetricTypeCPU:
			currentValue = metrics.CPUUsage
		case models.MetricTypeMemory:
			currentValue = metrics.MemoryUsage
		case models.MetricTypeStorage:
			currentValue = metrics.StorageUsage
		default:
			continue
		}

		alertKey := fmt.Sprintf("%s:%s:%s", environmentID, rule.Type, rule.Severity)

		if currentValue >= rule.Threshold {
			// Check if alert already exists
			if existingAlertID, exists := s.activeAlertKeys[alertKey]; exists {
				// Update existing alert
				if alert, ok := s.alerts[existingAlertID]; ok {
					alert.Value = currentValue
					alert.UpdatedAt = time.Now()
				}
				continue
			}

			// Create new alert
			alert := s.createAlert(environmentID, rule, currentValue)
			s.alerts[alert.ID] = alert
			s.activeAlertKeys[alertKey] = alert.ID

			// Send notification
			s.sendAlertNotification(alert)

			s.logger.Info("Alert created",
				zap.String("alertId", alert.ID),
				zap.String("environmentId", environmentID),
				zap.String("type", string(rule.Type)),
				zap.Float64("value", currentValue),
				zap.Float64("threshold", rule.Threshold))
		} else {
			// Check if we should resolve an existing alert
			if existingAlertID, exists := s.activeAlertKeys[alertKey]; exists {
				if alert, ok := s.alerts[existingAlertID]; ok {
					s.resolveAlert(alert)
					delete(s.activeAlertKeys, alertKey)
				}
			}
		}
	}
}

// createAlert creates a new alert
func (s *AlertService) createAlert(environmentID string, rule *models.AlertRule, value float64) *models.Alert {
	now := time.Now()
	
	var title, message string
	switch rule.Type {
	case models.MetricTypeStorage:
		title = "存储空间告警"
		message = fmt.Sprintf("环境存储使用率已达 %.1f%%，超过阈值 %.1f%%", value, rule.Threshold)
	case models.MetricTypeMemory:
		title = "内存使用告警"
		message = fmt.Sprintf("环境内存使用率已达 %.1f%%，超过阈值 %.1f%%", value, rule.Threshold)
	case models.MetricTypeCPU:
		title = "CPU 使用告警"
		message = fmt.Sprintf("环境 CPU 使用率已达 %.1f%%，超过阈值 %.1f%%", value, rule.Threshold)
	default:
		title = "资源告警"
		message = fmt.Sprintf("资源使用率已达 %.1f%%，超过阈值 %.1f%%", value, rule.Threshold)
	}

	return &models.Alert{
		ID:            uuid.New().String(),
		EnvironmentID: environmentID,
		Type:          rule.Type,
		Severity:      rule.Severity,
		Status:        models.AlertStatusActive,
		Title:         title,
		Message:       message,
		Value:         value,
		Threshold:     rule.Threshold,
		CreatedAt:     now,
		UpdatedAt:     now,
		NotifyMethods: rule.NotifyMethods,
	}
}


// resolveAlert resolves an alert
func (s *AlertService) resolveAlert(alert *models.Alert) {
	now := time.Now()
	alert.Status = models.AlertStatusResolved
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	s.logger.Info("Alert resolved",
		zap.String("alertId", alert.ID),
		zap.String("environmentId", alert.EnvironmentID))
}

// sendAlertNotification sends a notification for an alert
func (s *AlertService) sendAlertNotification(alert *models.Alert) {
	for _, method := range alert.NotifyMethods {
		notification := &models.Notification{
			ID:        uuid.New().String(),
			UserID:    alert.UserID,
			AlertID:   alert.ID,
			Type:      method,
			Status:    "pending",
			Title:     alert.Title,
			Message:   alert.Message,
			CreatedAt: time.Now(),
		}

		s.notifications[notification.ID] = notification

		// Send to notification channel (non-blocking)
		select {
		case s.notificationCh <- notification:
		default:
			s.logger.Warn("Notification channel full, dropping notification",
				zap.String("notificationId", notification.ID))
		}
	}

	now := time.Now()
	alert.NotifiedAt = &now
}

// notificationWorker processes notifications
func (s *AlertService) notificationWorker() {
	for notification := range s.notificationCh {
		s.processNotification(notification)
	}
}

// processNotification processes a single notification
func (s *AlertService) processNotification(notification *models.Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	switch notification.Type {
	case "email":
		err = s.sendEmailNotification(notification)
	case "web":
		err = s.sendWebNotification(notification)
	default:
		err = fmt.Errorf("unknown notification type: %s", notification.Type)
	}

	now := time.Now()
	notification.SentAt = &now

	if err != nil {
		notification.Status = "failed"
		notification.Error = err.Error()
		s.logger.Error("Failed to send notification",
			zap.String("notificationId", notification.ID),
			zap.Error(err))
	} else {
		notification.Status = "sent"
		s.logger.Info("Notification sent",
			zap.String("notificationId", notification.ID),
			zap.String("type", notification.Type))
	}
}

// sendEmailNotification sends an email notification
func (s *AlertService) sendEmailNotification(notification *models.Notification) error {
	// In production, integrate with email service (SendGrid, SES, etc.)
	s.logger.Info("Sending email notification",
		zap.String("notificationId", notification.ID),
		zap.String("title", notification.Title))
	return nil
}

// sendWebNotification sends a web notification
func (s *AlertService) sendWebNotification(notification *models.Notification) error {
	// In production, push to WebSocket or store for polling
	s.logger.Info("Sending web notification",
		zap.String("notificationId", notification.ID),
		zap.String("title", notification.Title))
	return nil
}


// ListAlerts lists alerts with filtering
func (s *AlertService) ListAlerts(ctx context.Context, userID string, req *models.ListAlertsRequest) (*models.ListAlertsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []models.Alert
	for _, alert := range s.alerts {
		// Filter by user
		if userID != "" && alert.UserID != userID {
			continue
		}
		// Filter by environment
		if req.EnvironmentID != "" && alert.EnvironmentID != req.EnvironmentID {
			continue
		}
		// Filter by status
		if req.Status != "" && alert.Status != req.Status {
			continue
		}
		// Filter by severity
		if req.Severity != "" && alert.Severity != req.Severity {
			continue
		}
		filtered = append(filtered, *alert)
	}

	// Pagination
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return &models.ListAlertsResponse{
		Alerts:   filtered[start:end],
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetAlert gets an alert by ID
func (s *AlertService) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, ok := s.alerts[alertID]
	if !ok {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}
	return alert, nil
}

// AcknowledgeAlert acknowledges an alert
func (s *AlertService) AcknowledgeAlert(ctx context.Context, alertID, userID string) (*models.Alert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, ok := s.alerts[alertID]
	if !ok {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	if alert.Status != models.AlertStatusActive {
		return nil, fmt.Errorf("alert is not active")
	}

	now := time.Now()
	alert.Status = models.AlertStatusAcked
	alert.AckedAt = &now
	alert.AckedBy = userID
	alert.UpdatedAt = now

	s.logger.Info("Alert acknowledged",
		zap.String("alertId", alertID),
		zap.String("userId", userID))

	return alert, nil
}

// GetAlertStats returns alert statistics
func (s *AlertService) GetAlertStats(ctx context.Context, userID string) (*models.AlertStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &models.AlertStats{}
	for _, alert := range s.alerts {
		if userID != "" && alert.UserID != userID {
			continue
		}
		stats.TotalAlerts++
		switch alert.Status {
		case models.AlertStatusActive:
			stats.ActiveAlerts++
		case models.AlertStatusResolved:
			stats.ResolvedAlerts++
		}
		switch alert.Severity {
		case models.AlertSeverityCritical:
			stats.CriticalAlerts++
		case models.AlertSeverityWarning:
			stats.WarningAlerts++
		case models.AlertSeverityInfo:
			stats.InfoAlerts++
		}
	}
	return stats, nil
}


// CreateAlertRule creates a new alert rule
func (s *AlertService) CreateAlertRule(ctx context.Context, userID string, req *models.CreateAlertRuleRequest) (*models.AlertRule, error) {
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	now := time.Now()
	rule := &models.AlertRule{
		ID:            uuid.New().String(),
		UserID:        userID,
		EnvironmentID: req.EnvironmentID,
		Type:          req.Type,
		Severity:      req.Severity,
		Threshold:     req.Threshold,
		Duration:      duration,
		Enabled:       true,
		NotifyMethods: req.NotifyMethods,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.mu.Lock()
	s.alertRules[rule.ID] = rule
	s.mu.Unlock()

	s.logger.Info("Alert rule created",
		zap.String("ruleId", rule.ID),
		zap.String("type", string(rule.Type)),
		zap.Float64("threshold", rule.Threshold))

	return rule, nil
}

// UpdateAlertRule updates an alert rule
func (s *AlertService) UpdateAlertRule(ctx context.Context, ruleID string, req *models.UpdateAlertRuleRequest) (*models.AlertRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.alertRules[ruleID]
	if !ok {
		return nil, fmt.Errorf("alert rule not found: %s", ruleID)
	}

	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Duration != nil {
		duration, err := time.ParseDuration(*req.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
		rule.Duration = duration
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.NotifyMethods != nil {
		rule.NotifyMethods = req.NotifyMethods
	}
	rule.UpdatedAt = time.Now()

	return rule, nil
}

// DeleteAlertRule deletes an alert rule
func (s *AlertService) DeleteAlertRule(ctx context.Context, ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.alertRules[ruleID]; !ok {
		return fmt.Errorf("alert rule not found: %s", ruleID)
	}

	delete(s.alertRules, ruleID)
	return nil
}

// ListAlertRules lists alert rules
func (s *AlertService) ListAlertRules(ctx context.Context, userID string) ([]*models.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []*models.AlertRule
	for _, rule := range s.alertRules {
		// Include global rules (no userID) and user-specific rules
		if rule.UserID == "" || rule.UserID == userID {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// GetActiveAlertsForEnvironment returns active alerts for an environment
func (s *AlertService) GetActiveAlertsForEnvironment(ctx context.Context, environmentID string) ([]*models.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alerts []*models.Alert
	for _, alert := range s.alerts {
		if alert.EnvironmentID == environmentID && alert.Status == models.AlertStatusActive {
			alerts = append(alerts, alert)
		}
	}
	return alerts, nil
}

// GetAlertResponseTime calculates the time from alert creation to notification
func (s *AlertService) GetAlertResponseTime(alert *models.Alert) time.Duration {
	if alert.NotifiedAt == nil {
		return 0
	}
	return alert.NotifiedAt.Sub(alert.CreatedAt)
}

// CheckAlertResponseTimeCompliance checks if alert response time is within SLA
func (s *AlertService) CheckAlertResponseTimeCompliance(alert *models.Alert) bool {
	responseTime := s.GetAlertResponseTime(alert)
	return responseTime <= s.config.AlertResponseTimeout
}
