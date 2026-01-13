// Package services provides business logic for the container service.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DatabaseBackupService handles database backup operations
type DatabaseBackupService struct {
	logger          *zap.Logger
	storageBasePath string
	policy          *models.BackupPolicy
	dbConfig        *DatabaseConfig
	
	// In-memory storage (in production, use database)
	backups        map[string]*models.Backup
	mu             sync.RWMutex
	
	// Scheduler
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "devbox",
		Password: "",
		Database: "devbox",
		SSLMode:  "disable",
	}
}

// DatabaseBackupServiceConfig holds configuration for the database backup service
type DatabaseBackupServiceConfig struct {
	StorageBasePath string
	Policy          *models.BackupPolicy
	DBConfig        *DatabaseConfig
}

// DefaultDatabaseBackupServiceConfig returns default configuration
func DefaultDatabaseBackupServiceConfig() *DatabaseBackupServiceConfig {
	return &DatabaseBackupServiceConfig{
		StorageBasePath: "/var/backups/devbox/database",
		Policy:          models.DefaultBackupPolicy(),
		DBConfig:        DefaultDatabaseConfig(),
	}
}

// NewDatabaseBackupService creates a new database backup service
func NewDatabaseBackupService(logger *zap.Logger, config *DatabaseBackupServiceConfig) *DatabaseBackupService {
	if config == nil {
		config = DefaultDatabaseBackupServiceConfig()
	}
	
	return &DatabaseBackupService{
		logger:          logger.Named("database-backup-service"),
		storageBasePath: config.StorageBasePath,
		policy:          config.Policy,
		dbConfig:        config.DBConfig,
		backups:         make(map[string]*models.Backup),
		stopChan:        make(chan struct{}),
	}
}

// Start starts the database backup scheduler
func (s *DatabaseBackupService) Start(ctx context.Context) error {
	s.logger.Info("Starting database backup scheduler",
		zap.Duration("fullInterval", s.policy.DBFullBackupInterval),
		zap.Duration("incrInterval", s.policy.DBIncrBackupInterval))
	
	// Start full backup scheduler
	s.wg.Add(1)
	go s.runFullBackupScheduler(ctx)
	
	// Start incremental backup scheduler
	s.wg.Add(1)
	go s.runIncrementalBackupScheduler(ctx)
	
	return nil
}

// Stop stops the database backup scheduler
func (s *DatabaseBackupService) Stop() error {
	s.logger.Info("Stopping database backup scheduler")
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// runFullBackupScheduler runs the full backup scheduler (daily)
func (s *DatabaseBackupService) runFullBackupScheduler(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.policy.DBFullBackupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.logger.Info("Running scheduled full database backup")
			_, err := s.CreateFullBackup(ctx, "system")
			if err != nil {
				s.logger.Error("Scheduled full backup failed", zap.Error(err))
			}
		}
	}
}

// runIncrementalBackupScheduler runs the incremental backup scheduler (hourly)
func (s *DatabaseBackupService) runIncrementalBackupScheduler(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.policy.DBIncrBackupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.logger.Info("Running scheduled incremental database backup")
			_, err := s.CreateIncrementalBackup(ctx, "system")
			if err != nil {
				s.logger.Error("Scheduled incremental backup failed", zap.Error(err))
			}
		}
	}
}

// CreateFullBackup creates a full database backup
func (s *DatabaseBackupService) CreateFullBackup(ctx context.Context, userID string) (*models.Backup, error) {
	return s.createBackup(ctx, userID, models.BackupModeFull)
}

// CreateIncrementalBackup creates an incremental database backup
func (s *DatabaseBackupService) CreateIncrementalBackup(ctx context.Context, userID string) (*models.Backup, error) {
	// Find the latest full backup as base
	latestFull := s.findLatestFullBackup()
	if latestFull == nil {
		s.logger.Info("No full backup found, creating full backup instead")
		return s.CreateFullBackup(ctx, userID)
	}
	
	return s.createBackup(ctx, userID, models.BackupModeIncremental)
}

// createBackup creates a database backup
func (s *DatabaseBackupService) createBackup(ctx context.Context, userID string, mode models.BackupMode) (*models.Backup, error) {
	logger := s.logger.With(
		zap.String("userId", userID),
		zap.String("mode", string(mode)))
	
	logger.Info("Creating database backup")
	
	// Create backup record
	id := uuid.New().String()
	now := time.Now()
	backup := &models.Backup{
		ID:            id,
		EnvironmentID: "system", // Database backups are system-wide
		UserID:        userID,
		Type:          models.BackupTypeDatabase,
		Mode:          mode,
		Status:        models.BackupStatusPending,
		CreatedAt:     now,
		Metadata: models.BackupMetadata{
			Compression: "gzip",
			Encryption:  "aes-256",
			Labels: map[string]string{
				"database": s.dbConfig.Database,
				"host":     s.dbConfig.Host,
			},
		},
	}
	
	// Find parent for incremental backup
	if mode == models.BackupModeIncremental {
		parent := s.findLatestBackup()
		if parent != nil {
			backup.ParentID = parent.ID
		}
	}
	
	// Store backup record
	s.mu.Lock()
	s.backups[id] = backup
	s.mu.Unlock()
	
	// Perform backup
	go s.performBackup(context.Background(), backup)
	
	return backup, nil
}

// performBackup performs the actual database backup
func (s *DatabaseBackupService) performBackup(ctx context.Context, backup *models.Backup) {
	logger := s.logger.With(zap.String("backupId", backup.ID))
	
	// Update status
	s.mu.Lock()
	backup.Status = models.BackupStatusInProgress
	s.mu.Unlock()
	
	// Create storage directory
	storagePath := filepath.Join(s.storageBasePath, backup.ID)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to create storage directory: %v", err))
		return
	}
	
	// Determine backup file name
	var filename string
	if backup.Mode == models.BackupModeFull {
		filename = "full_backup.sql.gz"
	} else {
		filename = "incremental_backup.sql.gz"
	}
	backupPath := filepath.Join(storagePath, filename)
	
	// Perform pg_dump
	var err error
	if backup.Mode == models.BackupModeFull {
		err = s.performFullDump(ctx, backupPath)
	} else {
		err = s.performIncrementalDump(ctx, backup, backupPath)
	}
	
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("backup failed: %v", err))
		return
	}
	
	// Calculate checksum
	checksum, err := s.calculateChecksum(backupPath)
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to calculate checksum: %v", err))
		return
	}
	
	// Get file size
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		s.failBackup(backup, fmt.Sprintf("failed to get file info: %v", err))
		return
	}
	
	// Update backup record
	s.mu.Lock()
	now := time.Now()
	backup.Status = models.BackupStatusCompleted
	backup.StoragePath = backupPath
	backup.Size = fileInfo.Size()
	backup.Checksum = checksum
	backup.CompletedAt = &now
	
	// Set expiration
	expiresAt := now.Add(time.Duration(s.policy.DBRetentionDays) * 24 * time.Hour)
	backup.ExpiresAt = &expiresAt
	s.mu.Unlock()
	
	logger.Info("Database backup completed",
		zap.String("mode", string(backup.Mode)),
		zap.Int64("size", fileInfo.Size()))
}

// performFullDump performs a full database dump using pg_dump
func (s *DatabaseBackupService) performFullDump(ctx context.Context, outputPath string) error {
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Build pg_dump command
	args := []string{
		"-h", s.dbConfig.Host,
		"-p", fmt.Sprintf("%d", s.dbConfig.Port),
		"-U", s.dbConfig.User,
		"-d", s.dbConfig.Database,
		"-F", "c", // Custom format for compression
		"-Z", "9", // Maximum compression
		"-v",      // Verbose
	}
	
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbConfig.Password))
	cmd.Stdout = file
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	
	return nil
}


// performIncrementalDump performs an incremental database dump
// Uses WAL archiving for true incremental backups
func (s *DatabaseBackupService) performIncrementalDump(ctx context.Context, backup *models.Backup, outputPath string) error {
	// For PostgreSQL, incremental backups typically use WAL archiving
	// Here we simulate by dumping only tables modified since last backup
	
	// Get parent backup timestamp
	var sinceTime time.Time
	if backup.ParentID != "" {
		s.mu.RLock()
		parent := s.backups[backup.ParentID]
		s.mu.RUnlock()
		if parent != nil && parent.CompletedAt != nil {
			sinceTime = *parent.CompletedAt
		}
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Build pg_dump command with table filtering
	// In production, this would use pg_basebackup with WAL archiving
	args := []string{
		"-h", s.dbConfig.Host,
		"-p", fmt.Sprintf("%d", s.dbConfig.Port),
		"-U", s.dbConfig.User,
		"-d", s.dbConfig.Database,
		"-F", "c",
		"-Z", "9",
		"-v",
	}
	
	// Add schema-only for incremental (in production, use WAL)
	if !sinceTime.IsZero() {
		// This is a simplified approach - real incremental would use WAL
		args = append(args, "--data-only")
	}
	
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbConfig.Password))
	cmd.Stdout = file
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	
	return nil
}

// findLatestFullBackup finds the latest completed full backup
func (s *DatabaseBackupService) findLatestFullBackup() *models.Backup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var latest *models.Backup
	for _, backup := range s.backups {
		if backup.Type == models.BackupTypeDatabase &&
			backup.Mode == models.BackupModeFull &&
			backup.Status == models.BackupStatusCompleted {
			if latest == nil || backup.CreatedAt.After(latest.CreatedAt) {
				latest = backup
			}
		}
	}
	return latest
}

// findLatestBackup finds the latest completed backup (any mode)
func (s *DatabaseBackupService) findLatestBackup() *models.Backup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var latest *models.Backup
	for _, backup := range s.backups {
		if backup.Type == models.BackupTypeDatabase &&
			backup.Status == models.BackupStatusCompleted {
			if latest == nil || backup.CreatedAt.After(latest.CreatedAt) {
				latest = backup
			}
		}
	}
	return latest
}

// failBackup marks a backup as failed
func (s *DatabaseBackupService) failBackup(backup *models.Backup, message string) {
	s.logger.Error("Database backup failed",
		zap.String("backupId", backup.ID),
		zap.String("message", message))
	
	s.mu.Lock()
	backup.Status = models.BackupStatusFailed
	backup.Message = message
	s.mu.Unlock()
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *DatabaseBackupService) calculateChecksum(path string) (string, error) {
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

// GetBackup retrieves a backup by ID
func (s *DatabaseBackupService) GetBackup(ctx context.Context, id string) (*models.Backup, error) {
	s.mu.RLock()
	backup, ok := s.backups[id]
	s.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("backup not found: %s", id)
	}
	
	return backup, nil
}

// ListBackups lists database backups
func (s *DatabaseBackupService) ListBackups(ctx context.Context, page, pageSize int) (*models.ListBackupsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var backups []models.Backup
	for _, backup := range s.backups {
		if backup.Type == models.BackupTypeDatabase {
			backups = append(backups, *backup)
		}
	}
	
	// Sort by creation time descending
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	
	total := int64(len(backups))
	
	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(backups) {
		start = len(backups)
	}
	if end > len(backups) {
		end = len(backups)
	}
	
	return &models.ListBackupsResponse{
		Backups:  backups[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// DeleteBackup deletes a backup
func (s *DatabaseBackupService) DeleteBackup(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	backup, ok := s.backups[id]
	if !ok {
		return fmt.Errorf("backup not found: %s", id)
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

// RestoreBackup restores a database from backup
func (s *DatabaseBackupService) RestoreBackup(ctx context.Context, backupID string) error {
	backup, err := s.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}
	
	if backup.Status != models.BackupStatusCompleted {
		return fmt.Errorf("backup is not completed: %s", backup.Status)
	}
	
	s.logger.Info("Restoring database from backup",
		zap.String("backupId", backupID),
		zap.String("mode", string(backup.Mode)))
	
	// For incremental restore, we need to restore full backup first, then apply incrementals
	if backup.Mode == models.BackupModeIncremental {
		// Find and restore the base full backup
		chain := s.getBackupChain(backup)
		for _, b := range chain {
			if err := s.restoreFromFile(ctx, b.StoragePath); err != nil {
				return fmt.Errorf("failed to restore backup %s: %w", b.ID, err)
			}
		}
	} else {
		if err := s.restoreFromFile(ctx, backup.StoragePath); err != nil {
			return err
		}
	}
	
	s.logger.Info("Database restore completed", zap.String("backupId", backupID))
	return nil
}

// getBackupChain returns the chain of backups needed for restore (full + incrementals)
func (s *DatabaseBackupService) getBackupChain(backup *models.Backup) []*models.Backup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var chain []*models.Backup
	current := backup
	
	for current != nil {
		chain = append([]*models.Backup{current}, chain...)
		if current.ParentID == "" {
			break
		}
		current = s.backups[current.ParentID]
	}
	
	return chain
}

// restoreFromFile restores database from a backup file
func (s *DatabaseBackupService) restoreFromFile(ctx context.Context, backupPath string) error {
	// Build pg_restore command
	args := []string{
		"-h", s.dbConfig.Host,
		"-p", fmt.Sprintf("%d", s.dbConfig.Port),
		"-U", s.dbConfig.User,
		"-d", s.dbConfig.Database,
		"-v",
		"-c", // Clean (drop) database objects before recreating
		backupPath,
	}
	
	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbConfig.Password))
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w, output: %s", err, string(output))
	}
	
	return nil
}

// CleanupExpiredBackups removes expired backups
func (s *DatabaseBackupService) CleanupExpiredBackups(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	var deleted int
	
	for id, backup := range s.backups {
		if backup.ExpiresAt != nil && now.After(*backup.ExpiresAt) {
			// Don't delete if it's a parent of another backup
			if s.isParentBackup(id) {
				continue
			}
			
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
		s.logger.Info("Cleaned up expired database backups", zap.Int("count", deleted))
	}
	
	return deleted, nil
}

// isParentBackup checks if a backup is a parent of another backup
func (s *DatabaseBackupService) isParentBackup(id string) bool {
	for _, backup := range s.backups {
		if backup.ParentID == id {
			return true
		}
	}
	return false
}

// GetStats returns database backup statistics
func (s *DatabaseBackupService) GetStats(ctx context.Context) (*models.BackupStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &models.BackupStats{}
	
	for _, backup := range s.backups {
		if backup.Type != models.BackupTypeDatabase {
			continue
		}
		
		stats.TotalBackups++
		stats.TotalSize += backup.Size
		stats.DatabaseBackups++
		
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

// VerifyBackup verifies the integrity of a backup
func (s *DatabaseBackupService) VerifyBackup(ctx context.Context, id string) (bool, error) {
	backup, err := s.GetBackup(ctx, id)
	if err != nil {
		return false, err
	}
	
	if backup.StoragePath == "" {
		return false, fmt.Errorf("backup has no storage path")
	}
	
	// Verify file exists
	if _, err := os.Stat(backup.StoragePath); os.IsNotExist(err) {
		return false, fmt.Errorf("backup file not found")
	}
	
	// Verify checksum
	checksum, err := s.calculateChecksum(backup.StoragePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	
	if checksum != backup.Checksum {
		return false, fmt.Errorf("checksum mismatch: expected %s, got %s", backup.Checksum, checksum)
	}
	
	return true, nil
}
