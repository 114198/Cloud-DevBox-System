// Package services provides business logic for the container service.
package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// CodeBackupService handles code backup operations
type CodeBackupService struct {
	logger          *zap.Logger
	storageBasePath string
	policy          *models.BackupPolicy
	
	// In-memory storage (in production, use database)
	backups    map[string]*models.Backup
	schedules  map[string]*models.BackupSchedule
	mu         sync.RWMutex
	
	// Backup tracking for incremental backups
	fileHashes map[string]map[string]string // environmentID -> filepath -> hash
	hashMu     sync.RWMutex
	
	// Scheduler
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// CodeBackupServiceConfig holds configuration for the code backup service
type CodeBackupServiceConfig struct {
	StorageBasePath string
	Policy          *models.BackupPolicy
}

// DefaultCodeBackupServiceConfig returns default configuration
func DefaultCodeBackupServiceConfig() *CodeBackupServiceConfig {
	return &CodeBackupServiceConfig{
		StorageBasePath: "/var/backups/devbox/code",
		Policy:          models.DefaultBackupPolicy(),
	}
}

// NewCodeBackupService creates a new code backup service
func NewCodeBackupService(logger *zap.Logger, config *CodeBackupServiceConfig) *CodeBackupService {
	if config == nil {
		config = DefaultCodeBackupServiceConfig()
	}
	
	return &CodeBackupService{
		logger:          logger.Named("code-backup-service"),
		storageBasePath: config.StorageBasePath,
		policy:          config.Policy,
		backups:         make(map[string]*models.Backup),
		schedules:       make(map[string]*models.BackupSchedule),
		fileHashes:      make(map[string]map[string]string),
		stopChan:        make(chan struct{}),
	}
}

// Start starts the backup scheduler
func (s *CodeBackupService) Start(ctx context.Context) error {
	s.logger.Info("Starting code backup scheduler",
		zap.Duration("interval", s.policy.CodeBackupInterval))
	
	s.wg.Add(1)
	go s.runScheduler(ctx)
	
	return nil
}

// Stop stops the backup scheduler
func (s *CodeBackupService) Stop() error {
	s.logger.Info("Stopping code backup scheduler")
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// runScheduler runs the backup scheduler loop
func (s *CodeBackupService) runScheduler(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.policy.CodeBackupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.runScheduledBackups(ctx)
		}
	}
}

// runScheduledBackups runs all scheduled backups
func (s *CodeBackupService) runScheduledBackups(ctx context.Context) {
	s.mu.RLock()
	schedules := make([]*models.BackupSchedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		if schedule.Enabled {
			schedules = append(schedules, schedule)
		}
	}
	s.mu.RUnlock()
	
	for _, schedule := range schedules {
		if schedule.NextRunAt != nil && time.Now().After(*schedule.NextRunAt) {
			s.logger.Info("Running scheduled backup",
				zap.String("scheduleId", schedule.ID),
				zap.String("environmentId", schedule.EnvironmentID))
			
			_, err := s.CreateBackup(ctx, schedule.UserID, &models.CreateBackupRequest{
				EnvironmentID: schedule.EnvironmentID,
				Type:          schedule.Type,
				Mode:          schedule.Mode,
			})
			if err != nil {
				s.logger.Error("Scheduled backup failed",
					zap.String("scheduleId", schedule.ID),
					zap.Error(err))
			}
			
			// Update schedule
			s.mu.Lock()
			now := time.Now()
			schedule.LastRunAt = &now
			nextRun := now.Add(s.policy.CodeBackupInterval)
			schedule.NextRunAt = &nextRun
			s.mu.Unlock()
		}
	}
}


// CreateBackup creates a new code backup
func (s *CodeBackupService) CreateBackup(ctx context.Context, userID string, req *models.CreateBackupRequest) (*models.Backup, error) {
	logger := s.logger.With(
		zap.String("userId", userID),
		zap.String("environmentId", req.EnvironmentID),
		zap.String("type", string(req.Type)))
	
	logger.Info("Creating code backup")
	
	// Determine backup mode
	mode := req.Mode
	if mode == "" {
		mode = models.BackupModeIncremental
	}
	
	// Create backup record
	id := uuid.New().String()
	now := time.Now()
	backup := &models.Backup{
		ID:            id,
		EnvironmentID: req.EnvironmentID,
		UserID:        userID,
		Type:          models.BackupTypeCode,
		Mode:          mode,
		Status:        models.BackupStatusPending,
		CreatedAt:     now,
		Metadata: models.BackupMetadata{
			Compression: "gzip",
			Encryption:  "aes-256",
			Labels:      make(map[string]string),
		},
	}
	
	// Find parent backup for incremental
	if mode == models.BackupModeIncremental {
		parentBackup := s.findLatestBackup(req.EnvironmentID)
		if parentBackup != nil {
			backup.ParentID = parentBackup.ID
		} else {
			// No parent found, do full backup
			backup.Mode = models.BackupModeFull
		}
	}
	
	// Store backup record
	s.mu.Lock()
	s.backups[id] = backup
	s.mu.Unlock()
	
	// Perform backup asynchronously
	go s.performBackup(context.Background(), backup)
	
	return backup, nil
}

// performBackup performs the actual backup operation
func (s *CodeBackupService) performBackup(ctx context.Context, backup *models.Backup) {
	logger := s.logger.With(
		zap.String("backupId", backup.ID),
		zap.String("environmentId", backup.EnvironmentID))
	
	// Update status to in progress
	s.mu.Lock()
	backup.Status = models.BackupStatusInProgress
	s.mu.Unlock()
	
	// Simulate getting source path from environment
	sourcePath := filepath.Join("/var/devbox/environments", backup.EnvironmentID, "workspace")
	backup.Metadata.SourcePath = sourcePath
	
	// Create storage directory
	storagePath := filepath.Join(s.storageBasePath, backup.EnvironmentID, backup.ID)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to create storage directory: %v", err))
		return
	}
	
	// Create backup archive
	archivePath := filepath.Join(storagePath, "backup.tar.gz")
	
	var fileCount, totalSize, changedFiles int64
	var err error
	
	if backup.Mode == models.BackupModeIncremental {
		fileCount, totalSize, changedFiles, err = s.createIncrementalBackup(ctx, backup, sourcePath, archivePath)
	} else {
		fileCount, totalSize, err = s.createFullBackup(ctx, backup, sourcePath, archivePath)
		changedFiles = fileCount
	}
	
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("backup failed: %v", err))
		return
	}
	
	// Calculate checksum
	checksum, err := s.calculateChecksum(archivePath)
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to calculate checksum: %v", err))
		return
	}
	
	// Get archive size
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to get archive info: %v", err))
		return
	}
	
	// Update backup record
	s.mu.Lock()
	now := time.Now()
	backup.Status = models.BackupStatusCompleted
	backup.StoragePath = archivePath
	backup.Size = fileInfo.Size()
	backup.Checksum = checksum
	backup.CompletedAt = &now
	backup.Metadata.FileCount = fileCount
	backup.Metadata.TotalSize = totalSize
	backup.Metadata.ChangedFiles = changedFiles
	
	// Set expiration
	expiresAt := now.Add(time.Duration(s.policy.CodeRetentionDays) * 24 * time.Hour)
	backup.ExpiresAt = &expiresAt
	s.mu.Unlock()
	
	logger.Info("Code backup completed",
		zap.Int64("fileCount", fileCount),
		zap.Int64("changedFiles", changedFiles),
		zap.Int64("size", fileInfo.Size()))
}

// createFullBackup creates a full backup archive
func (s *CodeBackupService) createFullBackup(ctx context.Context, backup *models.Backup, sourcePath, archivePath string) (int64, int64, error) {
	// Create archive file
	file, err := os.Create(archivePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create archive: %w", err)
	}
	defer file.Close()
	
	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	
	var fileCount, totalSize int64
	newHashes := make(map[string]string)
	
	// Walk source directory
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Get relative path
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		
		// Calculate file hash
		hash, err := s.hashFile(path)
		if err != nil {
			return err
		}
		newHashes[relPath] = hash
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath
		
		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// Write file content
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		
		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}
		
		fileCount++
		totalSize += info.Size()
		
		return nil
	})
	
	if err != nil {
		return 0, 0, err
	}
	
	// Update file hashes for incremental backups
	s.hashMu.Lock()
	s.fileHashes[backup.EnvironmentID] = newHashes
	s.hashMu.Unlock()
	
	return fileCount, totalSize, nil
}


// createIncrementalBackup creates an incremental backup with only changed files
func (s *CodeBackupService) createIncrementalBackup(ctx context.Context, backup *models.Backup, sourcePath, archivePath string) (int64, int64, int64, error) {
	// Get previous file hashes
	s.hashMu.RLock()
	prevHashes := s.fileHashes[backup.EnvironmentID]
	s.hashMu.RUnlock()
	
	if prevHashes == nil {
		prevHashes = make(map[string]string)
	}
	
	// Create archive file
	file, err := os.Create(archivePath)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create archive: %w", err)
	}
	defer file.Close()
	
	// Create gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	
	var fileCount, totalSize, changedFiles int64
	newHashes := make(map[string]string)
	
	// Walk source directory
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Get relative path
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		
		// Calculate file hash
		hash, err := s.hashFile(path)
		if err != nil {
			return err
		}
		newHashes[relPath] = hash
		fileCount++
		totalSize += info.Size()
		
		// Check if file changed
		prevHash, exists := prevHashes[relPath]
		if exists && prevHash == hash {
			// File unchanged, skip
			return nil
		}
		
		// File is new or changed, include in backup
		changedFiles++
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath
		
		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// Write file content
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		
		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}
		
		return nil
	})
	
	if err != nil {
		return 0, 0, 0, err
	}
	
	// Track deleted files
	var deletedFiles int64
	for relPath := range prevHashes {
		if _, exists := newHashes[relPath]; !exists {
			deletedFiles++
		}
	}
	
	// Update file hashes
	s.hashMu.Lock()
	s.fileHashes[backup.EnvironmentID] = newHashes
	s.hashMu.Unlock()
	
	// Update backup metadata
	s.mu.Lock()
	backup.Metadata.DeletedFiles = deletedFiles
	s.mu.Unlock()
	
	return fileCount, totalSize, changedFiles, nil
}

// hashFile calculates SHA256 hash of a file
func (s *CodeBackupService) hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(h.Sum(nil)), nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *CodeBackupService) calculateChecksum(path string) (string, error) {
	return s.hashFile(path)
}

// failBackup marks a backup as failed
func (s *CodeBackupService) failBackup(backup *models.Backup, message string) {
	s.logger.Error("Backup failed",
		zap.String("backupId", backup.ID),
		zap.String("message", message))
	
	s.mu.Lock()
	backup.Status = models.BackupStatusFailed
	backup.Message = message
	s.mu.Unlock()
}

// findLatestBackup finds the latest completed backup for an environment
func (s *CodeBackupService) findLatestBackup(environmentID string) *models.Backup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var latest *models.Backup
	for _, backup := range s.backups {
		if backup.EnvironmentID == environmentID &&
			backup.Type == models.BackupTypeCode &&
			backup.Status == models.BackupStatusCompleted {
			if latest == nil || backup.CreatedAt.After(latest.CreatedAt) {
				latest = backup
			}
		}
	}
	return latest
}

// GetBackup retrieves a backup by ID
func (s *CodeBackupService) GetBackup(ctx context.Context, userID, id string) (*models.Backup, error) {
	s.mu.RLock()
	backup, ok := s.backups[id]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("backup not found: %s", id)
	}
	
	if backup.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	
	return backup, nil
}

// ListBackups lists backups for a user or environment
func (s *CodeBackupService) ListBackups(ctx context.Context, req *models.ListBackupsRequest) (*models.ListBackupsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var filtered []models.Backup
	for _, backup := range s.backups {
		// Filter by user
		if req.UserID != "" && backup.UserID != req.UserID {
			continue
		}
		// Filter by environment
		if req.EnvironmentID != "" && backup.EnvironmentID != req.EnvironmentID {
			continue
		}
		// Filter by type
		if req.Type != "" && backup.Type != req.Type {
			continue
		}
		// Filter by status
		if req.Status != "" && backup.Status != req.Status {
			continue
		}
		// Only include code backups
		if backup.Type != models.BackupTypeCode {
			continue
		}
		filtered = append(filtered, *backup)
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
	
	return &models.ListBackupsResponse{
		Backups:  filtered[start:end],
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// DeleteBackup deletes a backup
func (s *CodeBackupService) DeleteBackup(ctx context.Context, userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	backup, ok := s.backups[id]
	if !ok {
		return fmt.Errorf("backup not found: %s", id)
	}
	
	if backup.UserID != userID {
		return fmt.Errorf("access denied")
	}
	
	// Delete storage
	if backup.StoragePath != "" {
		storageDir := filepath.Dir(backup.StoragePath)
		if err := os.RemoveAll(storageDir); err != nil {
			s.logger.Warn("Failed to delete backup storage",
				zap.String("backupId", id),
				zap.Error(err))
		}
	}
	
	delete(s.backups, id)
	return nil
}

// CreateSchedule creates a backup schedule
func (s *CodeBackupService) CreateSchedule(ctx context.Context, userID string, environmentID string) (*models.BackupSchedule, error) {
	id := uuid.New().String()
	now := time.Now()
	nextRun := now.Add(s.policy.CodeBackupInterval)
	
	schedule := &models.BackupSchedule{
		ID:            id,
		EnvironmentID: environmentID,
		UserID:        userID,
		Type:          models.BackupTypeCode,
		Mode:          models.BackupModeIncremental,
		CronExpr:      "0 * * * *", // Every hour
		Enabled:       true,
		RetentionDays: s.policy.CodeRetentionDays,
		NextRunAt:     &nextRun,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	
	s.mu.Lock()
	s.schedules[id] = schedule
	s.mu.Unlock()
	
	s.logger.Info("Created backup schedule",
		zap.String("scheduleId", id),
		zap.String("environmentId", environmentID))
	
	return schedule, nil
}

// GetSchedule retrieves a backup schedule
func (s *CodeBackupService) GetSchedule(ctx context.Context, userID, id string) (*models.BackupSchedule, error) {
	s.mu.RLock()
	schedule, ok := s.schedules[id]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	
	if schedule.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	
	return schedule, nil
}

// EnableSchedule enables a backup schedule
func (s *CodeBackupService) EnableSchedule(ctx context.Context, userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	schedule, ok := s.schedules[id]
	if !ok {
		return fmt.Errorf("schedule not found: %s", id)
	}
	
	if schedule.UserID != userID {
		return fmt.Errorf("access denied")
	}
	
	schedule.Enabled = true
	schedule.UpdatedAt = time.Now()
	return nil
}

// DisableSchedule disables a backup schedule
func (s *CodeBackupService) DisableSchedule(ctx context.Context, userID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	schedule, ok := s.schedules[id]
	if !ok {
		return fmt.Errorf("schedule not found: %s", id)
	}
	
	if schedule.UserID != userID {
		return fmt.Errorf("access denied")
	}
	
	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()
	return nil
}

// CleanupExpiredBackups removes expired backups
func (s *CodeBackupService) CleanupExpiredBackups(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	var deleted int
	
	for id, backup := range s.backups {
		if backup.ExpiresAt != nil && now.After(*backup.ExpiresAt) {
			// Delete storage
			if backup.StoragePath != "" {
				storageDir := filepath.Dir(backup.StoragePath)
				if err := os.RemoveAll(storageDir); err != nil {
					s.logger.Warn("Failed to delete expired backup storage",
						zap.String("backupId", id),
						zap.Error(err))
				}
			}
			delete(s.backups, id)
			deleted++
		}
	}
	
	if deleted > 0 {
		s.logger.Info("Cleaned up expired backups", zap.Int("count", deleted))
	}
	
	return deleted, nil
}

// GetStats returns backup statistics for a user
func (s *CodeBackupService) GetStats(ctx context.Context, userID string) (*models.BackupStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &models.BackupStats{}
	
	for _, backup := range s.backups {
		if backup.UserID != userID {
			continue
		}
		if backup.Type != models.BackupTypeCode {
			continue
		}
		
		stats.TotalBackups++
		stats.TotalSize += backup.Size
		stats.CodeBackups++
		
		if stats.LastBackupAt == nil || backup.CreatedAt.After(*stats.LastBackupAt) {
			t := backup.CreatedAt
			stats.LastBackupAt = &t
		}
		if stats.OldestBackupAt == nil || backup.CreatedAt.Before(*stats.OldestBackupAt) {
			t := backup.CreatedAt
			stats.OldestBackupAt = &t
		}
	}
	
	return stats, nil
}
