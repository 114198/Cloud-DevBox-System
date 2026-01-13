// Package services provides business logic services for the container service.
package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
)

var (
	ErrPackageManagerNotDetected = errors.New("package manager not detected")
	ErrDependencyInstallFailed   = errors.New("dependency installation failed")
)

// PackageManager represents a detected package manager
type PackageManager string

const (
	PackageManagerNPM    PackageManager = "npm"
	PackageManagerYarn   PackageManager = "yarn"
	PackageManagerPNPM   PackageManager = "pnpm"
	PackageManagerPip    PackageManager = "pip"
	PackageManagerPoetry PackageManager = "poetry"
	PackageManagerGoMod  PackageManager = "go"
	PackageManagerCargo  PackageManager = "cargo"
	PackageManagerMaven  PackageManager = "maven"
	PackageManagerGradle PackageManager = "gradle"
	PackageManagerComposer PackageManager = "composer"
)

// EnvironmentUpdateResult represents the result of an environment update
type EnvironmentUpdateResult struct {
	Success          bool              `json:"success"`
	PackageManager   PackageManager    `json:"packageManager,omitempty"`
	DependenciesUpdated bool           `json:"dependenciesUpdated"`
	ServerRestarted  bool              `json:"serverRestarted"`
	DurationMs       int               `json:"durationMs"`
	Error            string            `json:"error,omitempty"`
	Logs             []string          `json:"logs,omitempty"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// EnvironmentUpdateConfig represents configuration for environment updates
type EnvironmentUpdateConfig struct {
	AutoInstallDependencies bool     `json:"autoInstallDependencies"`
	AutoRestartServer       bool     `json:"autoRestartServer"`
	PreUpdateCommands       []string `json:"preUpdateCommands,omitempty"`
	PostUpdateCommands      []string `json:"postUpdateCommands,omitempty"`
}

// GitEnvironmentUpdateService handles automatic environment updates after Git operations
type GitEnvironmentUpdateService struct {
	gitSyncService *GitSyncService
	updateHistory  sync.Map
	mu             sync.RWMutex
}

// NewGitEnvironmentUpdateService creates a new GitEnvironmentUpdateService instance
func NewGitEnvironmentUpdateService(gitSyncService *GitSyncService) *GitEnvironmentUpdateService {
	return &GitEnvironmentUpdateService{
		gitSyncService: gitSyncService,
	}
}

// UpdateEnvironmentAfterSync updates an environment after a Git sync operation
func (s *GitEnvironmentUpdateService) UpdateEnvironmentAfterSync(ctx context.Context, environmentID string, syncResult *models.GitSyncResult, config *EnvironmentUpdateConfig) (*EnvironmentUpdateResult, error) {
	startTime := time.Now()
	result := &EnvironmentUpdateResult{
		Success:   true,
		Logs:      []string{},
		UpdatedAt: time.Now(),
	}

	workDir := fmt.Sprintf("/workspace/%s", environmentID)

	// Run pre-update commands
	if config != nil && len(config.PreUpdateCommands) > 0 {
		for _, cmd := range config.PreUpdateCommands {
			output, err := s.executeCommand(ctx, workDir, "sh", "-c", cmd)
			result.Logs = append(result.Logs, fmt.Sprintf("Pre-update: %s", cmd))
			if err != nil {
				result.Logs = append(result.Logs, fmt.Sprintf("Error: %v", err))
			} else {
				result.Logs = append(result.Logs, output)
			}
		}
	}

	// Detect package manager
	pm, err := s.detectPackageManager(workDir)
	if err != nil {
		result.Logs = append(result.Logs, "No package manager detected, skipping dependency installation")
	} else {
		result.PackageManager = pm
		result.Logs = append(result.Logs, fmt.Sprintf("Detected package manager: %s", pm))

		// Install dependencies if enabled
		if config == nil || config.AutoInstallDependencies {
			err = s.installDependencies(ctx, workDir, pm, result)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				result.DurationMs = int(time.Since(startTime).Milliseconds())
				return result, nil
			}
			result.DependenciesUpdated = true
		}
	}

	// Run post-update commands
	if config != nil && len(config.PostUpdateCommands) > 0 {
		for _, cmd := range config.PostUpdateCommands {
			output, err := s.executeCommand(ctx, workDir, "sh", "-c", cmd)
			result.Logs = append(result.Logs, fmt.Sprintf("Post-update: %s", cmd))
			if err != nil {
				result.Logs = append(result.Logs, fmt.Sprintf("Error: %v", err))
			} else {
				result.Logs = append(result.Logs, output)
			}
		}
	}

	// Restart development server if enabled
	if config != nil && config.AutoRestartServer {
		err = s.restartDevServer(ctx, environmentID, pm)
		if err != nil {
			result.Logs = append(result.Logs, fmt.Sprintf("Failed to restart server: %v", err))
		} else {
			result.ServerRestarted = true
			result.Logs = append(result.Logs, "Development server restarted")
		}
	}

	result.DurationMs = int(time.Since(startTime).Milliseconds())

	// Store update history
	s.storeUpdateHistory(environmentID, result)

	return result, nil
}


// detectPackageManager detects the package manager used in a project
func (s *GitEnvironmentUpdateService) detectPackageManager(workDir string) (PackageManager, error) {
	// Check for Node.js package managers
	if _, err := os.Stat(filepath.Join(workDir, "pnpm-lock.yaml")); err == nil {
		return PackageManagerPNPM, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "yarn.lock")); err == nil {
		return PackageManagerYarn, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "package-lock.json")); err == nil {
		return PackageManagerNPM, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "package.json")); err == nil {
		return PackageManagerNPM, nil
	}

	// Check for Python package managers
	if _, err := os.Stat(filepath.Join(workDir, "poetry.lock")); err == nil {
		return PackageManagerPoetry, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "Pipfile.lock")); err == nil {
		return PackageManagerPip, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "requirements.txt")); err == nil {
		return PackageManagerPip, nil
	}

	// Check for Go
	if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
		return PackageManagerGoMod, nil
	}

	// Check for Rust
	if _, err := os.Stat(filepath.Join(workDir, "Cargo.toml")); err == nil {
		return PackageManagerCargo, nil
	}

	// Check for Java
	if _, err := os.Stat(filepath.Join(workDir, "pom.xml")); err == nil {
		return PackageManagerMaven, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "build.gradle")); err == nil {
		return PackageManagerGradle, nil
	}
	if _, err := os.Stat(filepath.Join(workDir, "build.gradle.kts")); err == nil {
		return PackageManagerGradle, nil
	}

	// Check for PHP
	if _, err := os.Stat(filepath.Join(workDir, "composer.json")); err == nil {
		return PackageManagerComposer, nil
	}

	return "", ErrPackageManagerNotDetected
}

// installDependencies installs dependencies using the detected package manager
func (s *GitEnvironmentUpdateService) installDependencies(ctx context.Context, workDir string, pm PackageManager, result *EnvironmentUpdateResult) error {
	var cmd string
	var args []string

	switch pm {
	case PackageManagerNPM:
		cmd = "npm"
		args = []string{"install", "--prefer-offline"}
	case PackageManagerYarn:
		cmd = "yarn"
		args = []string{"install", "--prefer-offline"}
	case PackageManagerPNPM:
		cmd = "pnpm"
		args = []string{"install", "--prefer-offline"}
	case PackageManagerPip:
		cmd = "pip"
		args = []string{"install", "-r", "requirements.txt"}
	case PackageManagerPoetry:
		cmd = "poetry"
		args = []string{"install"}
	case PackageManagerGoMod:
		cmd = "go"
		args = []string{"mod", "download"}
	case PackageManagerCargo:
		cmd = "cargo"
		args = []string{"fetch"}
	case PackageManagerMaven:
		cmd = "mvn"
		args = []string{"dependency:resolve", "-q"}
	case PackageManagerGradle:
		cmd = "./gradlew"
		args = []string{"dependencies", "--quiet"}
	case PackageManagerComposer:
		cmd = "composer"
		args = []string{"install", "--no-interaction"}
	default:
		return ErrPackageManagerNotDetected
	}

	result.Logs = append(result.Logs, fmt.Sprintf("Running: %s %s", cmd, strings.Join(args, " ")))

	output, err := s.executeCommand(ctx, workDir, cmd, args...)
	if err != nil {
		result.Logs = append(result.Logs, fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("%w: %v", ErrDependencyInstallFailed, err)
	}

	// Truncate output if too long
	if len(output) > 1000 {
		output = output[:1000] + "... (truncated)"
	}
	result.Logs = append(result.Logs, output)

	return nil
}

// restartDevServer restarts the development server
func (s *GitEnvironmentUpdateService) restartDevServer(ctx context.Context, environmentID string, pm PackageManager) error {
	// This would integrate with the environment service to restart the dev server
	// For now, this is a placeholder that would:
	// 1. Find running dev server process
	// 2. Send SIGTERM to gracefully stop it
	// 3. Start a new dev server process

	// The actual implementation would depend on how dev servers are managed
	// (e.g., systemd, supervisor, or direct process management)

	return nil
}

// executeCommand executes a command in the specified directory
func (s *GitEnvironmentUpdateService) executeCommand(ctx context.Context, workDir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// storeUpdateHistory stores an update result in history
func (s *GitEnvironmentUpdateService) storeUpdateHistory(environmentID string, result *EnvironmentUpdateResult) {
	key := fmt.Sprintf("%s:%s", environmentID, uuid.New().String())
	s.updateHistory.Store(key, result)
}

// GetUpdateHistory returns update history for an environment
func (s *GitEnvironmentUpdateService) GetUpdateHistory(ctx context.Context, environmentID string, limit int) ([]*EnvironmentUpdateResult, error) {
	var history []*EnvironmentUpdateResult
	s.updateHistory.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if strings.HasPrefix(keyStr, environmentID+":") {
			history = append(history, value.(*EnvironmentUpdateResult))
		}
		return true
	})

	// Sort by updated_at descending and limit
	if len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}

// CheckForUpdates checks if there are pending updates for an environment
func (s *GitEnvironmentUpdateService) CheckForUpdates(ctx context.Context, environmentID string, repositoryID string) (bool, *models.GitCommit, error) {
	// Get repository
	repo, err := s.gitSyncService.gitRepoService.GetRepositoryByID(repositoryID)
	if err != nil {
		return false, nil, err
	}

	// Get connection
	connection, err := s.gitSyncService.gitOAuthService.GetConnectionByID(repo.ConnectionID)
	if err != nil {
		return false, nil, err
	}

	// Get valid token
	token, err := s.gitSyncService.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return false, nil, err
	}

	// Get latest commit from remote
	latestCommit, err := s.getLatestRemoteCommit(ctx, repo.Provider, repo.RepoFullName, repo.DefaultBranch, token)
	if err != nil {
		return false, nil, err
	}

	// Get current commit in environment
	workDir := fmt.Sprintf("/workspace/%s", environmentID)
	currentSHA, _, err := s.gitSyncService.getLatestCommit(ctx, workDir)
	if err != nil {
		return false, nil, err
	}

	// Compare commits
	hasUpdates := currentSHA != latestCommit.SHA

	return hasUpdates, latestCommit, nil
}

// getLatestRemoteCommit gets the latest commit from the remote repository
func (s *GitEnvironmentUpdateService) getLatestRemoteCommit(ctx context.Context, provider models.GitProvider, repoFullName, branch, token string) (*models.GitCommit, error) {
	// This would call the provider's API to get the latest commit
	// For now, return a placeholder
	return &models.GitCommit{
		SHA:       "placeholder",
		Message:   "Latest commit",
		Author:    "Unknown",
		Timestamp: time.Now(),
	}, nil
}

// ScheduleAutoSync schedules automatic sync for repositories
func (s *GitEnvironmentUpdateService) ScheduleAutoSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndSyncAllRepositories(ctx)
		}
	}
}

// checkAndSyncAllRepositories checks all repositories for updates and syncs if needed
func (s *GitEnvironmentUpdateService) checkAndSyncAllRepositories(ctx context.Context) {
	s.gitSyncService.gitRepoService.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)

		// Check for updates
		hasUpdates, _, err := s.CheckForUpdates(ctx, repo.EnvironmentID, repo.ID)
		if err != nil {
			fmt.Printf("Error checking updates for repo %s: %v\n", repo.ID, err)
			return true
		}

		if hasUpdates {
			// Trigger sync
			_, err := s.gitSyncService.SyncRepository(ctx, repo.UserID, models.SyncRepositoryRequest{
				RepositoryID:  repo.ID,
				EnvironmentID: repo.EnvironmentID,
			})
			if err != nil {
				fmt.Printf("Error syncing repo %s: %v\n", repo.ID, err)
			}
		}

		return true
	})
}
