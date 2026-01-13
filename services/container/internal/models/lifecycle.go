// Package models defines the data models for the container service.
package models

import (
	"fmt"
	"time"
)

// EnvironmentPhase constants - extended with Archived
const (
	EnvironmentPhaseArchived EnvironmentPhase = "Archived"
)

// StateTransition represents a valid state transition
type StateTransition struct {
	From EnvironmentPhase
	To   EnvironmentPhase
}

// StateMachine manages environment state transitions
type StateMachine struct {
	validTransitions map[StateTransition]bool
}

// NewStateMachine creates a new state machine with valid transitions
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		validTransitions: make(map[StateTransition]bool),
	}

	// Define valid transitions based on requirements:
	// Creating → Running → Stopped → Deleted
	// Failed, Suspended, Archived states

	// From Creating
	sm.addTransition(EnvironmentPhaseCreating, EnvironmentPhaseRunning)  // Success
	sm.addTransition(EnvironmentPhaseCreating, EnvironmentPhaseFailed)   // Creation failed

	// From Running
	sm.addTransition(EnvironmentPhaseRunning, EnvironmentPhaseStopped)   // Manual stop or auto-stop
	sm.addTransition(EnvironmentPhaseRunning, EnvironmentPhaseSuspended) // 72h inactivity
	sm.addTransition(EnvironmentPhaseRunning, EnvironmentPhaseFailed)    // Runtime failure
	sm.addTransition(EnvironmentPhaseRunning, EnvironmentPhaseDeleting)  // Direct delete

	// From Stopped
	sm.addTransition(EnvironmentPhaseStopped, EnvironmentPhaseCreating)  // Restart (goes through creating)
	sm.addTransition(EnvironmentPhaseStopped, EnvironmentPhaseDeleting)  // Delete
	sm.addTransition(EnvironmentPhaseStopped, EnvironmentPhaseArchived)  // 7 days stopped → archived

	// From Suspended
	sm.addTransition(EnvironmentPhaseSuspended, EnvironmentPhaseCreating) // Reactivate
	sm.addTransition(EnvironmentPhaseSuspended, EnvironmentPhaseStopped)  // Stop
	sm.addTransition(EnvironmentPhaseSuspended, EnvironmentPhaseArchived) // 30 days suspended → archived
	sm.addTransition(EnvironmentPhaseSuspended, EnvironmentPhaseDeleting) // Delete

	// From Failed
	sm.addTransition(EnvironmentPhaseFailed, EnvironmentPhaseCreating)   // Retry
	sm.addTransition(EnvironmentPhaseFailed, EnvironmentPhaseDeleting)   // Delete

	// From Archived
	sm.addTransition(EnvironmentPhaseArchived, EnvironmentPhaseCreating) // Restore
	sm.addTransition(EnvironmentPhaseArchived, EnvironmentPhaseDeleting) // Delete (90 days auto-delete)

	// From Deleting (terminal state, no transitions out except completion)
	// Deleting is a terminal state - environment is removed after this

	return sm
}

// addTransition adds a valid transition
func (sm *StateMachine) addTransition(from, to EnvironmentPhase) {
	sm.validTransitions[StateTransition{From: from, To: to}] = true
}

// CanTransition checks if a transition is valid
func (sm *StateMachine) CanTransition(from, to EnvironmentPhase) bool {
	return sm.validTransitions[StateTransition{From: from, To: to}]
}

// Transition attempts to transition from one state to another
func (sm *StateMachine) Transition(env *Environment, to EnvironmentPhase) error {
	if !sm.CanTransition(env.Phase, to) {
		return fmt.Errorf("invalid state transition from %s to %s", env.Phase, to)
	}

	now := time.Now()
	env.Phase = to
	env.UpdatedAt = now

	// Update timestamps based on transition
	switch to {
	case EnvironmentPhaseRunning:
		env.StartedAt = &now
		env.StoppedAt = nil
	case EnvironmentPhaseStopped, EnvironmentPhaseSuspended:
		env.StoppedAt = &now
	}

	return nil
}

// GetValidTransitions returns all valid transitions from a given phase
func (sm *StateMachine) GetValidTransitions(from EnvironmentPhase) []EnvironmentPhase {
	var transitions []EnvironmentPhase
	for t := range sm.validTransitions {
		if t.From == from {
			transitions = append(transitions, t.To)
		}
	}
	return transitions
}

// LifecycleConfig holds configuration for lifecycle management
type LifecycleConfig struct {
	// AutoStopInactivityDuration is the duration of inactivity before auto-stop (default 72h)
	AutoStopInactivityDuration time.Duration

	// AutoDeleteStoppedDuration is the duration after stop before auto-delete (default 7 days)
	AutoDeleteStoppedDuration time.Duration

	// AutoArchiveSuspendedDuration is the duration after suspended before archive (default 30 days)
	AutoArchiveSuspendedDuration time.Duration

	// AutoDeleteArchivedDuration is the duration after archived before permanent delete (default 90 days)
	AutoDeleteArchivedDuration time.Duration

	// CreatingTimeoutDuration is the max time an environment can be in Creating state (default 5 min)
	CreatingTimeoutDuration time.Duration

	// FailedRetentionDuration is how long to keep failed environments (default 24h)
	FailedRetentionDuration time.Duration

	// NotificationLeadTime is how long before action to send notification (default 24h)
	NotificationLeadTime time.Duration
}

// DefaultLifecycleConfig returns default lifecycle configuration
func DefaultLifecycleConfig() *LifecycleConfig {
	return &LifecycleConfig{
		AutoStopInactivityDuration:   72 * time.Hour,
		AutoDeleteStoppedDuration:    7 * 24 * time.Hour,
		AutoArchiveSuspendedDuration: 30 * 24 * time.Hour,
		AutoDeleteArchivedDuration:   90 * 24 * time.Hour,
		CreatingTimeoutDuration:      5 * time.Minute,
		FailedRetentionDuration:      24 * time.Hour,
		NotificationLeadTime:         24 * time.Hour,
	}
}

// LifecycleAction represents an action to be taken on an environment
type LifecycleAction string

const (
	LifecycleActionNone                       LifecycleAction = "none"
	LifecycleActionAutoStop                   LifecycleAction = "auto_stop"
	LifecycleActionAutoSuspend                LifecycleAction = "auto_suspend"
	LifecycleActionAutoArchive                LifecycleAction = "auto_archive"
	LifecycleActionAutoDelete                 LifecycleAction = "auto_delete"
	LifecycleActionCreatingTimeout            LifecycleAction = "creating_timeout"
	LifecycleActionFailedCleanup              LifecycleAction = "failed_cleanup"
	LifecycleActionNotifyAutoStop             LifecycleAction = "notify_auto_stop"
	LifecycleActionNotifyAutoDelete           LifecycleAction = "notify_auto_delete"
	LifecycleActionNotifyAutoArchive          LifecycleAction = "notify_auto_archive"
	LifecycleActionRequestDeleteConfirmation  LifecycleAction = "request_delete_confirmation"
)

// LifecycleCheckResult represents the result of a lifecycle check
type LifecycleCheckResult struct {
	EnvironmentID string
	Action        LifecycleAction
	Reason        string
	ScheduledAt   time.Time
}

// LifecycleNotification represents a notification to be sent
type LifecycleNotification struct {
	EnvironmentID   string
	UserID          string
	Type            LifecycleAction
	Message         string
	ScheduledAction time.Time
	CreatedAt       time.Time
}

// DeleteConfirmation tracks pending delete confirmations
type DeleteConfirmation struct {
	EnvironmentID   string
	UserID          string
	NotifiedAt      time.Time
	ConfirmDeadline time.Time
	Confirmed       bool
	ConfirmedAt     *time.Time
}
