// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// SSHHandler handles SSH-related HTTP requests
type SSHHandler struct {
	sshService *services.SSHService
	envService *services.EnvironmentService
}

// NewSSHHandler creates a new SSH handler
func NewSSHHandler(sshService *services.SSHService, envService *services.EnvironmentService) *SSHHandler {
	return &SSHHandler{
		sshService: sshService,
		envService: envService,
	}
}

// GenerateKeyPair generates a new SSH key pair
func (h *SSHHandler) GenerateKeyPair(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.CreateSSHKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keyPair, err := h.sshService.GenerateKeyPair(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return key pair with private key (only on creation)
	c.JSON(http.StatusCreated, gin.H{
		"key":        keyPair,
		"privateKey": keyPair.PrivateKey,
	})
}

// GetKey retrieves an SSH key by ID
func (h *SSHHandler) GetKey(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keyID := c.Param("keyId")
	key, err := h.sshService.GetKey(c.Request.Context(), userID, keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Don't return private key
	key.PrivateKey = ""
	c.JSON(http.StatusOK, key)
}

// ListKeys lists SSH keys for a user
func (h *SSHHandler) ListKeys(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.ListSSHKeysRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.UserID = userID

	resp, err := h.sshService.ListKeys(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteKey deletes an SSH key
func (h *SSHHandler) DeleteKey(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keyID := c.Param("keyId")
	if err := h.sshService.DeleteKey(c.Request.Context(), userID, keyID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "key deleted"})
}

// RotateKey rotates an SSH key
func (h *SSHHandler) RotateKey(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keyID := c.Param("keyId")
	newKey, err := h.sshService.RotateKey(c.Request.Context(), userID, keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return new key with private key
	c.JSON(http.StatusOK, gin.H{
		"key":        newKey,
		"privateKey": newKey.PrivateKey,
	})
}

// SetDefaultKey sets a key as the default
func (h *SSHHandler) SetDefaultKey(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keyID := c.Param("keyId")
	if err := h.sshService.SetDefaultKey(c.Request.Context(), userID, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default key set"})
}


// GenerateSSHConfig generates SSH configuration for an environment
func (h *SSHHandler) GenerateSSHConfig(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Get environment
	env, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Parse IDE type from query
	ideTypeStr := c.DefaultQuery("ideType", "generic")
	ideType := models.IDEType(ideTypeStr)

	// Get key path (optional)
	keyPath := c.Query("keyPath")
	if keyPath == "" {
		keyPath = "~/.ssh/devbox_key"
	}

	// Generate config
	startTime := time.Now()
	config, err := h.sshService.GenerateSSHConfig(c.Request.Context(), env, ideType, keyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add generation time to response
	c.JSON(http.StatusOK, gin.H{
		"config":         config,
		"generationTime": time.Since(startTime).Milliseconds(),
	})
}

// GetSSHCommand returns a direct SSH command for an environment
func (h *SSHHandler) GetSSHCommand(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Get environment
	env, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	keyPath := c.Query("keyPath")
	command := h.sshService.GenerateSSHCommand(env, keyPath)

	c.JSON(http.StatusOK, gin.H{
		"command": command,
	})
}

// GetProxyConfig returns the proxy gateway configuration
func (h *SSHHandler) GetProxyConfig(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Get environment
	env, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	proxyConfig := h.sshService.GetProxyConfig(env)
	if proxyConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no connection info available"})
		return
	}

	c.JSON(http.StatusOK, proxyConfig)
}

// RegisterConnection registers a new SSH connection
func (h *SSHHandler) RegisterConnection(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var conn models.SSHConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conn.UserID = userID

	if err := h.sshService.RegisterConnection(c.Request.Context(), &conn); err != nil {
		if services.IsConnectionLimitError(err) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conn)
}

// UnregisterConnection removes an SSH connection
func (h *SSHHandler) UnregisterConnection(c *gin.Context) {
	connectionID := c.Param("connectionId")
	
	if err := h.sshService.UnregisterConnection(c.Request.Context(), connectionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "connection unregistered"})
}

// GetConnectionStats returns connection statistics for an environment
func (h *SSHHandler) GetConnectionStats(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Verify user has access to environment
	_, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.sshService.GetConnectionStats(c.Request.Context(), envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetEnvironmentConnections returns all connections for an environment
func (h *SSHHandler) GetEnvironmentConnections(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Verify user has access to environment
	_, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	connections, err := h.sshService.GetEnvironmentConnections(c.Request.Context(), envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
		"count":       len(connections),
	})
}

// CanConnect checks if a new connection can be established
func (h *SSHHandler) CanConnect(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	envID := c.Param("envId")
	
	// Verify user has access to environment
	_, err := h.envService.Get(c.Request.Context(), userID, envID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	canConnect, current, max := h.sshService.CanConnect(c.Request.Context(), envID)

	c.JSON(http.StatusOK, gin.H{
		"canConnect":        canConnect,
		"currentConnections": current,
		"maxConnections":    max,
	})
}

// RegisterSSHRoutes registers SSH-related routes
func RegisterSSHRoutes(router *gin.RouterGroup, sshService *services.SSHService, envService *services.EnvironmentService) {
	handler := NewSSHHandler(sshService, envService)

	// SSH Key management
	router.POST("/ssh/keys", handler.GenerateKeyPair)
	router.GET("/ssh/keys", handler.ListKeys)
	router.GET("/ssh/keys/:keyId", handler.GetKey)
	router.DELETE("/ssh/keys/:keyId", handler.DeleteKey)
	router.POST("/ssh/keys/:keyId/rotate", handler.RotateKey)
	router.POST("/ssh/keys/:keyId/default", handler.SetDefaultKey)

	// SSH Config generation
	router.GET("/environments/:envId/ssh/config", handler.GenerateSSHConfig)
	router.GET("/environments/:envId/ssh/command", handler.GetSSHCommand)
	router.GET("/environments/:envId/ssh/proxy", handler.GetProxyConfig)

	// Connection management
	router.POST("/ssh/connections", handler.RegisterConnection)
	router.DELETE("/ssh/connections/:connectionId", handler.UnregisterConnection)
	router.GET("/environments/:envId/ssh/connections", handler.GetEnvironmentConnections)
	router.GET("/environments/:envId/ssh/stats", handler.GetConnectionStats)
	router.GET("/environments/:envId/ssh/can-connect", handler.CanConnect)
}
