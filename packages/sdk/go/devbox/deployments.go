package devbox

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// DeploymentsService handles deployment operations
type DeploymentsService struct {
	client *Client
}

// Deployment represents a deployment
type Deployment struct {
	ID            string     `json:"id"`
	EnvironmentID string     `json:"environment_id"`
	Version       string     `json:"version"`
	Status        string     `json:"status"`
	ImageTag      string     `json:"image_tag,omitempty"`
	CommitSHA     string     `json:"commit_sha,omitempty"`
	CommitMessage string     `json:"commit_message,omitempty"`
	DeployedBy    string     `json:"deployed_by,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DeploymentLogs contains deployment logs
type DeploymentLogs struct {
	Logs []LogEntry `json:"logs"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Stage     string    `json:"stage,omitempty"`
}

// CreateDeploymentRequest contains parameters for creating a deployment
type CreateDeploymentRequest struct {
	EnvironmentID string            `json:"environment_id"`
	Branch        string            `json:"branch,omitempty"`
	CommitSHA     string            `json:"commit_sha,omitempty"`
	BuildArgs     map[string]string `json:"build_args,omitempty"`
}

// DeploymentListOptions contains options for listing deployments
type DeploymentListOptions struct {
	ListOptions
	EnvironmentID string
	Status        string
}

// DeploymentListResponse contains a paginated list of deployments
type DeploymentListResponse struct {
	Data       []Deployment `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// List returns a list of deployments
func (s *DeploymentsService) List(ctx context.Context, opts *DeploymentListOptions) (*DeploymentListResponse, error) {
	path := "/deployments"
	if opts != nil {
		params := url.Values{}
		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.PageSize > 0 {
			params.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.EnvironmentID != "" {
			params.Set("environment_id", opts.EnvironmentID)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var resp DeploymentListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns a deployment by ID
func (s *DeploymentsService) Get(ctx context.Context, id string) (*Deployment, error) {
	var dep Deployment
	if err := s.client.request(ctx, "GET", "/deployments/"+id, nil, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// Create creates a new deployment
func (s *DeploymentsService) Create(ctx context.Context, req *CreateDeploymentRequest) (*Deployment, error) {
	var dep Deployment
	if err := s.client.request(ctx, "POST", "/deployments", req, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// Rollback rolls back a deployment
func (s *DeploymentsService) Rollback(ctx context.Context, id string) (*Deployment, error) {
	var dep Deployment
	if err := s.client.request(ctx, "POST", "/deployments/"+id+"/rollback", nil, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// GetLogs returns logs for a deployment
func (s *DeploymentsService) GetLogs(ctx context.Context, id string) (*DeploymentLogs, error) {
	var logs DeploymentLogs
	if err := s.client.request(ctx, "GET", "/deployments/"+id+"/logs", nil, &logs); err != nil {
		return nil, err
	}
	return &logs, nil
}
