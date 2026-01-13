// Package services provides business logic for the container service.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RecoveryObjective defines RTO and RPO targets
type RecoveryObjective struct {
	RTO time.Duration `json:"rto"` // Recovery Time Objective
	RPO time.Duration `json:"rpo"` // Recovery Point Objective
}

// DefaultRecoveryObjective returns default recovery objectives
func DefaultRecoveryObjective() *RecoveryObjective {
	return &RecoveryObjective{
		RTO: 15 * time.Minute, // < 15 minutes
		RPO: 5 * time.Minute,  // < 5 minutes
	}
}

// RegionConfig represents a region configuration
type RegionConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	StoragePath string `json:"storagePath"`
	IsPrimary   bool   `json:"isPrimary"`
	IsActive    bool   `json:"isActive"`
}

// ReplicationStatus represents the status of cross-region replication
type ReplicationStatus string

const (
	ReplicationStatusPending    ReplicationStatus = "pending"
	ReplicationStatusInProgress ReplicationStatus = "in_progress"
	ReplicationStatusCompleted  ReplicationStatus = "completed"
	ReplicationStatusFailed     ReplicationStatus = "failed"
)

// ReplicationRecord represents a replication operation record
type ReplicationRecord struct {
	ID            string            `json:"id"`
	BackupID      string            `json:"backupId"`
	SourceRegion  string            `json:"sourceRegion"`
	TargetRegion  string            `json:"targetRegion"`
	Status        ReplicationStatus `json:"status"`
	Size          int64             `json:"size"`
	BytesTransferred int64          `json:"bytesTransferred"`
	Message       string            `json:"message,omitempty"`
	StartedAt     time.Time         `json:"startedAt"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty"`
}

// FailoverStatus represents the status of a failover operation
type FailoverStatus string

const (
	FailoverStatusIdle       FailoverStatus = "idle"
	FailoverStatusInProgress FailoverStatus = "in_progress"
	FailoverStatusCompleted  FailoverStatus = "completed"
	FailoverStatusFailed     FailoverStatus = "failed"
)

// FailoverRecord represents a failover operation record
type FailoverRecord struct {
	ID            string         `json:"id"`
	FromRegion    string         `json:"fromRegion"`
	ToRegion      string         `json:"toRegion"`
	Status        FailoverStatus `json:"status"`
	Reason        string         `json:"reason"`
	Message       string         `json:"message,omitempty"`
	StartedAt     time.Time      `json:"startedAt"`
	CompletedAt   *time.Time     `json:"completedAt,omitempty"`
	DataLoss      time.Duration  `json:"dataLoss,omitempty"` // Actual RPO achieved
	RecoveryTime  time.Duration  `json:"recoveryTime,omitempty"` // Actual RTO achieved
}

// DisasterRecoveryService handles disaster recovery operations
type DisasterRecoveryService struct {
	logger           *zap.Logger
	objectives       *RecoveryObjective
	primaryRegion    *RegionConfig
	secondaryRegions []*RegionConfig
	
	// Services
	codeBackupSvc     *CodeBackupService
	configBackupSvc   *ConfigBackupService
	databaseBackupSvc *DatabaseBackupService
	
	// In-memory storage (in production, use database)
	replications     map[string]*ReplicationRecord
	failovers        map[string]*FailoverRecord
	mu               sync.RWMutex
	
	// Replication queue
	replicationQueue chan *models.Backup
	
	// Health monitoring
	regionHealth     map[string]bool
	healthMu         sync.RWMutex
	
	// Scheduler
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

// DisasterRecoveryServiceConfig holds configuration for the disaster recovery service
type DisasterRecoveryServiceConfig struct {
	Objectives       *RecoveryObjective
	PrimaryRegion    *RegionConfig
	SecondaryRegions []*RegionConfig
}

// DefaultDisasterRecoveryServiceConfig returns default configuration
func DefaultDisasterRecoveryServiceConfig() *DisasterRecoveryServiceConfig {
	return &DisasterRecoveryServiceConfig{
		Objectives: DefaultRecoveryObjective(),
		PrimaryRegion: &RegionConfig{
			ID:          "cn-north-1",
			Name:        "China North 1",
			Endpoint:    "https://backup.cn-north-1.devbox.com",
			StoragePath: "/var/backups/devbox",
			IsPrimary:   true,
			IsActive:    true,
		},
		SecondaryRegions: []*RegionConfig{
			{
				ID:          "cn-east-1",
				Name:        "China East 1",
				Endpoint:    "https://backup.cn-east-1.devbox.com",
				StoragePath: "/var/backups/devbox-dr",
				IsPrimary:   false,
				IsActive:    true,
			},
		},
	}
}

// NewDisasterRecoveryService creates a new disaster recovery service
func NewDisasterRecoveryService(
	logger *zap.Logger,
	config *DisasterRecoveryServiceConfig,
	codeBackupSvc *CodeBackupService,
	configBackupSvc *ConfigBackupService,
	databaseBackupSvc *DatabaseBackupService,
) *DisasterRecoveryService {
	if config == nil {
		config = DefaultDisasterRecoveryServiceConfig()
	}
	
	svc := &DisasterRecoveryService{
		logger:            logger.Named("disaster-recovery-service"),
		objectives:        config.Objectives,
		primaryRegion:     config.PrimaryRegion,
		secondaryRegions:  config.SecondaryRegions,
		codeBackupSvc:     codeBackupSvc,
		configBackupSvc:   configBackupSvc,
		databaseBackupSvc: databaseBackupSvc,
		replications:      make(map[string]*ReplicationRecord),
		failovers:         make(map[string]*FailoverRecord),
		replicationQueue:  make(chan *models.Backup, 100),
		regionHealth:      make(map[string]bool),
		stopChan:          make(chan struct{}),
	}
	
	// Initialize region health
	svc.regionHealth[config.PrimaryRegion.ID] = true
	for _, region := range config.SecondaryRegions {
		svc.regionHealth[region.ID] = true
	}
	
	return svc
}

// Start starts the disaster recovery service
func (s *DisasterRecoveryService) Start(ctx context.Context) error {
	s.logger.Info("Starting disaster recovery service",
		zap.Duration("rto", s.objectives.RTO),
		zap.Duration("rpo", s.objectives.RPO))
	
	// Start replication worker
	s.wg.Add(1)
	go s.replicationWorker(ctx)
	
	// Start health monitor
	s.wg.Add(1)
	go s.healthMonitor(ctx)
	
	// Start RPO monitor
	s.wg.Add(1)
	go s.rpoMonitor(ctx)
	
	return nil
}

// Stop stops the disaster recovery service
func (s *DisasterRecoveryService) Stop() error {
	s.logger.Info("Stopping disaster recovery service")
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// replicationWorker processes the replication queue
func (s *DisasterRecoveryService) replicationWorker(ctx context.Context) {
	defer s.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case backup := <-s.replicationQueue:
			s.replicateBackup(ctx, backup)
		}
	}
}

// healthMonitor monitors region health
func (s *DisasterRecoveryService) healthMonitor(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkRegionHealth(ctx)
		}
	}
}

// rpoMonitor monitors RPO compliance
func (s *DisasterRecoveryService) rpoMonitor(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkRPOCompliance(ctx)
		}
	}
}


// QueueReplication queues a backup for cross-region replication
func (s *DisasterRecoveryService) QueueReplication(backup *models.Backup) {
	select {
	case s.replicationQueue <- backup:
		s.logger.Debug("Backup queued for replication", zap.String("backupId", backup.ID))
	default:
		s.logger.Warn("Replication queue full, backup not queued", zap.String("backupId", backup.ID))
	}
}

// replicateBackup replicates a backup to all secondary regions
func (s *DisasterRecoveryService) replicateBackup(ctx context.Context, backup *models.Backup) {
	for _, region := range s.secondaryRegions {
		if !region.IsActive {
			continue
		}
		
		record := &ReplicationRecord{
			ID:           uuid.New().String(),
			BackupID:     backup.ID,
			SourceRegion: s.primaryRegion.ID,
			TargetRegion: region.ID,
			Status:       ReplicationStatusPending,
			Size:         backup.Size,
			StartedAt:    time.Now(),
		}
		
		s.mu.Lock()
		s.replications[record.ID] = record
		s.mu.Unlock()
		
		go s.performReplication(ctx, record, backup, region)
	}
}

// performReplication performs the actual replication to a target region
func (s *DisasterRecoveryService) performReplication(ctx context.Context, record *ReplicationRecord, backup *models.Backup, targetRegion *RegionConfig) {
	logger := s.logger.With(
		zap.String("replicationId", record.ID),
		zap.String("backupId", backup.ID),
		zap.String("targetRegion", targetRegion.ID))
	
	logger.Info("Starting backup replication")
	
	// Update status
	s.mu.Lock()
	record.Status = ReplicationStatusInProgress
	s.mu.Unlock()
	
	// Create target directory
	targetPath := filepath.Join(targetRegion.StoragePath, backup.EnvironmentID, backup.ID)
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		s.failReplication(record, fmt.Sprintf("failed to create target directory: %v", err))
		return
	}
	
	// Copy backup file
	if backup.StoragePath != "" {
		srcFile, err := os.Open(backup.StoragePath)
		if err != nil {
			s.failReplication(record, fmt.Sprintf("failed to open source file: %v", err))
			return
		}
		defer srcFile.Close()
		
		targetFile := filepath.Join(targetPath, filepath.Base(backup.StoragePath))
		dstFile, err := os.Create(targetFile)
		if err != nil {
			s.failReplication(record, fmt.Sprintf("failed to create target file: %v", err))
			return
		}
		defer dstFile.Close()
		
		// Copy with progress tracking
		written, err := io.Copy(dstFile, srcFile)
		if err != nil {
			s.failReplication(record, fmt.Sprintf("failed to copy file: %v", err))
			return
		}
		
		s.mu.Lock()
		record.BytesTransferred = written
		s.mu.Unlock()
		
		// Verify checksum
		dstFile.Seek(0, 0)
		h := sha256.New()
		if _, err := io.Copy(h, dstFile); err != nil {
			s.failReplication(record, fmt.Sprintf("failed to calculate checksum: %v", err))
			return
		}
		checksum := hex.EncodeToString(h.Sum(nil))
		
		if checksum != backup.Checksum {
			s.failReplication(record, "checksum mismatch after replication")
			return
		}
	}
	
	// Create metadata file
	metadataPath := filepath.Join(targetPath, "metadata.json")
	metadataJSON, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		s.failReplication(record, fmt.Sprintf("failed to marshal metadata: %v", err))
		return
	}
	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		s.failReplication(record, fmt.Sprintf("failed to write metadata: %v", err))
		return
	}
	
	// Update status
	s.mu.Lock()
	now := time.Now()
	record.Status = ReplicationStatusCompleted
	record.CompletedAt = &now
	s.mu.Unlock()
	
	logger.Info("Backup replication completed",
		zap.Int64("bytesTransferred", record.BytesTransferred),
		zap.Duration("duration", now.Sub(record.StartedAt)))
}

// failReplication marks a replication as failed
func (s *DisasterRecoveryService) failReplication(record *ReplicationRecord, message string) {
	s.logger.Error("Replication failed",
		zap.String("replicationId", record.ID),
		zap.String("message", message))
	
	s.mu.Lock()
	record.Status = ReplicationStatusFailed
	record.Message = message
	s.mu.Unlock()
}

// checkRegionHealth checks the health of all regions
func (s *DisasterRecoveryService) checkRegionHealth(ctx context.Context) {
	// Check primary region
	primaryHealthy := s.pingRegion(ctx, s.primaryRegion)
	
	s.healthMu.Lock()
	s.regionHealth[s.primaryRegion.ID] = primaryHealthy
	s.healthMu.Unlock()
	
	if !primaryHealthy {
		s.logger.Warn("Primary region unhealthy", zap.String("region", s.primaryRegion.ID))
		// Consider automatic failover
		s.considerFailover(ctx)
	}
	
	// Check secondary regions
	for _, region := range s.secondaryRegions {
		healthy := s.pingRegion(ctx, region)
		
		s.healthMu.Lock()
		s.regionHealth[region.ID] = healthy
		s.healthMu.Unlock()
		
		if !healthy {
			s.logger.Warn("Secondary region unhealthy", zap.String("region", region.ID))
		}
	}
}

// pingRegion checks if a region is healthy
func (s *DisasterRecoveryService) pingRegion(ctx context.Context, region *RegionConfig) bool {
	// In production, this would make an HTTP request to the region's health endpoint
	// For now, check if the storage path is accessible
	_, err := os.Stat(region.StoragePath)
	return err == nil
}

// considerFailover considers whether to initiate automatic failover
func (s *DisasterRecoveryService) considerFailover(ctx context.Context) {
	// Find a healthy secondary region
	var targetRegion *RegionConfig
	for _, region := range s.secondaryRegions {
		s.healthMu.RLock()
		healthy := s.regionHealth[region.ID]
		s.healthMu.RUnlock()
		
		if healthy && region.IsActive {
			targetRegion = region
			break
		}
	}
	
	if targetRegion == nil {
		s.logger.Error("No healthy secondary region available for failover")
		return
	}
	
	s.logger.Warn("Considering automatic failover",
		zap.String("fromRegion", s.primaryRegion.ID),
		zap.String("toRegion", targetRegion.ID))
	
	// In production, this might trigger automatic failover or alert operators
}

// checkRPOCompliance checks if RPO is being met
func (s *DisasterRecoveryService) checkRPOCompliance(ctx context.Context) {
	// Get latest replication for each secondary region
	for _, region := range s.secondaryRegions {
		latestReplication := s.getLatestReplication(region.ID)
		
		if latestReplication == nil {
			s.logger.Warn("No replication found for region", zap.String("region", region.ID))
			continue
		}
		
		// Check if replication is within RPO
		var lastReplicationTime time.Time
		if latestReplication.CompletedAt != nil {
			lastReplicationTime = *latestReplication.CompletedAt
		} else {
			lastReplicationTime = latestReplication.StartedAt
		}
		
		timeSinceReplication := time.Since(lastReplicationTime)
		if timeSinceReplication > s.objectives.RPO {
			s.logger.Warn("RPO violation detected",
				zap.String("region", region.ID),
				zap.Duration("timeSinceReplication", timeSinceReplication),
				zap.Duration("rpo", s.objectives.RPO))
		}
	}
}

// getLatestReplication gets the latest completed replication for a region
func (s *DisasterRecoveryService) getLatestReplication(regionID string) *ReplicationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var latest *ReplicationRecord
	for _, record := range s.replications {
		if record.TargetRegion == regionID && record.Status == ReplicationStatusCompleted {
			if latest == nil || record.StartedAt.After(latest.StartedAt) {
				latest = record
			}
		}
	}
	return latest
}

// InitiateFailover initiates a manual failover to a target region
func (s *DisasterRecoveryService) InitiateFailover(ctx context.Context, targetRegionID, reason string) (*FailoverRecord, error) {
	// Find target region
	var targetRegion *RegionConfig
	for _, region := range s.secondaryRegions {
		if region.ID == targetRegionID {
			targetRegion = region
			break
		}
	}
	
	if targetRegion == nil {
		return nil, fmt.Errorf("target region not found: %s", targetRegionID)
	}
	
	// Check if target region is healthy
	s.healthMu.RLock()
	healthy := s.regionHealth[targetRegionID]
	s.healthMu.RUnlock()
	
	if !healthy {
		return nil, fmt.Errorf("target region is not healthy: %s", targetRegionID)
	}
	
	// Create failover record
	record := &FailoverRecord{
		ID:         uuid.New().String(),
		FromRegion: s.primaryRegion.ID,
		ToRegion:   targetRegionID,
		Status:     FailoverStatusInProgress,
		Reason:     reason,
		StartedAt:  time.Now(),
	}
	
	s.mu.Lock()
	s.failovers[record.ID] = record
	s.mu.Unlock()
	
	s.logger.Info("Initiating failover",
		zap.String("failoverId", record.ID),
		zap.String("fromRegion", s.primaryRegion.ID),
		zap.String("toRegion", targetRegionID),
		zap.String("reason", reason))
	
	// Perform failover asynchronously
	go s.performFailover(ctx, record, targetRegion)
	
	return record, nil
}

// performFailover performs the actual failover operation
func (s *DisasterRecoveryService) performFailover(ctx context.Context, record *FailoverRecord, targetRegion *RegionConfig) {
	logger := s.logger.With(zap.String("failoverId", record.ID))
	startTime := time.Now()
	
	// Step 1: Verify target region has latest data
	logger.Info("Verifying target region data")
	latestReplication := s.getLatestReplication(targetRegion.ID)
	if latestReplication == nil {
		s.failFailover(record, "no replicated data found in target region")
		return
	}
	
	// Calculate data loss (RPO achieved)
	var dataLoss time.Duration
	if latestReplication.CompletedAt != nil {
		dataLoss = time.Since(*latestReplication.CompletedAt)
	}
	
	// Step 2: Update region configuration
	logger.Info("Updating region configuration")
	s.primaryRegion.IsActive = false
	targetRegion.IsPrimary = true
	targetRegion.IsActive = true
	
	// Step 3: Update DNS/routing (simulated)
	logger.Info("Updating DNS routing")
	time.Sleep(100 * time.Millisecond) // Simulate DNS update
	
	// Step 4: Verify services are running in target region
	logger.Info("Verifying services in target region")
	if !s.pingRegion(ctx, targetRegion) {
		s.failFailover(record, "target region services not responding")
		return
	}
	
	// Calculate recovery time (RTO achieved)
	recoveryTime := time.Since(startTime)
	
	// Update failover record
	s.mu.Lock()
	now := time.Now()
	record.Status = FailoverStatusCompleted
	record.CompletedAt = &now
	record.DataLoss = dataLoss
	record.RecoveryTime = recoveryTime
	s.mu.Unlock()
	
	// Update primary region reference
	s.primaryRegion = targetRegion
	
	logger.Info("Failover completed",
		zap.Duration("recoveryTime", recoveryTime),
		zap.Duration("dataLoss", dataLoss),
		zap.Bool("rtoMet", recoveryTime <= s.objectives.RTO),
		zap.Bool("rpoMet", dataLoss <= s.objectives.RPO))
}

// failFailover marks a failover as failed
func (s *DisasterRecoveryService) failFailover(record *FailoverRecord, message string) {
	s.logger.Error("Failover failed",
		zap.String("failoverId", record.ID),
		zap.String("message", message))
	
	s.mu.Lock()
	record.Status = FailoverStatusFailed
	record.Message = message
	s.mu.Unlock()
}

// GetFailoverStatus gets the status of a failover operation
func (s *DisasterRecoveryService) GetFailoverStatus(ctx context.Context, id string) (*FailoverRecord, error) {
	s.mu.RLock()
	record, ok := s.failovers[id]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("failover not found: %s", id)
	}
	
	return record, nil
}

// ListFailovers lists all failover records
func (s *DisasterRecoveryService) ListFailovers(ctx context.Context) ([]*FailoverRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	records := make([]*FailoverRecord, 0, len(s.failovers))
	for _, record := range s.failovers {
		records = append(records, record)
	}
	
	return records, nil
}

// GetReplicationStatus gets the status of a replication operation
func (s *DisasterRecoveryService) GetReplicationStatus(ctx context.Context, id string) (*ReplicationRecord, error) {
	s.mu.RLock()
	record, ok := s.replications[id]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("replication not found: %s", id)
	}
	
	return record, nil
}

// ListReplications lists all replication records
func (s *DisasterRecoveryService) ListReplications(ctx context.Context, page, pageSize int) ([]*ReplicationRecord, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	records := make([]*ReplicationRecord, 0, len(s.replications))
	for _, record := range s.replications {
		records = append(records, record)
	}
	
	total := int64(len(records))
	
	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(records) {
		start = len(records)
	}
	if end > len(records) {
		end = len(records)
	}
	
	return records[start:end], total, nil
}

// GetRegionHealth gets the health status of all regions
func (s *DisasterRecoveryService) GetRegionHealth(ctx context.Context) map[string]bool {
	s.healthMu.RLock()
	defer s.healthMu.RUnlock()
	
	health := make(map[string]bool)
	for k, v := range s.regionHealth {
		health[k] = v
	}
	return health
}

// GetRecoveryObjectives gets the current RTO/RPO objectives
func (s *DisasterRecoveryService) GetRecoveryObjectives() *RecoveryObjective {
	return s.objectives
}

// GetDRStatus returns the overall disaster recovery status
func (s *DisasterRecoveryService) GetDRStatus(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	s.healthMu.RLock()
	defer s.mu.RUnlock()
	defer s.healthMu.RUnlock()
	
	// Count replications by status
	replicationStats := make(map[ReplicationStatus]int)
	for _, record := range s.replications {
		replicationStats[record.Status]++
	}
	
	// Get latest replication time
	var latestReplicationTime *time.Time
	for _, record := range s.replications {
		if record.Status == ReplicationStatusCompleted && record.CompletedAt != nil {
			if latestReplicationTime == nil || record.CompletedAt.After(*latestReplicationTime) {
				latestReplicationTime = record.CompletedAt
			}
		}
	}
	
	// Calculate current RPO gap
	var rpoGap time.Duration
	if latestReplicationTime != nil {
		rpoGap = time.Since(*latestReplicationTime)
	}
	
	return map[string]interface{}{
		"primaryRegion":        s.primaryRegion.ID,
		"secondaryRegions":     len(s.secondaryRegions),
		"regionHealth":         s.regionHealth,
		"rto":                  s.objectives.RTO.String(),
		"rpo":                  s.objectives.RPO.String(),
		"currentRpoGap":        rpoGap.String(),
		"rpoCompliant":         rpoGap <= s.objectives.RPO,
		"replicationStats":     replicationStats,
		"latestReplicationTime": latestReplicationTime,
		"totalReplications":    len(s.replications),
		"totalFailovers":       len(s.failovers),
	}
}
