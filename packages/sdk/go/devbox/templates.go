package devbox

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// TemplatesService handles template operations
type TemplatesService struct {
	client *Client
}

// Template represents an environment template
type Template struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	IconURL          string          `json:"icon_url,omitempty"`
	DefaultResources *ResourceConfig `json:"default_resources,omitempty"`
	Dockerfile       string          `json:"dockerfile,omitempty"`
	SetupCommands    []string        `json:"setup_commands,omitempty"`
	IsOfficial       bool            `json:"is_official"`
	IsPublic         bool            `json:"is_public"`
	Version          string          `json:"version,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CreateTemplateRequest contains parameters for creating a template
type CreateTemplateRequest struct {
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DefaultResources *ResourceConfig `json:"default_resources,omitempty"`
	Dockerfile       string          `json:"dockerfile"`
	SetupCommands    []string        `json:"setup_commands,omitempty"`
	IsPublic         bool            `json:"is_public"`
}

// UpdateTemplateRequest contains parameters for updating a template
type UpdateTemplateRequest struct {
	Name             string          `json:"name,omitempty"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DefaultResources *ResourceConfig `json:"default_resources,omitempty"`
	Dockerfile       string          `json:"dockerfile,omitempty"`
	SetupCommands    []string        `json:"setup_commands,omitempty"`
	IsPublic         *bool           `json:"is_public,omitempty"`
}

// TemplateListOptions contains options for listing templates
type TemplateListOptions struct {
	ListOptions
	Category string
	Search   string
}

// TemplateListResponse contains a paginated list of templates
type TemplateListResponse struct {
	Data       []Template `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// List returns a list of templates
func (s *TemplatesService) List(ctx context.Context, opts *TemplateListOptions) (*TemplateListResponse, error) {
	path := "/templates"
	if opts != nil {
		params := url.Values{}
		if opts.Page > 0 {
			params.Set("page", fmt.Sprintf("%d", opts.Page))
		}
		if opts.PageSize > 0 {
			params.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.Category != "" {
			params.Set("category", opts.Category)
		}
		if opts.Search != "" {
			params.Set("search", opts.Search)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var resp TemplateListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns a template by ID
func (s *TemplatesService) Get(ctx context.Context, id string) (*Template, error) {
	var tmpl Template
	if err := s.client.request(ctx, "GET", "/templates/"+id, nil, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Create creates a new template
func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	var tmpl Template
	if err := s.client.request(ctx, "POST", "/templates", req, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Update updates a template
func (s *TemplatesService) Update(ctx context.Context, id string, req *UpdateTemplateRequest) (*Template, error) {
	var tmpl Template
	if err := s.client.request(ctx, "PUT", "/templates/"+id, req, &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Delete deletes a template
func (s *TemplatesService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", "/templates/"+id, nil, nil)
}
