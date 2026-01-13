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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
)

var (
	ErrGitProviderNotSupported = errors.New("git provider not supported")
	ErrGitConnectionNotFound   = errors.New("git connection not found")
	ErrGitTokenExpired         = errors.New("git token expired")
	ErrGitTokenRefreshFailed   = errors.New("failed to refresh git token")
)

// GitOAuthConfig holds OAuth configuration for a Git provider
type GitOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// GitOAuthService handles Git platform OAuth operations
type GitOAuthService struct {
	configs     map[models.GitProvider]*oauth2.Config
	connections sync.Map // In-memory cache, replace with DB in production
	mu          sync.RWMutex
}

// NewGitOAuthService creates a new GitOAuthService instance
func NewGitOAuthService(configs map[models.GitProvider]*GitOAuthConfig) *GitOAuthService {
	oauthConfigs := make(map[models.GitProvider]*oauth2.Config)

	// GitHub OAuth config
	if cfg, ok := configs[models.GitProviderGitHub]; ok && cfg.ClientID != "" {
		scopes := cfg.Scopes
		if len(scopes) == 0 {
			scopes = []string{"repo", "read:user", "user:email"}
		}
		oauthConfigs[models.GitProviderGitHub] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     github.Endpoint,
		}
	}

	// GitLab OAuth config
	if cfg, ok := configs[models.GitProviderGitLab]; ok && cfg.ClientID != "" {
		scopes := cfg.Scopes
		if len(scopes) == 0 {
			scopes = []string{"read_user", "read_repository", "write_repository", "api"}
		}
		oauthConfigs[models.GitProviderGitLab] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     gitlab.Endpoint,
		}
	}

	// Gitee OAuth config
	if cfg, ok := configs[models.GitProviderGitee]; ok && cfg.ClientID != "" {
		scopes := cfg.Scopes
		if len(scopes) == 0 {
			scopes = []string{"user_info", "projects", "pull_requests", "hook"}
		}
		oauthConfigs[models.GitProviderGitee] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://gitee.com/oauth/authorize",
				TokenURL: "https://gitee.com/oauth/token",
			},
		}
	}

	// Bitbucket OAuth config
	if cfg, ok := configs[models.GitProviderBitbucket]; ok && cfg.ClientID != "" {
		scopes := cfg.Scopes
		if len(scopes) == 0 {
			scopes = []string{"repository", "repository:write", "account", "webhook"}
		}
		oauthConfigs[models.GitProviderBitbucket] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     bitbucket.Endpoint,
		}
	}

	return &GitOAuthService{
		configs: oauthConfigs,
	}
}


// GetAuthURL returns the OAuth authorization URL for a Git provider
func (s *GitOAuthService) GetAuthURL(provider models.GitProvider, state string) (string, error) {
	config, ok := s.configs[provider]
	if !ok {
		return "", ErrGitProviderNotSupported
	}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCode exchanges an authorization code for tokens
func (s *GitOAuthService) ExchangeCode(ctx context.Context, provider models.GitProvider, code string) (*oauth2.Token, error) {
	config, ok := s.configs[provider]
	if !ok {
		return nil, ErrGitProviderNotSupported
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// Connect connects a Git provider and returns connection info
func (s *GitOAuthService) Connect(ctx context.Context, userID string, provider models.GitProvider, code string) (*models.GitConnection, *models.GitUserInfo, error) {
	// Exchange code for token
	token, err := s.ExchangeCode(ctx, provider, code)
	if err != nil {
		return nil, nil, err
	}

	// Get user info from provider
	userInfo, err := s.GetUserInfo(ctx, provider, token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Create connection
	connection := &models.GitConnection{
		ID:               uuid.New().String(),
		UserID:           userID,
		Provider:         provider,
		ProviderUserID:   userInfo.ID,
		ProviderUsername: userInfo.Username,
		AccessToken:      token.AccessToken,
		RefreshToken:     token.RefreshToken,
		Scopes:           s.configs[provider].Scopes,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if !token.Expiry.IsZero() {
		connection.TokenExpiresAt = &token.Expiry
	}

	// Store connection (in production, save to database)
	s.storeConnection(connection)

	return connection, userInfo, nil
}

// GetUserInfo fetches user info from the Git provider
func (s *GitOAuthService) GetUserInfo(ctx context.Context, provider models.GitProvider, accessToken string) (*models.GitUserInfo, error) {
	switch provider {
	case models.GitProviderGitHub:
		return s.getGitHubUserInfo(ctx, accessToken)
	case models.GitProviderGitLab:
		return s.getGitLabUserInfo(ctx, accessToken)
	case models.GitProviderGitee:
		return s.getGiteeUserInfo(ctx, accessToken)
	case models.GitProviderBitbucket:
		return s.getBitbucketUserInfo(ctx, accessToken)
	default:
		return nil, ErrGitProviderNotSupported
	}
}

// getGitHubUserInfo fetches user info from GitHub
func (s *GitOAuthService) getGitHubUserInfo(ctx context.Context, accessToken string) (*models.GitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// If email is not public, fetch from emails endpoint
	email := data.Email
	if email == "" {
		email, _ = s.getGitHubPrimaryEmail(ctx, accessToken)
	}

	return &models.GitUserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Login,
		Email:     email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

// getGitHubPrimaryEmail fetches the primary email from GitHub
func (s *GitOAuthService) getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", nil
}


// getGitLabUserInfo fetches user info from GitLab
func (s *GitOAuthService) getGitLabUserInfo(ctx context.Context, accessToken string) (*models.GitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitlab.com/api/v4/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &models.GitUserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Username,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

// getGiteeUserInfo fetches user info from Gitee
func (s *GitOAuthService) getGiteeUserInfo(ctx context.Context, accessToken string) (*models.GitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitee.com/api/v5/user?access_token="+accessToken, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gitee API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &models.GitUserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Login,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

// getBitbucketUserInfo fetches user info from Bitbucket
func (s *GitOAuthService) getBitbucketUserInfo(ctx context.Context, accessToken string) (*models.GitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.bitbucket.org/2.0/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bitbucket API error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		UUID        string `json:"uuid"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Get email from separate endpoint
	email, _ := s.getBitbucketPrimaryEmail(ctx, accessToken)

	return &models.GitUserInfo{
		ID:        data.UUID,
		Username:  data.Username,
		Email:     email,
		Name:      data.DisplayName,
		AvatarURL: data.Links.Avatar.Href,
	}, nil
}

// getBitbucketPrimaryEmail fetches the primary email from Bitbucket
func (s *GitOAuthService) getBitbucketPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.bitbucket.org/2.0/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data struct {
		Values []struct {
			Email       string `json:"email"`
			IsPrimary   bool   `json:"is_primary"`
			IsConfirmed bool   `json:"is_confirmed"`
		} `json:"values"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	for _, e := range data.Values {
		if e.IsPrimary && e.IsConfirmed {
			return e.Email, nil
		}
	}

	return "", nil
}


// RefreshToken refreshes the access token for a connection
func (s *GitOAuthService) RefreshToken(ctx context.Context, connection *models.GitConnection) (*models.GitConnection, error) {
	if connection.RefreshToken == "" {
		return nil, ErrGitTokenRefreshFailed
	}

	config, ok := s.configs[connection.Provider]
	if !ok {
		return nil, ErrGitProviderNotSupported
	}

	// Create token source with refresh token
	tokenSource := config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: connection.RefreshToken,
	})

	// Get new token
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGitTokenRefreshFailed, err)
	}

	// Update connection
	connection.AccessToken = newToken.AccessToken
	if newToken.RefreshToken != "" {
		connection.RefreshToken = newToken.RefreshToken
	}
	if !newToken.Expiry.IsZero() {
		connection.TokenExpiresAt = &newToken.Expiry
	}
	connection.UpdatedAt = time.Now()

	// Store updated connection
	s.storeConnection(connection)

	return connection, nil
}

// GetConnection retrieves a Git connection by user ID and provider
func (s *GitOAuthService) GetConnection(userID string, provider models.GitProvider) (*models.GitConnection, error) {
	key := fmt.Sprintf("%s:%s", userID, provider)
	if conn, ok := s.connections.Load(key); ok {
		return conn.(*models.GitConnection), nil
	}
	return nil, ErrGitConnectionNotFound
}

// GetConnectionByID retrieves a Git connection by ID
func (s *GitOAuthService) GetConnectionByID(connectionID string) (*models.GitConnection, error) {
	var found *models.GitConnection
	s.connections.Range(func(key, value interface{}) bool {
		conn := value.(*models.GitConnection)
		if conn.ID == connectionID {
			found = conn
			return false
		}
		return true
	})
	if found == nil {
		return nil, ErrGitConnectionNotFound
	}
	return found, nil
}

// GetUserConnections retrieves all Git connections for a user
func (s *GitOAuthService) GetUserConnections(userID string) ([]*models.GitConnection, error) {
	var connections []*models.GitConnection
	s.connections.Range(func(key, value interface{}) bool {
		conn := value.(*models.GitConnection)
		if conn.UserID == userID {
			connections = append(connections, conn)
		}
		return true
	})
	return connections, nil
}

// DeleteConnection deletes a Git connection
func (s *GitOAuthService) DeleteConnection(userID string, provider models.GitProvider) error {
	key := fmt.Sprintf("%s:%s", userID, provider)
	s.connections.Delete(key)
	return nil
}

// storeConnection stores a connection in the cache
func (s *GitOAuthService) storeConnection(connection *models.GitConnection) {
	key := fmt.Sprintf("%s:%s", connection.UserID, connection.Provider)
	s.connections.Store(key, connection)
}

// IsTokenValid checks if the token is still valid
func (s *GitOAuthService) IsTokenValid(connection *models.GitConnection) bool {
	if connection.TokenExpiresAt == nil {
		return true // No expiry set, assume valid
	}
	// Add 5 minute buffer
	return time.Now().Add(5 * time.Minute).Before(*connection.TokenExpiresAt)
}

// GetValidToken returns a valid access token, refreshing if necessary
func (s *GitOAuthService) GetValidToken(ctx context.Context, connection *models.GitConnection) (string, error) {
	if s.IsTokenValid(connection) {
		return connection.AccessToken, nil
	}

	// Token expired, try to refresh
	refreshed, err := s.RefreshToken(ctx, connection)
	if err != nil {
		return "", err
	}

	return refreshed.AccessToken, nil
}

// IsProviderSupported checks if a provider is supported
func (s *GitOAuthService) IsProviderSupported(provider models.GitProvider) bool {
	_, ok := s.configs[provider]
	return ok
}

// GetSupportedProviders returns a list of supported providers
func (s *GitOAuthService) GetSupportedProviders() []models.GitProvider {
	providers := make([]models.GitProvider, 0, len(s.configs))
	for provider := range s.configs {
		providers = append(providers, provider)
	}
	return providers
}
