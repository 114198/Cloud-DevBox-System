// Package services provides business logic services for the container service.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
)

var (
	ErrRepositoryNotFound      = errors.New("repository not found")
	ErrRepositoryAlreadyLinked = errors.New("repository already linked to this environment")
	ErrNoGitConnection         = errors.New("no git connection found for this provider")
)

// GitRepositoryService handles Git repository operations
type GitRepositoryService struct {
	gitOAuthService *GitOAuthService
	repositories    sync.Map // In-memory cache, replace with DB in production
	mu              sync.RWMutex
}

// NewGitRepositoryService creates a new GitRepositoryService instance
func NewGitRepositoryService(gitOAuthService *GitOAuthService) *GitRepositoryService {
	return &GitRepositoryService{
		gitOAuthService: gitOAuthService,
	}
}

// ListRemoteRepositories lists repositories from a Git provider
func (s *GitRepositoryService) ListRemoteRepositories(ctx context.Context, userID string, req models.ListRepositoriesRequest) ([]models.GitRepoInfo, int, error) {
	// Get connection for the provider
	connection, err := s.gitOAuthService.GetConnection(userID, req.Provider)
	if err != nil {
		return nil, 0, ErrNoGitConnection
	}

	// Get valid token
	token, err := s.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return nil, 0, err
	}

	// Fetch repositories based on provider
	switch req.Provider {
	case models.GitProviderGitHub:
		return s.listGitHubRepositories(ctx, token, req)
	case models.GitProviderGitLab:
		return s.listGitLabRepositories(ctx, token, req)
	case models.GitProviderGitee:
		return s.listGiteeRepositories(ctx, token, req)
	case models.GitProviderBitbucket:
		return s.listBitbucketRepositories(ctx, token, req)
	default:
		return nil, 0, ErrGitProviderNotSupported
	}
}

// listGitHubRepositories lists repositories from GitHub
func (s *GitRepositoryService) listGitHubRepositories(ctx context.Context, token string, req models.ListRepositoriesRequest) ([]models.GitRepoInfo, int, error) {
	url := fmt.Sprintf("https://api.github.com/user/repos?page=%d&per_page=%d&sort=updated", req.Page, req.PageSize)
	if req.Search != "" {
		url = fmt.Sprintf("https://api.github.com/search/repositories?q=%s+user:@me&page=%d&per_page=%d", req.Search, req.Page, req.PageSize)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	var repos []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		CloneURL      string `json:"clone_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
		Size          int64  `json:"size"`
		Language      string `json:"language"`
	}

	if req.Search != "" {
		var searchResult struct {
			TotalCount int `json:"total_count"`
			Items      []struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				FullName      string `json:"full_name"`
				Description   string `json:"description"`
				CloneURL      string `json:"clone_url"`
				SSHURL        string `json:"ssh_url"`
				DefaultBranch string `json:"default_branch"`
				Private       bool   `json:"private"`
				Size          int64  `json:"size"`
				Language      string `json:"language"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &searchResult); err != nil {
			return nil, 0, err
		}
		for _, item := range searchResult.Items {
			repos = append(repos, struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				FullName      string `json:"full_name"`
				Description   string `json:"description"`
				CloneURL      string `json:"clone_url"`
				SSHURL        string `json:"ssh_url"`
				DefaultBranch string `json:"default_branch"`
				Private       bool   `json:"private"`
				Size          int64  `json:"size"`
				Language      string `json:"language"`
			}(item))
		}
	} else {
		if err := json.Unmarshal(body, &repos); err != nil {
			return nil, 0, err
		}
	}

	result := make([]models.GitRepoInfo, len(repos))
	for i, repo := range repos {
		result[i] = models.GitRepoInfo{
			ID:            fmt.Sprintf("%d", repo.ID),
			Name:          repo.Name,
			FullName:      repo.FullName,
			Description:   repo.Description,
			CloneURL:      repo.CloneURL,
			SSHURL:        repo.SSHURL,
			DefaultBranch: repo.DefaultBranch,
			IsPrivate:     repo.Private,
			Size:          repo.Size,
			Language:      repo.Language,
		}
	}

	return result, len(result), nil
}


// listGitLabRepositories lists repositories from GitLab
func (s *GitRepositoryService) listGitLabRepositories(ctx context.Context, token string, req models.ListRepositoriesRequest) ([]models.GitRepoInfo, int, error) {
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects?membership=true&page=%d&per_page=%d&order_by=updated_at", req.Page, req.PageSize)
	if req.Search != "" {
		url += "&search=" + req.Search
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	var repos []struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		SSHURLToRepo      string `json:"ssh_url_to_repo"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
	}
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, 0, err
	}

	result := make([]models.GitRepoInfo, len(repos))
	for i, repo := range repos {
		result[i] = models.GitRepoInfo{
			ID:            fmt.Sprintf("%d", repo.ID),
			Name:          repo.Name,
			FullName:      repo.PathWithNamespace,
			Description:   repo.Description,
			CloneURL:      repo.HTTPURLToRepo,
			SSHURL:        repo.SSHURLToRepo,
			DefaultBranch: repo.DefaultBranch,
			IsPrivate:     repo.Visibility == "private",
		}
	}

	return result, len(result), nil
}

// listGiteeRepositories lists repositories from Gitee
func (s *GitRepositoryService) listGiteeRepositories(ctx context.Context, token string, req models.ListRepositoriesRequest) ([]models.GitRepoInfo, int, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/user/repos?access_token=%s&page=%d&per_page=%d&sort=updated", token, req.Page, req.PageSize)
	if req.Search != "" {
		url += "&q=" + req.Search
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	var repos []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		HTMLURL       string `json:"html_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
	}
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, 0, err
	}

	result := make([]models.GitRepoInfo, len(repos))
	for i, repo := range repos {
		cloneURL := repo.HTMLURL + ".git"
		result[i] = models.GitRepoInfo{
			ID:            fmt.Sprintf("%d", repo.ID),
			Name:          repo.Name,
			FullName:      repo.FullName,
			Description:   repo.Description,
			CloneURL:      cloneURL,
			SSHURL:        repo.SSHURL,
			DefaultBranch: repo.DefaultBranch,
			IsPrivate:     repo.Private,
		}
	}

	return result, len(result), nil
}

// listBitbucketRepositories lists repositories from Bitbucket
func (s *GitRepositoryService) listBitbucketRepositories(ctx context.Context, token string, req models.ListRepositoriesRequest) ([]models.GitRepoInfo, int, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories?role=member&page=%d&pagelen=%d", req.Page, req.PageSize)
	if req.Search != "" {
		url += "&q=name~\"" + req.Search + "\""
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	var response struct {
		Values []struct {
			UUID        string `json:"uuid"`
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			IsPrivate   bool   `json:"is_private"`
			MainBranch  struct {
				Name string `json:"name"`
			} `json:"mainbranch"`
			Links struct {
				Clone []struct {
					Href string `json:"href"`
					Name string `json:"name"`
				} `json:"clone"`
			} `json:"links"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, 0, err
	}

	result := make([]models.GitRepoInfo, len(response.Values))
	for i, repo := range response.Values {
		var cloneURL, sshURL string
		for _, link := range repo.Links.Clone {
			if link.Name == "https" {
				cloneURL = link.Href
			} else if link.Name == "ssh" {
				sshURL = link.Href
			}
		}

		result[i] = models.GitRepoInfo{
			ID:            repo.UUID,
			Name:          repo.Name,
			FullName:      repo.FullName,
			Description:   repo.Description,
			CloneURL:      cloneURL,
			SSHURL:        sshURL,
			DefaultBranch: repo.MainBranch.Name,
			IsPrivate:     repo.IsPrivate,
		}
	}

	return result, len(result), nil
}


// GetRepositoryInfo gets detailed info about a repository
func (s *GitRepositoryService) GetRepositoryInfo(ctx context.Context, userID string, provider models.GitProvider, repoID string) (*models.GitRepoInfo, error) {
	connection, err := s.gitOAuthService.GetConnection(userID, provider)
	if err != nil {
		return nil, ErrNoGitConnection
	}

	token, err := s.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return nil, err
	}

	switch provider {
	case models.GitProviderGitHub:
		return s.getGitHubRepoInfo(ctx, token, repoID)
	case models.GitProviderGitLab:
		return s.getGitLabRepoInfo(ctx, token, repoID)
	case models.GitProviderGitee:
		return s.getGiteeRepoInfo(ctx, token, repoID)
	case models.GitProviderBitbucket:
		return s.getBitbucketRepoInfo(ctx, token, repoID)
	default:
		return nil, ErrGitProviderNotSupported
	}
}

// getGitHubRepoInfo gets repository info from GitHub
func (s *GitRepositoryService) getGitHubRepoInfo(ctx context.Context, token string, repoID string) (*models.GitRepoInfo, error) {
	// GitHub API requires owner/repo format, but we have numeric ID
	// First, get repo by ID
	url := fmt.Sprintf("https://api.github.com/repositories/%s", repoID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrRepositoryNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repo struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		CloneURL      string `json:"clone_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
		Size          int64  `json:"size"`
		Language      string `json:"language"`
	}
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, err
	}

	return &models.GitRepoInfo{
		ID:            fmt.Sprintf("%d", repo.ID),
		Name:          repo.Name,
		FullName:      repo.FullName,
		Description:   repo.Description,
		CloneURL:      repo.CloneURL,
		SSHURL:        repo.SSHURL,
		DefaultBranch: repo.DefaultBranch,
		IsPrivate:     repo.Private,
		Size:          repo.Size,
		Language:      repo.Language,
	}, nil
}

// getGitLabRepoInfo gets repository info from GitLab
func (s *GitRepositoryService) getGitLabRepoInfo(ctx context.Context, token string, repoID string) (*models.GitRepoInfo, error) {
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", repoID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrRepositoryNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repo struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		SSHURLToRepo      string `json:"ssh_url_to_repo"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
	}
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, err
	}

	return &models.GitRepoInfo{
		ID:            fmt.Sprintf("%d", repo.ID),
		Name:          repo.Name,
		FullName:      repo.PathWithNamespace,
		Description:   repo.Description,
		CloneURL:      repo.HTTPURLToRepo,
		SSHURL:        repo.SSHURLToRepo,
		DefaultBranch: repo.DefaultBranch,
		IsPrivate:     repo.Visibility == "private",
	}, nil
}

// getGiteeRepoInfo gets repository info from Gitee
func (s *GitRepositoryService) getGiteeRepoInfo(ctx context.Context, token string, repoID string) (*models.GitRepoInfo, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/projects/%s?access_token=%s", repoID, token)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrRepositoryNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repo struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		HTMLURL       string `json:"html_url"`
		SSHURL        string `json:"ssh_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
	}
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, err
	}

	return &models.GitRepoInfo{
		ID:            fmt.Sprintf("%d", repo.ID),
		Name:          repo.Name,
		FullName:      repo.FullName,
		Description:   repo.Description,
		CloneURL:      repo.HTMLURL + ".git",
		SSHURL:        repo.SSHURL,
		DefaultBranch: repo.DefaultBranch,
		IsPrivate:     repo.Private,
	}, nil
}

// getBitbucketRepoInfo gets repository info from Bitbucket
func (s *GitRepositoryService) getBitbucketRepoInfo(ctx context.Context, token string, repoID string) (*models.GitRepoInfo, error) {
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s", repoID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrRepositoryNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repo struct {
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		IsPrivate   bool   `json:"is_private"`
		MainBranch  struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
		Links struct {
			Clone []struct {
				Href string `json:"href"`
				Name string `json:"name"`
			} `json:"clone"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, err
	}

	var cloneURL, sshURL string
	for _, link := range repo.Links.Clone {
		if link.Name == "https" {
			cloneURL = link.Href
		} else if link.Name == "ssh" {
			sshURL = link.Href
		}
	}

	return &models.GitRepoInfo{
		ID:            repo.UUID,
		Name:          repo.Name,
		FullName:      repo.FullName,
		Description:   repo.Description,
		CloneURL:      cloneURL,
		SSHURL:        sshURL,
		DefaultBranch: repo.MainBranch.Name,
		IsPrivate:     repo.IsPrivate,
	}, nil
}


// LinkRepository links a repository to an environment
func (s *GitRepositoryService) LinkRepository(ctx context.Context, userID string, req models.LinkRepositoryRequest) (*models.GitRepository, error) {
	// Get connection
	connection, err := s.gitOAuthService.GetConnection(userID, req.Provider)
	if err != nil {
		return nil, ErrNoGitConnection
	}

	// Get repository info
	repoInfo, err := s.GetRepositoryInfo(ctx, userID, req.Provider, req.RepoID)
	if err != nil {
		return nil, err
	}

	// Check if already linked
	key := fmt.Sprintf("%s:%s:%s", req.EnvironmentID, req.Provider, req.RepoID)
	if _, ok := s.repositories.Load(key); ok {
		return nil, ErrRepositoryAlreadyLinked
	}

	// Create repository record
	branch := req.Branch
	if branch == "" {
		branch = repoInfo.DefaultBranch
	}

	repo := &models.GitRepository{
		ID:            uuid.New().String(),
		UserID:        userID,
		ConnectionID:  connection.ID,
		EnvironmentID: req.EnvironmentID,
		Provider:      req.Provider,
		RepoID:        req.RepoID,
		RepoName:      repoInfo.Name,
		RepoFullName:  repoInfo.FullName,
		CloneURL:      repoInfo.CloneURL,
		SSHURL:        repoInfo.SSHURL,
		DefaultBranch: branch,
		IsPrivate:     repoInfo.IsPrivate,
		SyncStatus:    models.SyncStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Setup webhook if requested
	if req.EnableWebhook {
		webhookID, webhookSecret, err := s.setupWebhook(ctx, userID, req.Provider, repoInfo.FullName)
		if err != nil {
			// Log error but don't fail the link operation
			fmt.Printf("Failed to setup webhook: %v\n", err)
		} else {
			repo.WebhookID = webhookID
			repo.WebhookSecret = webhookSecret
		}
	}

	// Store repository
	s.repositories.Store(key, repo)

	return repo, nil
}

// UnlinkRepository unlinks a repository from an environment
func (s *GitRepositoryService) UnlinkRepository(ctx context.Context, userID string, repoID string) error {
	// Find and delete the repository
	var found bool
	s.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)
		if repo.ID == repoID && repo.UserID == userID {
			// Delete webhook if exists
			if repo.WebhookID != "" {
				s.deleteWebhook(ctx, userID, repo.Provider, repo.RepoFullName, repo.WebhookID)
			}
			s.repositories.Delete(key)
			found = true
			return false
		}
		return true
	})

	if !found {
		return ErrRepositoryNotFound
	}

	return nil
}

// GetLinkedRepositories returns all repositories linked to an environment
func (s *GitRepositoryService) GetLinkedRepositories(ctx context.Context, userID string, environmentID string) ([]*models.GitRepository, error) {
	var repos []*models.GitRepository
	s.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)
		if repo.UserID == userID && repo.EnvironmentID == environmentID {
			repos = append(repos, repo)
		}
		return true
	})
	return repos, nil
}

// GetRepositoryByID returns a repository by ID
func (s *GitRepositoryService) GetRepositoryByID(repoID string) (*models.GitRepository, error) {
	var found *models.GitRepository
	s.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)
		if repo.ID == repoID {
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

// UpdateSyncStatus updates the sync status of a repository
func (s *GitRepositoryService) UpdateSyncStatus(repoID string, status models.SyncStatus) error {
	var found bool
	s.repositories.Range(func(key, value interface{}) bool {
		repo := value.(*models.GitRepository)
		if repo.ID == repoID {
			repo.SyncStatus = status
			repo.UpdatedAt = time.Now()
			if status == models.SyncStatusSynced {
				now := time.Now()
				repo.LastSyncAt = &now
			}
			found = true
			return false
		}
		return true
	})
	if !found {
		return ErrRepositoryNotFound
	}
	return nil
}

// setupWebhook sets up a webhook for a repository
func (s *GitRepositoryService) setupWebhook(ctx context.Context, userID string, provider models.GitProvider, repoFullName string) (string, string, error) {
	connection, err := s.gitOAuthService.GetConnection(userID, provider)
	if err != nil {
		return "", "", err
	}

	token, err := s.gitOAuthService.GetValidToken(ctx, connection)
	if err != nil {
		return "", "", err
	}

	// Generate webhook secret
	secret := uuid.New().String()

	// Webhook URL (should be configured)
	webhookURL := fmt.Sprintf("https://api.devbox.com/api/v1/git/webhook/%s", provider)

	switch provider {
	case models.GitProviderGitHub:
		return s.setupGitHubWebhook(ctx, token, repoFullName, webhookURL, secret)
	case models.GitProviderGitLab:
		return s.setupGitLabWebhook(ctx, token, repoFullName, webhookURL, secret)
	case models.GitProviderGitee:
		return s.setupGiteeWebhook(ctx, token, repoFullName, webhookURL, secret)
	case models.GitProviderBitbucket:
		return s.setupBitbucketWebhook(ctx, token, repoFullName, webhookURL, secret)
	default:
		return "", "", ErrGitProviderNotSupported
	}
}

// setupGitHubWebhook sets up a webhook on GitHub
func (s *GitRepositoryService) setupGitHubWebhook(ctx context.Context, token, repoFullName, webhookURL, secret string) (string, string, error) {
	// Implementation would create webhook via GitHub API
	// For now, return placeholder
	return "github-webhook-id", secret, nil
}

// setupGitLabWebhook sets up a webhook on GitLab
func (s *GitRepositoryService) setupGitLabWebhook(ctx context.Context, token, repoFullName, webhookURL, secret string) (string, string, error) {
	return "gitlab-webhook-id", secret, nil
}

// setupGiteeWebhook sets up a webhook on Gitee
func (s *GitRepositoryService) setupGiteeWebhook(ctx context.Context, token, repoFullName, webhookURL, secret string) (string, string, error) {
	return "gitee-webhook-id", secret, nil
}

// setupBitbucketWebhook sets up a webhook on Bitbucket
func (s *GitRepositoryService) setupBitbucketWebhook(ctx context.Context, token, repoFullName, webhookURL, secret string) (string, string, error) {
	return "bitbucket-webhook-id", secret, nil
}

// deleteWebhook deletes a webhook from a repository
func (s *GitRepositoryService) deleteWebhook(ctx context.Context, userID string, provider models.GitProvider, repoFullName, webhookID string) error {
	// Implementation would delete webhook via provider API
	return nil
}
