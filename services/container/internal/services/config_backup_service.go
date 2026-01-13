// Package services provides business logic for the container service.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConfigVersion represents a version of configuration
type ConfigVersion struct {
	ID            string                 `json:"id"`
	EnvironmentID string                 `json:"environmentId"`
	UserID        string                 `json:"userId"`
	Version       int                    `json:"version"`
	Config        map[string]interface{} `json:"config"`
	Checksum      string                 `json:"checksum"`
	Message       string                 `json:"message,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	CreatedBy     string                 `json:"createdBy"`
}

// ConfigDiff represents the difference between two config versions
type ConfigDiff struct {
	FromVersion int                    `json:"fromVersion"`
	ToVersion   int                    `json:"toVersion"`
	Added       map[string]interface{} `json:"added"`
	Removed     map[string]interface{} `json:"removed"`
	Modified    map[string]ConfigChange `json:"modified"`
}

// ConfigChange represents a single config change
type ConfigChange struct {
	OldValue interface{} `json:"oldValue"`
	NewValue interface{} `json:"newValue"`
}

// ConfigBackupService handles configuration backup operations with real-time backup and version history
type ConfigBackupService struct {
	logger          *zap.Logger
	storageBasePath string
	policy          *models.BackupPolicy
	
	// In-memory storage (in production, use database)
	versions       map[string][]*ConfigVersion // environmentID -> versions
	latestVersion  map[string]int              // environmentID -> latest version number
	mu             sync.RWMutex
	
	// Change listeners for real-time backup
	listeners      map[string][]chan *ConfigVersion
	listenerMu     sync.RWMutex
}

// ConfigBackupServiceConfig holds configuration for the config backup service
type ConfigBackupServiceConfig struct {
	StorageBasePath string
	Policy          *models.BackupPolicy
}

// DefaultConfigBackupServiceConfig returns default configuration
func DefaultConfigBackupServiceConfig() *ConfigBackupServiceConfig {
	return &ConfigBackupServiceConfig{
		StorageBasePath: "/var/backups/devbox/config",
		Policy:          models.DefaultBackupPolicy(),
	}
}

// NewConfigBackupService creates a new config backup service
func NewConfigBackupService(logger *zap.Logger, config *ConfigBackupServiceConfig) *ConfigBackupService {
	if config == nil {
		config = DefaultConfigBackupServiceConfig()
	}
	
	return &ConfigBackupService{
		logger:          logger.Named("config-backup-service"),
		storageBasePath: config.StorageBasePath,
		policy:          config.Policy,
		versions:        make(map[string][]*ConfigVersion),
		latestVersion:   make(map[string]int),
		listeners:       make(map[string][]chan *ConfigVersion),
	}
}

// SaveConfig saves a new configuration version (real-time backup)
func (s *ConfigBackupService) SaveConfig(ctx context.Context, userID, environmentID string, config map[string]interface{}, message string) (*ConfigVersion, error) {
	logger := s.logger.With(
		zap.String("userId", userID),
		zap.String("environmentId", environmentID))
	
	// Calculate checksum
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	checksum := s.calculateChecksum(configJSON)
	
	// Check if config actually changed
	s.mu.RLock()
	versions := s.versions[environmentID]
	if len(versions) > 0 {
		latestVersion := versions[len(versions)-1]
		if latestVersion.Checksum == checksum {
			s.mu.RUnlock()
			logger.Debug("Config unchanged, skipping backup")
			return latestVersion, nil
		}
	}
	s.mu.RUnlock()
	
	// Create new version
	s.mu.Lock()
	defer s.mu.Unlock()
	
	versionNum := s.latestVersion[environmentID] + 1
	s.latestVersion[environmentID] = versionNum
	
	version := &ConfigVersion{
		ID:            uuid.New().String(),
		EnvironmentID: environmentID,
		UserID:        userID,
		Version:       versionNum,
		Config:        config,
		Checksum:      checksum,
		Message:       message,
		CreatedAt:     time.Now(),
		CreatedBy:     userID,
	}
	
	// Store version
	s.versions[environmentID] = append(s.versions[environmentID], version)
	
	// Enforce version limit
	if s.policy.ConfigVersionLimit > 0 && len(s.versions[environmentID]) > s.policy.ConfigVersionLimit {
		// Remove oldest versions
		excess := len(s.versions[environmentID]) - s.policy.ConfigVersionLimit
		s.versions[environmentID] = s.versions[environmentID][excess:]
	}
	
	// Persist to storage
	go s.persistVersion(version)
	
	// Notify listeners
	go s.notifyListeners(environmentID, version)
	
	logger.Info("Config version saved",
		zap.Int("version", versionNum),
		zap.String("checksum", checksum))
	
	return version, nil
}

// persistVersion persists a config version to storage
func (s *ConfigBackupService) persistVersion(version *ConfigVersion) {
	storagePath := filepath.Join(s.storageBasePath, version.EnvironmentID)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		s.logger.Error("Failed to create storage directory",
			zap.String("path", storagePath),
			zap.Error(err))
		return
	}
	
	filename := fmt.Sprintf("v%d_%s.json", version.Version, version.ID)
	filePath := filepath.Join(storagePath, filename)
	
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal version",
			zap.String("versionId", version.ID),
			zap.Error(err))
		return
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		s.logger.Error("Failed to write version file",
			zap.String("path", filePath),
			zap.Error(err))
	}
}

// notifyListeners notifies all listeners of a config change
func (s *ConfigBackupService) notifyListeners(environmentID string, version *ConfigVersion) {
	s.listenerMu.RLock()
	listeners := s.listeners[environmentID]
	s.listenerMu.RUnlock()
	
	for _, ch := range listeners {
		select {
		case ch <- version:
		default:
			// Channel full, skip
		}
	}
}

// Subscribe subscribes to config changes for an environment
func (s *ConfigBackupService) Subscribe(environmentID string) <-chan *ConfigVersion {
	ch := make(chan *ConfigVersion, 10)
	
	s.listenerMu.Lock()
	s.listeners[environmentID] = append(s.listeners[environmentID], ch)
	s.listenerMu.Unlock()
	
	return ch
}

// Unsubscribe unsubscribes from config changes
func (s *ConfigBackupService) Unsubscribe(environmentID string, ch <-chan *ConfigVersion) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	
	listeners := s.listeners[environmentID]
	for i, listener := range listeners {
		if listener == ch {
			s.listeners[environmentID] = append(listeners[:i], listeners[i+1:]...)
			close(listener)
			break
		}
	}
}


// GetVersion retrieves a specific config version
func (s *ConfigBackupService) GetVersion(ctx context.Context, userID, environmentID string, versionNum int) (*ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	versions := s.versions[environmentID]
	for _, v := range versions {
		if v.Version == versionNum {
			if v.UserID != userID {
				return nil, fmt.Errorf("access denied")
			}
			return v, nil
		}
	}
	
	return nil, fmt.Errorf("version not found: %d", versionNum)
}

// GetLatestVersion retrieves the latest config version
func (s *ConfigBackupService) GetLatestVersion(ctx context.Context, userID, environmentID string) (*ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	versions := s.versions[environmentID]
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for environment: %s", environmentID)
	}
	
	latest := versions[len(versions)-1]
	if latest.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	
	return latest, nil
}

// ListVersions lists all config versions for an environment
func (s *ConfigBackupService) ListVersions(ctx context.Context, userID, environmentID string, page, pageSize int) ([]*ConfigVersion, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	versions := s.versions[environmentID]
	
	// Filter by user
	var filtered []*ConfigVersion
	for _, v := range versions {
		if v.UserID == userID {
			filtered = append(filtered, v)
		}
	}
	
	// Sort by version descending (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Version > filtered[j].Version
	})
	
	total := int64(len(filtered))
	
	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	
	return filtered[start:end], total, nil
}

// CompareVersions compares two config versions and returns the diff
func (s *ConfigBackupService) CompareVersions(ctx context.Context, userID, environmentID string, fromVersion, toVersion int) (*ConfigDiff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var fromConfig, toConfig *ConfigVersion
	
	versions := s.versions[environmentID]
	for _, v := range versions {
		if v.Version == fromVersion {
			fromConfig = v
		}
		if v.Version == toVersion {
			toConfig = v
		}
	}
	
	if fromConfig == nil {
		return nil, fmt.Errorf("from version not found: %d", fromVersion)
	}
	if toConfig == nil {
		return nil, fmt.Errorf("to version not found: %d", toVersion)
	}
	
	// Check access
	if fromConfig.UserID != userID || toConfig.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	
	diff := &ConfigDiff{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Added:       make(map[string]interface{}),
		Removed:     make(map[string]interface{}),
		Modified:    make(map[string]ConfigChange),
	}
	
	// Find added and modified keys
	for key, newValue := range toConfig.Config {
		oldValue, exists := fromConfig.Config[key]
		if !exists {
			diff.Added[key] = newValue
		} else if !s.deepEqual(oldValue, newValue) {
			diff.Modified[key] = ConfigChange{
				OldValue: oldValue,
				NewValue: newValue,
			}
		}
	}
	
	// Find removed keys
	for key, oldValue := range fromConfig.Config {
		if _, exists := toConfig.Config[key]; !exists {
			diff.Removed[key] = oldValue
		}
	}
	
	return diff, nil
}

// RestoreVersion restores a config to a specific version
func (s *ConfigBackupService) RestoreVersion(ctx context.Context, userID, environmentID string, versionNum int) (*ConfigVersion, error) {
	// Get the version to restore
	targetVersion, err := s.GetVersion(ctx, userID, environmentID, versionNum)
	if err != nil {
		return nil, err
	}
	
	// Create a new version with the restored config
	message := fmt.Sprintf("Restored from version %d", versionNum)
	return s.SaveConfig(ctx, userID, environmentID, targetVersion.Config, message)
}

// DeleteVersion deletes a specific config version
func (s *ConfigBackupService) DeleteVersion(ctx context.Context, userID, environmentID string, versionNum int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	versions := s.versions[environmentID]
	for i, v := range versions {
		if v.Version == versionNum {
			if v.UserID != userID {
				return fmt.Errorf("access denied")
			}
			
			// Remove from slice
			s.versions[environmentID] = append(versions[:i], versions[i+1:]...)
			
			// Delete from storage
			go s.deleteVersionFile(v)
			
			return nil
		}
	}
	
	return fmt.Errorf("version not found: %d", versionNum)
}

// deleteVersionFile deletes a version file from storage
func (s *ConfigBackupService) deleteVersionFile(version *ConfigVersion) {
	storagePath := filepath.Join(s.storageBasePath, version.EnvironmentID)
	filename := fmt.Sprintf("v%d_%s.json", version.Version, version.ID)
	filePath := filepath.Join(storagePath, filename)
	
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("Failed to delete version file",
			zap.String("path", filePath),
			zap.Error(err))
	}
}

// CleanupOldVersions removes versions older than retention period
func (s *ConfigBackupService) CleanupOldVersions(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-time.Duration(s.policy.ConfigRetentionDays) * 24 * time.Hour)
	var deleted int
	
	for envID, versions := range s.versions {
		var kept []*ConfigVersion
		for _, v := range versions {
			if v.CreatedAt.After(cutoff) {
				kept = append(kept, v)
			} else {
				go s.deleteVersionFile(v)
				deleted++
			}
		}
		s.versions[envID] = kept
	}
	
	if deleted > 0 {
		s.logger.Info("Cleaned up old config versions", zap.Int("count", deleted))
	}
	
	return deleted, nil
}

// GetStats returns config backup statistics for a user
func (s *ConfigBackupService) GetStats(ctx context.Context, userID string) (*models.BackupStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &models.BackupStats{}
	
	for _, versions := range s.versions {
		for _, v := range versions {
			if v.UserID != userID {
				continue
			}
			
			stats.TotalBackups++
			stats.ConfigBackups++
			
			// Estimate size from config
			if configJSON, err := json.Marshal(v.Config); err == nil {
				stats.TotalSize += int64(len(configJSON))
			}
			
			if stats.LastBackupAt == nil || v.CreatedAt.After(*stats.LastBackupAt) {
				t := v.CreatedAt
				stats.LastBackupAt = &t
			}
			if stats.OldestBackupAt == nil || v.CreatedAt.Before(*stats.OldestBackupAt) {
				t := v.CreatedAt
				stats.OldestBackupAt = &t
			}
		}
	}
	
	return stats, nil
}

// ExportVersions exports all versions for an environment as JSON
func (s *ConfigBackupService) ExportVersions(ctx context.Context, userID, environmentID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	versions := s.versions[environmentID]
	
	// Filter by user
	var filtered []*ConfigVersion
	for _, v := range versions {
		if v.UserID == userID {
			filtered = append(filtered, v)
		}
	}
	
	return json.MarshalIndent(filtered, "", "  ")
}

// ImportVersions imports versions from JSON
func (s *ConfigBackupService) ImportVersions(ctx context.Context, userID, environmentID string, data []byte) (int, error) {
	var versions []*ConfigVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return 0, fmt.Errorf("failed to unmarshal versions: %w", err)
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	imported := 0
	for _, v := range versions {
		// Update ownership
		v.UserID = userID
		v.EnvironmentID = environmentID
		v.ID = uuid.New().String()
		
		// Update version number
		v.Version = s.latestVersion[environmentID] + 1
		s.latestVersion[environmentID] = v.Version
		
		s.versions[environmentID] = append(s.versions[environmentID], v)
		go s.persistVersion(v)
		imported++
	}
	
	return imported, nil
}

// calculateChecksum calculates SHA256 checksum of data
func (s *ConfigBackupService) calculateChecksum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// deepEqual compares two values for equality
func (s *ConfigBackupService) deepEqual(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
