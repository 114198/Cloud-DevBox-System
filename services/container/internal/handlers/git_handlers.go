// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// GitHandler handles Git-related HTTP requests
type GitHandler struct {
	gitOAuthService *services.GitOAuthService
	gitRepoService  *services.GitRepositoryService
	gitSyncService  *services.GitSyncService
}

// NewGitHandler creates a new GitHandler instance
func NewGitHandler(
	gitOAuthService *services.GitOAuthService,
	gitRepoService *services.GitRepositoryService,
	gitSyncService *services.GitSyncService,
) *GitHandler {
	return &GitHandler{
		gitOAuthService: gitOAuthService,
		gitRepoService:  gitRepoService,
		gitSyncService:  gitSyncService,
	}
}

// GetAuthURL returns the OAuth authorization URL for a Git provider
// @Summary Get Git OAuth URL
// @Description Get the OAuth authorization URL for connecting a Git provider
// @Tags git
// @Accept json
// @Produce json
// @Param provider path string true "Git provider (github, gitlab, gitee, bitbucket)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/auth/{provider} [get]
func (h *GitHandler) GetAuthURL(c *gin.Context) {
	provider := models.GitProvider(c.Param("provider"))

	if !h.gitOAuthService.IsProviderSupported(provider) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "unsupported_provider",
			Message: "Git provider not supported",
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

	// Store state in cookie for verification
	c.SetCookie("git_oauth_state", state, 600, "/", "", false, true)

	url, err := h.gitOAuthService.GetAuthURL(provider, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "url_generation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      url,
		"state":    state,
		"provider": provider,
	})
}

// Callback handles the OAuth callback from Git provider
// @Summary Handle Git OAuth callback
// @Description Handle the OAuth callback and connect the Git provider
// @Tags git
// @Accept json
// @Produce json
// @Param provider path string true "Git provider"
// @Param code query string true "Authorization code"
// @Param state query string true "State parameter"
// @Success 200 {object} models.ConnectGitResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/callback/{provider} [get]
func (h *GitHandler) Callback(c *gin.Context) {
	provider := models.GitProvider(c.Param("provider"))

	if !h.gitOAuthService.IsProviderSupported(provider) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "unsupported_provider",
			Message: "Git provider not supported",
		})
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_code",
			Message: "Authorization code is required",
		})
		return
	}

	// Verify state
	storedState, _ := c.Cookie("git_oauth_state")
	if storedState != "" && storedState != state {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "state_mismatch",
			Message: "OAuth state mismatch",
		})
		return
	}

	// Clear state cookie
	c.SetCookie("git_oauth_state", "", -1, "/", "", false, true)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Connect Git provider
	connection, userInfo, err := h.gitOAuthService.Connect(c.Request.Context(), userID.(string), provider, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "connection_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ConnectGitResponse{
		Connection: connection,
		UserInfo:   userInfo,
	})
}


// GetConnections returns all Git connections for the current user
// @Summary Get Git connections
// @Description Get all connected Git providers for the current user
// @Tags git
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/git/connections [get]
func (h *GitHandler) GetConnections(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	connections, err := h.gitOAuthService.GetUserConnections(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	// Return connections without sensitive data
	result := make([]map[string]interface{}, len(connections))
	for i, conn := range connections {
		result[i] = map[string]interface{}{
			"id":               conn.ID,
			"provider":         conn.Provider,
			"providerUsername": conn.ProviderUsername,
			"scopes":           conn.Scopes,
			"createdAt":        conn.CreatedAt,
			"updatedAt":        conn.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"connections": result,
	})
}

// DeleteConnection disconnects a Git provider
// @Summary Disconnect Git provider
// @Description Disconnect a Git provider from the current user
// @Tags git
// @Accept json
// @Produce json
// @Param provider path string true "Git provider"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/connections/{provider} [delete]
func (h *GitHandler) DeleteConnection(c *gin.Context) {
	provider := models.GitProvider(c.Param("provider"))

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	if err := h.gitOAuthService.DeleteConnection(userID.(string), provider); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "delete_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Git provider disconnected successfully",
	})
}

// ListRepositories lists repositories from a connected Git provider
// @Summary List repositories
// @Description List repositories from a connected Git provider
// @Tags git
// @Accept json
// @Produce json
// @Param provider query string true "Git provider"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param search query string false "Search query"
// @Success 200 {object} models.ListRepositoriesResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/repositories [get]
func (h *GitHandler) ListRepositories(c *gin.Context) {
	var req models.ListRepositoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	repos, total, err := h.gitRepoService.ListRemoteRepositories(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ListRepositoriesResponse{
		Repositories: repos,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
}

// LinkRepository links a repository to an environment
// @Summary Link repository
// @Description Link a Git repository to a development environment
// @Tags git
// @Accept json
// @Produce json
// @Param request body models.LinkRepositoryRequest true "Link request"
// @Success 200 {object} models.GitRepository
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/repositories/link [post]
func (h *GitHandler) LinkRepository(c *gin.Context) {
	var req models.LinkRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	repo, err := h.gitRepoService.LinkRepository(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "link_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// UnlinkRepository unlinks a repository from an environment
// @Summary Unlink repository
// @Description Unlink a Git repository from a development environment
// @Tags git
// @Accept json
// @Produce json
// @Param id path string true "Repository ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/repositories/{id} [delete]
func (h *GitHandler) UnlinkRepository(c *gin.Context) {
	repoID := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	if err := h.gitRepoService.UnlinkRepository(c.Request.Context(), userID.(string), repoID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "unlink_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Repository unlinked successfully",
	})
}

// GetLinkedRepositories returns repositories linked to an environment
// @Summary Get linked repositories
// @Description Get all repositories linked to a specific environment
// @Tags git
// @Accept json
// @Produce json
// @Param environmentId query string true "Environment ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/repositories/linked [get]
func (h *GitHandler) GetLinkedRepositories(c *gin.Context) {
	environmentID := c.Query("environmentId")
	if environmentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_environment_id",
			Message: "Environment ID is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	repos, err := h.gitRepoService.GetLinkedRepositories(c.Request.Context(), userID.(string), environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repos,
	})
}

// HandleWebhook handles incoming webhooks from Git providers
// @Summary Handle Git webhook
// @Description Handle incoming webhook events from Git providers
// @Tags git
// @Accept json
// @Produce json
// @Param provider path string true "Git provider"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/webhook/{provider} [post]
func (h *GitHandler) HandleWebhook(c *gin.Context) {
	provider := models.GitProvider(c.Param("provider"))

	// Parse webhook payload based on provider
	payload, err := h.gitSyncService.ParseWebhookPayload(c.Request, provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_payload",
			Message: err.Error(),
		})
		return
	}

	// Process webhook asynchronously
	go h.gitSyncService.ProcessWebhook(c.Request.Context(), payload)

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook received",
	})
}

// CloneRepository clones a repository to an environment
// @Summary Clone repository
// @Description Clone a Git repository to a development environment
// @Tags git
// @Accept json
// @Produce json
// @Param request body models.CloneRepositoryRequest true "Clone request"
// @Success 200 {object} models.GitSyncResult
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/clone [post]
func (h *GitHandler) CloneRepository(c *gin.Context) {
	var req models.CloneRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.gitSyncService.CloneRepository(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "clone_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SyncRepository syncs a repository with the latest changes
// @Summary Sync repository
// @Description Sync a Git repository with the latest changes
// @Tags git
// @Accept json
// @Produce json
// @Param request body models.SyncRepositoryRequest true "Sync request"
// @Success 200 {object} models.GitSyncResult
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/sync [post]
func (h *GitHandler) SyncRepository(c *gin.Context) {
	var req models.SyncRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.gitSyncService.SyncRepository(c.Request.Context(), userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "sync_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSyncHistory returns sync history for a repository
// @Summary Get sync history
// @Description Get sync history for a Git repository
// @Tags git
// @Accept json
// @Produce json
// @Param repositoryId query string true "Repository ID"
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/git/sync/history [get]
func (h *GitHandler) GetSyncHistory(c *gin.Context) {
	repositoryID := c.Query("repositoryId")
	if repositoryID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_repository_id",
			Message: "Repository ID is required",
		})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		// Parse limit
	}

	history, err := h.gitSyncService.GetSyncHistory(c.Request.Context(), repositoryID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}
