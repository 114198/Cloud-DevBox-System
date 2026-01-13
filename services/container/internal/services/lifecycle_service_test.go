// Package services provides business logic for the container service.
package services

import (
	"context"
	"testing"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestLifecycleService creates a test lifecycle service
func setupTestLifecycleService(t *testing.T) (*LifecycleService, *EnvironmentService) {
	logger, _ := zap.NewDevelopment()
	envConfig := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  100, // High limit for testing
		CreationRateWindow: time.Hour,
	}
	envService := NewEnvironmentService(nil, logger, envConfig)

	// Use short durations for testing
	lifecycleConfig := &models.LifecycleConfig{
		AutoStopInactivityDuration:   100 * time.Millisecond,
		AutoDeleteStoppedDuration:    200 * time.Millisecond,
		AutoArchiveSuspendedDuration: 300 * time.Millisecond,
		AutoDeleteArchivedDuration:   400 * time.Millisecond,
		CreatingTimeoutDuration:      50 * time.Millisecond,
		FailedRetentionDuration:      100 * time.Millisecond,
		NotificationLeadTime:         50 * time.Millisecond,
	}

	lifecycleService := NewLifecycleService(envService, lifecycleConfig, logger)
	return lifecycleService, envService
}

func TestStateMachine_ValidTransitions(t *testing.T) {
	sm := models.NewStateMachine()

	testCases := []struct {
		from     models.EnvironmentPhase
		to       models.EnvironmentPhase
		expected bool
	}{
		// From Creating
		{models.EnvironmentPhaseCreating, models.EnvironmentPhaseRunning, true},
		{models.EnvironmentPhaseCreating, models.EnvironmentPhaseFailed, true},
		{models.EnvironmentPhaseCreating, models.EnvironmentPhaseStopped, false},

		// From Running
		{models.EnvironmentPhaseRunning, models.EnvironmentPhaseStopped, true},
		{models.EnvironmentPhaseRunning, models.EnvironmentPhaseSuspended, true},
		{models.EnvironmentPhaseRunning, models.EnvironmentPhaseFailed, true},
		{models.EnvironmentPhaseRunning, models.EnvironmentPhaseDeleting, true},
		{models.EnvironmentPhaseRunning, models.EnvironmentPhaseCreating, false},

		// From Stopped
		{models.EnvironmentPhaseStopped, models.EnvironmentPhaseCreating, true},
		{models.EnvironmentPhaseStopped, models.EnvironmentPhaseDeleting, true},
		{models.EnvironmentPhaseStopped, models.EnvironmentPhaseArchived, true},
		{models.EnvironmentPhaseStopped, models.EnvironmentPhaseRunning, false},

		// From Suspended
		{models.EnvironmentPhaseSuspended, models.EnvironmentPhaseCreating, true},
		{models.EnvironmentPhaseSuspended, models.EnvironmentPhaseStopped, true},
		{models.EnvironmentPhaseSuspended, models.EnvironmentPhaseArchived, true},
		{models.EnvironmentPhaseSuspended, models.EnvironmentPhaseDeleting, true},
		{models.EnvironmentPhaseSuspended, models.EnvironmentPhaseRunning, false},

		// From Failed
		{models.EnvironmentPhaseFailed, models.EnvironmentPhaseCreating, true},
		{models.EnvironmentPhaseFailed, models.EnvironmentPhaseDeleting, true},
		{models.EnvironmentPhaseFailed, models.EnvironmentPhaseRunning, false},

		// From Archived
		{models.EnvironmentPhaseArchived, models.EnvironmentPhaseCreating, true},
		{models.EnvironmentPhaseArchived, models.EnvironmentPhaseDeleting, true},
		{models.EnvironmentPhaseArchived, models.EnvironmentPhaseRunning, false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.from)+"_to_"+string(tc.to), func(t *testing.T) {
			result := sm.CanTransition(tc.from, tc.to)
			assert.Equal(t, tc.expected, result,
				"Transition from %s to %s should be %v", tc.from, tc.to, tc.expected)
		})
	}
}

func TestStateMachine_Transition(t *testing.T) {
	sm := models.NewStateMachine()

	t.Run("valid transition updates phase", func(t *testing.T) {
		env := &models.Environment{
			Phase:     models.EnvironmentPhaseCreating,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := sm.Transition(env, models.EnvironmentPhaseRunning)
		require.NoError(t, err)
		assert.Equal(t, models.EnvironmentPhaseRunning, env.Phase)
		assert.NotNil(t, env.StartedAt)
	})

	t.Run("invalid transition returns error", func(t *testing.T) {
		env := &models.Environment{
			Phase:     models.EnvironmentPhaseCreating,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := sm.Transition(env, models.EnvironmentPhaseStopped)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state transition")
		// Phase should not change
		assert.Equal(t, models.EnvironmentPhaseCreating, env.Phase)
	})

	t.Run("transition to stopped sets StoppedAt", func(t *testing.T) {
		env := &models.Environment{
			Phase:     models.EnvironmentPhaseRunning,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := sm.Transition(env, models.EnvironmentPhaseStopped)
		require.NoError(t, err)
		assert.Equal(t, models.EnvironmentPhaseStopped, env.Phase)
		assert.NotNil(t, env.StoppedAt)
	})
}

// **Feature: cloud-devbox, Property 12: 自动停止和删除**
// **Validates: Requirements 7.7, 7.8**
func TestProperty12_AutoStopAndDelete(t *testing.T) {
	lifecycleService, envService := setupTestLifecycleService(t)
	ctx := context.Background()

	const iterations = 100

	t.Run("auto-suspend after inactivity", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			// Create an environment
			req := &models.CreateEnvironmentRequest{
				Name:       "auto-stop-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
				TemplateID: "node-18",
			}
			env, err := envService.Create(ctx, "auto-stop-user", req)
			require.NoError(t, err)

			// Manually set to Running state
			envService.mu.Lock()
			env.Phase = models.EnvironmentPhaseRunning
			startTime := time.Now().Add(-200 * time.Millisecond) // Started 200ms ago
			env.StartedAt = &startTime
			env.LastActivityTime = &startTime // No activity since start
			envService.mu.Unlock()

			// Check lifecycle - should trigger auto-suspend
			result := lifecycleService.checkEnvironment(env)

			// Property: Environment should be auto-suspended after inactivity period
			assert.Equal(t, models.LifecycleActionAutoSuspend, result.Action,
				"Iteration %d: Environment should be auto-suspended after inactivity", i)
		}
	})

	t.Run("auto-delete after stopped duration", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			// Create an environment
			req := &models.CreateEnvironmentRequest{
				Name:       "auto-delete-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
				TemplateID: "node-18",
			}
			env, err := envService.Create(ctx, "auto-delete-user", req)
			require.NoError(t, err)

			// Manually set to Stopped state
			envService.mu.Lock()
			env.Phase = models.EnvironmentPhaseStopped
			stoppedTime := time.Now().Add(-300 * time.Millisecond) // Stopped 300ms ago
			env.StoppedAt = &stoppedTime
			envService.mu.Unlock()

			// Check lifecycle - should trigger auto-delete
			result := lifecycleService.checkEnvironment(env)

			// Property: Environment should be auto-deleted after stopped duration
			assert.Equal(t, models.LifecycleActionAutoDelete, result.Action,
				"Iteration %d: Environment should be auto-deleted after stopped duration", i)
		}
	})

	t.Run("no action when recently active", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			// Create an environment
			req := &models.CreateEnvironmentRequest{
				Name:       "active-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
				TemplateID: "node-18",
			}
			env, err := envService.Create(ctx, "active-user", req)
			require.NoError(t, err)

			// Manually set to Running state with recent activity
			envService.mu.Lock()
			env.Phase = models.EnvironmentPhaseRunning
			now := time.Now()
			env.StartedAt = &now
			env.LastActivityTime = &now // Just active
			envService.mu.Unlock()

			// Check lifecycle - should not trigger any action
			result := lifecycleService.checkEnvironment(env)

			// Property: Recently active environment should not be auto-stopped
			assert.Equal(t, models.LifecycleActionNone, result.Action,
				"Iteration %d: Recently active environment should not be auto-stopped", i)
		}
	})

	t.Logf("Property 12 Results: Verified auto-stop and auto-delete for %d iterations each", iterations)
}

func TestLifecycleService_CreatingTimeout(t *testing.T) {
	lifecycleService, envService := setupTestLifecycleService(t)
	ctx := context.Background()

	// Create an environment
	req := &models.CreateEnvironmentRequest{
		Name:       "timeout-test",
		TemplateID: "node-18",
	}
	env, err := envService.Create(ctx, "timeout-user", req)
	require.NoError(t, err)

	// Set creation time to past the timeout
	envService.mu.Lock()
	env.CreatedAt = time.Now().Add(-100 * time.Millisecond)
	envService.mu.Unlock()

	// Check lifecycle
	result := lifecycleService.checkEnvironment(env)

	assert.Equal(t, models.LifecycleActionCreatingTimeout, result.Action)
	assert.Contains(t, result.Reason, "timed out")
}

func TestLifecycleService_FailedCleanup(t *testing.T) {
	lifecycleService, envService := setupTestLifecycleService(t)
	ctx := context.Background()

	// Create an environment
	req := &models.CreateEnvironmentRequest{
		Name:       "failed-test",
		TemplateID: "node-18",
	}
	env, err := envService.Create(ctx, "failed-user", req)
	require.NoError(t, err)

	// Set to Failed state with old update time
	envService.mu.Lock()
	env.Phase = models.EnvironmentPhaseFailed
	env.UpdatedAt = time.Now().Add(-200 * time.Millisecond)
	envService.mu.Unlock()

	// Check lifecycle
	result := lifecycleService.checkEnvironment(env)

	assert.Equal(t, models.LifecycleActionFailedCleanup, result.Action)
	assert.Contains(t, result.Reason, "cleaned up")
}

func TestLifecycleService_NotificationBeforeAction(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	envConfig := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  100,
		CreationRateWindow: time.Hour,
	}
	envService := NewEnvironmentService(nil, logger, envConfig)

	// Configure with notification lead time
	lifecycleConfig := &models.LifecycleConfig{
		AutoStopInactivityDuration: 100 * time.Millisecond,
		NotificationLeadTime:       50 * time.Millisecond,
		// Other durations
		AutoDeleteStoppedDuration:    200 * time.Millisecond,
		AutoArchiveSuspendedDuration: 300 * time.Millisecond,
		AutoDeleteArchivedDuration:   400 * time.Millisecond,
		CreatingTimeoutDuration:      50 * time.Millisecond,
		FailedRetentionDuration:      100 * time.Millisecond,
	}

	lifecycleService := NewLifecycleService(envService, lifecycleConfig, logger)
	ctx := context.Background()

	// Create an environment
	req := &models.CreateEnvironmentRequest{
		Name:       "notify-test",
		TemplateID: "node-18",
	}
	env, err := envService.Create(ctx, "notify-user", req)
	require.NoError(t, err)

	// Set to Running with activity time in notification window
	// (past notification threshold but before auto-stop threshold)
	envService.mu.Lock()
	env.Phase = models.EnvironmentPhaseRunning
	// 60ms ago - past 50ms notification threshold, before 100ms auto-stop
	activityTime := time.Now().Add(-60 * time.Millisecond)
	env.StartedAt = &activityTime
	env.LastActivityTime = &activityTime
	envService.mu.Unlock()

	// Check lifecycle - should trigger notification
	result := lifecycleService.checkEnvironment(env)

	assert.Equal(t, models.LifecycleActionNotifyAutoStop, result.Action)
	assert.Contains(t, result.Reason, "will be auto-stopped")
}

func TestLifecycleService_DeleteConfirmation(t *testing.T) {
	lifecycleService, _ := setupTestLifecycleService(t)

	t.Run("request delete confirmation", func(t *testing.T) {
		lifecycleService.RequestDeleteConfirmation("env-123", "user-456")

		confirmation := lifecycleService.GetPendingDeleteConfirmation("env-123")
		require.NotNil(t, confirmation)
		assert.Equal(t, "env-123", confirmation.EnvironmentID)
		assert.Equal(t, "user-456", confirmation.UserID)
		assert.False(t, confirmation.Confirmed)
	})

	t.Run("confirm keep environment", func(t *testing.T) {
		lifecycleService.RequestDeleteConfirmation("env-789", "user-abc")

		err := lifecycleService.ConfirmKeepEnvironment("env-789")
		require.NoError(t, err)

		// Should be removed from pending
		assert.False(t, lifecycleService.HasPendingDeleteConfirmation("env-789"))
	})

	t.Run("confirm non-existent returns error", func(t *testing.T) {
		err := lifecycleService.ConfirmKeepEnvironment("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no pending delete confirmation")
	})
}

func TestActivityTracker(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	envConfig := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  100,
		CreationRateWindow: time.Hour,
	}
	envService := NewEnvironmentService(nil, logger, envConfig)
	tracker := NewActivityTracker(envService, logger)

	ctx := context.Background()

	// Create an environment first
	req := &models.CreateEnvironmentRequest{
		Name:       "activity-test",
		TemplateID: "node-18",
	}
	env, err := envService.Create(ctx, "activity-user", req)
	require.NoError(t, err)

	// Start the tracker
	tracker.Start(ctx)
	defer tracker.Stop()

	t.Run("record SSH connect", func(t *testing.T) {
		tracker.RecordSSHConnect(env.ID, "activity-user")
		time.Sleep(10 * time.Millisecond) // Allow event processing

		connections := tracker.GetActiveSSHConnections(env.ID)
		assert.Equal(t, 1, connections)
	})

	t.Run("record SSH disconnect", func(t *testing.T) {
		tracker.RecordSSHDisconnect(env.ID, "activity-user")
		time.Sleep(10 * time.Millisecond) // Allow event processing

		connections := tracker.GetActiveSSHConnections(env.ID)
		assert.Equal(t, 0, connections)
	})

	t.Run("has active connections", func(t *testing.T) {
		tracker.RecordSSHConnect(env.ID, "activity-user")
		time.Sleep(10 * time.Millisecond)

		assert.True(t, tracker.HasActiveConnections(env.ID))

		tracker.RecordSSHDisconnect(env.ID, "activity-user")
		time.Sleep(10 * time.Millisecond)

		assert.False(t, tracker.HasActiveConnections(env.ID))
	})
}

func TestRateLimiter_DetailedResult(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	t.Run("check returns detailed result", func(t *testing.T) {
		result := limiter.Check("user-1")
		assert.True(t, result.Allowed)
		assert.Equal(t, 0, result.CurrentCount)
		assert.Equal(t, 3, result.Limit)
		assert.Equal(t, time.Minute, result.Window)
	})

	t.Run("record increments count", func(t *testing.T) {
		limiter.Record("user-1")
		limiter.Record("user-1")

		result := limiter.Check("user-1")
		assert.True(t, result.Allowed)
		assert.Equal(t, 2, result.CurrentCount)
	})

	t.Run("blocked when at limit", func(t *testing.T) {
		limiter.Record("user-1") // 3rd record

		result := limiter.Check("user-1")
		assert.False(t, result.Allowed)
		assert.Equal(t, 3, result.CurrentCount)
		assert.Greater(t, result.RetryAfter, time.Duration(0))
	})

	t.Run("get count", func(t *testing.T) {
		count := limiter.GetCount("user-1")
		assert.Equal(t, 3, count)
	})
}

// **Feature: cloud-devbox, Property: Rate Limit 429 Error**
// **Validates: Requirements 7.9**
func TestProperty_RateLimitReturns429(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  20, // 20 per hour as per requirements
		CreationRateWindow: time.Hour,
	}
	service := NewEnvironmentService(nil, logger, config)
	ctx := context.Background()

	const iterations = 100

	// Create 20 environments (the limit)
	for i := 0; i < 20; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "rate-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			TemplateID: "node-18",
		}
		_, err := service.Create(ctx, "rate-limit-user", req)
		require.NoError(t, err, "Creation %d should succeed", i)
	}

	// Now test that subsequent creations fail with rate limit error
	for i := 0; i < iterations; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "rate-exceeded-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			TemplateID: "node-18",
		}
		_, err := service.Create(ctx, "rate-limit-user", req)

		// Property: After 20 creations per hour, subsequent creations should fail
		require.Error(t, err, "Iteration %d: Creation should fail due to rate limit", i)

		// Verify it's a RateLimitError
		rateLimitErr, ok := err.(*RateLimitError)
		require.True(t, ok, "Iteration %d: Error should be RateLimitError", i)
		assert.Equal(t, 20, rateLimitErr.Limit)
		assert.Equal(t, time.Hour, rateLimitErr.Window)
		assert.Contains(t, rateLimitErr.Message, "rate limit exceeded")
	}

	t.Logf("Property Rate Limit Results: Verified 429 error for %d iterations after limit reached", iterations)
}
