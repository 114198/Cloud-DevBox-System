// Package models defines the data models for the container service.
package models

import (
	"time"
)

// DomainStatus represents the status of a domain
type DomainStatus string

const (
	DomainStatusPending   DomainStatus = "Pending"
	DomainStatusActive    DomainStatus = "Active"
	DomainStatusFailed    DomainStatus = "Failed"
	DomainStatusExpired   DomainStatus = "Expired"
	DomainStatusDeleting  DomainStatus = "Deleting"
)

// DomainType represents the type of domain
type DomainType string

const (
	DomainTypeAuto   DomainType = "Auto"   // Auto-assigned subdomain
	DomainTypeCustom DomainType = "Custom" // Custom domain
)

// Domain represents a domain assignment for an environment
type Domain struct {
	ID            string       `json:"id"`
	EnvironmentID string       `json:"environmentId"`
	UserID        string       `json:"userId"`
	Type          DomainType   `json:"type"`
	Subdomain     string       `json:"subdomain"`     // e.g., "abc123" for abc123.preview.devbox.com
	CustomDomain  string       `json:"customDomain,omitempty"` // e.g., "myapp.example.com"
	FullDomain    string       `json:"fullDomain"`    // Full domain URL
	Port          int32        `json:"port"`          // Target port in the environment
	Status        DomainStatus `json:"status"`
	Message       string       `json:"message,omitempty"`
	
	// SSL/TLS configuration
	SSLEnabled    bool         `json:"sslEnabled"`
	SSLCertID     string       `json:"sslCertId,omitempty"`
	SSLExpiresAt  *time.Time   `json:"sslExpiresAt,omitempty"`
	
	// DNS configuration
	DNSRecordID   string       `json:"dnsRecordId,omitempty"`
	DNSVerified   bool         `json:"dnsVerified"`
	CNAMETarget   string       `json:"cnameTarget,omitempty"` // For custom domains
	
	// Timestamps
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	VerifiedAt    *time.Time   `json:"verifiedAt,omitempty"`
}

// ShareLink represents a temporary share link for preview
type ShareLink struct {
	ID            string       `json:"id"`
	DomainID      string       `json:"domainId"`
	EnvironmentID string       `json:"environmentId"`
	UserID        string       `json:"userId"`
	Token         string       `json:"token"`         // Unique token for the link
	URL           string       `json:"url"`           // Full share URL
	
	// Access control
	Password      string       `json:"password,omitempty"` // Optional password protection
	HasPassword   bool         `json:"hasPassword"`
	MaxViews      int          `json:"maxViews,omitempty"` // 0 = unlimited
	ViewCount     int          `json:"viewCount"`
	
	// Validity
	ExpiresAt     time.Time    `json:"expiresAt"`
	Duration      string       `json:"duration"` // e.g., "1h", "24h", "7d", "30d"
	IsActive      bool         `json:"isActive"`
	
	// Timestamps
	CreatedAt     time.Time    `json:"createdAt"`
	LastAccessedAt *time.Time  `json:"lastAccessedAt,omitempty"`
}

// HotReloadConfig represents hot reload configuration
type HotReloadConfig struct {
	Enabled       bool     `json:"enabled"`
	WatchPaths    []string `json:"watchPaths"`    // Paths to watch for changes
	IgnorePaths   []string `json:"ignorePaths"`   // Paths to ignore
	DebounceMs    int      `json:"debounceMs"`    // Debounce time in milliseconds
	NotifyClients bool     `json:"notifyClients"` // Whether to notify connected clients
}

// FileChangeEvent represents a file change event for hot reload
type FileChangeEvent struct {
	EnvironmentID string    `json:"environmentId"`
	Path          string    `json:"path"`
	Type          string    `json:"type"` // "create", "modify", "delete"
	Timestamp     time.Time `json:"timestamp"`
}

// PreviewSession represents an active preview session
type PreviewSession struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	DomainID      string    `json:"domainId"`
	ClientID      string    `json:"clientId"`
	UserAgent     string    `json:"userAgent,omitempty"`
	IPAddress     string    `json:"ipAddress,omitempty"`
	ConnectedAt   time.Time `json:"connectedAt"`
	LastPingAt    time.Time `json:"lastPingAt"`
}


// CreateDomainRequest represents a request to create a domain
type CreateDomainRequest struct {
	EnvironmentID string     `json:"environmentId" binding:"required"`
	Port          int32      `json:"port" binding:"required,min=1,max=65535"`
	Type          DomainType `json:"type,omitempty"` // Defaults to Auto
	CustomDomain  string     `json:"customDomain,omitempty"` // Required if Type is Custom
}

// UpdateDomainRequest represents a request to update a domain
type UpdateDomainRequest struct {
	Port         *int32  `json:"port,omitempty"`
	CustomDomain *string `json:"customDomain,omitempty"`
}

// VerifyDomainRequest represents a request to verify a custom domain
type VerifyDomainRequest struct {
	DomainID string `json:"domainId" binding:"required"`
}

// CreateShareLinkRequest represents a request to create a share link
type CreateShareLinkRequest struct {
	DomainID string `json:"domainId" binding:"required"`
	Duration string `json:"duration" binding:"required"` // "1h", "24h", "7d", "30d"
	Password string `json:"password,omitempty"`
	MaxViews int    `json:"maxViews,omitempty"` // 0 = unlimited
}

// ValidateShareLinkRequest represents a request to validate share link access
type ValidateShareLinkRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password,omitempty"`
}

// ListDomainsRequest represents a request to list domains
type ListDomainsRequest struct {
	EnvironmentID string       `form:"environmentId"`
	UserID        string       `form:"userId"`
	Type          DomainType   `form:"type"`
	Status        DomainStatus `form:"status"`
	Page          int          `form:"page,default=1"`
	PageSize      int          `form:"pageSize,default=20"`
}

// ListDomainsResponse represents a response containing a list of domains
type ListDomainsResponse struct {
	Domains  []Domain `json:"domains"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

// ListShareLinksRequest represents a request to list share links
type ListShareLinksRequest struct {
	DomainID      string `form:"domainId"`
	EnvironmentID string `form:"environmentId"`
	UserID        string `form:"userId"`
	ActiveOnly    bool   `form:"activeOnly"`
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"pageSize,default=20"`
}

// ListShareLinksResponse represents a response containing a list of share links
type ListShareLinksResponse struct {
	ShareLinks []ShareLink `json:"shareLinks"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
}

// DNSRecord represents a DNS record configuration
type DNSRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`      // "A", "CNAME", "TXT"
	Name      string `json:"name"`      // Subdomain or full domain
	Value     string `json:"value"`     // IP address or target domain
	TTL       int    `json:"ttl"`       // Time to live in seconds
	Priority  int    `json:"priority,omitempty"` // For MX records
	Proxied   bool   `json:"proxied"`   // Whether traffic is proxied (e.g., Cloudflare)
}

// SSLCertificate represents an SSL certificate
type SSLCertificate struct {
	ID          string    `json:"id"`
	DomainID    string    `json:"domainId"`
	Domain      string    `json:"domain"`
	Type        string    `json:"type"` // "auto", "custom"
	Status      string    `json:"status"` // "pending", "active", "expired", "failed"
	IssuedAt    time.Time `json:"issuedAt,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	Issuer      string    `json:"issuer,omitempty"`
	Certificate string    `json:"certificate,omitempty"` // PEM encoded (for custom certs)
	PrivateKey  string    `json:"privateKey,omitempty"`  // PEM encoded (for custom certs)
}

// ProxyConfig represents reverse proxy configuration
type ProxyConfig struct {
	DomainID       string            `json:"domainId"`
	UpstreamHost   string            `json:"upstreamHost"`
	UpstreamPort   int32             `json:"upstreamPort"`
	SSLEnabled     bool              `json:"sslEnabled"`
	WebSocketSupport bool            `json:"webSocketSupport"`
	Headers        map[string]string `json:"headers,omitempty"`
	Timeout        int               `json:"timeout"` // in seconds
	MaxBodySize    string            `json:"maxBodySize,omitempty"` // e.g., "10m"
}

// PreviewStats represents preview statistics
type PreviewStats struct {
	TotalDomains      int64 `json:"totalDomains"`
	ActiveDomains     int64 `json:"activeDomains"`
	CustomDomains     int64 `json:"customDomains"`
	TotalShareLinks   int64 `json:"totalShareLinks"`
	ActiveShareLinks  int64 `json:"activeShareLinks"`
	TotalViews        int64 `json:"totalViews"`
}

// DomainValidationResult represents the result of domain validation
type DomainValidationResult struct {
	Valid       bool     `json:"valid"`
	DNSVerified bool     `json:"dnsVerified"`
	SSLValid    bool     `json:"sslValid"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}
