package devbox

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// EnvironmentsService handles environment operations
type EnvironmentsService struct {
	client *Client
}

// Environment represents a development environment
type Environment struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	TemplateID     string            `json:"template_id"`
	Status         string            `json:"status"`
	Resources      *ResourceConfig   `json:"resources,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
	Ports          []PortMapping     `json:"ports,omitempty"`
	PreviewURL     string            `json:"preview_url,omitempty"`
	SSHHost        string            `json:"ssh_host,omitempty"`
	SSHPort        int               `json:"ssh_port,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastActiveAt   *time.Time        `json:"last_active_at,omitempty"`
}

// ResourceConfig defines resource allocation
type ResourceConfig struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}

// PortMapping defines a port mapping
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	Public        bool   `json:"public"`
}

// SSHConfig contains SSH connection information
type SSHConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	PrivateKey    string `json:"private_key"`
	ConfigSnippet string `json:"config_snippet"`
}

// EnvironmentMetrics contains environment metrics
type EnvironmentMetrics struct {
	CPUUsage     []MetricPoint `json:"cpu_usage"`
	MemoryUsage  []MetricPoint `json:"memory_usage"`
	StorageUsage []MetricPoint `json:"storage_usage"`
	NetworkRx    []MetricPoint `json:"network_rx"`
	NetworkTx    []MetricPoint `json:"network_tx"`
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// CreateEnvironmentRequest contains parameters for creating an environment
type CreateEnvironmentRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	TemplateID  string            `json:"template_id"`
	Resources   *ResourceConfig   `json:"resources,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	Ports       []PortMapping     `json:"ports,omitempty"`
	GitRepoURL  string            `json:"git_repo_url,omitempty"`
}

// UpdateEnvironmentRequest contains parameters for updating an environment
type UpdateEnvironmentRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Resources   *ResourceConfig   `json:"resources,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

// EnvironmentListOptions contains options for listing environments
type EnvironmentListOptions struct {
	ListOptions
	Status     string
	TemplateID string
}

// EnvironmentListResponse contains a paginated list of environments
type EnvironmentListResponse struct {
	Data       []Environment `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

// List returns a list of environments
func (s *EnvironmentsService) List(ctx context.Context, opts *EnvironmentListOptions) (*EnvironmentListResponse, error) {
	path := "/environments"
	if opts != nil {
		params := url.Values{}
		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.PageSize > 0 {
			params.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.TemplateID != "" {
			params.Set("template_id", opts.TemplateID)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var resp EnvironmentListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns an environment by ID
func (s *EnvironmentsService) Get(ctx context.Context, id string) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "GET", "/environments/"+id, nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Create creates a new environment
func (s *EnvironmentsService) Create(ctx context.Context, req *CreateEnvironmentRequest) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "POST", "/environments", req, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Update updates an environment
func (s *EnvironmentsService) Update(ctx context.Context, id string, req *UpdateEnvironmentRequest) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "PUT", "/environments/"+id, req, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Delete deletes an environment
func (s *EnvironmentsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", "/environments/"+id, nil, nil)
}

// Start starts an environment
func (s *EnvironmentsService) Start(ctx context.Context, id string) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "POST", "/environments/"+id+"/start", nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Stop stops an environment
func (s *EnvironmentsService) Stop(ctx context.Context, id string) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "POST", "/environments/"+id+"/stop", nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Restart restarts an environment
func (s *EnvironmentsService) Restart(ctx context.Context, id string) (*Environment, error) {
	var env Environment
	if err := s.client.request(ctx, "POST", "/environments/"+id+"/restart", nil, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

// GetSSHConfig returns SSH configuration for an environment
func (s *EnvironmentsService) GetSSHConfig(ctx context.Context, id string) (*SSHConfig, error) {
	var config SSHConfig
	if err := s.client.request(ctx, "GET", "/environments/"+id+"/ssh-config", nil, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetMetrics returns metrics for an environment
func (s *EnvironmentsService) GetMetrics(ctx context.Context, id string, startTime, endTime *time.Time) (*EnvironmentMetrics, error) {
	path := "/environments/" + id + "/metrics"
	params := url.Values{}
	if startTime != nil {
		params.Set("start_time", startTime.Format(time.RFC3339))
	}
	if endTime != nil {
		params.Set("end_time", endTime.Format(time.RFC3339))
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var metrics EnvironmentMetrics
	if err := s.client.request(ctx, "GET", path, nil, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}
