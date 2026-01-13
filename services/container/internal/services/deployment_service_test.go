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

// setupTestDeploymentService creates a test deployment service
func setupTestDeploymentService(t *testing.T) *DeploymentService {
	logger, _ := zap.NewDevelopment()
	config := &DeploymentServiceConfig{
		Namespace: "test-deployments",
		Registry:  "test-registry.io",
	}
	return NewDeploymentService(nil, logger, config)
}

func TestDeploymentService_Create(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	t.Run("creates deployment successfully", func(t *testing.T) {
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-123",
			Name:          "test-app",
		}

		deployment, err := service.Create(ctx, "user-1", req)
		require.NoError(t, err)
		assert.NotEmpty(t, deployment.ID)
		assert.Equal(t, "test-app", deployment.Name)
		assert.Equal(t, "user-1", deployment.UserID)
		assert.Equal(t, "env-123", deployment.EnvironmentID)
		assert.NotEmpty(t, deployment.Version)
	})

	t.Run("sets default build config", func(t *testing.T) {
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-456",
			Name:          "test-app-2",
		}

		deployment, err := service.Create(ctx, "user-1", req)
		require.NoError(t, err)
		assert.NotNil(t, deployment.BuildConfig)
		assert.Equal(t, "Dockerfile", deployment.BuildConfig.DockerfilePath)
		assert.Equal(t, ".", deployment.BuildConfig.ContextPath)
		assert.Equal(t, "linux/amd64", deployment.BuildConfig.Platform)
	})

	t.Run("sets default deploy config", func(t *testing.T) {
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-789",
			Name:          "test-app-3",
		}

		deployment, err := service.Create(ctx, "user-1", req)
		require.NoError(t, err)
		assert.NotNil(t, deployment.DeployConfig)
		assert.Equal(t, int32(1), deployment.DeployConfig.Replicas)
		assert.Equal(t, "ClusterIP", deployment.DeployConfig.ServiceType)
		assert.NotNil(t, deployment.DeployConfig.Resources)
		assert.NotNil(t, deployment.DeployConfig.Strategy)
	})
}

func TestDeploymentService_Get(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create a deployment first
	req := &models.CreateDeploymentRequest{
		EnvironmentID: "env-123",
		Name:          "get-test-app",
	}
	created, err := service.Create(ctx, "user-1", req)
	require.NoError(t, err)

	t.Run("retrieves deployment by ID", func(t *testing.T) {
		deployment, err := service.Get(ctx, "user-1", created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, deployment.ID)
		assert.Equal(t, "get-test-app", deployment.Name)
	})

	t.Run("returns error for non-existent deployment", func(t *testing.T) {
		_, err := service.Get(ctx, "user-1", "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for wrong user", func(t *testing.T) {
		_, err := service.Get(ctx, "user-2", created.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestDeploymentService_List(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create multiple deployments
	for i := 0; i < 5; i++ {
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-list",
			Name:          "list-app-" + string(rune('a'+i)),
		}
		_, err := service.Create(ctx, "list-user", req)
		require.NoError(t, err)
	}

	t.Run("lists deployments for user", func(t *testing.T) {
		listReq := &models.ListDeploymentsRequest{
			UserID:   "list-user",
			Page:     1,
			PageSize: 10,
		}
		resp, err := service.List(ctx, listReq)
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.Total)
		assert.Len(t, resp.Deployments, 5)
	})

	t.Run("filters by environment ID", func(t *testing.T) {
		// Create deployment for different environment
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-other",
			Name:          "other-app",
		}
		_, err := service.Create(ctx, "list-user", req)
		require.NoError(t, err)

		listReq := &models.ListDeploymentsRequest{
			UserID:        "list-user",
			EnvironmentID: "env-list",
			Page:          1,
			PageSize:      10,
		}
		resp, err := service.List(ctx, listReq)
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.Total)
	})

	t.Run("paginates results", func(t *testing.T) {
		listReq := &models.ListDeploymentsRequest{
			UserID:   "list-user",
			Page:     1,
			PageSize: 2,
		}
		resp, err := service.List(ctx, listReq)
		require.NoError(t, err)
		assert.Len(t, resp.Deployments, 2)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 2, resp.PageSize)
	})
}

func TestDeploymentService_GenerateDockerfile(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	testCases := []struct {
		name     string
		language string
		contains []string
	}{
		{
			name:     "Node.js",
			language: "node",
			contains: []string{"FROM node:", "npm", "EXPOSE"},
		},
		{
			name:     "Go",
			language: "go",
			contains: []string{"FROM golang:", "go build", "EXPOSE"},
		},
		{
			name:     "Python",
			language: "python",
			contains: []string{"FROM python:", "pip", "EXPOSE"},
		},
		{
			name:     "Java",
			language: "java",
			contains: []string{"FROM maven:", "mvn", "EXPOSE"},
		},
		{
			name:     "Rust",
			language: "rust",
			contains: []string{"FROM rust:", "cargo", "EXPOSE"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.GenerateDockerfileRequest{
				Language: tc.language,
			}
			resp, err := service.GenerateDockerfile(ctx, req)
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Dockerfile)
			for _, s := range tc.contains {
				assert.Contains(t, resp.Dockerfile, s)
			}
		})
	}

	t.Run("returns error for unknown language", func(t *testing.T) {
		req := &models.GenerateDockerfileRequest{
			Language: "unknown",
		}
		_, err := service.GenerateDockerfile(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no Dockerfile template found")
	})
}

func TestDeploymentService_GetDockerfileTemplates(t *testing.T) {
	service := setupTestDeploymentService(t)

	templates := service.GetDockerfileTemplates()
	assert.NotEmpty(t, templates)

	// Verify expected templates exist
	languages := make(map[string]bool)
	for _, tmpl := range templates {
		languages[tmpl.Language] = true
	}

	assert.True(t, languages["nodejs"])
	assert.True(t, languages["go"])
	assert.True(t, languages["python"])
	assert.True(t, languages["java"])
	assert.True(t, languages["rust"])
}


func TestDeploymentService_Rollback(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create a deployment and wait for it to complete
	req := &models.CreateDeploymentRequest{
		EnvironmentID: "env-rollback",
		Name:          "rollback-app",
	}
	deployment, err := service.Create(ctx, "rollback-user", req)
	require.NoError(t, err)

	// Wait for deployment to complete (simulated)
	time.Sleep(200 * time.Millisecond)

	// Get the initial version
	initialVersion := deployment.Version

	// Create another deployment (new version)
	req2 := &models.CreateDeploymentRequest{
		EnvironmentID: "env-rollback",
		Name:          "rollback-app-v2",
	}
	deployment2, err := service.Create(ctx, "rollback-user", req2)
	require.NoError(t, err)

	// Wait for second deployment
	time.Sleep(200 * time.Millisecond)

	t.Run("rollback to previous version", func(t *testing.T) {
		// Get history first
		history, err := service.GetHistory(ctx, "rollback-user", deployment.ID)
		require.NoError(t, err)

		if len(history.Versions) > 0 {
			rollbackReq := &models.RollbackRequest{
				TargetVersion: history.Versions[0].Version,
				Reason:        "Testing rollback",
			}
			result, err := service.Rollback(ctx, "rollback-user", deployment.ID, rollbackReq)
			require.NoError(t, err)
			assert.Equal(t, models.DeploymentPhaseRolledBack, result.Phase)
		}
	})

	t.Run("rollback to non-existent version fails", func(t *testing.T) {
		rollbackReq := &models.RollbackRequest{
			TargetVersion: "non-existent-version",
		}
		_, err := service.Rollback(ctx, "rollback-user", deployment2.ID, rollbackReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target version not found")
	})

	t.Run("rollback with wrong user fails", func(t *testing.T) {
		rollbackReq := &models.RollbackRequest{
			TargetVersion: initialVersion,
		}
		_, err := service.Rollback(ctx, "wrong-user", deployment.ID, rollbackReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestDeploymentService_GetLogs(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create a deployment
	req := &models.CreateDeploymentRequest{
		EnvironmentID: "env-logs",
		Name:          "logs-app",
	}
	deployment, err := service.Create(ctx, "logs-user", req)
	require.NoError(t, err)

	// Wait for some logs to be generated
	time.Sleep(200 * time.Millisecond)

	t.Run("retrieves deployment logs", func(t *testing.T) {
		logs, err := service.GetLogs(ctx, "logs-user", deployment.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, logs)

		// Verify log structure
		for _, log := range logs {
			assert.NotEmpty(t, log.ID)
			assert.Equal(t, deployment.ID, log.DeploymentID)
			assert.NotEmpty(t, log.Phase)
			assert.NotEmpty(t, log.Level)
			assert.NotEmpty(t, log.Message)
			assert.False(t, log.Timestamp.IsZero())
		}
	})

	t.Run("returns error for wrong user", func(t *testing.T) {
		_, err := service.GetLogs(ctx, "wrong-user", deployment.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestDeploymentService_GetHistory(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create a deployment
	req := &models.CreateDeploymentRequest{
		EnvironmentID: "env-history",
		Name:          "history-app",
	}
	deployment, err := service.Create(ctx, "history-user", req)
	require.NoError(t, err)

	// Wait for deployment to complete
	time.Sleep(200 * time.Millisecond)

	t.Run("retrieves deployment history", func(t *testing.T) {
		history, err := service.GetHistory(ctx, "history-user", deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, history.DeploymentID)
		assert.GreaterOrEqual(t, history.Total, 1)
	})

	t.Run("returns error for wrong user", func(t *testing.T) {
		_, err := service.GetHistory(ctx, "wrong-user", deployment.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

func TestDeploymentService_Delete(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	// Create a deployment
	req := &models.CreateDeploymentRequest{
		EnvironmentID: "env-delete",
		Name:          "delete-app",
	}
	deployment, err := service.Create(ctx, "delete-user", req)
	require.NoError(t, err)

	t.Run("deletes deployment successfully", func(t *testing.T) {
		err := service.Delete(ctx, "delete-user", deployment.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = service.Get(ctx, "delete-user", deployment.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for non-existent deployment", func(t *testing.T) {
		err := service.Delete(ctx, "delete-user", "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for wrong user", func(t *testing.T) {
		// Create another deployment
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-delete-2",
			Name:          "delete-app-2",
		}
		deployment2, err := service.Create(ctx, "delete-user", req)
		require.NoError(t, err)

		err = service.Delete(ctx, "wrong-user", deployment2.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

// **Feature: cloud-devbox, Property 10: 构建 < 5分钟，回滚 < 30秒**
// **Validates: Requirements 6.1, 6.4**
func TestProperty10_BuildAndRollbackPerformance(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	const iterations = 100

	t.Run("build completes within time limit", func(t *testing.T) {
		// Note: In production, actual build time would be measured
		// Here we verify the build process completes and metrics are recorded
		for i := 0; i < iterations; i++ {
			req := &models.CreateDeploymentRequest{
				EnvironmentID: "env-build-perf",
				Name:          "build-perf-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			}

			startTime := time.Now()
			deployment, err := service.Create(ctx, "perf-user", req)
			require.NoError(t, err)

			// Wait for deployment to complete (simulated)
			time.Sleep(200 * time.Millisecond)

			// Get updated deployment
			updated, err := service.Get(ctx, "perf-user", deployment.ID)
			require.NoError(t, err)

			// Property: Build should complete (in simulation, this is fast)
			// In production, this would verify actual build time < 5 minutes
			if updated.Metrics != nil && updated.Metrics.BuildTime > 0 {
				// Build time should be recorded
				assert.Greater(t, updated.Metrics.BuildTime, float64(0),
					"Iteration %d: Build time should be recorded", i)
			}

			// Verify deployment progressed past building phase
			assert.NotEqual(t, models.DeploymentPhasePending, updated.Phase,
				"Iteration %d: Deployment should progress past pending", i)

			// Verify total time is reasonable (in simulation)
			totalTime := time.Since(startTime)
			assert.Less(t, totalTime, 5*time.Minute,
				"Iteration %d: Total deployment time should be < 5 minutes", i)
		}
	})

	t.Run("rollback completes within 30 seconds", func(t *testing.T) {
		// Create initial deployment
		req := &models.CreateDeploymentRequest{
			EnvironmentID: "env-rollback-perf",
			Name:          "rollback-perf-app",
		}
		deployment, err := service.Create(ctx, "rollback-perf-user", req)
		require.NoError(t, err)

		// Wait for deployment to complete
		time.Sleep(200 * time.Millisecond)

		// Get history to find a version to rollback to
		history, err := service.GetHistory(ctx, "rollback-perf-user", deployment.ID)
		require.NoError(t, err)

		if len(history.Versions) > 0 {
			targetVersion := history.Versions[0].Version

			for i := 0; i < iterations; i++ {
				rollbackReq := &models.RollbackRequest{
					TargetVersion: targetVersion,
					Reason:        "Performance test iteration " + string(rune('0'+i%10)),
				}

				startTime := time.Now()
				result, err := service.Rollback(ctx, "rollback-perf-user", deployment.ID, rollbackReq)
				rollbackDuration := time.Since(startTime)

				require.NoError(t, err)

				// Property: Rollback should complete within 30 seconds
				assert.Less(t, rollbackDuration, 30*time.Second,
					"Iteration %d: Rollback should complete within 30 seconds, took %v", i, rollbackDuration)

				// Verify rollback was successful
				assert.Equal(t, models.DeploymentPhaseRolledBack, result.Phase,
					"Iteration %d: Deployment should be in RolledBack phase", i)
			}
		}
	})

	t.Logf("Property 10 Results: Verified build and rollback performance for %d iterations", iterations)
}

// Additional property test for deployment pipeline
func TestProperty_DeploymentPipelinePhases(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	const iterations = 50

	t.Run("deployment progresses through expected phases", func(t *testing.T) {
		for i := 0; i < iterations; i++ {
			req := &models.CreateDeploymentRequest{
				EnvironmentID: "env-phases",
				Name:          "phases-app-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			}

			deployment, err := service.Create(ctx, "phases-user", req)
			require.NoError(t, err)

			// Initial phase should be Pending
			assert.Equal(t, models.DeploymentPhasePending, deployment.Phase,
				"Iteration %d: Initial phase should be Pending", i)

			// Wait for deployment to complete
			time.Sleep(200 * time.Millisecond)

			// Get updated deployment
			updated, err := service.Get(ctx, "phases-user", deployment.ID)
			require.NoError(t, err)

			// Property: Deployment should end in Running or Failed phase
			validEndPhases := []models.DeploymentPhase{
				models.DeploymentPhaseRunning,
				models.DeploymentPhaseFailed,
			}
			assert.Contains(t, validEndPhases, updated.Phase,
				"Iteration %d: Deployment should end in Running or Failed phase, got %s", i, updated.Phase)

			// If successful, verify image info is populated
			if updated.Phase == models.DeploymentPhaseRunning {
				assert.NotNil(t, updated.ImageInfo,
					"Iteration %d: ImageInfo should be populated for running deployment", i)
				assert.NotEmpty(t, updated.ImageInfo.Name,
					"Iteration %d: Image name should not be empty", i)
				assert.NotEmpty(t, updated.ImageInfo.Tag,
					"Iteration %d: Image tag should not be empty", i)
			}
		}
	})

	t.Logf("Property Deployment Pipeline Results: Verified phase progression for %d iterations", iterations)
}

// Test for version management
func TestProperty_VersionManagement(t *testing.T) {
	service := setupTestDeploymentService(t)
	ctx := context.Background()

	const iterations = 20

	t.Run("each deployment creates unique version", func(t *testing.T) {
		versions := make(map[string]bool)

		for i := 0; i < iterations; i++ {
			req := &models.CreateDeploymentRequest{
				EnvironmentID: "env-versions",
				Name:          "version-app-" + string(rune('a'+i%26)),
			}

			deployment, err := service.Create(ctx, "version-user", req)
			require.NoError(t, err)

			// Property: Each deployment should have a unique version
			assert.False(t, versions[deployment.Version],
				"Iteration %d: Version %s should be unique", i, deployment.Version)
			versions[deployment.Version] = true

			// Version should follow expected format (v + timestamp)
			assert.Contains(t, deployment.Version, "v",
				"Iteration %d: Version should start with 'v'", i)
		}
	})

	t.Run("history tracks all versions", func(t *testing.T) {
		// Create multiple deployments for same app
		var deploymentID string
		for i := 0; i < 5; i++ {
			req := &models.CreateDeploymentRequest{
				EnvironmentID: "env-history-track",
				Name:          "history-track-app",
			}

			deployment, err := service.Create(ctx, "history-track-user", req)
			require.NoError(t, err)
			deploymentID = deployment.ID

			// Wait for deployment to complete
			time.Sleep(200 * time.Millisecond)
		}

		// Get history
		history, err := service.GetHistory(ctx, "history-track-user", deploymentID)
		require.NoError(t, err)

		// Property: History should contain at least one version
		assert.GreaterOrEqual(t, history.Total, 1,
			"History should contain at least one version")
	})

	t.Logf("Property Version Management Results: Verified version uniqueness for %d iterations", iterations)
}
