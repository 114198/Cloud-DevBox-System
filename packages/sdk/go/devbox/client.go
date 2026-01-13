// Package devbox provides a Go SDK for the Cloud DevBox API.
package devbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultBaseURL = "https://api.clouddevbox.io/api/v1"
	DefaultTimeout = 30 * time.Second
)

// Client is the Cloud DevBox API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	userAgent  string
}

// ClientOption is a function that configures the client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new Cloud DevBox API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		userAgent: "CloudDevBox-Go-SDK/1.0",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// request makes an HTTP request to the API
func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// APIError represents an API error response
type APIError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ListOptions contains common list options
type ListOptions struct {
	Page     int
	PageSize int
}

func (o *ListOptions) toQuery() string {
	if o == nil {
		return ""
	}
	params := url.Values{}
	if o.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", o.Page))
	}
	if o.PageSize > 0 {
		params.Set("page_size", fmt.Sprintf("%d", o.PageSize))
	}
	if len(params) > 0 {
		return "?" + params.Encode()
	}
	return ""
}

// Environments returns the environments service
func (c *Client) Environments() *EnvironmentsService {
	return &EnvironmentsService{client: c}
}

// Templates returns the templates service
func (c *Client) Templates() *TemplatesService {
	return &TemplatesService{client: c}
}

// Deployments returns the deployments service
func (c *Client) Deployments() *DeploymentsService {
	return &DeploymentsService{client: c}
}

// APIKeys returns the API keys service
func (c *Client) APIKeys() *APIKeysService {
	return &APIKeysService{client: c}
}

// Webhooks returns the webhooks service
func (c *Client) Webhooks() *WebhooksService {
	return &WebhooksService{client: c}
}
