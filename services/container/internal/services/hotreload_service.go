// Package services provides business logic for the container service.
package services

import (
	"context"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HotReloadServiceConfig holds configuration for the hot reload service
type HotReloadServiceConfig struct {
	DefaultDebounceMs    int
	MaxWatchPaths        int
	NotificationTimeout  time.Duration
	MaxSessionsPerDomain int
}

// DefaultHotReloadServiceConfig returns default configuration
func DefaultHotReloadServiceConfig() *HotReloadServiceConfig {
	return &HotReloadServiceConfig{
		DefaultDebounceMs:    300,
		MaxWatchPaths:        100,
		NotificationTimeout:  5 * time.Second,
		MaxSessionsPerDomain: 100,
	}
}

// HotReloadService handles hot reload functionality
type HotReloadService struct {
	config *HotReloadServiceConfig
	logger *zap.Logger

	// Configuration per environment
	configs map[string]*models.HotReloadConfig
	
	// Active preview sessions
	sessions map[string]*models.PreviewSession
	
	// File change event channels per environment
	eventChannels map[string][]chan *models.FileChangeEvent
	
	mu sync.RWMutex
}

// NewHotReloadService creates a new hot reload service
func NewHotReloadService(logger *zap.Logger, config *HotReloadServiceConfig) *HotReloadService {
	if config == nil {
		config = DefaultHotReloadServiceConfig()
	}

	return &HotReloadService{
		config:        config,
		logger:        logger.Named("hotreload-service"),
		configs:       make(map[string]*models.HotReloadConfig),
		sessions:      make(map[string]*models.PreviewSession),
		eventChannels: make(map[string][]chan *models.FileChangeEvent),
	}
}

// GetConfig retrieves hot reload configuration for an environment
func (s *HotReloadService) GetConfig(ctx context.Context, environmentID string) (*models.HotReloadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, ok := s.configs[environmentID]
	if !ok {
		// Return default config
		return &models.HotReloadConfig{
			Enabled:       true,
			WatchPaths:    []string{"."},
			IgnorePaths:   []string{"node_modules", ".git", "dist", "build", "__pycache__", ".venv"},
			DebounceMs:    s.config.DefaultDebounceMs,
			NotifyClients: true,
		}, nil
	}

	return config, nil
}

// SetConfig sets hot reload configuration for an environment
func (s *HotReloadService) SetConfig(ctx context.Context, environmentID string, config *models.HotReloadConfig) error {
	// Validate watch paths
	if len(config.WatchPaths) > s.config.MaxWatchPaths {
		config.WatchPaths = config.WatchPaths[:s.config.MaxWatchPaths]
		s.logger.Warn("Watch paths truncated to max limit",
			zap.String("environmentId", environmentID),
			zap.Int("maxPaths", s.config.MaxWatchPaths),
		)
	}

	s.mu.Lock()
	s.configs[environmentID] = config
	s.mu.Unlock()

	s.logger.Info("Hot reload config updated",
		zap.String("environmentId", environmentID),
		zap.Bool("enabled", config.Enabled),
	)

	return nil
}

// RegisterSession registers a new preview session for hot reload notifications
func (s *HotReloadService) RegisterSession(ctx context.Context, environmentID, domainID, clientID, userAgent, ipAddress string) (*models.PreviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check session limit
	sessionCount := 0
	for _, session := range s.sessions {
		if session.DomainID == domainID {
			sessionCount++
		}
	}

	if sessionCount >= s.config.MaxSessionsPerDomain {
		return nil, nil // Silently reject, don't error
	}

	now := time.Now()
	session := &models.PreviewSession{
		ID:            uuid.New().String(),
		EnvironmentID: environmentID,
		DomainID:      domainID,
		ClientID:      clientID,
		UserAgent:     userAgent,
		IPAddress:     ipAddress,
		ConnectedAt:   now,
		LastPingAt:    now,
	}

	s.sessions[session.ID] = session

	s.logger.Debug("Preview session registered",
		zap.String("sessionId", session.ID),
		zap.String("environmentId", environmentID),
		zap.String("clientId", clientID),
	)

	return session, nil
}

// UnregisterSession removes a preview session
func (s *HotReloadService) UnregisterSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// UpdateSessionPing updates the last ping time for a session
func (s *HotReloadService) UpdateSessionPing(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil // Session not found, ignore
	}

	session.LastPingAt = time.Now()
	return nil
}

// GetActiveSessions returns active sessions for an environment
func (s *HotReloadService) GetActiveSessions(ctx context.Context, environmentID string) ([]*models.PreviewSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.PreviewSession
	cutoff := time.Now().Add(-30 * time.Second) // Sessions inactive for 30s are considered stale

	for _, session := range s.sessions {
		if session.EnvironmentID == environmentID && session.LastPingAt.After(cutoff) {
			result = append(result, session)
		}
	}

	return result, nil
}

// SubscribeToChanges subscribes to file change events for an environment
func (s *HotReloadService) SubscribeToChanges(ctx context.Context, environmentID string) (<-chan *models.FileChangeEvent, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *models.FileChangeEvent, 100)
	s.eventChannels[environmentID] = append(s.eventChannels[environmentID], ch)

	// Return cleanup function
	cleanup := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		channels := s.eventChannels[environmentID]
		for i, c := range channels {
			if c == ch {
				s.eventChannels[environmentID] = append(channels[:i], channels[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, cleanup
}

// NotifyFileChange notifies all subscribers of a file change
func (s *HotReloadService) NotifyFileChange(ctx context.Context, event *models.FileChangeEvent) error {
	s.mu.RLock()
	config, hasConfig := s.configs[event.EnvironmentID]
	channels := s.eventChannels[event.EnvironmentID]
	s.mu.RUnlock()

	// Check if hot reload is enabled
	if hasConfig && !config.Enabled {
		return nil
	}

	// Check if notifications are enabled
	if hasConfig && !config.NotifyClients {
		return nil
	}

	// Check if path should be ignored
	if hasConfig {
		for _, ignorePath := range config.IgnorePaths {
			if matchPath(event.Path, ignorePath) {
				s.logger.Debug("File change ignored",
					zap.String("path", event.Path),
					zap.String("ignorePath", ignorePath),
				)
				return nil
			}
		}
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Notify all subscribers
	notified := 0
	for _, ch := range channels {
		select {
		case ch <- event:
			notified++
		case <-time.After(s.config.NotificationTimeout):
			s.logger.Warn("Notification timeout",
				zap.String("environmentId", event.EnvironmentID),
			)
		}
	}

	s.logger.Debug("File change notified",
		zap.String("environmentId", event.EnvironmentID),
		zap.String("path", event.Path),
		zap.String("type", event.Type),
		zap.Int("subscribers", notified),
	)

	return nil
}

// matchPath checks if a path matches a pattern (simple prefix/suffix matching)
func matchPath(path, pattern string) bool {
	// Simple matching: check if path contains the pattern
	if pattern == "" {
		return false
	}
	
	// Check exact match
	if path == pattern {
		return true
	}
	
	// Check if path starts with pattern (directory match)
	if len(path) > len(pattern) && path[:len(pattern)] == pattern && (path[len(pattern)] == '/' || path[len(pattern)] == '\\') {
		return true
	}
	
	// Check if path contains pattern as a directory component
	for i := 0; i <= len(path)-len(pattern); i++ {
		if path[i:i+len(pattern)] == pattern {
			// Check boundaries
			leftOk := i == 0 || path[i-1] == '/' || path[i-1] == '\\'
			rightOk := i+len(pattern) == len(path) || path[i+len(pattern)] == '/' || path[i+len(pattern)] == '\\'
			if leftOk && rightOk {
				return true
			}
		}
	}
	
	return false
}

// CleanupStaleSessions removes sessions that haven't pinged recently
func (s *HotReloadService) CleanupStaleSessions(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-60 * time.Second) // 60 seconds timeout
	count := 0

	for id, session := range s.sessions {
		if session.LastPingAt.Before(cutoff) {
			delete(s.sessions, id)
			count++
		}
	}

	if count > 0 {
		s.logger.Info("Cleaned up stale sessions", zap.Int("count", count))
	}

	return count, nil
}

// GetSessionCount returns the number of active sessions for an environment
func (s *HotReloadService) GetSessionCount(ctx context.Context, environmentID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	cutoff := time.Now().Add(-30 * time.Second)

	for _, session := range s.sessions {
		if session.EnvironmentID == environmentID && session.LastPingAt.After(cutoff) {
			count++
		}
	}

	return count
}
