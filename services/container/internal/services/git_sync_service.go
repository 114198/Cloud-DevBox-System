// Package services provides business logic services for the container service.
package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
)

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrCloneFailed             = errors.New("clone operation failed")
	ErrSyncFailed              = errors.New("sync operation failed")
	ErrEnvironmentNotFound     = errors.New("environment not found")
)

// GitSyncService handles Git synchronization operations
type GitSyncService struct {
	gitOAuthService *GitOAuthService
	gitRepoService  *GitRepositoryService
	syncHistory     sync.Map // In-memory cache for sync history
	webhooks        sync.Map // In-memory cache for webhooks
	mu              sync.RWMutex
}

// NewGitSyncService creates a new GitSyncService instance
func NewGitSyncService(gitOAuthService *GitOAuthService, gitRepoService *GitRepositoryService) *GitSyncService {
	return &GitSyncService{
		gitOAuthService: gitOAuthService,
		gitRepoService:  gitRepoService,
	}
}

// CloneRepository clones a repository to an environment
func (s *GitSyncService) CloneRepository(ctx context.Context, userID string, req models.CloneRepositoryRequest) (*models.GitSyncResult, error) {
	startTime := time.Now()

	// Get repository
	repo, err := s.gitRepoService.GetRepositoryByID(req.RepositoryID)
	if err != nil {
		return nil, err
	}

	// Verify user owns the repository
	if repo.UserID != userID {
		return nil, ErrRepositoryNotFound
	}

	// Get connection for authentication
	connection, err := s.gitOAuthService.GetConnectionByID(repo.ConnectionID)
	if err != nil {
		return nil, err
	}

	// Get valid token
	token, err := s.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return nil, err
	}

	// Update sync status
	s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusSyncing)

	// Build clone URL with authentication
	cloneURL := s.buildAuthenticatedURL(repo.CloneURL, token, repo.Provider)

	// Determine branch
	branch := req.Branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Build clone command
	args := []string{"clone"}

	// Add shallow clone options for optimization
	if req.Shallow {
		depth := req.Depth
		if depth <= 0 {
			depth = 1 // Default shallow depth
		}
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}

	// Add branch
	args = append(args, "--branch", branch)

	// Add single-branch for optimization
	args = append(args, "--single-branch")

	// Add URL and destination
	workDir := fmt.Sprintf("/workspace/%s", req.EnvironmentID)
	args = append(args, cloneURL, workDir)

	// Execute clone command
	result, err := s.executeGitCommand(ctx, args, "")
	if err != nil {
		s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusFailed)
		return &models.GitSyncResult{
			Success:    false,
			Error:      err.Error(),
			DurationMs: int(time.Since(startTime).Milliseconds()),
			SyncedAt:   time.Now(),
		}, nil
	}

	// Get latest commit info
	commitSHA, commitMessage, err := s.getLatestCommit(ctx, workDir)
	if err != nil {
		commitSHA = "unknown"
		commitMessage = ""
	}

	// Update sync status
	s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusSynced)

	// Record sync history
	s.recordSyncHistory(repo.ID, req.EnvironmentID, commitSHA, commitMessage, branch, models.SyncTypeClone, models.SyncStatusSynced, startTime, nil)

	return &models.GitSyncResult{
		Success:       true,
		CommitSHA:     commitSHA,
		CommitMessage: commitMessage,
		FilesChanged:  result.FilesChanged,
		DurationMs:    int(time.Since(startTime).Milliseconds()),
		SyncedAt:      time.Now(),
	}, nil
}


// SyncRepository syncs a repository with the latest changes
func (s *GitSyncService) SyncRepository(ctx context.Context, userID string, req models.SyncRepositoryRequest) (*models.GitSyncResult, error) {
	startTime := time.Now()

	// Get repository
	repo, err := s.gitRepoService.GetRepositoryByID(req.RepositoryID)
	if err != nil {
		return nil, err
	}

	// Verify user owns the repository
	if repo.UserID != userID {
		return nil, ErrRepositoryNotFound
	}

	// Get connection for authentication
	connection, err := s.gitOAuthService.GetConnectionByID(repo.ConnectionID)
	if err != nil {
		return nil, err
	}

	// Get valid token
	token, err := s.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return nil, err
	}

	// Update sync status
	s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusSyncing)

	workDir := fmt.Sprintf("/workspace/%s", req.EnvironmentID)

	// Configure remote URL with authentication
	remoteURL := s.buildAuthenticatedURL(repo.CloneURL, token, repo.Provider)
	_, err = s.executeGitCommand(ctx, []string{"remote", "set-url", "origin", remoteURL}, workDir)
	if err != nil {
		// If remote doesn't exist, this is expected for first sync
	}

	// Determine branch
	branch := req.Branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Fetch latest changes
	_, err = s.executeGitCommand(ctx, []string{"fetch", "origin", branch}, workDir)
	if err != nil {
		s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusFailed)
		return &models.GitSyncResult{
			Success:    false,
			Error:      fmt.Sprintf("fetch failed: %v", err),
			DurationMs: int(time.Since(startTime).Milliseconds()),
			SyncedAt:   time.Now(),
		}, nil
	}

	// Reset to latest commit (hard reset to handle any local changes)
	_, err = s.executeGitCommand(ctx, []string{"reset", "--hard", fmt.Sprintf("origin/%s", branch)}, workDir)
	if err != nil {
		s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusFailed)
		return &models.GitSyncResult{
			Success:    false,
			Error:      fmt.Sprintf("reset failed: %v", err),
			DurationMs: int(time.Since(startTime).Milliseconds()),
			SyncedAt:   time.Now(),
		}, nil
	}

	// Get latest commit info
	commitSHA, commitMessage, err := s.getLatestCommit(ctx, workDir)
	if err != nil {
		commitSHA = "unknown"
		commitMessage = ""
	}

	// Update sync status
	s.gitRepoService.UpdateSyncStatus(repo.ID, models.SyncStatusSynced)

	// Record sync history
	s.recordSyncHistory(repo.ID, req.EnvironmentID, commitSHA, commitMessage, branch, models.SyncTypePull, models.SyncStatusSynced, startTime, nil)

	return &models.GitSyncResult{
		Success:       true,
		CommitSHA:     commitSHA,
		CommitMessage: commitMessage,
		DurationMs:    int(time.Since(startTime).Milliseconds()),
		SyncedAt:      time.Now(),
	}, nil
}

// buildAuthenticatedURL builds a clone URL with embedded authentication
func (s *GitSyncService) buildAuthenticatedURL(cloneURL, token string, provider models.GitProvider) string {
	// For HTTPS URLs, embed the token
	if strings.HasPrefix(cloneURL, "https://") {
		switch provider {
		case models.GitProviderGitHub:
			// GitHub: https://x-access-token:TOKEN@github.com/owner/repo.git
			return strings.Replace(cloneURL, "https://", fmt.Sprintf("https://x-access-token:%s@", token), 1)
		case models.GitProviderGitLab:
			// GitLab: https://oauth2:TOKEN@gitlab.com/owner/repo.git
			return strings.Replace(cloneURL, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)
		case models.GitProviderGitee:
			// Gitee: https://oauth2:TOKEN@gitee.com/owner/repo.git
			return strings.Replace(cloneURL, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)
		case models.GitProviderBitbucket:
			// Bitbucket: https://x-token-auth:TOKEN@bitbucket.org/owner/repo.git
			return strings.Replace(cloneURL, "https://", fmt.Sprintf("https://x-token-auth:%s@", token), 1)
		}
	}
	return cloneURL
}

// GitCommandResult represents the result of a git command
type GitCommandResult struct {
	Output       string
	FilesChanged int
}

// executeGitCommand executes a git command
func (s *GitSyncService) executeGitCommand(ctx context.Context, args []string, workDir string) (*GitCommandResult, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}

	return &GitCommandResult{
		Output: stdout.String(),
	}, nil
}

// getLatestCommit gets the latest commit SHA and message
func (s *GitSyncService) getLatestCommit(ctx context.Context, workDir string) (string, string, error) {
	// Get commit SHA
	result, err := s.executeGitCommand(ctx, []string{"rev-parse", "HEAD"}, workDir)
	if err != nil {
		return "", "", err
	}
	sha := strings.TrimSpace(result.Output)

	// Get commit message
	result, err = s.executeGitCommand(ctx, []string{"log", "-1", "--pretty=%B"}, workDir)
	if err != nil {
		return sha, "", nil
	}
	message := strings.TrimSpace(result.Output)

	return sha, message, nil
}

// recordSyncHistory records a sync operation in history
func (s *GitSyncService) recordSyncHistory(repoID, envID, commitSHA, commitMessage, branch string, syncType models.SyncType, status models.SyncStatus, startTime time.Time, errMsg *string) {
	history := &models.GitSyncHistory{
		ID:            uuid.New().String(),
		RepositoryID:  repoID,
		EnvironmentID: envID,
		CommitSHA:     commitSHA,
		CommitMessage: commitMessage,
		Branch:        branch,
		SyncType:      syncType,
		Status:        status,
		StartedAt:     startTime,
		DurationMs:    int(time.Since(startTime).Milliseconds()),
	}

	now := time.Now()
	history.CompletedAt = &now

	if errMsg != nil {
		history.ErrorMessage = *errMsg
	}

	// Store in history
	key := fmt.Sprintf("%s:%s", repoID, history.ID)
	s.syncHistory.Store(key, history)
}

// GetSyncHistory returns sync history for a repository
func (s *GitSyncService) GetSyncHistory(ctx context.Context, repositoryID string, limit int) ([]*models.GitSyncHistory, error) {
	var history []*models.GitSyncHistory
	s.syncHistory.Range(func(key, value interface{}) bool {
		h := value.(*models.GitSyncHistory)
		if h.RepositoryID == repositoryID {
			history = append(history, h)
		}
		return true
	})

	// Sort by started_at descending and limit
	// In production, this would be done by the database
	if len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}


// ParseWebhookPayload parses a webhook payload from a Git provider
func (s *GitSyncService) ParseWebhookPayload(r *http.Request, provider models.GitProvider) (*models.WebhookPayload, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	switch provider {
	case models.GitProviderGitHub:
		return s.parseGitHubWebhook(r, body)
	case models.GitProviderGitLab:
		return s.parseGitLabWebhook(r, body)
	case models.GitProviderGitee:
		return s.parseGiteeWebhook(r, body)
	case models.GitProviderBitbucket:
		return s.parseBitbucketWebhook(r, body)
	default:
		return nil, ErrGitProviderNotSupported
	}
}

// parseGitHubWebhook parses a GitHub webhook payload
func (s *GitSyncService) parseGitHubWebhook(r *http.Request, body []byte) (*models.WebhookPayload, error) {
	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")
	signature := r.Header.Get("X-Hub-Signature-256")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Extract repository ID
	var repoID string
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if id, ok := repo["id"].(float64); ok {
			repoID = fmt.Sprintf("%.0f", id)
		}
	}

	// Verify signature if webhook secret is configured
	if signature != "" {
		// Find repository by ID to get webhook secret
		repo, err := s.findRepositoryByProviderRepoID(models.GitProviderGitHub, repoID)
		if err == nil && repo.WebhookSecret != "" {
			if !s.verifyGitHubSignature(body, signature, repo.WebhookSecret) {
				return nil, ErrInvalidWebhookSignature
			}
		}
	}

	// Extract branch and commit info for push events
	var branch, commitSHA string
	if eventType == "push" {
		if ref, ok := payload["ref"].(string); ok {
			branch = strings.TrimPrefix(ref, "refs/heads/")
		}
		if after, ok := payload["after"].(string); ok {
			commitSHA = after
		}
	}

	return &models.WebhookPayload{
		Provider:  models.GitProviderGitHub,
		EventType: eventType,
		EventID:   deliveryID,
		RepoID:    repoID,
		Branch:    branch,
		CommitSHA: commitSHA,
		Payload:   payload,
	}, nil
}

// parseGitLabWebhook parses a GitLab webhook payload
func (s *GitSyncService) parseGitLabWebhook(r *http.Request, body []byte) (*models.WebhookPayload, error) {
	eventType := r.Header.Get("X-Gitlab-Event")
	token := r.Header.Get("X-Gitlab-Token")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Extract project ID
	var repoID string
	if project, ok := payload["project"].(map[string]interface{}); ok {
		if id, ok := project["id"].(float64); ok {
			repoID = fmt.Sprintf("%.0f", id)
		}
	}

	// Verify token if configured
	if token != "" {
		repo, err := s.findRepositoryByProviderRepoID(models.GitProviderGitLab, repoID)
		if err == nil && repo.WebhookSecret != "" && token != repo.WebhookSecret {
			return nil, ErrInvalidWebhookSignature
		}
	}

	// Extract branch and commit info
	var branch, commitSHA string
	if ref, ok := payload["ref"].(string); ok {
		branch = strings.TrimPrefix(ref, "refs/heads/")
	}
	if after, ok := payload["after"].(string); ok {
		commitSHA = after
	}

	// Map GitLab event types
	mappedEventType := "push"
	switch eventType {
	case "Push Hook":
		mappedEventType = "push"
	case "Merge Request Hook":
		mappedEventType = "pull_request"
	case "Tag Push Hook":
		mappedEventType = "create"
	}

	return &models.WebhookPayload{
		Provider:  models.GitProviderGitLab,
		EventType: mappedEventType,
		EventID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		RepoID:    repoID,
		Branch:    branch,
		CommitSHA: commitSHA,
		Payload:   payload,
	}, nil
}

// parseGiteeWebhook parses a Gitee webhook payload
func (s *GitSyncService) parseGiteeWebhook(r *http.Request, body []byte) (*models.WebhookPayload, error) {
	eventType := r.Header.Get("X-Gitee-Event")
	timestamp := r.Header.Get("X-Gitee-Timestamp")
	token := r.Header.Get("X-Gitee-Token")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Extract repository ID
	var repoID string
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if id, ok := repo["id"].(float64); ok {
			repoID = fmt.Sprintf("%.0f", id)
		}
	}

	// Verify token if configured
	if token != "" {
		repo, err := s.findRepositoryByProviderRepoID(models.GitProviderGitee, repoID)
		if err == nil && repo.WebhookSecret != "" && token != repo.WebhookSecret {
			return nil, ErrInvalidWebhookSignature
		}
	}

	// Extract branch and commit info
	var branch, commitSHA string
	if ref, ok := payload["ref"].(string); ok {
		branch = strings.TrimPrefix(ref, "refs/heads/")
	}
	if after, ok := payload["after"].(string); ok {
		commitSHA = after
	}

	// Map Gitee event types
	mappedEventType := "push"
	switch eventType {
	case "Push Hook":
		mappedEventType = "push"
	case "Merge Request Hook":
		mappedEventType = "pull_request"
	case "Tag Push Hook":
		mappedEventType = "create"
	}

	_ = timestamp // Can be used for additional validation

	return &models.WebhookPayload{
		Provider:  models.GitProviderGitee,
		EventType: mappedEventType,
		EventID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		RepoID:    repoID,
		Branch:    branch,
		CommitSHA: commitSHA,
		Payload:   payload,
	}, nil
}

// parseBitbucketWebhook parses a Bitbucket webhook payload
func (s *GitSyncService) parseBitbucketWebhook(r *http.Request, body []byte) (*models.WebhookPayload, error) {
	eventType := r.Header.Get("X-Event-Key")
	hookUUID := r.Header.Get("X-Hook-UUID")

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Extract repository UUID
	var repoID string
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if uuid, ok := repo["uuid"].(string); ok {
			repoID = uuid
		}
	}

	// Extract branch and commit info for push events
	var branch, commitSHA string
	if eventType == "repo:push" {
		if push, ok := payload["push"].(map[string]interface{}); ok {
			if changes, ok := push["changes"].([]interface{}); ok && len(changes) > 0 {
				if change, ok := changes[0].(map[string]interface{}); ok {
					if newObj, ok := change["new"].(map[string]interface{}); ok {
						if name, ok := newObj["name"].(string); ok {
							branch = name
						}
						if target, ok := newObj["target"].(map[string]interface{}); ok {
							if hash, ok := target["hash"].(string); ok {
								commitSHA = hash
							}
						}
					}
				}
			}
		}
	}

	// Map Bitbucket event types
	mappedEventType := "push"
	switch eventType {
	case "repo:push":
		mappedEventType = "push"
	case "pullrequest:created", "pullrequest:updated":
		mappedEventType = "pull_request"
	case "repo:commit_status_created":
		mappedEventType = "create"
	}

	return &models.WebhookPayload{
		Provider:  models.GitProviderBitbucket,
		EventType: mappedEventType,
		EventID:   hookUUID,
		RepoID:    repoID,
		Branch:    branch,
		CommitSHA: commitSHA,
		Payload:   payload,
	}, nil
}

// verifyGitHubSignature verifies a GitHub webhook signature
func (s *GitSyncService) verifyGitHubSignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedMAC := signature[7:]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	actualMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedMAC), []byte(actualMAC))
}

// findRepositoryByProviderRepoID finds a repository by provider and repo ID
func (s *GitSyncService) findRepositoryByProviderRepoID(provider models.GitProvider, repoID string) (*models.GitRepository, error) {
	var found *models.GitRepository
	s.gitRepoService.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)
		if repo.Provider == provider && repo.RepoID == repoID {
			found = repo
			return false
		}
		return true
	})
	if found == nil {
		return nil, ErrRepositoryNotFound
	}
	return found, nil
}


// ProcessWebhook processes a webhook event
func (s *GitSyncService) ProcessWebhook(ctx context.Context, payload *models.WebhookPayload) error {
	// Find repository by provider and repo ID
	repo, err := s.findRepositoryByProviderRepoID(payload.Provider, payload.RepoID)
	if err != nil {
		return err
	}

	// Store webhook event
	webhook := &models.GitWebhook{
		ID:           uuid.New().String(),
		RepositoryID: repo.ID,
		EventType:    payload.EventType,
		EventID:      payload.EventID,
		Payload:      payload.Payload,
		Processed:    false,
		CreatedAt:    time.Now(),
	}
	s.webhooks.Store(webhook.ID, webhook)

	// Process based on event type
	switch payload.EventType {
	case "push":
		return s.handlePushEvent(ctx, repo, payload)
	case "pull_request":
		return s.handlePullRequestEvent(ctx, repo, payload)
	default:
		// Mark as processed without action
		s.markWebhookProcessed(webhook.ID, nil)
		return nil
	}
}

// handlePushEvent handles a push webhook event
func (s *GitSyncService) handlePushEvent(ctx context.Context, repo *models.GitRepository, payload *models.WebhookPayload) error {
	// Only sync if the push is to the tracked branch
	if payload.Branch != "" && payload.Branch != repo.DefaultBranch {
		return nil
	}

	// Trigger sync
	result, err := s.SyncRepository(ctx, repo.UserID, models.SyncRepositoryRequest{
		RepositoryID:  repo.ID,
		EnvironmentID: repo.EnvironmentID,
		Branch:        payload.Branch,
	})

	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("sync failed: %s", result.Error)
	}

	// Trigger environment update (dependency installation, etc.)
	go s.triggerEnvironmentUpdate(ctx, repo.EnvironmentID, result)

	return nil
}

// handlePullRequestEvent handles a pull request webhook event
func (s *GitSyncService) handlePullRequestEvent(ctx context.Context, repo *models.GitRepository, payload *models.WebhookPayload) error {
	// For now, just log the event
	// In a full implementation, this could create preview environments
	return nil
}

// markWebhookProcessed marks a webhook as processed
func (s *GitSyncService) markWebhookProcessed(webhookID string, err error) {
	if value, ok := s.webhooks.Load(webhookID); ok {
		webhook := value.(*models.GitWebhook)
		webhook.Processed = true
		now := time.Now()
		webhook.ProcessedAt = &now
		if err != nil {
			webhook.ErrorMessage = err.Error()
		}
	}
}

// triggerEnvironmentUpdate triggers an environment update after sync
func (s *GitSyncService) triggerEnvironmentUpdate(ctx context.Context, environmentID string, syncResult *models.GitSyncResult) {
	// This would integrate with the environment service to:
	// 1. Detect package manager (npm, pip, go mod, etc.)
	// 2. Install/update dependencies
	// 3. Restart development servers if needed
	// 4. Notify the user of the update

	// For now, this is a placeholder
	fmt.Printf("Environment %s updated with commit %s\n", environmentID, syncResult.CommitSHA)
}

// GetWebhookEvents returns webhook events for a repository
func (s *GitSyncService) GetWebhookEvents(ctx context.Context, repositoryID string, limit int) ([]*models.GitWebhook, error) {
	var webhooks []*models.GitWebhook
	s.webhooks.Range(func(key, value interface{}) bool {
		webhook := value.(*models.GitWebhook)
		if webhook.RepositoryID == repositoryID {
			webhooks = append(webhooks, webhook)
		}
		return true
	})

	// Sort by created_at descending and limit
	if len(webhooks) > limit {
		webhooks = webhooks[:limit]
	}

	return webhooks, nil
}

// RetryWebhook retries processing a failed webhook
func (s *GitSyncService) RetryWebhook(ctx context.Context, webhookID string) error {
	value, ok := s.webhooks.Load(webhookID)
	if !ok {
		return errors.New("webhook not found")
	}

	webhook := value.(*models.GitWebhook)

	// Reconstruct payload
	payload := &models.WebhookPayload{
		EventType: webhook.EventType,
		EventID:   webhook.EventID,
		Payload:   webhook.Payload,
	}

	// Get repository to determine provider
	repo, err := s.gitRepoService.GetRepositoryByID(webhook.RepositoryID)
	if err != nil {
		return err
	}

	payload.Provider = repo.Provider
	payload.RepoID = repo.RepoID

	// Reprocess
	return s.ProcessWebhook(ctx, payload)
}
