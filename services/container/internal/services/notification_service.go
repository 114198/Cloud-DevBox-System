// Package services provides business logic for the container service.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// NotificationServiceConfig holds configuration for the notification service
type NotificationServiceConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
	WebhookURL   string
}

// DefaultNotificationServiceConfig returns default configuration
func DefaultNotificationServiceConfig() *NotificationServiceConfig {
	return &NotificationServiceConfig{
		SMTPHost:   "smtp.example.com",
		SMTPPort:   587,
		FromEmail:  "alerts@devbox.com",
		FromName:   "DevBox Alerts",
		WebhookURL: "",
	}
}

// NotificationService handles sending notifications
type NotificationService struct {
	config     *NotificationServiceConfig
	logger     *zap.Logger
	httpClient *http.Client

	// In-memory queue for notifications
	queue      chan *models.Notification
	mu         sync.RWMutex
	sent       map[string]*models.Notification
}

// NewNotificationService creates a new notification service
func NewNotificationService(config *NotificationServiceConfig, logger *zap.Logger) *NotificationService {
	if config == nil {
		config = DefaultNotificationServiceConfig()
	}

	s := &NotificationService{
		config:     config,
		logger:     logger.Named("notification-service"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		queue:      make(chan *models.Notification, 1000),
		sent:       make(map[string]*models.Notification),
	}

	// Start worker
	go s.worker()

	return s
}


// Send queues a notification for sending
func (s *NotificationService) Send(notification *models.Notification) error {
	select {
	case s.queue <- notification:
		s.logger.Debug("Notification queued",
			zap.String("id", notification.ID),
			zap.String("type", notification.Type))
		return nil
	default:
		return fmt.Errorf("notification queue is full")
	}
}

// worker processes notifications from the queue
func (s *NotificationService) worker() {
	for notification := range s.queue {
		s.processNotification(notification)
	}
}

// processNotification sends a single notification
func (s *NotificationService) processNotification(notification *models.Notification) {
	var err error

	switch notification.Type {
	case "email":
		err = s.sendEmail(notification)
	case "web":
		err = s.sendWebNotification(notification)
	case "webhook":
		err = s.sendWebhook(notification)
	default:
		err = fmt.Errorf("unknown notification type: %s", notification.Type)
	}

	now := time.Now()
	notification.SentAt = &now

	s.mu.Lock()
	if err != nil {
		notification.Status = "failed"
		notification.Error = err.Error()
		s.logger.Error("Failed to send notification",
			zap.String("id", notification.ID),
			zap.String("type", notification.Type),
			zap.Error(err))
	} else {
		notification.Status = "sent"
		s.logger.Info("Notification sent",
			zap.String("id", notification.ID),
			zap.String("type", notification.Type))
	}
	s.sent[notification.ID] = notification
	s.mu.Unlock()
}

// sendEmail sends an email notification
func (s *NotificationService) sendEmail(notification *models.Notification) error {
	// Email template
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #1890ff; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .alert-critical { border-left: 4px solid #ff4d4f; }
        .alert-warning { border-left: 4px solid #faad14; }
        .alert-info { border-left: 4px solid #1890ff; }
        .footer { padding: 20px; text-align: center; color: #999; font-size: 12px; }
        .button { display: inline-block; padding: 10px 20px; background: #1890ff; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>DevBox 告警通知</h1>
        </div>
        <div class="content">
            <h2>{{.Title}}</h2>
            <p>{{.Message}}</p>
            <p><a href="https://devbox.com/alerts" class="button">查看详情</a></p>
        </div>
        <div class="footer">
            <p>此邮件由 DevBox 系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, notification); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// In production, use actual SMTP client
	// For now, just log the email
	s.logger.Info("Email notification prepared",
		zap.String("to", notification.UserID),
		zap.String("subject", notification.Title),
		zap.Int("bodyLength", body.Len()))

	return nil
}


// sendWebNotification sends a web notification (stored for polling or WebSocket)
func (s *NotificationService) sendWebNotification(notification *models.Notification) error {
	// In production, this would:
	// 1. Store in Redis for polling
	// 2. Push via WebSocket if user is connected
	// 3. Store in database for notification history

	s.logger.Info("Web notification sent",
		zap.String("userId", notification.UserID),
		zap.String("title", notification.Title))

	return nil
}

// sendWebhook sends a webhook notification
func (s *NotificationService) sendWebhook(notification *models.Notification) error {
	if s.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload := map[string]interface{}{
		"id":        notification.ID,
		"alertId":   notification.AlertID,
		"userId":    notification.UserID,
		"title":     notification.Title,
		"message":   notification.Message,
		"timestamp": notification.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DevBox-Alert-Webhook/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GetSentNotifications returns sent notifications for a user
func (s *NotificationService) GetSentNotifications(ctx context.Context, userID string, limit int) ([]*models.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var notifications []*models.Notification
	for _, n := range s.sent {
		if n.UserID == userID {
			notifications = append(notifications, n)
		}
	}

	// Sort by created time (newest first) and limit
	// In production, this would be done by the database
	if len(notifications) > limit {
		notifications = notifications[:limit]
	}

	return notifications, nil
}

// GetNotificationStats returns notification statistics
func (s *NotificationService) GetNotificationStats(ctx context.Context, userID string) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]int64{
		"total":  0,
		"sent":   0,
		"failed": 0,
	}

	for _, n := range s.sent {
		if userID != "" && n.UserID != userID {
			continue
		}
		stats["total"]++
		if n.Status == "sent" {
			stats["sent"]++
		} else if n.Status == "failed" {
			stats["failed"]++
		}
	}

	return stats, nil
}

// Close closes the notification service
func (s *NotificationService) Close() {
	close(s.queue)
}
