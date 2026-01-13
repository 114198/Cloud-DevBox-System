// Package services provides business logic for the container service.
package services

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ActivityType represents the type of activity
type ActivityType string

const (
	ActivityTypeSSHConnect    ActivityType = "ssh_connect"
	ActivityTypeSSHDisconnect ActivityType = "ssh_disconnect"
	ActivityTypeAPICall       ActivityType = "api_call"
	ActivityTypeWebTerminal   ActivityType = "web_terminal"
	ActivityTypeFileAccess    ActivityType = "file_access"
	ActivityTypePortAccess    ActivityType = "port_access"
)

// ActivityEvent represents an activity event for an environment
type ActivityEvent struct {
	EnvironmentID string
	UserID        string
	Type          ActivityType
	Timestamp     time.Time
	Metadata      map[string]string
}

// ActivityTracker tracks activity for environments
type ActivityTracker struct {
	envService *EnvironmentService
	logger     *zap.Logger

	// Track active SSH connections per environment
	sshConnections map[string]int
	mu             sync.RWMutex

	// Activity event channel
	events chan *ActivityEvent

	// Stop channel
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewActivityTracker creates a new activity tracker
func NewActivityTracker(envService *EnvironmentService, logger *zap.Logger) *ActivityTracker {
	return &ActivityTracker{
		envService:     envService,
		logger:         logger.Named("activity-tracker"),
		sshConnections: make(map[string]int),
		events:         make(chan *ActivityEvent, 1000),
		stopCh:         make(chan struct{}),
	}
}

// Start starts the activity tracker
func (t *ActivityTracker) Start(ctx context.Context) {
	t.logger.Info("Starting activity tracker")

	t.wg.Add(1)
	go t.processEvents(ctx)
}

// Stop stops the activity tracker
func (t *ActivityTracker) Stop() {
	t.logger.Info("Stopping activity tracker")
	close(t.stopCh)
	t.wg.Wait()
}

// processEvents processes activity events
func (t *ActivityTracker) processEvents(ctx context.Context) {
	defer t.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		case event := <-t.events:
			t.handleEvent(event)
		}
	}
}

// handleEvent handles a single activity event
func (t *ActivityTracker) handleEvent(event *ActivityEvent) {
	logger := t.logger.With(
		zap.String("environmentId", event.EnvironmentID),
		zap.String("type", string(event.Type)),
	)

	// Update last activity time
	if err := t.envService.UpdateActivity(context.Background(), event.EnvironmentID); err != nil {
		logger.Warn("Failed to update activity", zap.Error(err))
		return
	}

	// Handle SSH connection tracking
	switch event.Type {
	case ActivityTypeSSHConnect:
		t.mu.Lock()
		t.sshConnections[event.EnvironmentID]++
		t.mu.Unlock()
		logger.Debug("SSH connection opened", zap.Int("connections", t.sshConnections[event.EnvironmentID]))

	case ActivityTypeSSHDisconnect:
		t.mu.Lock()
		if t.sshConnections[event.EnvironmentID] > 0 {
			t.sshConnections[event.EnvironmentID]--
		}
		t.mu.Unlock()
		logger.Debug("SSH connection closed", zap.Int("connections", t.sshConnections[event.EnvironmentID]))
	}
}

// RecordActivity records an activity event
func (t *ActivityTracker) RecordActivity(environmentID, userID string, activityType ActivityType, metadata map[string]string) {
	event := &ActivityEvent{
		EnvironmentID: environmentID,
		UserID:        userID,
		Type:          activityType,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}

	select {
	case t.events <- event:
	default:
		t.logger.Warn("Activity event channel full, dropping event",
			zap.String("environmentId", environmentID),
			zap.String("type", string(activityType)))
	}
}

// RecordSSHConnect records an SSH connection
func (t *ActivityTracker) RecordSSHConnect(environmentID, userID string) {
	t.RecordActivity(environmentID, userID, ActivityTypeSSHConnect, nil)
}

// RecordSSHDisconnect records an SSH disconnection
func (t *ActivityTracker) RecordSSHDisconnect(environmentID, userID string) {
	t.RecordActivity(environmentID, userID, ActivityTypeSSHDisconnect, nil)
}

// RecordAPICall records an API call
func (t *ActivityTracker) RecordAPICall(environmentID, userID, endpoint string) {
	t.RecordActivity(environmentID, userID, ActivityTypeAPICall, map[string]string{
		"endpoint": endpoint,
	})
}

// RecordWebTerminal records web terminal activity
func (t *ActivityTracker) RecordWebTerminal(environmentID, userID string) {
	t.RecordActivity(environmentID, userID, ActivityTypeWebTerminal, nil)
}

// GetActiveSSHConnections returns the number of active SSH connections for an environment
func (t *ActivityTracker) GetActiveSSHConnections(environmentID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sshConnections[environmentID]
}

// HasActiveConnections checks if an environment has any active connections
func (t *ActivityTracker) HasActiveConnections(environmentID string) bool {
	return t.GetActiveSSHConnections(environmentID) > 0
}

// GetAllActiveConnections returns all environments with active connections
func (t *ActivityTracker) GetAllActiveConnections() map[string]int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]int)
	for envID, count := range t.sshConnections {
		if count > 0 {
			result[envID] = count
		}
	}
	return result
}
