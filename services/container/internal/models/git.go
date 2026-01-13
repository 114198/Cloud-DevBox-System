// Package models defines the data models for the container service.
package models

import (
	"time"
)

// GitProvider represents supported Git providers
type GitProvider string

const (
	GitProviderGitHub    GitProvider = "github"
	GitProviderGitLab    GitProvider = "gitlab"
	GitProviderGitee     GitProvider = "gitee"
	GitProviderBitbucket GitProvider = "bitbucket"
)

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSyncing SyncStatus = "syncing"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusFailed  SyncStatus = "failed"
)

// SyncType represents the type of sync operation
type SyncType string

const (
	SyncTypeClone   SyncType = "clone"
	SyncTypePull    SyncType = "pull"
	SyncTypeWebhook SyncType = "webhook"
	SyncTypeManual  SyncType = "manual"
)

// GitConnection represents an OAuth connection to a Git provider
type GitConnection struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	Provider         GitProvider `json:"provider"`
	ProviderUserID   string     `json:"providerUserId"`
	ProviderUsername string     `json:"providerUsername,omitempty"`
	AccessToken      string     `json:"-"`
	RefreshToken     string     `json:"-"`
	TokenExpiresAt   *time.Time `json:"tokenExpiresAt,omitempty"`
	Scopes           []string   `json:"scopes,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// GitRepository represents a linked Git repository
type GitRepository struct {
	ID            string      `json:"id"`
	UserID        string      `json:"userId"`
	ConnectionID  string      `json:"connectionId"`
	EnvironmentID string      `json:"environmentId,omitempty"`
	Provider      GitProvider `json:"provider"`
	RepoID        string      `json:"repoId"`
	RepoName      string      `json:"repoName"`
	RepoFullName  string      `json:"repoFullName"`
	CloneURL      string      `json:"cloneUrl"`
	SSHURL        string      `json:"sshUrl,omitempty"`
	DefaultBranch string      `json:"defaultBranch"`
	IsPrivate     bool        `json:"isPrivate"`
	WebhookID     string      `json:"webhookId,omitempty"`
	WebhookSecret string      `json:"-"`
	LastSyncAt    *time.Time  `json:"lastSyncAt,omitempty"`
	SyncStatus    SyncStatus  `json:"syncStatus"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}


// GitWebhook represents a webhook event from a Git provider
type GitWebhook struct {
	ID           string                 `json:"id"`
	RepositoryID string                 `json:"repositoryId"`
	EventType    string                 `json:"eventType"`
	EventID      string                 `json:"eventId,omitempty"`
	Payload      map[string]interface{} `json:"payload"`
	Processed    bool                   `json:"processed"`
	ProcessedAt  *time.Time             `json:"processedAt,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
}

// GitSyncHistory represents a sync operation history entry
type GitSyncHistory struct {
	ID            string     `json:"id"`
	RepositoryID  string     `json:"repositoryId"`
	EnvironmentID string     `json:"environmentId"`
	CommitSHA     string     `json:"commitSha"`
	CommitMessage string     `json:"commitMessage,omitempty"`
	CommitAuthor  string     `json:"commitAuthor,omitempty"`
	Branch        string     `json:"branch"`
	SyncType      SyncType   `json:"syncType"`
	Status        SyncStatus `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	DurationMs    int        `json:"durationMs,omitempty"`
	ErrorMessage  string     `json:"errorMessage,omitempty"`
}

// GitUserInfo represents user info from a Git provider
type GitUserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// GitRepoInfo represents repository info from a Git provider
type GitRepoInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	Description   string `json:"description,omitempty"`
	CloneURL      string `json:"cloneUrl"`
	SSHURL        string `json:"sshUrl,omitempty"`
	DefaultBranch string `json:"defaultBranch"`
	IsPrivate     bool   `json:"isPrivate"`
	Size          int64  `json:"size,omitempty"`
	Language      string `json:"language,omitempty"`
}

// GitBranch represents a branch in a repository
type GitBranch struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commitSha"`
	IsDefault bool   `json:"isDefault"`
	Protected bool   `json:"protected"`
}

// GitCommit represents a commit in a repository
type GitCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Request/Response types

// ConnectGitRequest represents a request to connect a Git provider
type ConnectGitRequest struct {
	Provider GitProvider `json:"provider" binding:"required"`
	Code     string      `json:"code" binding:"required"`
	State    string      `json:"state"`
}

// ConnectGitResponse represents a response after connecting a Git provider
type ConnectGitResponse struct {
	Connection *GitConnection `json:"connection"`
	UserInfo   *GitUserInfo   `json:"userInfo"`
}

// ListRepositoriesRequest represents a request to list repositories
type ListRepositoriesRequest struct {
	Provider GitProvider `form:"provider" binding:"required"`
	Page     int         `form:"page,default=1"`
	PageSize int         `form:"pageSize,default=20"`
	Search   string      `form:"search"`
}

// ListRepositoriesResponse represents a response containing repositories
type ListRepositoriesResponse struct {
	Repositories []GitRepoInfo `json:"repositories"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
}

// LinkRepositoryRequest represents a request to link a repository
type LinkRepositoryRequest struct {
	Provider      GitProvider `json:"provider" binding:"required"`
	RepoID        string      `json:"repoId" binding:"required"`
	EnvironmentID string      `json:"environmentId" binding:"required"`
	Branch        string      `json:"branch"`
	EnableWebhook bool        `json:"enableWebhook"`
}

// CloneRepositoryRequest represents a request to clone a repository
type CloneRepositoryRequest struct {
	RepositoryID  string `json:"repositoryId" binding:"required"`
	EnvironmentID string `json:"environmentId" binding:"required"`
	Branch        string `json:"branch"`
	Shallow       bool   `json:"shallow"`
	Depth         int    `json:"depth"`
}

// SyncRepositoryRequest represents a request to sync a repository
type SyncRepositoryRequest struct {
	RepositoryID  string `json:"repositoryId" binding:"required"`
	EnvironmentID string `json:"environmentId" binding:"required"`
	Branch        string `json:"branch"`
}

// WebhookPayload represents a generic webhook payload
type WebhookPayload struct {
	Provider  GitProvider            `json:"provider"`
	EventType string                 `json:"eventType"`
	EventID   string                 `json:"eventId"`
	RepoID    string                 `json:"repoId"`
	Branch    string                 `json:"branch,omitempty"`
	CommitSHA string                 `json:"commitSha,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
}

// GitSyncResult represents the result of a sync operation
type GitSyncResult struct {
	Success       bool       `json:"success"`
	CommitSHA     string     `json:"commitSha,omitempty"`
	CommitMessage string     `json:"commitMessage,omitempty"`
	FilesChanged  int        `json:"filesChanged,omitempty"`
	DurationMs    int        `json:"durationMs"`
	Error         string     `json:"error,omitempty"`
	SyncedAt      time.Time  `json:"syncedAt"`
}
