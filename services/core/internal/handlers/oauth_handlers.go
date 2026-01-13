// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OAuthHandler handles OAuth-related HTTP requests
type OAuthHandler struct {
	oauthService *services.OAuthService
}

// NewOAuthHandler creates a new OAuthHandler instance
func NewOAuthHandler(oauthService *services.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
	}
}

// OAuthCallbackRequest represents the OAuth callback request
type OAuthCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

// OAuthLoginResponse represents the OAuth login response
type OAuthLoginResponse struct {
	User      *UserResponse  `json:"user"`
	Tokens    *TokenResponse `json:"tokens"`
	IsNewUser bool           `json:"is_new_user"`
}

// LinkedAccountResponse represents a linked OAuth account
type LinkedAccountResponse struct {
	Provider  string `json:"provider"`
	AccountID string `json:"account_id"`
	LinkedAt  string `json:"linked_at"`
}

// GetAuthURL returns the OAuth authorization URL
func (h *OAuthHandler) GetAuthURL(c *gin.Context) {
	provider := services.OAuthProvider(c.Param("provider"))

	if !h.oauthService.IsProviderSupported(provider) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "unsupported_provider",
			Message: "OAuth provider not supported",
		})
		return
	}

	// Generate random state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "state_generation_failed",
			Message: "Failed to generate state",
		})
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store state in session/cookie for verification
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	url, err := h.oauthService.GetAuthURL(provider, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "url_generation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":   url,
		"state": state,
	})
}

// Callback handles the OAuth callback
func (h *OAuthHandler) Callback(c *gin.Context) {
	provider := services.OAuthProvider(c.Param("provider"))

	if !h.oauthService.IsProviderSupported(provider) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "unsupported_provider",
			Message: "OAuth provider not supported",
		})
		return
	}

	var req OAuthCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Verify state (optional - can be done via cookie)
	storedState, _ := c.Cookie("oauth_state")
	if storedState != "" && storedState != req.State {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "state_mismatch",
			Message: "OAuth state mismatch",
		})
		return
	}

	// Clear state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Exchange code and login/register
	user, tokens, isNewUser, err := h.oauthService.LoginOrRegisterWithOAuth(c.Request.Context(), provider, req.Code)
	if err != nil {
		switch err {
		case services.ErrOAuthAccountAlreadyLinked:
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "account_already_linked",
				Message: "This OAuth account is already linked to another user",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "oauth_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, OAuthLoginResponse{
		User:      toUserResponse(user),
		Tokens: &TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			TokenType:    tokens.TokenType,
		},
		IsNewUser: isNewUser,
	})
}

// LinkAccount links an OAuth account to the current user
func (h *OAuthHandler) LinkAccount(c *gin.Context) {
	provider := services.OAuthProvider(c.Param("provider"))

	if !h.oauthService.IsProviderSupported(provider) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "unsupported_provider",
			Message: "OAuth provider not supported",
		})
		return
	}

	// Generate state with user ID
	userID, _ := c.Get("user_id")
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store state and user ID for linking
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	c.SetCookie("oauth_link_user", userID.(string), 600, "/", "", false, true)

	url, err := h.oauthService.GetAuthURL(provider, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "url_generation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":   url,
		"state": state,
	})
}

// UnlinkAccount unlinks an OAuth account from the current user
func (h *OAuthHandler) UnlinkAccount(c *gin.Context) {
	provider := services.OAuthProvider(c.Param("provider"))

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := h.oauthService.UnlinkOAuthAccount(userID, provider); err != nil {
		switch err {
		case services.ErrOAuthAccountNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "account_not_found",
				Message: "OAuth account not found",
			})
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "unlink_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OAuth account unlinked successfully"})
}

// GetLinkedAccounts returns all OAuth accounts linked to the current user
func (h *OAuthHandler) GetLinkedAccounts(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID",
		})
		return
	}

	accounts, err := h.oauthService.GetLinkedAccounts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	response := make([]LinkedAccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = LinkedAccountResponse{
			Provider:  acc.Provider,
			AccountID: acc.ProviderAccountID,
			LinkedAt:  acc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, gin.H{"accounts": response})
}
