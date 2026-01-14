// Package handlers provides HTTP/WebSocket request handlers for the realtime service.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/cloud-devbox/services/realtime/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// CollaborationHandler handles collaboration-related requests
type CollaborationHandler struct {
	collabService *services.CollaborationService
	hub           *services.Hub
	logger        *zap.Logger
	upgrader      websocket.Upgrader
}

// NewCollaborationHandler creates a new collaboration handler
func NewCollaborationHandler(collabService *services.CollaborationService, hub *services.Hub, logger *zap.Logger) *CollaborationHandler {
	return &CollaborationHandler{
		collabService: collabService,
		hub:           hub,
		logger:        logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for collaboration
func (h *CollaborationHandler) HandleWebSocket(c *gin.Context) {
	environmentID := c.Param("environmentId")
	if environmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Environment ID is required"})
		return
	}

	// Get user info from query params or headers
	userID := c.Query("userId")
	username := c.Query("username")
	displayName := c.Query("displayName")
	avatarURL := c.Query("avatarUrl")
	token := extractAccessToken(c)

	if userID == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID and username are required"})
		return
	}

	if err := validateAccessToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}

	// Get or create session for environment
	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": "Invalid environment ID"})
		conn.Close()
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": "Invalid user ID"})
		conn.Close()
		return
	}

	session, err := h.collabService.GetOrCreateSessionForEnvironment(envUUID, userUUID)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": "Failed to create session"})
		conn.Close()
		return
	}

	// Create client
	colorIndex := len(session.Participants)
	client := services.NewClient(
		h.hub,
		conn,
		userID,
		username,
		displayName,
		avatarURL,
		session.ID.String(),
		environmentID,
		colorIndex,
	)

	// Join session
	response, err := h.collabService.JoinSession(session.ID.String(), client)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		conn.Close()
		return
	}

	// Send join response
	conn.WriteJSON(gin.H{
		"type":         "connected",
		"sessionId":    response.SessionID,
		"userId":       response.UserID,
		"color":        response.Color,
		"participants": response.Participants,
	})

	// Start read and write pumps
	go client.WritePump()
	go client.ReadPump()
}

// CreateSession creates a new collaboration session
func (h *CollaborationHandler) CreateSession(c *gin.Context) {
	var req struct {
		EnvironmentID string                  `json:"environmentId" binding:"required"`
		UserID        string                  `json:"userId" binding:"required"`
		Settings      *models.SessionSettings `json:"settings"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	envUUID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environment ID"})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	session, err := h.collabService.CreateSession(envUUID, userUUID, req.Settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession gets a collaboration session
func (h *CollaborationHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	session, err := h.collabService.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// EndSession ends a collaboration session
func (h *CollaborationHandler) EndSession(c *gin.Context) {
	sessionID := c.Param("sessionId")

	if err := h.collabService.EndSession(sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session ended"})
}

// GetSessionParticipants gets participants in a session
func (h *CollaborationHandler) GetSessionParticipants(c *gin.Context) {
	sessionID := c.Param("sessionId")

	participants, err := h.collabService.GetSessionParticipants(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, participants)
}

// GetSessionHistory gets history for a session
func (h *CollaborationHandler) GetSessionHistory(c *gin.Context) {
	sessionID := c.Param("sessionId")
	limit := 100
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history := h.collabService.GetSessionHistory(sessionID, limit, offset)
	c.JSON(http.StatusOK, history)
}

// GetSessionStats gets statistics for a session
func (h *CollaborationHandler) GetSessionStats(c *gin.Context) {
	sessionID := c.Param("sessionId")

	stats := h.collabService.GetSessionStats(sessionID)
	c.JSON(http.StatusOK, stats)
}
