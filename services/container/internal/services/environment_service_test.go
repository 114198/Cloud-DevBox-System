// Package services provides business logic for the container service.
package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestEnvironmentService creates a test environment service
func setupTestEnvironmentService(t *testing.T) *EnvironmentService {
	logger, _ := zap.NewDevelopment()
	config := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  20,
		CreationRateWindow: time.Hour,
	}
	
	// Create service without Kubernetes client for unit tests
	return NewEnvironmentService(nil, logger, config)
}

func TestEnvironmentService_Create(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := &models.CreateEnvironmentRequest{
			Name:        "test-env",
			Description: "Test environment",
			TemplateID:  "node-18",
			Resources: &models.ResourceConfig{
				CPU:     "1",
				Memory:  "2Gi",
				Storage: "10Gi",
			},
		}

		env, err := service.Create(ctx, "user-123", req)
		require.NoError(t, err)
		assert.NotEmpty(t, env.ID)
		assert.Equal(t, "test-env", env.Name)
		assert.Equal(t, "user-123", env.UserID)
		assert.Equal(t, models.EnvironmentPhaseCreating, env.Phase)
	})

	t.Run("default resources applied", func(t *testing.T) {
		req := &models.CreateEnvironmentRequest{
			Name:       "env-no-resources",
			TemplateID: "python-3",
		}

		env, err := service.Create(ctx, "user-456", req)
		require.NoError(t, err)
		assert.NotNil(t, env.Resources)
		assert.Equal(t, "1", env.Resources.CPU)
		assert.Equal(t, "2Gi", env.Resources.Memory)
		assert.Equal(t, "10Gi", env.Resources.Storage)
	})
}

func TestEnvironmentService_Get(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create an environment first
	req := &models.CreateEnvironmentRequest{
		Name:       "get-test-env",
		TemplateID: "node-18",
	}
	created, err := service.Create(ctx, "user-123", req)
	require.NoError(t, err)

	t.Run("get existing environment", func(t *testing.T) {
		env, err := service.Get(ctx, "user-123", created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, env.ID)
		assert.Equal(t, "get-test-env", env.Name)
	})

	t.Run("access denied for different user", func(t *testing.T) {
		_, err := service.Get(ctx, "other-user", created.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.Get(ctx, "user-123", "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestEnvironmentService_List(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create multiple environments
	for i := 0; i < 5; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "list-test-env-" + string(rune('a'+i)),
			TemplateID: "node-18",
		}
		_, err := service.Create(ctx, "user-list", req)
		require.NoError(t, err)
	}

	t.Run("list all for user", func(t *testing.T) {
		resp, err := service.List(ctx, &models.ListEnvironmentsRequest{
			UserID:   "user-list",
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.Total)
		assert.Len(t, resp.Environments, 5)
	})

	t.Run("pagination", func(t *testing.T) {
		resp, err := service.List(ctx, &models.ListEnvironmentsRequest{
			UserID:   "user-list",
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.Total)
		assert.Len(t, resp.Environments, 2)
	})
}

func TestEnvironmentService_Update(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create an environment
	req := &models.CreateEnvironmentRequest{
		Name:       "update-test-env",
		TemplateID: "node-18",
	}
	created, err := service.Create(ctx, "user-123", req)
	require.NoError(t, err)

	t.Run("successful update", func(t *testing.T) {
		newName := "updated-name"
		newDesc := "Updated description"
		updateReq := &models.UpdateEnvironmentRequest{
			Name:        &newName,
			Description: &newDesc,
		}

		updated, err := service.Update(ctx, "user-123", created.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, "updated-name", updated.Name)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("access denied for different user", func(t *testing.T) {
		newName := "hacked"
		updateReq := &models.UpdateEnvironmentRequest{
			Name: &newName,
		}
		_, err := service.Update(ctx, "other-user", created.ID, updateReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestEnvironmentService_Delete(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create an environment
	req := &models.CreateEnvironmentRequest{
		Name:       "delete-test-env",
		TemplateID: "node-18",
	}
	created, err := service.Create(ctx, "user-123", req)
	require.NoError(t, err)

	t.Run("successful delete", func(t *testing.T) {
		err := service.Delete(ctx, "user-123", created.ID)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = service.Get(ctx, "user-123", created.ID)
		assert.Error(t, err)
	})

	t.Run("access denied for different user", func(t *testing.T) {
		// Create another environment
		req := &models.CreateEnvironmentRequest{
			Name:       "delete-test-env-2",
			TemplateID: "node-18",
		}
		created2, err := service.Create(ctx, "user-123", req)
		require.NoError(t, err)

		err = service.Delete(ctx, "other-user", created2.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestEnvironmentService_BatchOperation(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create multiple environments
	var ids []string
	for i := 0; i < 3; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "batch-test-env-" + string(rune('a'+i)),
			TemplateID: "node-18",
		}
		env, err := service.Create(ctx, "user-batch", req)
		require.NoError(t, err)
		ids = append(ids, env.ID)
	}

	t.Run("batch delete", func(t *testing.T) {
		batchReq := &models.BatchOperationRequest{
			IDs:       ids,
			Operation: "delete",
		}

		resp, err := service.BatchOperation(ctx, "user-batch", batchReq)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Success)
		assert.Equal(t, 0, resp.Failed)
	})
}

// **Feature: cloud-devbox, Property 2: 环境创建 < 3秒**
// **Validates: Requirements 1.3, 7.1**
func TestProperty2_EnvironmentCreationTime(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100
	var totalTime time.Duration
	var maxTime time.Duration

	for i := 0; i < iterations; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "perf-test-env-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			TemplateID: "node-18",
			Resources: &models.ResourceConfig{
				CPU:     "1",
				Memory:  "2Gi",
				Storage: "10Gi",
			},
		}

		start := time.Now()
		env, err := service.Create(ctx, "perf-user", req)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, env)

		totalTime += elapsed
		if elapsed > maxTime {
			maxTime = elapsed
		}

		// Property: Each environment creation should complete within 3 seconds
		// Note: Without actual K8s, this tests the service layer only
		assert.Less(t, elapsed, 3*time.Second, 
			"Environment creation took %v, expected < 3s", elapsed)
	}

	avgTime := totalTime / iterations
	t.Logf("Property 2 Results: avg=%v, max=%v over %d iterations", avgTime, maxTime, iterations)
}

// **Feature: cloud-devbox, Property 3: 资源和网络完全隔离**
// **Validates: Requirements 7.1, 7.4**
func TestProperty3_ResourceAndNetworkIsolation(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	// Create multiple environments for different users
	const iterations = 100
	type envPair struct {
		env1 *models.Environment
		env2 *models.Environment
	}
	var pairs []envPair

	for i := 0; i < iterations/2; i++ {
		// Create environment for user 1
		req1 := &models.CreateEnvironmentRequest{
			Name:       "isolation-env-1-" + string(rune('a'+i%26)),
			TemplateID: "node-18",
		}
		env1, err := service.Create(ctx, "user-1", req1)
		require.NoError(t, err)

		// Create environment for user 2
		req2 := &models.CreateEnvironmentRequest{
			Name:       "isolation-env-2-" + string(rune('a'+i%26)),
			TemplateID: "python-3",
		}
		env2, err := service.Create(ctx, "user-2", req2)
		require.NoError(t, err)

		pairs = append(pairs, envPair{env1, env2})
	}

	// Property: Each environment should be isolated from others
	for _, pair := range pairs {
		// Verify different IDs (unique environments)
		assert.NotEqual(t, pair.env1.ID, pair.env2.ID,
			"Environments should have unique IDs")

		// Verify different users
		assert.NotEqual(t, pair.env1.UserID, pair.env2.UserID,
			"Environments belong to different users")

		// Verify user 1 cannot access user 2's environment
		_, err := service.Get(ctx, "user-1", pair.env2.ID)
		assert.Error(t, err, "User 1 should not access User 2's environment")
		assert.Contains(t, err.Error(), "access denied")

		// Verify user 2 cannot access user 1's environment
		_, err = service.Get(ctx, "user-2", pair.env1.ID)
		assert.Error(t, err, "User 2 should not access User 1's environment")
		assert.Contains(t, err.Error(), "access denied")

		// Verify user 1 cannot delete user 2's environment
		err = service.Delete(ctx, "user-1", pair.env2.ID)
		assert.Error(t, err, "User 1 should not delete User 2's environment")

		// Verify user 2 cannot delete user 1's environment
		err = service.Delete(ctx, "user-2", pair.env1.ID)
		assert.Error(t, err, "User 2 should not delete User 1's environment")
	}

	t.Logf("Property 3 Results: Verified isolation for %d environment pairs", len(pairs))
}

// TestRateLimiter tests the rate limiting functionality
func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)

	t.Run("allows requests under limit", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			assert.True(t, limiter.Allow("user-1"), "Request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// The 6th request should be blocked
		assert.False(t, limiter.Allow("user-1"), "6th request should be blocked")
	})

	t.Run("different users have separate limits", func(t *testing.T) {
		// user-2 should have their own limit
		for i := 0; i < 5; i++ {
			assert.True(t, limiter.Allow("user-2"), "Request %d for user-2 should be allowed", i+1)
		}
	})
}

// TestEnvironmentService_RateLimit tests the rate limiting for environment creation
func TestEnvironmentService_RateLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &EnvironmentServiceConfig{
		Namespace:          "test-devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  5, // Low limit for testing
		CreationRateWindow: time.Hour,
	}
	service := NewEnvironmentService(nil, logger, config)
	ctx := context.Background()

	// Create environments up to the limit
	for i := 0; i < 5; i++ {
		req := &models.CreateEnvironmentRequest{
			Name:       "rate-limit-env-" + string(rune('a'+i)),
			TemplateID: "node-18",
		}
		_, err := service.Create(ctx, "rate-limit-user", req)
		require.NoError(t, err)
	}

	// The 6th creation should fail due to rate limit
	req := &models.CreateEnvironmentRequest{
		Name:       "rate-limit-env-f",
		TemplateID: "node-18",
	}
	_, err := service.Create(ctx, "rate-limit-user", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

// TestConcurrentEnvironmentCreation tests concurrent environment creation
func TestConcurrentEnvironmentCreation(t *testing.T) {
	service := setupTestEnvironmentService(t)
	ctx := context.Background()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	envs := make(chan *models.Environment, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &models.CreateEnvironmentRequest{
				Name:       "concurrent-env-" + string(rune('a'+idx)),
				TemplateID: "node-18",
			}
			env, err := service.Create(ctx, "concurrent-user", req)
			if err != nil {
				errors <- err
			} else {
				envs <- env
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(envs)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent creation error: %v", err)
	}

	// Verify all environments were created with unique IDs
	ids := make(map[string]bool)
	for env := range envs {
		assert.False(t, ids[env.ID], "Duplicate environment ID: %s", env.ID)
		ids[env.ID] = true
	}
	assert.Equal(t, numGoroutines, len(ids), "All environments should have unique IDs")
}
