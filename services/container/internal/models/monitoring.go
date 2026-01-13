// Package models defines the data models for the container service.
package models

import (
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCPU     MetricType = "cpu"
	MetricTypeMemory  MetricType = "memory"
	MetricTypeStorage MetricType = "storage"
	MetricTypeNetwork MetricType = "network"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusAcked    AlertStatus = "acknowledged"
)

// MetricDataPoint represents a single metric data point
type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// EnvironmentMetrics represents metrics for an environment
type EnvironmentMetrics struct {
	EnvironmentID string    `json:"environmentId"`
	Timestamp     time.Time `json:"timestamp"`

	// CPU metrics
	CPUUsage       float64 `json:"cpuUsage"`       // Percentage (0-100)
	CPUCores       float64 `json:"cpuCores"`       // Number of cores used
	CPULimit       float64 `json:"cpuLimit"`       // CPU limit in cores

	// Memory metrics
	MemoryUsage    float64 `json:"memoryUsage"`    // Percentage (0-100)
	MemoryUsed     int64   `json:"memoryUsed"`     // Bytes used
	MemoryLimit    int64   `json:"memoryLimit"`    // Memory limit in bytes

	// Storage metrics
	StorageUsage   float64 `json:"storageUsage"`   // Percentage (0-100)
	StorageUsed    int64   `json:"storageUsed"`    // Bytes used
	StorageLimit   int64   `json:"storageLimit"`   // Storage limit in bytes

	// Network metrics
	NetworkRxBytes int64   `json:"networkRxBytes"` // Bytes received
	NetworkTxBytes int64   `json:"networkTxBytes"` // Bytes transmitted
	NetworkRxRate  float64 `json:"networkRxRate"`  // Bytes/second received
	NetworkTxRate  float64 `json:"networkTxRate"`  // Bytes/second transmitted
}

// MetricsHistory represents historical metrics data
type MetricsHistory struct {
	EnvironmentID string            `json:"environmentId"`
	Period        string            `json:"period"`        // e.g., "1h", "24h", "7d"
	Resolution    string            `json:"resolution"`    // e.g., "1m", "5m", "1h"
	StartTime     time.Time         `json:"startTime"`
	EndTime       time.Time         `json:"endTime"`
	CPU           []MetricDataPoint `json:"cpu"`
	Memory        []MetricDataPoint `json:"memory"`
	Storage       []MetricDataPoint `json:"storage"`
	NetworkRx     []MetricDataPoint `json:"networkRx"`
	NetworkTx     []MetricDataPoint `json:"networkTx"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID            string        `json:"id"`
	EnvironmentID string        `json:"environmentId"`
	UserID        string        `json:"userId"`
	Type          MetricType    `json:"type"`
	Severity      AlertSeverity `json:"severity"`
	Status        AlertStatus   `json:"status"`
	Title         string        `json:"title"`
	Message       string        `json:"message"`
	Value         float64       `json:"value"`         // Current value that triggered alert
	Threshold     float64       `json:"threshold"`     // Threshold that was exceeded
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	ResolvedAt    *time.Time    `json:"resolvedAt,omitempty"`
	AckedAt       *time.Time    `json:"ackedAt,omitempty"`
	AckedBy       string        `json:"ackedBy,omitempty"`
	NotifiedAt    *time.Time    `json:"notifiedAt,omitempty"`
	NotifyMethods []string      `json:"notifyMethods,omitempty"` // email, web, etc.
}

// AlertRule represents a rule for generating alerts
type AlertRule struct {
	ID            string        `json:"id"`
	UserID        string        `json:"userId"`
	EnvironmentID string        `json:"environmentId,omitempty"` // Empty means all environments
	Type          MetricType    `json:"type"`
	Severity      AlertSeverity `json:"severity"`
	Threshold     float64       `json:"threshold"`
	Duration      time.Duration `json:"duration"`      // How long threshold must be exceeded
	Enabled       bool          `json:"enabled"`
	NotifyMethods []string      `json:"notifyMethods"` // email, web, etc.
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// Notification represents a notification to be sent
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AlertID   string    `json:"alertId"`
	Type      string    `json:"type"`    // email, web, sms
	Status    string    `json:"status"`  // pending, sent, failed
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// GetMetricsRequest represents a request to get metrics
type GetMetricsRequest struct {
	EnvironmentID string `form:"environmentId" binding:"required"`
}

// GetMetricsHistoryRequest represents a request to get metrics history
type GetMetricsHistoryRequest struct {
	EnvironmentID string `form:"environmentId" binding:"required"`
	Period        string `form:"period" binding:"omitempty,oneof=1h 6h 24h 7d 30d"`
	Resolution    string `form:"resolution" binding:"omitempty,oneof=1m 5m 15m 1h"`
}

// ListAlertsRequest represents a request to list alerts
type ListAlertsRequest struct {
	EnvironmentID string        `form:"environmentId"`
	Status        AlertStatus   `form:"status"`
	Severity      AlertSeverity `form:"severity"`
	Page          int           `form:"page,default=1"`
	PageSize      int           `form:"pageSize,default=20"`
}

// ListAlertsResponse represents a response containing alerts
type ListAlertsResponse struct {
	Alerts   []Alert `json:"alerts"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

// AcknowledgeAlertRequest represents a request to acknowledge an alert
type AcknowledgeAlertRequest struct {
	AlertID string `json:"alertId" binding:"required"`
}

// CreateAlertRuleRequest represents a request to create an alert rule
type CreateAlertRuleRequest struct {
	EnvironmentID string        `json:"environmentId,omitempty"`
	Type          MetricType    `json:"type" binding:"required,oneof=cpu memory storage network"`
	Severity      AlertSeverity `json:"severity" binding:"required,oneof=info warning critical"`
	Threshold     float64       `json:"threshold" binding:"required,min=0,max=100"`
	Duration      string        `json:"duration" binding:"required"` // e.g., "5m", "10m"
	NotifyMethods []string      `json:"notifyMethods" binding:"required,min=1"`
}

// UpdateAlertRuleRequest represents a request to update an alert rule
type UpdateAlertRuleRequest struct {
	Threshold     *float64 `json:"threshold,omitempty"`
	Duration      *string  `json:"duration,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	NotifyMethods []string `json:"notifyMethods,omitempty"`
}

// AlertStats represents alert statistics
type AlertStats struct {
	TotalAlerts    int64 `json:"totalAlerts"`
	ActiveAlerts   int64 `json:"activeAlerts"`
	ResolvedAlerts int64 `json:"resolvedAlerts"`
	CriticalAlerts int64 `json:"criticalAlerts"`
	WarningAlerts  int64 `json:"warningAlerts"`
	InfoAlerts     int64 `json:"infoAlerts"`
}

// PrometheusMetric represents a metric from Prometheus
type PrometheusMetric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
	Time   time.Time         `json:"time"`
}

// PrometheusQueryResult represents a Prometheus query result
type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
			Values [][]interface{}   `json:"values,omitempty"`
		} `json:"result"`
	} `json:"data"`
}
