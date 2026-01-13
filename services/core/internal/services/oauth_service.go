// Package services provides business logic services for the core service.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
	"gorm.io/gorm"
)

var (
	ErrOAuthProviderNotSupported = errors.New("OAuth provider not supported")
	ErrOAuthStateMismatch        = errors.New("OAuth state mismatch")
	ErrOAuthAccountNotFound      = errors.New("OAuth account not found")
	ErrOAuthAccountAlreadyLinked = errors.New("OAuth account already linked to another user")
)

// OAuthProvider represents supported OAuth providers
type OAuthProvider string

const (
	ProviderGitHub OAuthProvider = "github"
	ProviderGitLab OAuthProvider = "gitlab"
	ProviderGitee  OAuthProvider = "gitee"
)

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
}

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OAuthService handles OAuth authentication operations
type OAuthService struct {
	db          *gorm.DB
	authService *AuthService
	configs     map[OAuthProvider]*oauth2.Config
}

// NewOAuthService creates a new OAuthService instance
func NewOAuthService(db *gorm.DB, authService *AuthService, configs map[OAuthProvider]*OAuthConfig) *OAuthService {
	oauthConfigs := make(map[OAuthProvider]*oauth2.Config)

	// GitHub OAuth config
	if cfg, ok := configs[ProviderGitHub]; ok && cfg.ClientID != "" {
		oauthConfigs[ProviderGitHub] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	}

	// GitLab OAuth config
	if cfg, ok := configs[ProviderGitLab]; ok && cfg.ClientID != "" {
		oauthConfigs[ProviderGitLab] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"read_user", "email"},
			Endpoint:     gitlab.Endpoint,
		}
	}

	// Gitee OAuth config
	if cfg, ok := configs[ProviderGitee]; ok && cfg.ClientID != "" {
		oauthConfigs[ProviderGitee] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"user_info", "emails"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://gitee.com/oauth/authorize",
				TokenURL: "https://gitee.com/oauth/token",
			},
		}
	}

	return &OAuthService{
		db:          db,
		authService: authService,
		configs:     oauthConfigs,
	}
}

// GetAuthURL returns the OAuth authorization URL for a provider
func (s *OAuthService) GetAuthURL(provider OAuthProvider, state string) (string, error) {
	config, ok := s.configs[provider]
	if !ok {
		return "", ErrOAuthProviderNotSupported
	}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}


// ExchangeCode exchanges an authorization code for tokens and user info
func (s *OAuthService) ExchangeCode(ctx context.Context, provider OAuthProvider, code string) (*oauth2.Token, *OAuthUserInfo, error) {
	config, ok := s.configs[provider]
	if !ok {
		return nil, nil, ErrOAuthProviderNotSupported
	}

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.getUserInfo(ctx, provider, token)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return token, userInfo, nil
}

// getUserInfo fetches user info from the OAuth provider
func (s *OAuthService) getUserInfo(ctx context.Context, provider OAuthProvider, token *oauth2.Token) (*OAuthUserInfo, error) {
	config := s.configs[provider]
	client := config.Client(ctx, token)

	switch provider {
	case ProviderGitHub:
		return s.getGitHubUserInfo(client)
	case ProviderGitLab:
		return s.getGitLabUserInfo(client)
	case ProviderGitee:
		return s.getGiteeUserInfo(client)
	default:
		return nil, ErrOAuthProviderNotSupported
	}
}

// getGitHubUserInfo fetches user info from GitHub
func (s *OAuthService) getGitHubUserInfo(client *http.Client) (*OAuthUserInfo, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
		email, _ = s.getGitHubPrimaryEmail(client)
	}

	displayName := data.Name
	if displayName == "" {
		displayName = data.Login
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", data.ID),
		Email:       email,
		Username:    data.Login,
		DisplayName: displayName,
		Avatar:      data.AvatarURL,
	}, nil
}

// getGitHubPrimaryEmail fetches the primary email from GitHub
func (s *OAuthService) getGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
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
func (s *OAuthService) getGitLabUserInfo(client *http.Client) (*OAuthUserInfo, error) {
	resp, err := client.Get("https://gitlab.com/api/v4/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	displayName := data.Name
	if displayName == "" {
		displayName = data.Username
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", data.ID),
		Email:       data.Email,
		Username:    data.Username,
		DisplayName: displayName,
		Avatar:      data.AvatarURL,
	}, nil
}

// getGiteeUserInfo fetches user info from Gitee
func (s *OAuthService) getGiteeUserInfo(client *http.Client) (*OAuthUserInfo, error) {
	resp, err := client.Get("https://gitee.com/api/v5/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	displayName := data.Name
	if displayName == "" {
		displayName = data.Login
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", data.ID),
		Email:       data.Email,
		Username:    data.Login,
		DisplayName: displayName,
		Avatar:      data.AvatarURL,
	}, nil
}


// LoginOrRegisterWithOAuth handles OAuth login or registration
func (s *OAuthService) LoginOrRegisterWithOAuth(ctx context.Context, provider OAuthProvider, code string) (*models.User, *TokenPair, bool, error) {
	// Exchange code for token and user info
	token, userInfo, err := s.ExchangeCode(ctx, provider, code)
	if err != nil {
		return nil, nil, false, err
	}

	// Check if OAuth account already exists
	var oauthAccount models.OAuthAccount
	err = s.db.Where("provider = ? AND provider_account_id = ?", provider, userInfo.ID).First(&oauthAccount).Error

	if err == nil {
		// OAuth account exists, login the user
		user, err := s.authService.GetUserByID(oauthAccount.UserID)
		if err != nil {
			return nil, nil, false, err
		}

		// Update OAuth tokens
		s.updateOAuthTokens(&oauthAccount, token)

		// Generate JWT tokens
		tokens, err := s.authService.generateTokenPair(user)
		if err != nil {
			return nil, nil, false, err
		}

		return user, tokens, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, false, fmt.Errorf("failed to check OAuth account: %w", err)
	}

	// OAuth account doesn't exist, check if user with email exists
	var existingUser models.User
	if userInfo.Email != "" {
		err = s.db.Where("email = ?", userInfo.Email).First(&existingUser).Error
		if err == nil {
			// User exists, link OAuth account
			if err := s.linkOAuthAccount(&existingUser, provider, userInfo, token); err != nil {
				return nil, nil, false, err
			}

			tokens, err := s.authService.generateTokenPair(&existingUser)
			if err != nil {
				return nil, nil, false, err
			}

			return &existingUser, tokens, false, nil
		}
	}

	// Create new user with OAuth account
	user, tokens, err := s.createUserWithOAuth(provider, userInfo, token)
	if err != nil {
		return nil, nil, false, err
	}

	return user, tokens, true, nil
}

// createUserWithOAuth creates a new user with an OAuth account
func (s *OAuthService) createUserWithOAuth(provider OAuthProvider, userInfo *OAuthUserInfo, token *oauth2.Token) (*models.User, *TokenPair, error) {
	// Generate unique username if needed
	username := userInfo.Username
	if username == "" {
		username = fmt.Sprintf("user_%s", uuid.New().String()[:8])
	}

	// Check if username exists and make it unique
	var count int64
	s.db.Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		username = fmt.Sprintf("%s_%s", username, uuid.New().String()[:4])
	}

	// Create user
	user := &models.User{
		Email:         userInfo.Email,
		Username:      username,
		DisplayName:   userInfo.DisplayName,
		Avatar:        userInfo.Avatar,
		Role:          "developer",
		EmailVerified: true, // OAuth emails are considered verified
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Link OAuth account
	if err := s.linkOAuthAccount(user, provider, userInfo, token); err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := s.authService.generateTokenPair(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// linkOAuthAccount links an OAuth account to a user
func (s *OAuthService) linkOAuthAccount(user *models.User, provider OAuthProvider, userInfo *OAuthUserInfo, token *oauth2.Token) error {
	// Check if this OAuth account is already linked to another user
	var existing models.OAuthAccount
	err := s.db.Where("provider = ? AND provider_account_id = ?", provider, userInfo.ID).First(&existing).Error
	if err == nil && existing.UserID != user.ID {
		return ErrOAuthAccountAlreadyLinked
	}

	oauthAccount := &models.OAuthAccount{
		UserID:            user.ID,
		Provider:          string(provider),
		ProviderAccountID: userInfo.ID,
		AccessToken:       token.AccessToken,
		RefreshToken:      token.RefreshToken,
	}

	if !token.Expiry.IsZero() {
		oauthAccount.TokenExpiresAt = &token.Expiry
	}

	return s.db.Create(oauthAccount).Error
}

// updateOAuthTokens updates the OAuth tokens for an account
func (s *OAuthService) updateOAuthTokens(account *models.OAuthAccount, token *oauth2.Token) {
	updates := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"updated_at":    time.Now(),
	}

	if !token.Expiry.IsZero() {
		updates["token_expires_at"] = token.Expiry
	}

	s.db.Model(account).Updates(updates)
}

// UnlinkOAuthAccount unlinks an OAuth account from a user
func (s *OAuthService) UnlinkOAuthAccount(userID uuid.UUID, provider OAuthProvider) error {
	// Check if user has password or other OAuth accounts
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return ErrUserNotFound
	}

	// Count OAuth accounts
	var count int64
	s.db.Model(&models.OAuthAccount{}).Where("user_id = ?", userID).Count(&count)

	// If no password and only one OAuth account, cannot unlink
	if user.PasswordHash == "" && count <= 1 {
		return errors.New("cannot unlink the only authentication method")
	}

	// Delete OAuth account
	result := s.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.OAuthAccount{})
	if result.RowsAffected == 0 {
		return ErrOAuthAccountNotFound
	}

	return nil
}

// GetLinkedAccounts returns all OAuth accounts linked to a user
func (s *OAuthService) GetLinkedAccounts(userID uuid.UUID) ([]models.OAuthAccount, error) {
	var accounts []models.OAuthAccount
	if err := s.db.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// IsProviderSupported checks if a provider is supported
func (s *OAuthService) IsProviderSupported(provider OAuthProvider) bool {
	_, ok := s.configs[provider]
	return ok
}
