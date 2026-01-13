// Package services provides business logic for the container service.
package services

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// FileWatcherServiceConfig holds configuration for the file watcher service
type FileWatcherServiceConfig struct {
	DefaultDebounceMs int
	MaxWatchedFiles   int
	PollInterval      time.Duration
}

// DefaultFileWatcherServiceConfig returns default configuration
func DefaultFileWatcherServiceConfig() *FileWatcherServiceConfig {
	return &FileWatcherServiceConfig{
		DefaultDebounceMs: 300,
		MaxWatchedFiles:   10000,
		PollInterval:      500 * time.Millisecond,
	}
}

// FileWatcherService handles file system watching for hot reload
type FileWatcherService struct {
	config           *FileWatcherServiceConfig
	logger           *zap.Logger
	hotReloadService *HotReloadService

	// Active watchers per environment
	watchers map[string]*EnvironmentWatcher
	mu       sync.RWMutex
}

// EnvironmentWatcher watches files for a specific environment
type EnvironmentWatcher struct {
	EnvironmentID string
	Config        *models.HotReloadConfig
	Running       bool
	StopChan      chan struct{}
	LastChange    time.Time
	FileHashes    map[string]string // path -> hash for change detection
	mu            sync.RWMutex
}

// NewFileWatcherService creates a new file watcher service
func NewFileWatcherService(logger *zap.Logger, config *FileWatcherServiceConfig, hotReloadService *HotReloadService) *FileWatcherService {
	if config == nil {
		config = DefaultFileWatcherServiceConfig()
	}

	return &FileWatcherService{
		config:           config,
		logger:           logger.Named("filewatcher-service"),
		hotReloadService: hotReloadService,
		watchers:         make(map[string]*EnvironmentWatcher),
	}
}

// StartWatching starts watching files for an environment
func (s *FileWatcherService) StartWatching(ctx context.Context, environmentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already watching
	if watcher, ok := s.watchers[environmentID]; ok && watcher.Running {
		s.logger.Debug("Already watching environment", zap.String("environmentId", environmentID))
		return nil
	}

	// Get hot reload config
	config, err := s.hotReloadService.GetConfig(ctx, environmentID)
	if err != nil {
		return err
	}

	if !config.Enabled {
		s.logger.Debug("Hot reload disabled for environment", zap.String("environmentId", environmentID))
		return nil
	}

	watcher := &EnvironmentWatcher{
		EnvironmentID: environmentID,
		Config:        config,
		Running:       true,
		StopChan:      make(chan struct{}),
		FileHashes:    make(map[string]string),
	}

	s.watchers[environmentID] = watcher

	// Start watching in background
	go s.watchLoop(watcher)

	s.logger.Info("Started watching environment",
		zap.String("environmentId", environmentID),
		zap.Strings("watchPaths", config.WatchPaths),
	)

	return nil
}

// StopWatching stops watching files for an environment
func (s *FileWatcherService) StopWatching(ctx context.Context, environmentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	watcher, ok := s.watchers[environmentID]
	if !ok || !watcher.Running {
		return nil
	}

	close(watcher.StopChan)
	watcher.Running = false
	delete(s.watchers, environmentID)

	s.logger.Info("Stopped watching environment", zap.String("environmentId", environmentID))
	return nil
}

// watchLoop is the main watching loop for an environment
func (s *FileWatcherService) watchLoop(watcher *EnvironmentWatcher) {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	var pendingEvents []*models.FileChangeEvent
	var pendingMu sync.Mutex

	for {
		select {
		case <-watcher.StopChan:
			return

		case <-ticker.C:
			// Poll for changes (in production, use fsnotify or inotify)
			events := s.pollForChanges(watcher)
			if len(events) > 0 {
				pendingMu.Lock()
				pendingEvents = append(pendingEvents, events...)
				pendingMu.Unlock()

				// Reset debounce timer
				debounceTimer.Reset(time.Duration(watcher.Config.DebounceMs) * time.Millisecond)
			}

		case <-debounceTimer.C:
			// Debounce period elapsed, send notifications
			pendingMu.Lock()
			eventsToSend := pendingEvents
			pendingEvents = nil
			pendingMu.Unlock()

			for _, event := range eventsToSend {
				if err := s.hotReloadService.NotifyFileChange(context.Background(), event); err != nil {
					s.logger.Error("Failed to notify file change",
						zap.String("environmentId", watcher.EnvironmentID),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// pollForChanges polls for file changes (simplified implementation)
func (s *FileWatcherService) pollForChanges(watcher *EnvironmentWatcher) []*models.FileChangeEvent {
	// In production, this would use fsnotify or similar
	// For now, return empty (changes would be reported via API)
	return nil
}

// ReportFileChange reports a file change from external source (e.g., container agent)
func (s *FileWatcherService) ReportFileChange(ctx context.Context, environmentID, path, changeType string) error {
	s.mu.RLock()
	watcher, ok := s.watchers[environmentID]
	s.mu.RUnlock()

	if !ok {
		// Not watching this environment, but still notify
		event := &models.FileChangeEvent{
			EnvironmentID: environmentID,
			Path:          path,
			Type:          changeType,
			Timestamp:     time.Now(),
		}
		return s.hotReloadService.NotifyFileChange(ctx, event)
	}

	// Check if path should be ignored
	if s.shouldIgnore(watcher.Config, path) {
		s.logger.Debug("Ignoring file change",
			zap.String("environmentId", environmentID),
			zap.String("path", path),
		)
		return nil
	}

	// Check if path is in watch paths
	if !s.isInWatchPaths(watcher.Config, path) {
		s.logger.Debug("File not in watch paths",
			zap.String("environmentId", environmentID),
			zap.String("path", path),
		)
		return nil
	}

	event := &models.FileChangeEvent{
		EnvironmentID: environmentID,
		Path:          path,
		Type:          changeType,
		Timestamp:     time.Now(),
	}

	return s.hotReloadService.NotifyFileChange(ctx, event)
}

// shouldIgnore checks if a path should be ignored
func (s *FileWatcherService) shouldIgnore(config *models.HotReloadConfig, path string) bool {
	for _, ignorePath := range config.IgnorePaths {
		if matchesPattern(path, ignorePath) {
			return true
		}
	}
	return false
}

// isInWatchPaths checks if a path is in the watch paths
func (s *FileWatcherService) isInWatchPaths(config *models.HotReloadConfig, path string) bool {
	// If watch paths is empty or contains ".", watch everything
	if len(config.WatchPaths) == 0 {
		return true
	}

	for _, watchPath := range config.WatchPaths {
		if watchPath == "." || watchPath == "/" {
			return true
		}
		if matchesPattern(path, watchPath) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a path matches a pattern
func matchesPattern(path, pattern string) bool {
	// Normalize paths
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	// Exact match
	if path == pattern {
		return true
	}

	// Check if path starts with pattern (directory match)
	if strings.HasPrefix(path, pattern+"/") || strings.HasPrefix(path, pattern+"\\") {
		return true
	}

	// Check if pattern is a directory component in path
	pathParts := strings.Split(path, "/")
	for _, part := range pathParts {
		if part == pattern {
			return true
		}
	}

	return false
}

// GetWatcherStatus returns the status of a watcher
func (s *FileWatcherService) GetWatcherStatus(ctx context.Context, environmentID string) (map[string]interface{}, error) {
	s.mu.RLock()
	watcher, ok := s.watchers[environmentID]
	s.mu.RUnlock()

	if !ok {
		return map[string]interface{}{
			"watching": false,
		}, nil
	}

	return map[string]interface{}{
		"watching":      watcher.Running,
		"watchPaths":    watcher.Config.WatchPaths,
		"ignorePaths":   watcher.Config.IgnorePaths,
		"debounceMs":    watcher.Config.DebounceMs,
		"notifyClients": watcher.Config.NotifyClients,
		"lastChange":    watcher.LastChange,
	}, nil
}

// UpdateWatcherConfig updates the watcher configuration
func (s *FileWatcherService) UpdateWatcherConfig(ctx context.Context, environmentID string, config *models.HotReloadConfig) error {
	// Update hot reload service config
	if err := s.hotReloadService.SetConfig(ctx, environmentID, config); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	watcher, ok := s.watchers[environmentID]
	if !ok {
		return nil // Not watching, config will be used when watching starts
	}

	watcher.Config = config

	// If disabled, stop watching
	if !config.Enabled && watcher.Running {
		close(watcher.StopChan)
		watcher.Running = false
		delete(s.watchers, environmentID)
		s.logger.Info("Stopped watching due to config change", zap.String("environmentId", environmentID))
	}

	return nil
}

// GetActiveWatchers returns the number of active watchers
func (s *FileWatcherService) GetActiveWatchers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, watcher := range s.watchers {
		if watcher.Running {
			count++
		}
	}
	return count
}
