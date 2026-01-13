// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"go.uber.org/zap"
)

// LifecycleService manages environment lifecycle operations
type LifecycleService struct {
	envService      *EnvironmentService
	activityTracker *ActivityTracker
	stateMachine    *models.StateMachine
	config          *models.LifecycleConfig
	logger          *zap.Logger

	// Notification channel
	notifications chan *models.LifecycleNotification

	// Track pending delete confirmations
	pendingDeletes map[string]*models.DeleteConfirmation
	deleteMu       sync.RWMutex

	// Stop channel for graceful shutdown
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewLifecycleService creates a new lifecycle service
func NewLifecycleService(
	envService *EnvironmentService,
	config *models.LifecycleConfig,
	logger *zap.Logger,
) *LifecycleService {
	if config == nil {
		config = models.DefaultLifecycleConfig()
	}

	return &LifecycleService{
		envService:      envService,
		activityTracker: NewActivityTracker(envService, logger),
		stateMachine:    models.NewStateMachine(),
		config:          config,
		logger:          logger.Named("lifecycle-service"),
		notifications:   make(chan *models.LifecycleNotification, 100),
		pendingDeletes:  make(map[string]*models.DeleteConfirmation),
		stopCh:          make(chan struct{}),
	}
}

// Start starts the lifecycle management background workers
func (s *LifecycleService) Start(ctx context.Context) {
	s.logger.Info("Starting lifecycle service")

	// Start the activity tracker
	s.activityTracker.Start(ctx)

	// Start the lifecycle checker
	s.wg.Add(1)
	go s.runLifecycleChecker(ctx)

	// Start the notification processor
	s.wg.Add(1)
	go s.runNotificationProcessor(ctx)
}

// Stop stops the lifecycle service
func (s *LifecycleService) Stop() {
	s.logger.Info("Stopping lifecycle service")
	close(s.stopCh)
	s.wg.Wait()
	s.activityTracker.Stop()
}

// GetActivityTracker returns the activity tracker
func (s *LifecycleService) GetActivityTracker() *ActivityTracker {
	return s.activityTracker
}

// GetNotifications returns the notification channel for external consumers
func (s *LifecycleService) GetNotifications() <-chan *models.LifecycleNotification {
	return s.notifications
}

// runLifecycleChecker periodically checks environments for lifecycle actions
func (s *LifecycleService) runLifecycleChecker(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAllEnvironments(ctx)
			s.CheckPendingDeletes(ctx)
		}
	}
}

// runNotificationProcessor processes notifications
func (s *LifecycleService) runNotificationProcessor(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case notification := <-s.notifications:
			s.processNotification(notification)
		}
	}
}

// checkAllEnvironments checks all environments for lifecycle actions
func (s *LifecycleService) checkAllEnvironments(ctx context.Context) {
	s.envService.mu.RLock()
	environments := make([]*models.Environment, 0, len(s.envService.environments))
	for _, env := range s.envService.environments {
		environments = append(environments, env)
	}
	s.envService.mu.RUnlock()

	for _, env := range environments {
		result := s.checkEnvironment(env)
		if result.Action != models.LifecycleActionNone {
			s.executeLifecycleAction(ctx, result)
		}
	}
}

// checkEnvironment checks a single environment for lifecycle actions
func (s *LifecycleService) checkEnvironment(env *models.Environment) *models.LifecycleCheckResult {
	now := time.Now()
	result := &models.LifecycleCheckResult{
		EnvironmentID: env.ID,
		Action:        models.LifecycleActionNone,
	}

	switch env.Phase {
	case models.EnvironmentPhaseCreating:
		// Check for creating timeout
		if now.Sub(env.CreatedAt) > s.config.CreatingTimeoutDuration {
			result.Action = models.LifecycleActionCreatingTimeout
			result.Reason = "Environment creation timed out"
			result.ScheduledAt = now
		}

	case models.EnvironmentPhaseRunning:
		// Skip auto-stop if there are active SSH connections
		if s.activityTracker.HasActiveConnections(env.ID) {
			// Update activity time since there are active connections
			s.envService.UpdateActivity(context.Background(), env.ID)
			return result
		}

		// Check for auto-stop due to inactivity
		lastActivity := env.LastActivityTime
		if lastActivity == nil {
			lastActivity = env.StartedAt
		}
		if lastActivity == nil {
			lastActivity = &env.CreatedAt
		}

		inactiveDuration := now.Sub(*lastActivity)
		
		// Check if we should send notification
		notifyThreshold := s.config.AutoStopInactivityDuration - s.config.NotificationLeadTime
		if inactiveDuration >= notifyThreshold && inactiveDuration < s.config.AutoStopInactivityDuration {
			result.Action = models.LifecycleActionNotifyAutoStop
			result.Reason = "Environment will be auto-stopped due to inactivity"
			result.ScheduledAt = lastActivity.Add(s.config.AutoStopInactivityDuration)
		} else if inactiveDuration >= s.config.AutoStopInactivityDuration {
			result.Action = models.LifecycleActionAutoSuspend
			result.Reason = "Environment auto-suspended due to 72 hours of inactivity"
			result.ScheduledAt = now
		}

	case models.EnvironmentPhaseStopped:
		// Check for auto-delete after being stopped (7 days)
		// Per Requirements 7.8: After 7 days stopped, prompt user for confirmation
		// If no response within 24 hours, auto-delete
		if env.StoppedAt != nil {
			stoppedDuration := now.Sub(*env.StoppedAt)
			
			// Check if there's already a pending delete confirmation
			if s.HasPendingDeleteConfirmation(env.ID) {
				// Don't send another notification, the pending delete check will handle it
				return result
			}
			
			// Check if we should send notification (24h before the 7-day mark)
			notifyThreshold := s.config.AutoDeleteStoppedDuration - s.config.NotificationLeadTime
			if stoppedDuration >= notifyThreshold && stoppedDuration < s.config.AutoDeleteStoppedDuration {
				result.Action = models.LifecycleActionNotifyAutoDelete
				result.Reason = "Environment will be auto-deleted after 7 days stopped"
				result.ScheduledAt = env.StoppedAt.Add(s.config.AutoDeleteStoppedDuration)
			} else if stoppedDuration >= s.config.AutoDeleteStoppedDuration {
				// 7 days reached - request delete confirmation (24h window)
				result.Action = models.LifecycleActionRequestDeleteConfirmation
				result.Reason = "Environment stopped for 7 days. Please confirm if you want to keep it. Auto-delete in 24 hours if no response."
				result.ScheduledAt = now.Add(24 * time.Hour)
			}
		}

	case models.EnvironmentPhaseSuspended:
		// Check for auto-archive after being suspended
		if env.StoppedAt != nil {
			suspendedDuration := now.Sub(*env.StoppedAt)
			
			notifyThreshold := s.config.AutoArchiveSuspendedDuration - s.config.NotificationLeadTime
			if suspendedDuration >= notifyThreshold && suspendedDuration < s.config.AutoArchiveSuspendedDuration {
				result.Action = models.LifecycleActionNotifyAutoArchive
				result.Reason = "Environment will be archived after 30 days suspended"
				result.ScheduledAt = env.StoppedAt.Add(s.config.AutoArchiveSuspendedDuration)
			} else if suspendedDuration >= s.config.AutoArchiveSuspendedDuration {
				result.Action = models.LifecycleActionAutoArchive
				result.Reason = "Environment archived after 30 days suspended"
				result.ScheduledAt = now
			}
		}

	case models.EnvironmentPhaseArchived:
		// Check for permanent deletion after being archived
		if env.StoppedAt != nil {
			archivedDuration := now.Sub(*env.StoppedAt)
			if archivedDuration >= s.config.AutoDeleteArchivedDuration {
				result.Action = models.LifecycleActionAutoDelete
				result.Reason = "Environment permanently deleted after 90 days archived"
				result.ScheduledAt = now
			}
		}

	case models.EnvironmentPhaseFailed:
		// Check for failed environment cleanup
		failedDuration := now.Sub(env.UpdatedAt)
		if failedDuration >= s.config.FailedRetentionDuration {
			result.Action = models.LifecycleActionFailedCleanup
			result.Reason = "Failed environment cleaned up after 24 hours"
			result.ScheduledAt = now
		}
	}

	return result
}

// executeLifecycleAction executes a lifecycle action
func (s *LifecycleService) executeLifecycleAction(ctx context.Context, result *models.LifecycleCheckResult) {
	logger := s.logger.With(
		zap.String("environmentId", result.EnvironmentID),
		zap.String("action", string(result.Action)),
	)

	s.envService.mu.Lock()
	env, ok := s.envService.environments[result.EnvironmentID]
	if !ok {
		s.envService.mu.Unlock()
		return
	}

	var err error
	switch result.Action {
	case models.LifecycleActionAutoSuspend:
		err = s.stateMachine.Transition(env, models.EnvironmentPhaseSuspended)
		if err == nil {
			env.Message = result.Reason
			logger.Info("Environment auto-suspended")
			s.sendNotification(env, result.Action, result.Reason)
		}

	case models.LifecycleActionAutoArchive:
		err = s.stateMachine.Transition(env, models.EnvironmentPhaseArchived)
		if err == nil {
			env.Message = result.Reason
			logger.Info("Environment auto-archived")
			s.sendNotification(env, result.Action, result.Reason)
		}

	case models.LifecycleActionAutoDelete, models.LifecycleActionFailedCleanup:
		s.envService.mu.Unlock()
		// Delete requires releasing the lock first
		if deleteErr := s.envService.Delete(ctx, env.UserID, env.ID); deleteErr != nil {
			logger.Error("Failed to auto-delete environment", zap.Error(deleteErr))
		} else {
			logger.Info("Environment auto-deleted")
			s.sendNotificationDirect(env.UserID, env.ID, result.Action, result.Reason)
		}
		return

	case models.LifecycleActionCreatingTimeout:
		err = s.stateMachine.Transition(env, models.EnvironmentPhaseFailed)
		if err == nil {
			env.Message = result.Reason
			logger.Info("Environment creation timed out")
			s.sendNotification(env, result.Action, result.Reason)
		}

	case models.LifecycleActionNotifyAutoStop, models.LifecycleActionNotifyAutoDelete, models.LifecycleActionNotifyAutoArchive:
		s.sendNotification(env, result.Action, result.Reason)

	case models.LifecycleActionRequestDeleteConfirmation:
		// Request delete confirmation - user has 24 hours to respond
		// Per Requirements 7.8: After 7 days stopped, prompt user for confirmation
		// If no response within 24 hours, auto-delete
		s.envService.mu.Unlock()
		s.RequestDeleteConfirmation(env.ID, env.UserID)
		return
	}

	s.envService.mu.Unlock()

	if err != nil {
		logger.Error("Failed to execute lifecycle action", zap.Error(err))
	}
}

// sendNotification sends a lifecycle notification
func (s *LifecycleService) sendNotification(env *models.Environment, action models.LifecycleAction, message string) {
	notification := &models.LifecycleNotification{
		EnvironmentID: env.ID,
		UserID:        env.UserID,
		Type:          action,
		Message:       message,
		CreatedAt:     time.Now(),
	}

	select {
	case s.notifications <- notification:
	default:
		s.logger.Warn("Notification channel full, dropping notification",
			zap.String("environmentId", env.ID),
			zap.String("action", string(action)))
	}
}

// sendNotificationDirect sends a notification without holding the lock
func (s *LifecycleService) sendNotificationDirect(userID, envID string, action models.LifecycleAction, message string) {
	notification := &models.LifecycleNotification{
		EnvironmentID: envID,
		UserID:        userID,
		Type:          action,
		Message:       message,
		CreatedAt:     time.Now(),
	}

	select {
	case s.notifications <- notification:
	default:
		s.logger.Warn("Notification channel full, dropping notification",
			zap.String("environmentId", envID),
			zap.String("action", string(action)))
	}
}

// processNotification processes a notification (e.g., send email, webhook)
func (s *LifecycleService) processNotification(notification *models.LifecycleNotification) {
	s.logger.Info("Processing lifecycle notification",
		zap.String("environmentId", notification.EnvironmentID),
		zap.String("userId", notification.UserID),
		zap.String("type", string(notification.Type)),
		zap.String("message", notification.Message))

	// TODO: Implement actual notification sending (email, webhook, etc.)
	// This would integrate with a notification service
}

// TransitionEnvironment transitions an environment to a new phase
func (s *LifecycleService) TransitionEnvironment(env *models.Environment, to models.EnvironmentPhase) error {
	return s.stateMachine.Transition(env, to)
}

// CanTransition checks if a transition is valid
func (s *LifecycleService) CanTransition(from, to models.EnvironmentPhase) bool {
	return s.stateMachine.CanTransition(from, to)
}

// GetValidTransitions returns valid transitions from a phase
func (s *LifecycleService) GetValidTransitions(from models.EnvironmentPhase) []models.EnvironmentPhase {
	return s.stateMachine.GetValidTransitions(from)
}

// RecordActivity records activity for an environment (resets inactivity timer)
func (s *LifecycleService) RecordActivity(ctx context.Context, environmentID string) error {
	return s.envService.UpdateActivity(ctx, environmentID)
}

// GetLifecycleStatus returns the lifecycle status of an environment
func (s *LifecycleService) GetLifecycleStatus(env *models.Environment) *LifecycleStatus {
	now := time.Now()
	status := &LifecycleStatus{
		Phase:            env.Phase,
		ValidTransitions: s.stateMachine.GetValidTransitions(env.Phase),
	}

	switch env.Phase {
	case models.EnvironmentPhaseRunning:
		lastActivity := env.LastActivityTime
		if lastActivity == nil {
			lastActivity = env.StartedAt
		}
		if lastActivity == nil {
			lastActivity = &env.CreatedAt
		}
		
		inactiveDuration := now.Sub(*lastActivity)
		remaining := s.config.AutoStopInactivityDuration - inactiveDuration
		if remaining > 0 {
			status.TimeUntilAutoStop = &remaining
		}

	case models.EnvironmentPhaseStopped:
		if env.StoppedAt != nil {
			stoppedDuration := now.Sub(*env.StoppedAt)
			remaining := s.config.AutoDeleteStoppedDuration - stoppedDuration
			if remaining > 0 {
				status.TimeUntilAutoDelete = &remaining
			}
		}

	case models.EnvironmentPhaseSuspended:
		if env.StoppedAt != nil {
			suspendedDuration := now.Sub(*env.StoppedAt)
			remaining := s.config.AutoArchiveSuspendedDuration - suspendedDuration
			if remaining > 0 {
				status.TimeUntilAutoArchive = &remaining
			}
		}

	case models.EnvironmentPhaseArchived:
		if env.StoppedAt != nil {
			archivedDuration := now.Sub(*env.StoppedAt)
			remaining := s.config.AutoDeleteArchivedDuration - archivedDuration
			if remaining > 0 {
				status.TimeUntilAutoDelete = &remaining
			}
		}
	}

	return status
}

// LifecycleStatus represents the lifecycle status of an environment
type LifecycleStatus struct {
	Phase                models.EnvironmentPhase   `json:"phase"`
	ValidTransitions     []models.EnvironmentPhase `json:"validTransitions"`
	TimeUntilAutoStop    *time.Duration            `json:"timeUntilAutoStop,omitempty"`
	TimeUntilAutoDelete  *time.Duration            `json:"timeUntilAutoDelete,omitempty"`
	TimeUntilAutoArchive *time.Duration            `json:"timeUntilAutoArchive,omitempty"`
}


// RequestDeleteConfirmation sends a delete confirmation request
func (s *LifecycleService) RequestDeleteConfirmation(environmentID, userID string) {
	s.deleteMu.Lock()
	defer s.deleteMu.Unlock()

	now := time.Now()
	confirmation := &models.DeleteConfirmation{
		EnvironmentID:   environmentID,
		UserID:          userID,
		NotifiedAt:      now,
		ConfirmDeadline: now.Add(24 * time.Hour),
		Confirmed:       false,
	}

	s.pendingDeletes[environmentID] = confirmation

	// Send notification
	s.sendNotificationDirect(userID, environmentID, models.LifecycleActionNotifyAutoDelete,
		"Your environment will be deleted in 24 hours. Please confirm if you want to keep it.")
}

// ConfirmKeepEnvironment confirms that the user wants to keep the environment
func (s *LifecycleService) ConfirmKeepEnvironment(environmentID string) error {
	s.deleteMu.Lock()
	defer s.deleteMu.Unlock()

	confirmation, ok := s.pendingDeletes[environmentID]
	if !ok {
		return fmt.Errorf("no pending delete confirmation for environment: %s", environmentID)
	}

	now := time.Now()
	confirmation.Confirmed = true
	confirmation.ConfirmedAt = &now

	// Remove from pending deletes
	delete(s.pendingDeletes, environmentID)

	s.logger.Info("User confirmed to keep environment",
		zap.String("environmentId", environmentID))

	return nil
}

// CheckPendingDeletes checks for expired delete confirmations
// This implements the 24-hour no-response auto-delete per Requirements 7.8
func (s *LifecycleService) CheckPendingDeletes(ctx context.Context) {
	s.deleteMu.Lock()
	now := time.Now()
	
	// Collect environments to delete and their confirmations
	type deleteInfo struct {
		envID        string
		confirmation *models.DeleteConfirmation
	}
	var toDelete []deleteInfo

	for envID, confirmation := range s.pendingDeletes {
		if !confirmation.Confirmed && now.After(confirmation.ConfirmDeadline) {
			toDelete = append(toDelete, deleteInfo{
				envID:        envID,
				confirmation: confirmation,
			})
		}
	}

	// Remove from pending deletes map while holding the lock
	for _, info := range toDelete {
		delete(s.pendingDeletes, info.envID)
	}
	s.deleteMu.Unlock()

	// Now perform deletions without holding the lock
	for _, info := range toDelete {
		s.logger.Info("Auto-deleting environment after 24h no response",
			zap.String("environmentId", info.envID),
			zap.String("userId", info.confirmation.UserID),
			zap.Time("notifiedAt", info.confirmation.NotifiedAt),
			zap.Time("deadline", info.confirmation.ConfirmDeadline))

		if err := s.envService.Delete(ctx, info.confirmation.UserID, info.envID); err != nil {
			s.logger.Error("Failed to auto-delete environment",
				zap.String("environmentId", info.envID),
				zap.Error(err))
		} else {
			s.sendNotificationDirect(info.confirmation.UserID, info.envID, models.LifecycleActionAutoDelete,
				"Your environment has been deleted due to no response within 24 hours.")
		}
	}
}

// GetPendingDeleteConfirmation returns the pending delete confirmation for an environment
func (s *LifecycleService) GetPendingDeleteConfirmation(environmentID string) *models.DeleteConfirmation {
	s.deleteMu.RLock()
	defer s.deleteMu.RUnlock()
	return s.pendingDeletes[environmentID]
}

// HasPendingDeleteConfirmation checks if an environment has a pending delete confirmation
func (s *LifecycleService) HasPendingDeleteConfirmation(environmentID string) bool {
	s.deleteMu.RLock()
	defer s.deleteMu.RUnlock()
	_, ok := s.pendingDeletes[environmentID]
	return ok
}
