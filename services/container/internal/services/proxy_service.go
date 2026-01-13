// Package services provides business logic for the container service.
package services

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProxyServiceConfig holds configuration for the proxy service
type ProxyServiceConfig struct {
	ProxyType           string // "nginx" or "traefik"
	ConfigPath          string
	CertPath            string
	DefaultUpstreamPort int32
	SSLEnabled          bool
	ACMEEmail           string
	ACMEServer          string // "production" or "staging"
}

// DefaultProxyServiceConfig returns default configuration
func DefaultProxyServiceConfig() *ProxyServiceConfig {
	return &ProxyServiceConfig{
		ProxyType:           "traefik",
		ConfigPath:          "/etc/traefik/dynamic",
		CertPath:            "/etc/traefik/certs",
		DefaultUpstreamPort: 3000,
		SSLEnabled:          true,
		ACMEEmail:           "admin@devbox.com",
		ACMEServer:          "production",
	}
}

// ProxyService handles reverse proxy configuration
type ProxyService struct {
	config *ProxyServiceConfig
	logger *zap.Logger

	// In-memory storage for proxy configs
	proxyConfigs map[string]*models.ProxyConfig
	mu           sync.RWMutex

	// Templates
	nginxTemplate   *template.Template
	traefikTemplate *template.Template
}

// NewProxyService creates a new proxy service
func NewProxyService(logger *zap.Logger, config *ProxyServiceConfig) *ProxyService {
	if config == nil {
		config = DefaultProxyServiceConfig()
	}

	service := &ProxyService{
		config:       config,
		logger:       logger.Named("proxy-service"),
		proxyConfigs: make(map[string]*models.ProxyConfig),
	}

	// Initialize templates
	service.initTemplates()

	return service
}

// initTemplates initializes the configuration templates
func (s *ProxyService) initTemplates() {
	// Nginx server block template
	nginxTmpl := `
# Auto-generated Nginx configuration for {{ .Domain }}
# Generated at: {{ .GeneratedAt }}

upstream {{ .UpstreamName }} {
    server {{ .UpstreamHost }}:{{ .UpstreamPort }};
    keepalive 32;
}

server {
    listen 80;
    server_name {{ .Domain }};
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name {{ .Domain }};

    # SSL Configuration
    ssl_certificate {{ .SSLCertPath }};
    ssl_certificate_key {{ .SSLKeyPath }};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Proxy settings
    client_max_body_size {{ .MaxBodySize }};
    proxy_connect_timeout {{ .Timeout }}s;
    proxy_send_timeout {{ .Timeout }}s;
    proxy_read_timeout {{ .Timeout }}s;

    location / {
        proxy_pass http://{{ .UpstreamName }};
        proxy_http_version 1.1;
        
        # Headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        {{- if .WebSocketSupport }}
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        {{- end }}
        
        # Buffering
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
`

	// Traefik dynamic configuration template (YAML)
	traefikTmpl := `
# Auto-generated Traefik configuration for {{ .Domain }}
# Generated at: {{ .GeneratedAt }}

http:
  routers:
    {{ .RouterName }}:
      rule: "Host(` + "`{{ .Domain }}`" + `)"
      service: {{ .ServiceName }}
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
      middlewares:
        - {{ .MiddlewareName }}-headers
        {{- if .RateLimitEnabled }}
        - {{ .MiddlewareName }}-ratelimit
        {{- end }}

  services:
    {{ .ServiceName }}:
      loadBalancer:
        servers:
          - url: "http://{{ .UpstreamHost }}:{{ .UpstreamPort }}"
        healthCheck:
          path: /health
          interval: 10s
          timeout: 3s

  middlewares:
    {{ .MiddlewareName }}-headers:
      headers:
        customRequestHeaders:
          X-Forwarded-Proto: "https"
        customResponseHeaders:
          X-Frame-Options: "SAMEORIGIN"
          X-Content-Type-Options: "nosniff"
          X-XSS-Protection: "1; mode=block"
    {{- if .RateLimitEnabled }}
    {{ .MiddlewareName }}-ratelimit:
      rateLimit:
        average: 100
        burst: 50
    {{- end }}
`

	s.nginxTemplate = template.Must(template.New("nginx").Parse(nginxTmpl))
	s.traefikTemplate = template.Must(template.New("traefik").Parse(traefikTmpl))
}

// ProxyConfigData holds data for template rendering
type ProxyConfigData struct {
	Domain           string
	UpstreamName     string
	UpstreamHost     string
	UpstreamPort     int32
	SSLCertPath      string
	SSLKeyPath       string
	MaxBodySize      string
	Timeout          int
	WebSocketSupport bool
	GeneratedAt      string
	RouterName       string
	ServiceName      string
	MiddlewareName   string
	RateLimitEnabled bool
}

// CreateProxyConfig creates a new proxy configuration for a domain
func (s *ProxyService) CreateProxyConfig(ctx context.Context, domain *models.Domain) (*models.ProxyConfig, error) {
	logger := s.logger.With(
		zap.String("domainId", domain.ID),
		zap.String("domain", domain.FullDomain),
	)
	logger.Info("Creating proxy configuration")

	config := &models.ProxyConfig{
		DomainID:         domain.ID,
		UpstreamHost:     fmt.Sprintf("env-%s.internal", domain.EnvironmentID),
		UpstreamPort:     domain.Port,
		SSLEnabled:       domain.SSLEnabled,
		WebSocketSupport: true,
		Headers: map[string]string{
			"X-Forwarded-Proto": "https",
			"X-Real-IP":         "$remote_addr",
			"X-Forwarded-For":   "$proxy_add_x_forwarded_for",
		},
		Timeout:     60,
		MaxBodySize: "50m",
	}

	// Store config
	s.mu.Lock()
	s.proxyConfigs[domain.ID] = config
	s.mu.Unlock()

	// Generate configuration file
	if err := s.generateConfigFile(domain, config); err != nil {
		logger.Error("Failed to generate config file", zap.Error(err))
		return config, err
	}

	logger.Info("Proxy configuration created")
	return config, nil
}

// generateConfigFile generates the proxy configuration file
func (s *ProxyService) generateConfigFile(domain *models.Domain, config *models.ProxyConfig) error {
	data := ProxyConfigData{
		Domain:           domain.FullDomain,
		UpstreamName:     fmt.Sprintf("upstream_%s", sanitizeName(domain.ID)),
		UpstreamHost:     config.UpstreamHost,
		UpstreamPort:     config.UpstreamPort,
		SSLCertPath:      fmt.Sprintf("%s/%s.crt", s.config.CertPath, domain.FullDomain),
		SSLKeyPath:       fmt.Sprintf("%s/%s.key", s.config.CertPath, domain.FullDomain),
		MaxBodySize:      config.MaxBodySize,
		Timeout:          config.Timeout,
		WebSocketSupport: config.WebSocketSupport,
		GeneratedAt:      time.Now().Format(time.RFC3339),
		RouterName:       fmt.Sprintf("router-%s", sanitizeName(domain.ID)),
		ServiceName:      fmt.Sprintf("service-%s", sanitizeName(domain.ID)),
		MiddlewareName:   fmt.Sprintf("mw-%s", sanitizeName(domain.ID)),
		RateLimitEnabled: true,
	}

	var buf bytes.Buffer
	var err error

	switch s.config.ProxyType {
	case "nginx":
		err = s.nginxTemplate.Execute(&buf, data)
	case "traefik":
		err = s.traefikTemplate.Execute(&buf, data)
	default:
		return fmt.Errorf("unsupported proxy type: %s", s.config.ProxyType)
	}

	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// In production, write to file system
	// For now, just log the generated config
	s.logger.Debug("Generated proxy config",
		zap.String("domain", domain.FullDomain),
		zap.String("config", buf.String()),
	)

	return nil
}

// sanitizeName sanitizes a name for use in configuration
func sanitizeName(name string) string {
	// Replace non-alphanumeric characters with underscores
	result := make([]byte, len(name))
	for i, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result[i] = byte(c)
		} else {
			result[i] = '_'
		}
	}
	return string(result)
}

// UpdateProxyConfig updates an existing proxy configuration
func (s *ProxyService) UpdateProxyConfig(ctx context.Context, domain *models.Domain, config *models.ProxyConfig) error {
	s.mu.Lock()
	s.proxyConfigs[domain.ID] = config
	s.mu.Unlock()

	return s.generateConfigFile(domain, config)
}

// DeleteProxyConfig deletes a proxy configuration
func (s *ProxyService) DeleteProxyConfig(ctx context.Context, domainID string) error {
	s.mu.Lock()
	delete(s.proxyConfigs, domainID)
	s.mu.Unlock()

	// In production, delete the config file
	s.logger.Info("Proxy configuration deleted", zap.String("domainId", domainID))
	return nil
}

// GetProxyConfig retrieves a proxy configuration
func (s *ProxyService) GetProxyConfig(ctx context.Context, domainID string) (*models.ProxyConfig, error) {
	s.mu.RLock()
	config, ok := s.proxyConfigs[domainID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("proxy config not found: %s", domainID)
	}

	return config, nil
}


// SSLCertificateRequest represents a request for SSL certificate
type SSLCertificateRequest struct {
	ID        string
	Domain    string
	Status    string // "pending", "issued", "failed"
	CreatedAt time.Time
	IssuedAt  *time.Time
	ExpiresAt *time.Time
	Error     string
}

// CertificateManager handles SSL certificate management
type CertificateManager struct {
	config   *ProxyServiceConfig
	logger   *zap.Logger
	requests map[string]*SSLCertificateRequest
	mu       sync.RWMutex
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(logger *zap.Logger, config *ProxyServiceConfig) *CertificateManager {
	return &CertificateManager{
		config:   config,
		logger:   logger.Named("cert-manager"),
		requests: make(map[string]*SSLCertificateRequest),
	}
}

// RequestCertificate requests an SSL certificate for a domain
func (m *CertificateManager) RequestCertificate(ctx context.Context, domain string) (*SSLCertificateRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if request already exists
	for _, req := range m.requests {
		if req.Domain == domain && req.Status == "pending" {
			return req, nil
		}
	}

	now := time.Now()
	request := &SSLCertificateRequest{
		ID:        uuid.New().String(),
		Domain:    domain,
		Status:    "pending",
		CreatedAt: now,
	}

	m.requests[request.ID] = request

	// Simulate async certificate issuance
	go m.issueCertificate(request)

	m.logger.Info("Certificate requested",
		zap.String("requestId", request.ID),
		zap.String("domain", domain),
	)

	return request, nil
}

// issueCertificate simulates certificate issuance (in production, use ACME)
func (m *CertificateManager) issueCertificate(request *SSLCertificateRequest) {
	// Simulate processing time
	time.Sleep(2 * time.Second)

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(90 * 24 * time.Hour) // 90 days

	request.Status = "issued"
	request.IssuedAt = &now
	request.ExpiresAt = &expiresAt

	m.logger.Info("Certificate issued",
		zap.String("requestId", request.ID),
		zap.String("domain", request.Domain),
		zap.Time("expiresAt", expiresAt),
	)
}

// GetCertificateStatus gets the status of a certificate request
func (m *CertificateManager) GetCertificateStatus(ctx context.Context, requestID string) (*SSLCertificateRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	request, ok := m.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("certificate request not found: %s", requestID)
	}

	return request, nil
}

// GetCertificateByDomain gets certificate info by domain
func (m *CertificateManager) GetCertificateByDomain(ctx context.Context, domain string) (*SSLCertificateRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, req := range m.requests {
		if req.Domain == domain && req.Status == "issued" {
			return req, nil
		}
	}

	return nil, fmt.Errorf("certificate not found for domain: %s", domain)
}

// RenewCertificate renews an expiring certificate
func (m *CertificateManager) RenewCertificate(ctx context.Context, domain string) (*SSLCertificateRequest, error) {
	m.logger.Info("Renewing certificate", zap.String("domain", domain))
	return m.RequestCertificate(ctx, domain)
}

// CheckExpiringCertificates checks for certificates expiring soon
func (m *CertificateManager) CheckExpiringCertificates(ctx context.Context, threshold time.Duration) ([]*SSLCertificateRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var expiring []*SSLCertificateRequest
	cutoff := time.Now().Add(threshold)

	for _, req := range m.requests {
		if req.Status == "issued" && req.ExpiresAt != nil && req.ExpiresAt.Before(cutoff) {
			expiring = append(expiring, req)
		}
	}

	return expiring, nil
}

// GenerateTraefikStaticConfig generates the static Traefik configuration
func (s *ProxyService) GenerateTraefikStaticConfig() string {
	return fmt.Sprintf(`
# Traefik Static Configuration
# Auto-generated by Cloud DevBox

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: %s
      storage: /etc/traefik/acme.json
      httpChallenge:
        entryPoint: web
      %s

providers:
  file:
    directory: %s
    watch: true

api:
  dashboard: true
  insecure: false

log:
  level: INFO

accessLog:
  filePath: /var/log/traefik/access.log
  format: json
`, s.config.ACMEEmail, s.getACMEServer(), s.config.ConfigPath)
}

// getACMEServer returns the ACME server URL
func (s *ProxyService) getACMEServer() string {
	if s.config.ACMEServer == "staging" {
		return "caServer: https://acme-staging-v02.api.letsencrypt.org/directory"
	}
	return "# Using production Let's Encrypt server"
}

// GenerateNginxMainConfig generates the main Nginx configuration
func (s *ProxyService) GenerateNginxMainConfig() string {
	return `
# Nginx Main Configuration
# Auto-generated by Cloud DevBox

user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 4096;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml;

    # SSL settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    # Include dynamic configurations
    include /etc/nginx/conf.d/*.conf;
}
`
}

// HealthCheck performs a health check on the proxy service
func (s *ProxyService) HealthCheck(ctx context.Context) error {
	// In production, check if proxy is running and responsive
	return nil
}

// ReloadConfig triggers a configuration reload
func (s *ProxyService) ReloadConfig(ctx context.Context) error {
	s.logger.Info("Reloading proxy configuration")
	// In production, send reload signal to proxy
	// For Nginx: nginx -s reload
	// For Traefik: automatic via file watch
	return nil
}
