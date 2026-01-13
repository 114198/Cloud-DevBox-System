// Package services provides business logic for the container service.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// GrafanaServiceConfig holds configuration for Grafana integration
type GrafanaServiceConfig struct {
	GrafanaURL string
	APIKey     string
	OrgID      int
}

// DefaultGrafanaServiceConfig returns default configuration
func DefaultGrafanaServiceConfig() *GrafanaServiceConfig {
	return &GrafanaServiceConfig{
		GrafanaURL: "http://grafana:3000",
		OrgID:      1,
	}
}

// GrafanaService handles Grafana dashboard management
type GrafanaService struct {
	config     *GrafanaServiceConfig
	logger     *zap.Logger
	httpClient *http.Client
}

// NewGrafanaService creates a new Grafana service
func NewGrafanaService(config *GrafanaServiceConfig, logger *zap.Logger) *GrafanaService {
	if config == nil {
		config = DefaultGrafanaServiceConfig()
	}

	return &GrafanaService{
		config:     config,
		logger:     logger.Named("grafana-service"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateSystemMonitoringDashboard creates the system monitoring dashboard
func (s *GrafanaService) CreateSystemMonitoringDashboard(ctx context.Context) (*models.GrafanaDashboard, error) {
	dashboard := s.buildSystemMonitoringDashboard()
	return s.createOrUpdateDashboard(ctx, dashboard)
}

// CreateBusinessMetricsDashboard creates the business metrics dashboard
func (s *GrafanaService) CreateBusinessMetricsDashboard(ctx context.Context) (*models.GrafanaDashboard, error) {
	dashboard := s.buildBusinessMetricsDashboard()
	return s.createOrUpdateDashboard(ctx, dashboard)
}

// buildSystemMonitoringDashboard builds the system monitoring dashboard configuration
func (s *GrafanaService) buildSystemMonitoringDashboard() *models.GrafanaDashboard {
	return &models.GrafanaDashboard{
		UID:         "devbox-system-monitoring",
		Title:       "Cloud DevBox - System Monitoring",
		Description: "System-level metrics for Cloud DevBox infrastructure",
		Tags:        []string{"devbox", "system", "monitoring"},
		Refresh:     "30s",
		Time: models.GrafanaTimeRange{
			From: "now-1h",
			To:   "now",
		},
		Panels: []models.GrafanaPanel{
			// CPU Usage Panel
			{
				ID:    1,
				Title: "CPU Usage",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 0, Y: 0},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`,
						LegendFormat: "CPU Usage %",
						RefID:        "A",
					},
				},
			},
			// Memory Usage Panel
			{
				ID:    2,
				Title: "Memory Usage",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 12, Y: 0},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`,
						LegendFormat: "Memory Usage %",
						RefID:        "A",
					},
				},
			},
			// Disk Usage Panel
			{
				ID:    3,
				Title: "Disk Usage",
				Type:  "gauge",
				GridPos: models.GrafanaGridPos{H: 8, W: 8, X: 0, Y: 8},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `(1 - (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"})) * 100`,
						LegendFormat: "Disk Usage %",
						RefID:        "A",
					},
				},
			},
			// Network I/O Panel
			{
				ID:    4,
				Title: "Network I/O",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 16, X: 8, Y: 8},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `sum(rate(node_network_receive_bytes_total[5m]))`,
						LegendFormat: "Received",
						RefID:        "A",
					},
					{
						Expr:         `sum(rate(node_network_transmit_bytes_total[5m]))`,
						LegendFormat: "Transmitted",
						RefID:        "B",
					},
				},
			},
			// Active Pods Panel
			{
				ID:    5,
				Title: "Active Pods",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 0, Y: 16},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(kube_pod_status_phase{phase="Running"})`,
						LegendFormat: "Running Pods",
						RefID:        "A",
					},
				},
			},
			// Node Count Panel
			{
				ID:    6,
				Title: "Cluster Nodes",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 6, Y: 16},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(kube_node_info)`,
						LegendFormat: "Total Nodes",
						RefID:        "A",
					},
				},
			},
		},
	}
}


// buildBusinessMetricsDashboard builds the business metrics dashboard configuration
func (s *GrafanaService) buildBusinessMetricsDashboard() *models.GrafanaDashboard {
	return &models.GrafanaDashboard{
		UID:         "devbox-business-metrics",
		Title:       "Cloud DevBox - Business Metrics",
		Description: "Business-level metrics for Cloud DevBox platform",
		Tags:        []string{"devbox", "business", "metrics"},
		Refresh:     "30s",
		Time: models.GrafanaTimeRange{
			From: "now-1h",
			To:   "now",
		},
		Panels: []models.GrafanaPanel{
			// Active Environments Panel
			{
				ID:    1,
				Title: "Active Environments",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 0, Y: 0},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(devbox_environment_status{status="running"})`,
						LegendFormat: "Active",
						RefID:        "A",
					},
				},
			},
			// Total Environments Panel
			{
				ID:    2,
				Title: "Total Environments",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 6, Y: 0},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(devbox_environment_status)`,
						LegendFormat: "Total",
						RefID:        "A",
					},
				},
			},
			// Active Users Panel
			{
				ID:    3,
				Title: "Active Users",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 12, Y: 0},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(count by (user_id) (devbox_environment_status{status="running"}))`,
						LegendFormat: "Active Users",
						RefID:        "A",
					},
				},
			},
			// API Request Rate Panel
			{
				ID:    4,
				Title: "API Request Rate",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 0, Y: 4},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `sum(rate(http_requests_total[5m]))`,
						LegendFormat: "Requests/sec",
						RefID:        "A",
					},
				},
			},
			// API Error Rate Panel
			{
				ID:    5,
				Title: "API Error Rate",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 12, Y: 4},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100`,
						LegendFormat: "Error Rate %",
						RefID:        "A",
					},
				},
			},
			// API Latency Panel
			{
				ID:    6,
				Title: "API Latency (P95)",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 0, Y: 12},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) * 1000`,
						LegendFormat: "P95 Latency (ms)",
						RefID:        "A",
					},
				},
			},
			// Environment Creation Time Panel
			{
				ID:    7,
				Title: "Environment Creation Time (P95)",
				Type:  "graph",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 12, Y: 12},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `histogram_quantile(0.95, sum(rate(devbox_environment_creation_seconds_bucket[5m])) by (le))`,
						LegendFormat: "P95 Creation Time (s)",
						RefID:        "A",
					},
				},
			},
			// Environment Status Distribution Panel
			{
				ID:    8,
				Title: "Environment Status Distribution",
				Type:  "piechart",
				GridPos: models.GrafanaGridPos{H: 8, W: 12, X: 0, Y: 20},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count by (status) (devbox_environment_status)`,
						LegendFormat: "{{status}}",
						RefID:        "A",
					},
				},
			},
			// Alerts Overview Panel
			{
				ID:    9,
				Title: "Active Alerts",
				Type:  "stat",
				GridPos: models.GrafanaGridPos{H: 4, W: 6, X: 12, Y: 20},
				Targets: []models.GrafanaTarget{
					{
						Expr:         `count(ALERTS{alertstate="firing"})`,
						LegendFormat: "Firing Alerts",
						RefID:        "A",
					},
				},
			},
		},
	}
}


// createOrUpdateDashboard creates or updates a dashboard in Grafana
func (s *GrafanaService) createOrUpdateDashboard(ctx context.Context, dashboard *models.GrafanaDashboard) (*models.GrafanaDashboard, error) {
	payload := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"uid":         dashboard.UID,
			"title":       dashboard.Title,
			"description": dashboard.Description,
			"tags":        dashboard.Tags,
			"panels":      dashboard.Panels,
			"refresh":     dashboard.Refresh,
			"time":        dashboard.Time,
			"schemaVersion": 30,
		},
		"overwrite": true,
		"message":   "Dashboard updated by Cloud DevBox",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/dashboards/db", s.config.GrafanaURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Debug("Failed to create dashboard in Grafana", zap.Error(err))
		// Return the dashboard config even if Grafana is not available
		return dashboard, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.logger.Debug("Grafana returned error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(bodyBytes)))
		return dashboard, nil
	}

	s.logger.Info("Dashboard created/updated in Grafana",
		zap.String("uid", dashboard.UID),
		zap.String("title", dashboard.Title))

	return dashboard, nil
}

// GetDashboard retrieves a dashboard from Grafana
func (s *GrafanaService) GetDashboard(ctx context.Context, uid string) (*models.GrafanaDashboard, error) {
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", s.config.GrafanaURL, uid)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if s.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}

	var result struct {
		Dashboard models.GrafanaDashboard `json:"dashboard"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Dashboard, nil
}

// DeleteDashboard deletes a dashboard from Grafana
func (s *GrafanaService) DeleteDashboard(ctx context.Context, uid string) error {
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", s.config.GrafanaURL, uid)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	if s.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.APIKey))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grafana returned status %d", resp.StatusCode)
	}

	s.logger.Info("Dashboard deleted from Grafana", zap.String("uid", uid))
	return nil
}

// GetSystemDashboardConfig returns the system monitoring dashboard configuration
func (s *GrafanaService) GetSystemDashboardConfig() *models.GrafanaDashboard {
	return s.buildSystemMonitoringDashboard()
}

// GetBusinessDashboardConfig returns the business metrics dashboard configuration
func (s *GrafanaService) GetBusinessDashboardConfig() *models.GrafanaDashboard {
	return s.buildBusinessMetricsDashboard()
}
