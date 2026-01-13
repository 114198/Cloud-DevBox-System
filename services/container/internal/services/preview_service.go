// Package services provides business logic for the container service.
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PreviewServiceConfig holds configuration for the preview service
type PreviewServiceConfig struct {
	BaseDomain          string        // e.g., "preview.devbox.com"
	CNAMETarget         string        // e.g., "proxy.devbox.com"
	DefaultSSLEnabled   bool
	DNSPropagationTime  time.Duration // Expected DNS propagation time
	MaxDomainsPerUser   int
	MaxShareLinksPerDomain int
	ShareLinkMinDuration time.Duration
	ShareLinkMaxDuration time.Duration
}

// DefaultPreviewServiceConfig returns default configuration
func DefaultPreviewServiceConfig() *PreviewServiceConfig {
	return &PreviewServiceConfig{
		BaseDomain:          "preview.devbox.com",
		CNAMETarget:         "proxy.devbox.com",
		DefaultSSLEnabled:   true,
		DNSPropagationTime:  5 * time.Minute,
		MaxDomainsPerUser:   50,
		MaxShareLinksPerDomain: 100,
		ShareLinkMinDuration: time.Hour,
		ShareLinkMaxDuration: 30 * 24 * time.Hour, // 30 days
	}
}

// PreviewService handles preview domain and share link operations
type PreviewService struct {
	config *PreviewServiceConfig
	logger *zap.Logger

	// In-memory storage (in production, use Redis/DB)
	domains    map[string]*models.Domain
	shareLinks map[string]*models.ShareLink
	mu         sync.RWMutex

	// DNS provider interface (mock for now)
	dnsProvider DNSProvider

	// SSL certificate manager interface (mock for now)
	sslManager SSLManager
}

// DNSProvider interface for DNS operations
type DNSProvider interface {
	CreateRecord(ctx context.Context, record *models.DNSRecord) (string, error)
	UpdateRecord(ctx context.Context, recordID string, record *models.DNSRecord) error
	DeleteRecord(ctx context.Context, recordID string) error
	VerifyRecord(ctx context.Context, domain string, expectedValue string) (bool, error)
}

// SSLManager interface for SSL certificate operations
type SSLManager interface {
	RequestCertificate(ctx context.Context, domain string) (*models.SSLCertificate, error)
	GetCertificate(ctx context.Context, certID string) (*models.SSLCertificate, error)
	RevokeCertificate(ctx context.Context, certID string) error
	VerifyCertificate(ctx context.Context, domain string) (bool, error)
}

// MockDNSProvider implements DNSProvider for development
type MockDNSProvider struct {
	records map[string]*models.DNSRecord
	mu      sync.RWMutex
}

// NewMockDNSProvider creates a new mock DNS provider
func NewMockDNSProvider() *MockDNSProvider {
	return &MockDNSProvider{
		records: make(map[string]*models.DNSRecord),
	}
}

func (m *MockDNSProvider) CreateRecord(ctx context.Context, record *models.DNSRecord) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	id := uuid.New().String()
	record.ID = id
	m.records[id] = record
	return id, nil
}

func (m *MockDNSProvider) UpdateRecord(ctx context.Context, recordID string, record *models.DNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.records[recordID]; !ok {
		return fmt.Errorf("record not found: %s", recordID)
	}
	record.ID = recordID
	m.records[recordID] = record
	return nil
}

func (m *MockDNSProvider) DeleteRecord(ctx context.Context, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.records, recordID)
	return nil
}

func (m *MockDNSProvider) VerifyRecord(ctx context.Context, domain string, expectedValue string) (bool, error) {
	// In mock, always return true after a short delay
	return true, nil
}


// MockSSLManager implements SSLManager for development
type MockSSLManager struct {
	certs map[string]*models.SSLCertificate
	mu    sync.RWMutex
}

// NewMockSSLManager creates a new mock SSL manager
func NewMockSSLManager() *MockSSLManager {
	return &MockSSLManager{
		certs: make(map[string]*models.SSLCertificate),
	}
}

func (m *MockSSLManager) RequestCertificate(ctx context.Context, domain string) (*models.SSLCertificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cert := &models.SSLCertificate{
		ID:        uuid.New().String(),
		Domain:    domain,
		Type:      "auto",
		Status:    "active",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 90 days
		Issuer:    "Let's Encrypt",
	}
	m.certs[cert.ID] = cert
	return cert, nil
}

func (m *MockSSLManager) GetCertificate(ctx context.Context, certID string) (*models.SSLCertificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	cert, ok := m.certs[certID]
	if !ok {
		return nil, fmt.Errorf("certificate not found: %s", certID)
	}
	return cert, nil
}

func (m *MockSSLManager) RevokeCertificate(ctx context.Context, certID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.certs, certID)
	return nil
}

func (m *MockSSLManager) VerifyCertificate(ctx context.Context, domain string) (bool, error) {
	return true, nil
}

// NewPreviewService creates a new preview service
func NewPreviewService(logger *zap.Logger, config *PreviewServiceConfig) *PreviewService {
	if config == nil {
		config = DefaultPreviewServiceConfig()
	}
	
	return &PreviewService{
		config:      config,
		logger:      logger.Named("preview-service"),
		domains:     make(map[string]*models.Domain),
		shareLinks:  make(map[string]*models.ShareLink),
		dnsProvider: NewMockDNSProvider(),
		sslManager:  NewMockSSLManager(),
	}
}

// generateSubdomain generates a unique subdomain
func (s *PreviewService) generateSubdomain() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return strings.ToLower(hex.EncodeToString(bytes))
}

// generateShareToken generates a unique share token
func (s *PreviewService) generateShareToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// validateCustomDomain validates a custom domain format
func (s *PreviewService) validateCustomDomain(domain string) error {
	// Basic domain validation regex
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}
	
	// Check for reserved domains
	reservedDomains := []string{"localhost", "devbox.com", "preview.devbox.com"}
	for _, reserved := range reservedDomains {
		if strings.HasSuffix(domain, reserved) {
			return fmt.Errorf("domain %s is reserved", domain)
		}
	}
	
	return nil
}

// parseDuration parses duration string to time.Duration
func (s *PreviewService) parseDuration(duration string) (time.Duration, error) {
	switch duration {
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "14d":
		return 14 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration: %s (valid: 1h, 6h, 12h, 24h, 7d, 14d, 30d)", duration)
	}
}

// CreateDomain creates a new domain for an environment
func (s *PreviewService) CreateDomain(ctx context.Context, userID string, req *models.CreateDomainRequest) (*models.Domain, error) {
	logger := s.logger.With(
		zap.String("userId", userID),
		zap.String("environmentId", req.EnvironmentID),
	)
	logger.Info("Creating domain")

	// Check user domain limit
	s.mu.RLock()
	userDomainCount := 0
	for _, d := range s.domains {
		if d.UserID == userID {
			userDomainCount++
		}
	}
	s.mu.RUnlock()

	if userDomainCount >= s.config.MaxDomainsPerUser {
		return nil, fmt.Errorf("maximum domains per user exceeded (%d)", s.config.MaxDomainsPerUser)
	}

	// Set default type
	domainType := req.Type
	if domainType == "" {
		domainType = models.DomainTypeAuto
	}

	// Validate custom domain if provided
	if domainType == models.DomainTypeCustom {
		if req.CustomDomain == "" {
			return nil, fmt.Errorf("custom domain is required when type is Custom")
		}
		if err := s.validateCustomDomain(req.CustomDomain); err != nil {
			return nil, err
		}
	}

	// Generate subdomain for auto type
	subdomain := ""
	fullDomain := ""
	if domainType == models.DomainTypeAuto {
		subdomain = s.generateSubdomain()
		fullDomain = fmt.Sprintf("%s.%s", subdomain, s.config.BaseDomain)
	} else {
		fullDomain = req.CustomDomain
	}

	now := time.Now()
	domain := &models.Domain{
		ID:            uuid.New().String(),
		EnvironmentID: req.EnvironmentID,
		UserID:        userID,
		Type:          domainType,
		Subdomain:     subdomain,
		CustomDomain:  req.CustomDomain,
		FullDomain:    fullDomain,
		Port:          req.Port,
		Status:        models.DomainStatusPending,
		SSLEnabled:    s.config.DefaultSSLEnabled,
		DNSVerified:   false,
		CNAMETarget:   s.config.CNAMETarget,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Create DNS record for auto domains
	if domainType == models.DomainTypeAuto {
		dnsRecord := &models.DNSRecord{
			Type:    "CNAME",
			Name:    subdomain,
			Value:   s.config.CNAMETarget,
			TTL:     300,
			Proxied: true,
		}
		
		recordID, err := s.dnsProvider.CreateRecord(ctx, dnsRecord)
		if err != nil {
			logger.Error("Failed to create DNS record", zap.Error(err))
			domain.Status = models.DomainStatusFailed
			domain.Message = fmt.Sprintf("DNS record creation failed: %v", err)
		} else {
			domain.DNSRecordID = recordID
			domain.DNSVerified = true
			domain.Status = models.DomainStatusActive
			verifiedAt := now
			domain.VerifiedAt = &verifiedAt
		}
	}

	// Request SSL certificate for active domains
	if domain.Status == models.DomainStatusActive && s.config.DefaultSSLEnabled {
		cert, err := s.sslManager.RequestCertificate(ctx, fullDomain)
		if err != nil {
			logger.Warn("Failed to request SSL certificate", zap.Error(err))
		} else {
			domain.SSLCertID = cert.ID
			domain.SSLExpiresAt = &cert.ExpiresAt
		}
	}

	// Store domain
	s.mu.Lock()
	s.domains[domain.ID] = domain
	s.mu.Unlock()

	logger.Info("Domain created",
		zap.String("domainId", domain.ID),
		zap.String("fullDomain", fullDomain),
		zap.String("status", string(domain.Status)),
	)

	return domain, nil
}


// GetDomain retrieves a domain by ID
func (s *PreviewService) GetDomain(ctx context.Context, userID, domainID string) (*models.Domain, error) {
	s.mu.RLock()
	domain, ok := s.domains[domainID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("domain not found: %s", domainID)
	}

	if domain.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return domain, nil
}

// GetDomainByEnvironment retrieves domains for an environment
func (s *PreviewService) GetDomainByEnvironment(ctx context.Context, userID, environmentID string) ([]*models.Domain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Domain
	for _, domain := range s.domains {
		if domain.EnvironmentID == environmentID && domain.UserID == userID {
			result = append(result, domain)
		}
	}

	return result, nil
}

// ListDomains lists domains with filtering
func (s *PreviewService) ListDomains(ctx context.Context, req *models.ListDomainsRequest) (*models.ListDomainsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*models.Domain
	for _, domain := range s.domains {
		// Filter by user
		if req.UserID != "" && domain.UserID != req.UserID {
			continue
		}
		// Filter by environment
		if req.EnvironmentID != "" && domain.EnvironmentID != req.EnvironmentID {
			continue
		}
		// Filter by type
		if req.Type != "" && domain.Type != req.Type {
			continue
		}
		// Filter by status
		if req.Status != "" && domain.Status != req.Status {
			continue
		}
		filtered = append(filtered, domain)
	}

	// Pagination
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	// Convert to slice of Domain (not pointers) for response
	domains := make([]models.Domain, 0, end-start)
	for _, d := range filtered[start:end] {
		domains = append(domains, *d)
	}

	return &models.ListDomainsResponse{
		Domains:  domains,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateDomain updates a domain
func (s *PreviewService) UpdateDomain(ctx context.Context, userID, domainID string, req *models.UpdateDomainRequest) (*models.Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain, ok := s.domains[domainID]
	if !ok {
		return nil, fmt.Errorf("domain not found: %s", domainID)
	}

	if domain.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Update port
	if req.Port != nil {
		domain.Port = *req.Port
	}

	// Update custom domain (only for custom type)
	if req.CustomDomain != nil && domain.Type == models.DomainTypeCustom {
		if err := s.validateCustomDomain(*req.CustomDomain); err != nil {
			return nil, err
		}
		domain.CustomDomain = *req.CustomDomain
		domain.FullDomain = *req.CustomDomain
		domain.DNSVerified = false
		domain.Status = models.DomainStatusPending
	}

	domain.UpdatedAt = time.Now()
	return domain, nil
}

// DeleteDomain deletes a domain
func (s *PreviewService) DeleteDomain(ctx context.Context, userID, domainID string) error {
	s.mu.Lock()
	domain, ok := s.domains[domainID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("domain not found: %s", domainID)
	}

	if domain.UserID != userID {
		s.mu.Unlock()
		return fmt.Errorf("access denied")
	}

	domain.Status = models.DomainStatusDeleting
	s.mu.Unlock()

	// Delete DNS record
	if domain.DNSRecordID != "" {
		if err := s.dnsProvider.DeleteRecord(ctx, domain.DNSRecordID); err != nil {
			s.logger.Warn("Failed to delete DNS record", zap.Error(err))
		}
	}

	// Revoke SSL certificate
	if domain.SSLCertID != "" {
		if err := s.sslManager.RevokeCertificate(ctx, domain.SSLCertID); err != nil {
			s.logger.Warn("Failed to revoke SSL certificate", zap.Error(err))
		}
	}

	// Delete associated share links
	s.mu.Lock()
	for id, link := range s.shareLinks {
		if link.DomainID == domainID {
			delete(s.shareLinks, id)
		}
	}
	delete(s.domains, domainID)
	s.mu.Unlock()

	return nil
}

// VerifyCustomDomain verifies DNS configuration for a custom domain
func (s *PreviewService) VerifyCustomDomain(ctx context.Context, userID, domainID string) (*models.DomainValidationResult, error) {
	s.mu.Lock()
	domain, ok := s.domains[domainID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("domain not found: %s", domainID)
	}

	if domain.UserID != userID {
		s.mu.Unlock()
		return nil, fmt.Errorf("access denied")
	}

	if domain.Type != models.DomainTypeCustom {
		s.mu.Unlock()
		return nil, fmt.Errorf("only custom domains need verification")
	}
	s.mu.Unlock()

	result := &models.DomainValidationResult{
		Valid:       true,
		DNSVerified: false,
		SSLValid:    false,
	}

	// Verify DNS CNAME record
	dnsVerified, err := s.dnsProvider.VerifyRecord(ctx, domain.CustomDomain, s.config.CNAMETarget)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("DNS verification failed: %v", err))
		result.Valid = false
	} else if !dnsVerified {
		result.Errors = append(result.Errors, fmt.Sprintf("CNAME record not found. Please add a CNAME record pointing %s to %s", domain.CustomDomain, s.config.CNAMETarget))
		result.Valid = false
	} else {
		result.DNSVerified = true
	}

	// Verify SSL certificate
	if result.DNSVerified {
		sslValid, err := s.sslManager.VerifyCertificate(ctx, domain.CustomDomain)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("SSL verification failed: %v", err))
		} else {
			result.SSLValid = sslValid
		}
	}

	// Update domain status if verified
	if result.DNSVerified {
		s.mu.Lock()
		domain.DNSVerified = true
		domain.Status = models.DomainStatusActive
		now := time.Now()
		domain.VerifiedAt = &now
		domain.UpdatedAt = now

		// Request SSL certificate if not already present
		if domain.SSLCertID == "" && s.config.DefaultSSLEnabled {
			cert, err := s.sslManager.RequestCertificate(ctx, domain.CustomDomain)
			if err != nil {
				s.logger.Warn("Failed to request SSL certificate", zap.Error(err))
			} else {
				domain.SSLCertID = cert.ID
				domain.SSLExpiresAt = &cert.ExpiresAt
				result.SSLValid = true
			}
		}
		s.mu.Unlock()
	}

	return result, nil
}

// GetDNSInstructions returns DNS configuration instructions for a custom domain
func (s *PreviewService) GetDNSInstructions(ctx context.Context, userID, domainID string) (map[string]interface{}, error) {
	s.mu.RLock()
	domain, ok := s.domains[domainID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("domain not found: %s", domainID)
	}

	if domain.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	if domain.Type != models.DomainTypeCustom {
		return nil, fmt.Errorf("DNS instructions only available for custom domains")
	}

	instructions := map[string]interface{}{
		"domain":      domain.CustomDomain,
		"cnameTarget": s.config.CNAMETarget,
		"recordType":  "CNAME",
		"ttl":         300,
		"instructions": []string{
			fmt.Sprintf("1. Log in to your domain registrar or DNS provider"),
			fmt.Sprintf("2. Navigate to DNS settings for %s", domain.CustomDomain),
			fmt.Sprintf("3. Add a CNAME record:"),
			fmt.Sprintf("   - Type: CNAME"),
			fmt.Sprintf("   - Name: %s (or @ for root domain)", strings.Split(domain.CustomDomain, ".")[0]),
			fmt.Sprintf("   - Value: %s", s.config.CNAMETarget),
			fmt.Sprintf("   - TTL: 300 (or Auto)"),
			fmt.Sprintf("4. Wait for DNS propagation (usually 5-30 minutes)"),
			fmt.Sprintf("5. Click 'Verify Domain' to confirm the configuration"),
		},
		"verified":    domain.DNSVerified,
		"status":      domain.Status,
	}

	return instructions, nil
}


// CreateShareLink creates a new share link for a domain
func (s *PreviewService) CreateShareLink(ctx context.Context, userID string, req *models.CreateShareLinkRequest) (*models.ShareLink, error) {
	logger := s.logger.With(
		zap.String("userId", userID),
		zap.String("domainId", req.DomainID),
	)
	logger.Info("Creating share link")

	// Get domain
	s.mu.RLock()
	domain, ok := s.domains[req.DomainID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("domain not found: %s", req.DomainID)
	}

	if domain.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	if domain.Status != models.DomainStatusActive {
		return nil, fmt.Errorf("domain is not active")
	}

	// Check share link limit
	s.mu.RLock()
	linkCount := 0
	for _, link := range s.shareLinks {
		if link.DomainID == req.DomainID && link.IsActive {
			linkCount++
		}
	}
	s.mu.RUnlock()

	if linkCount >= s.config.MaxShareLinksPerDomain {
		return nil, fmt.Errorf("maximum share links per domain exceeded (%d)", s.config.MaxShareLinksPerDomain)
	}

	// Parse duration
	duration, err := s.parseDuration(req.Duration)
	if err != nil {
		return nil, err
	}

	// Validate duration bounds
	if duration < s.config.ShareLinkMinDuration {
		return nil, fmt.Errorf("duration must be at least %v", s.config.ShareLinkMinDuration)
	}
	if duration > s.config.ShareLinkMaxDuration {
		return nil, fmt.Errorf("duration must not exceed %v", s.config.ShareLinkMaxDuration)
	}

	now := time.Now()
	token := s.generateShareToken()
	
	shareLink := &models.ShareLink{
		ID:            uuid.New().String(),
		DomainID:      req.DomainID,
		EnvironmentID: domain.EnvironmentID,
		UserID:        userID,
		Token:         token,
		URL:           fmt.Sprintf("https://%s/share/%s", domain.FullDomain, token),
		Password:      req.Password,
		HasPassword:   req.Password != "",
		MaxViews:      req.MaxViews,
		ViewCount:     0,
		ExpiresAt:     now.Add(duration),
		Duration:      req.Duration,
		IsActive:      true,
		CreatedAt:     now,
	}

	// Store share link
	s.mu.Lock()
	s.shareLinks[shareLink.ID] = shareLink
	s.mu.Unlock()

	logger.Info("Share link created",
		zap.String("shareLinkId", shareLink.ID),
		zap.String("token", token),
		zap.Time("expiresAt", shareLink.ExpiresAt),
	)

	return shareLink, nil
}

// GetShareLink retrieves a share link by ID
func (s *PreviewService) GetShareLink(ctx context.Context, userID, shareLinkID string) (*models.ShareLink, error) {
	s.mu.RLock()
	shareLink, ok := s.shareLinks[shareLinkID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("share link not found: %s", shareLinkID)
	}

	if shareLink.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return shareLink, nil
}

// GetShareLinkByToken retrieves a share link by token (for public access)
func (s *PreviewService) GetShareLinkByToken(ctx context.Context, token string) (*models.ShareLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, link := range s.shareLinks {
		if link.Token == token {
			return link, nil
		}
	}

	return nil, fmt.Errorf("share link not found")
}

// ValidateShareLink validates a share link for access
func (s *PreviewService) ValidateShareLink(ctx context.Context, req *models.ValidateShareLinkRequest) (*models.ShareLink, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var shareLink *models.ShareLink
	for _, link := range s.shareLinks {
		if link.Token == req.Token {
			shareLink = link
			break
		}
	}

	if shareLink == nil {
		return nil, fmt.Errorf("share link not found")
	}

	// Check if active
	if !shareLink.IsActive {
		return nil, fmt.Errorf("share link is no longer active")
	}

	// Check expiration
	if time.Now().After(shareLink.ExpiresAt) {
		shareLink.IsActive = false
		return nil, fmt.Errorf("share link has expired")
	}

	// Check view limit
	if shareLink.MaxViews > 0 && shareLink.ViewCount >= shareLink.MaxViews {
		shareLink.IsActive = false
		return nil, fmt.Errorf("share link view limit exceeded")
	}

	// Check password
	if shareLink.HasPassword && shareLink.Password != req.Password {
		return nil, fmt.Errorf("invalid password")
	}

	// Increment view count
	shareLink.ViewCount++
	now := time.Now()
	shareLink.LastAccessedAt = &now

	return shareLink, nil
}

// ListShareLinks lists share links with filtering
func (s *PreviewService) ListShareLinks(ctx context.Context, req *models.ListShareLinksRequest) (*models.ListShareLinksResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*models.ShareLink
	for _, link := range s.shareLinks {
		// Filter by user
		if req.UserID != "" && link.UserID != req.UserID {
			continue
		}
		// Filter by domain
		if req.DomainID != "" && link.DomainID != req.DomainID {
			continue
		}
		// Filter by environment
		if req.EnvironmentID != "" && link.EnvironmentID != req.EnvironmentID {
			continue
		}
		// Filter by active status
		if req.ActiveOnly && !link.IsActive {
			continue
		}
		// Check expiration for active filter
		if req.ActiveOnly && time.Now().After(link.ExpiresAt) {
			continue
		}
		filtered = append(filtered, link)
	}

	// Pagination
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	// Convert to slice of ShareLink (not pointers) for response
	links := make([]models.ShareLink, 0, end-start)
	for _, l := range filtered[start:end] {
		links = append(links, *l)
	}

	return &models.ListShareLinksResponse{
		ShareLinks: links,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

// RevokeShareLink revokes a share link
func (s *PreviewService) RevokeShareLink(ctx context.Context, userID, shareLinkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	shareLink, ok := s.shareLinks[shareLinkID]
	if !ok {
		return fmt.Errorf("share link not found: %s", shareLinkID)
	}

	if shareLink.UserID != userID {
		return fmt.Errorf("access denied")
	}

	shareLink.IsActive = false
	return nil
}

// DeleteShareLink deletes a share link
func (s *PreviewService) DeleteShareLink(ctx context.Context, userID, shareLinkID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	shareLink, ok := s.shareLinks[shareLinkID]
	if !ok {
		return fmt.Errorf("share link not found: %s", shareLinkID)
	}

	if shareLink.UserID != userID {
		return fmt.Errorf("access denied")
	}

	delete(s.shareLinks, shareLinkID)
	return nil
}

// CleanupExpiredShareLinks removes expired share links
func (s *PreviewService) CleanupExpiredShareLinks(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	for id, link := range s.shareLinks {
		if now.After(link.ExpiresAt) {
			delete(s.shareLinks, id)
			count++
		}
	}

	s.logger.Info("Cleaned up expired share links", zap.Int("count", count))
	return count, nil
}

// GetPreviewStats returns preview statistics for a user
func (s *PreviewService) GetPreviewStats(ctx context.Context, userID string) (*models.PreviewStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &models.PreviewStats{}
	now := time.Now()

	for _, domain := range s.domains {
		if domain.UserID != userID {
			continue
		}
		stats.TotalDomains++
		if domain.Status == models.DomainStatusActive {
			stats.ActiveDomains++
		}
		if domain.Type == models.DomainTypeCustom {
			stats.CustomDomains++
		}
	}

	for _, link := range s.shareLinks {
		if link.UserID != userID {
			continue
		}
		stats.TotalShareLinks++
		if link.IsActive && now.Before(link.ExpiresAt) {
			stats.ActiveShareLinks++
		}
		stats.TotalViews += int64(link.ViewCount)
	}

	return stats, nil
}

// GetProxyConfig returns the proxy configuration for a domain
func (s *PreviewService) GetProxyConfig(ctx context.Context, domainID string) (*models.ProxyConfig, error) {
	s.mu.RLock()
	domain, ok := s.domains[domainID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("domain not found: %s", domainID)
	}

	config := &models.ProxyConfig{
		DomainID:         domainID,
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

	return config, nil
}
