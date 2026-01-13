// Package models defines the data models for the container service.
package models

import (
	"time"
)

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	Timestamp           time.Time `json:"timestamp"`
	CPUUsagePercent     float64   `json:"cpuUsagePercent"`
	MemoryUsagePercent  float64   `json:"memoryUsagePercent"`
	DiskUsagePercent    float64   `json:"diskUsagePercent"`
	NetworkRxBytesPerSec float64  `json:"networkRxBytesPerSec"`
	NetworkTxBytesPerSec float64  `json:"networkTxBytesPerSec"`
	ActivePods          int64     `json:"activePods"`
	TotalNodes          int64     `json:"totalNodes"`
	HealthyNodes        int64     `json:"healthyNodes"`
}

// BusinessMetrics represents business-level metrics
type BusinessMetrics struct {
	Timestamp             time.Time `json:"timestamp"`
	ActiveEnvironments    int64     `json:"activeEnvironments"`
	TotalEnvironments     int64     `json:"totalEnvironments"`
	ActiveUsers           int64     `json:"activeUsers"`
	TotalUsers            int64     `json:"totalUsers"`
	APIRequestsPerSecond  float64   `json:"apiRequestsPerSecond"`
	APIErrorRate          float64   `json:"apiErrorRate"`
	APIP95LatencyMs       float64   `json:"apiP95LatencyMs"`
	EnvCreationP95Seconds float64   `json:"envCreationP95Seconds"`
}

// CollectionStats represents metrics collection statistics
type CollectionStats struct {
	LastCollectionTime time.Time     `json:"lastCollectionTime"`
	CollectionCount    int64         `json:"collectionCount"`
	CollectionErrors   int64         `json:"collectionErrors"`
	CollectionInterval time.Duration `json:"collectionInterval"`
}

// EnvironmentMetricsTimeSeries represents time series data for an environment
type EnvironmentMetricsTimeSeries struct {
	EnvironmentID string               `json:"environmentId"`
	DataPoints    []EnvironmentMetrics `json:"dataPoints"`
}

// PerformanceAlert represents a performance-related alert
type PerformanceAlert struct {
	ID          string        `json:"id"`
	Type        string        `json:"type"` // system, business, environment
	Severity    AlertSeverity `json:"severity"`
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	MetricName  string        `json:"metricName"`
	MetricValue float64       `json:"metricValue"`
	Threshold   float64       `json:"threshold"`
	CreatedAt   time.Time     `json:"createdAt"`
	ResolvedAt  *time.Time    `json:"resolvedAt,omitempty"`
}

// AlertManagerConfig represents AlertManager configuration
type AlertManagerConfig struct {
	URL              string        `json:"url"`
	Enabled          bool          `json:"enabled"`
	AlertCheckPeriod time.Duration `json:"alertCheckPeriod"`
}

// AlertManagerAlert represents an alert from AlertManager
type AlertManagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// AlertManagerResponse represents a response from AlertManager API
type AlertManagerResponse struct {
	Status string              `json:"status"`
	Data   []AlertManagerAlert `json:"data"`
}


// PerformanceAutoScalingConfig represents auto-scaling configuration for performance monitoring
type PerformanceAutoScalingConfig struct {
	Enabled           bool    `json:"enabled"`
	MinReplicas       int32   `json:"minReplicas"`
	MaxReplicas       int32   `json:"maxReplicas"`
	CPUThreshold      float64 `json:"cpuThreshold"`
	MemoryThreshold   float64 `json:"memoryThreshold"`
	ScaleUpCooldown   time.Duration `json:"scaleUpCooldown"`
	ScaleDownCooldown time.Duration `json:"scaleDownCooldown"`
}

// ScalingEvent represents an auto-scaling event
type ScalingEvent struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // scale_up, scale_down
	ServiceName   string    `json:"serviceName"`
	FromReplicas  int32     `json:"fromReplicas"`
	ToReplicas    int32     `json:"toReplicas"`
	Reason        string    `json:"reason"`
	MetricName    string    `json:"metricName"`
	MetricValue   float64   `json:"metricValue"`
	Threshold     float64   `json:"threshold"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"errorMessage,omitempty"`
}

// GrafanaDashboard represents a Grafana dashboard configuration
type GrafanaDashboard struct {
	ID          string            `json:"id"`
	UID         string            `json:"uid"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Panels      []GrafanaPanel    `json:"panels"`
	Variables   []GrafanaVariable `json:"variables"`
	Refresh     string            `json:"refresh"`
	Time        GrafanaTimeRange  `json:"time"`
}

// GrafanaPanel represents a panel in a Grafana dashboard
type GrafanaPanel struct {
	ID          int               `json:"id"`
	Title       string            `json:"title"`
	Type        string            `json:"type"` // graph, gauge, stat, table
	GridPos     GrafanaGridPos    `json:"gridPos"`
	Targets     []GrafanaTarget   `json:"targets"`
	FieldConfig *GrafanaFieldConfig `json:"fieldConfig,omitempty"`
}

// GrafanaGridPos represents panel position in Grafana
type GrafanaGridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

// GrafanaTarget represents a query target in Grafana
type GrafanaTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	RefID        string `json:"refId"`
}

// GrafanaVariable represents a dashboard variable
type GrafanaVariable struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Query   string `json:"query"`
	Current struct {
		Text  string `json:"text"`
		Value string `json:"value"`
	} `json:"current"`
}

// GrafanaTimeRange represents the time range for a dashboard
type GrafanaTimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// GrafanaFieldConfig represents field configuration for panels
type GrafanaFieldConfig struct {
	Defaults struct {
		Unit       string `json:"unit"`
		Min        *float64 `json:"min,omitempty"`
		Max        *float64 `json:"max,omitempty"`
		Thresholds struct {
			Mode  string `json:"mode"`
			Steps []struct {
				Color string   `json:"color"`
				Value *float64 `json:"value"`
			} `json:"steps"`
		} `json:"thresholds"`
	} `json:"defaults"`
}

// PerformanceReport represents a performance report
type PerformanceReport struct {
	GeneratedAt       time.Time       `json:"generatedAt"`
	Period            string          `json:"period"`
	SystemMetrics     *SystemMetrics  `json:"systemMetrics"`
	BusinessMetrics   *BusinessMetrics `json:"businessMetrics"`
	AlertsSummary     *AlertsSummary  `json:"alertsSummary"`
	ScalingEvents     []ScalingEvent  `json:"scalingEvents"`
	Recommendations   []string        `json:"recommendations"`
}

// AlertsSummary represents a summary of alerts
type AlertsSummary struct {
	TotalAlerts      int64 `json:"totalAlerts"`
	CriticalAlerts   int64 `json:"criticalAlerts"`
	WarningAlerts    int64 `json:"warningAlerts"`
	ResolvedAlerts   int64 `json:"resolvedAlerts"`
	AvgResponseTime  time.Duration `json:"avgResponseTime"`
}
