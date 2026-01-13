// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CustomDomainServiceConfig holds configuration for the custom domain service
type CustomDomainServiceConfig struct {
	CNAMETarget           string
	VerificationTimeout   time.Duration
	DNSCheckInterval      time.Duration
	MaxVerificationAttempts int
	SSLProvisioningTimeout time.Duration
}

// DefaultCustomDomainServiceConfig returns default configuration
func DefaultCustomDomainServiceConfig() *CustomDomainServiceConfig {
	return &CustomDomainServiceConfig{
		CNAMETarget:           "proxy.devbox.com",
		VerificationTimeout:   5 * time.Minute,
		DNSCheckInterval:      10 * time.Second,
		MaxVerificationAttempts: 30,
		SSLProvisioningTimeout: 10 * time.Minute,
	}
}

// CustomDomainService handles custom domain operations
type CustomDomainService struct {
	config     *CustomDomainServiceConfig
	logger     *zap.Logger
	certManager *CertificateManager

	// Verification jobs
	verificationJobs map[string]*VerificationJob
	mu               sync.RWMutex
}

// VerificationJob represents an ongoing domain verification
type VerificationJob struct {
	ID            string
	DomainID      string
	Domain        string
	Status        string // "pending", "verifying", "verified", "failed"
	Attempts      int
	StartedAt     time.Time
	CompletedAt   *time.Time
	Error         string
	DNSRecords    []DNSCheckResult
	SSLStatus     string
}

// DNSCheckResult represents the result of a DNS check
type DNSCheckResult struct {
	RecordType string
	Expected   string
	Actual     []string
	Valid      bool
	CheckedAt  time.Time
}

// NewCustomDomainService creates a new custom domain service
func NewCustomDomainService(logger *zap.Logger, config *CustomDomainServiceConfig, certManager *CertificateManager) *CustomDomainService {
	if config == nil {
		config = DefaultCustomDomainServiceConfig()
	}

	return &CustomDomainService{
		config:           config,
		logger:           logger.Named("customdomain-service"),
		certManager:      certManager,
		verificationJobs: make(map[string]*VerificationJob),
	}
}

// StartVerification starts the domain verification process
func (s *CustomDomainService) StartVerification(ctx context.Context, domainID, domain string) (*VerificationJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if verification is already in progress
	for _, job := range s.verificationJobs {
		if job.DomainID == domainID && job.Status == "verifying" {
			return job, nil
		}
	}

	job := &VerificationJob{
		ID:        uuid.New().String(),
		DomainID:  domainID,
		Domain:    domain,
		Status:    "pending",
		Attempts:  0,
		StartedAt: time.Now(),
	}

	s.verificationJobs[job.ID] = job

	// Start verification in background
	go s.runVerification(job)

	s.logger.Info("Started domain verification",
		zap.String("jobId", job.ID),
		zap.String("domain", domain),
	)

	return job, nil
}

// runVerification runs the verification process
func (s *CustomDomainService) runVerification(job *VerificationJob) {
	s.mu.Lock()
	job.Status = "verifying"
	s.mu.Unlock()

	ticker := time.NewTicker(s.config.DNSCheckInterval)
	defer ticker.Stop()

	timeout := time.After(s.config.VerificationTimeout)

	for {
		select {
		case <-timeout:
			s.mu.Lock()
			job.Status = "failed"
			job.Error = "verification timeout exceeded"
			now := time.Now()
			job.CompletedAt = &now
			s.mu.Unlock()
			return

		case <-ticker.C:
			s.mu.Lock()
			job.Attempts++
			s.mu.Unlock()

			if job.Attempts > s.config.MaxVerificationAttempts {
				s.mu.Lock()
				job.Status = "failed"
				job.Error = "maximum verification attempts exceeded"
				now := time.Now()
				job.CompletedAt = &now
				s.mu.Unlock()
				return
			}

			// Check DNS records
			result := s.checkDNS(job.Domain)
			
			s.mu.Lock()
			job.DNSRecords = append(job.DNSRecords, result)
			s.mu.Unlock()

			if result.Valid {
				// DNS verified, now provision SSL
				s.mu.Lock()
				job.SSLStatus = "provisioning"
				s.mu.Unlock()

				if err := s.provisionSSL(job); err != nil {
					s.mu.Lock()
					job.Status = "failed"
					job.Error = fmt.Sprintf("SSL provisioning failed: %v", err)
					job.SSLStatus = "failed"
					now := time.Now()
					job.CompletedAt = &now
					s.mu.Unlock()
					return
				}

				s.mu.Lock()
				job.Status = "verified"
				job.SSLStatus = "active"
				now := time.Now()
				job.CompletedAt = &now
				s.mu.Unlock()

				s.logger.Info("Domain verification completed",
					zap.String("jobId", job.ID),
					zap.String("domain", job.Domain),
				)
				return
			}
		}
	}
}

// checkDNS checks DNS records for a domain
func (s *CustomDomainService) checkDNS(domain string) DNSCheckResult {
	result := DNSCheckResult{
		RecordType: "CNAME",
		Expected:   s.config.CNAMETarget,
		CheckedAt:  time.Now(),
	}

	// Perform CNAME lookup
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		s.logger.Debug("CNAME lookup failed",
			zap.String("domain", domain),
			zap.Error(err),
		)
		result.Valid = false
		return result
	}

	// Normalize CNAME (remove trailing dot)
	cname = strings.TrimSuffix(cname, ".")
	result.Actual = []string{cname}

	// Check if CNAME matches expected target
	result.Valid = strings.EqualFold(cname, s.config.CNAMETarget) ||
		strings.HasSuffix(strings.ToLower(cname), "."+strings.ToLower(s.config.CNAMETarget))

	return result
}

// provisionSSL provisions an SSL certificate for the domain
func (s *CustomDomainService) provisionSSL(job *VerificationJob) error {
	if s.certManager == nil {
		// No cert manager, skip SSL provisioning
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.SSLProvisioningTimeout)
	defer cancel()

	_, err := s.certManager.RequestCertificate(ctx, job.Domain)
	return err
}

// GetVerificationJob retrieves a verification job
func (s *CustomDomainService) GetVerificationJob(ctx context.Context, jobID string) (*VerificationJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.verificationJobs[jobID]
	if !ok {
		return nil, fmt.Errorf("verification job not found: %s", jobID)
	}

	return job, nil
}

// GetVerificationJobByDomain retrieves a verification job by domain ID
func (s *CustomDomainService) GetVerificationJobByDomain(ctx context.Context, domainID string) (*VerificationJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.verificationJobs {
		if job.DomainID == domainID {
			return job, nil
		}
	}

	return nil, fmt.Errorf("verification job not found for domain: %s", domainID)
}

// CancelVerification cancels an ongoing verification
func (s *CustomDomainService) CancelVerification(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.verificationJobs[jobID]
	if !ok {
		return fmt.Errorf("verification job not found: %s", jobID)
	}

	if job.Status == "verified" || job.Status == "failed" {
		return fmt.Errorf("verification already completed")
	}

	job.Status = "failed"
	job.Error = "cancelled by user"
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// GetDNSInstructions returns detailed DNS configuration instructions
func (s *CustomDomainService) GetDNSInstructions(domain string) *DNSInstructions {
	return &DNSInstructions{
		Domain:      domain,
		CNAMETarget: s.config.CNAMETarget,
		Steps: []string{
			"1. Log in to your domain registrar or DNS provider (e.g., Cloudflare, GoDaddy, Namecheap)",
			"2. Navigate to DNS management for your domain",
			"3. Add a new DNS record with the following settings:",
			fmt.Sprintf("   - Type: CNAME"),
			fmt.Sprintf("   - Name: %s (or @ for root domain)", getSubdomain(domain)),
			fmt.Sprintf("   - Value/Target: %s", s.config.CNAMETarget),
			"   - TTL: 300 (or Auto)",
			"4. Save the DNS record",
			"5. Wait for DNS propagation (typically 5-30 minutes, up to 48 hours)",
			"6. Return here and click 'Verify Domain' to confirm the configuration",
		},
		Notes: []string{
			"DNS changes may take up to 48 hours to propagate globally",
			"Some DNS providers may require you to remove the domain suffix from the Name field",
			"If using Cloudflare, you can enable the proxy (orange cloud) for additional security",
			"For root domains (@), some providers require an ALIAS or ANAME record instead of CNAME",
		},
		Troubleshooting: []string{
			"If verification fails, check that the CNAME record is correctly configured",
			"Use a DNS lookup tool (e.g., dig, nslookup) to verify the record",
			"Ensure there are no conflicting A or AAAA records for the same hostname",
			"Clear your local DNS cache and try again",
		},
	}
}

// DNSInstructions contains DNS configuration instructions
type DNSInstructions struct {
	Domain          string   `json:"domain"`
	CNAMETarget     string   `json:"cnameTarget"`
	Steps           []string `json:"steps"`
	Notes           []string `json:"notes"`
	Troubleshooting []string `json:"troubleshooting"`
}

// getSubdomain extracts the subdomain from a full domain
func getSubdomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		return parts[0]
	}
	return "@"
}

// ValidateDomainOwnership validates that the user owns the domain
func (s *CustomDomainService) ValidateDomainOwnership(ctx context.Context, domain string) (*DomainOwnershipResult, error) {
	result := &DomainOwnershipResult{
		Domain:    domain,
		CheckedAt: time.Now(),
	}

	// Check CNAME record
	cnameResult := s.checkDNS(domain)
	result.CNAMEValid = cnameResult.Valid
	result.CNAMEActual = cnameResult.Actual

	// Check TXT record for additional verification (optional)
	txtRecords, err := net.LookupTXT(domain)
	if err == nil {
		result.TXTRecords = txtRecords
	}

	// Overall validation
	result.Valid = result.CNAMEValid

	return result, nil
}

// DomainOwnershipResult contains the result of domain ownership validation
type DomainOwnershipResult struct {
	Domain      string    `json:"domain"`
	Valid       bool      `json:"valid"`
	CNAMEValid  bool      `json:"cnameValid"`
	CNAMEActual []string  `json:"cnameActual,omitempty"`
	TXTRecords  []string  `json:"txtRecords,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// CleanupOldJobs removes completed verification jobs older than the specified duration
func (s *CustomDomainService) CleanupOldJobs(ctx context.Context, maxAge time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0

	for id, job := range s.verificationJobs {
		if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(s.verificationJobs, id)
			count++
		}
	}

	if count > 0 {
		s.logger.Info("Cleaned up old verification jobs", zap.Int("count", count))
	}

	return count, nil
}
