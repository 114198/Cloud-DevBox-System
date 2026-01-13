// Package handlers provides HTTP/WebSocket request handlers for the meeting service.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/cloud-devbox/services/realtime/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MeetingHandler handles meeting-related HTTP and WebSocket requests
type MeetingHandler struct {
	signalingService *services.SignalingService
	logger           *zap.Logger
}

// NewMeetingHandler creates a new meeting handler
func NewMeetingHandler(signalingService *services.SignalingService, logger *zap.Logger) *MeetingHandler {
	return &MeetingHandler{
		signalingService: signalingService,
		logger:           logger,
	}
}

// CreateMeeting handles meeting creation requests
func (h *MeetingHandler) CreateMeeting(c *gin.Context) {
	var req models.CreateMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	meeting, err := h.signalingService.CreateMeeting(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create meeting", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create meeting",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, meeting)
}

// GetMeeting handles meeting retrieval requests
func (h *MeetingHandler) GetMeeting(c *gin.Context) {
	meetingID := c.Param("meetingId")

	meeting, err := h.signalingService.GetMeeting(meetingID)
	if err != nil {
		if meetingErr, ok := err.(*models.MeetingError); ok {
			if meetingErr.Code == models.ErrMeetingNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": meetingErr.Message,
					"code": meetingErr.Code,
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get meeting",
		})
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// EndMeeting handles meeting end requests
func (h *MeetingHandler) EndMeeting(c *gin.Context) {
	meetingID := c.Param("meetingId")
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	err := h.signalingService.EndMeeting(meetingID, userID)
	if err != nil {
		if meetingErr, ok := err.(*models.MeetingError); ok {
			switch meetingErr.Code {
			case models.ErrMeetingNotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error": meetingErr.Message,
					"code": meetingErr.Code,
				})
				return
			case models.ErrUnauthorized:
				c.JSON(http.StatusForbidden, gin.H{
					"error": meetingErr.Message,
					"code": meetingErr.Code,
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to end meeting",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Meeting ended successfully",
	})
}

// GetMeetingParticipants handles participant list requests
func (h *MeetingHandler) GetMeetingParticipants(c *gin.Context) {
	meetingID := c.Param("meetingId")

	participants, err := h.signalingService.GetMeetingParticipants(meetingID)
	if err != nil {
		if meetingErr, ok := err.(*models.MeetingError); ok {
			if meetingErr.Code == models.ErrMeetingNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": meetingErr.Message,
					"code": meetingErr.Code,
				})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get participants",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"participants": participants,
	})
}

// ListMeetings handles listing meetings for an environment
func (h *MeetingHandler) ListMeetings(c *gin.Context) {
	environmentID := c.Query("environmentId")
	if environmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Environment ID is required",
		})
		return
	}

	meetings, err := h.signalingService.ListMeetings(environmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list meetings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meetings": meetings,
	})
}


// HandleSignalingWebSocket handles WebSocket connections for WebRTC signaling
func (h *MeetingHandler) HandleSignalingWebSocket(c *gin.Context) {
	meetingID := c.Param("meetingId")
	userID := c.Query("userId")
	displayName := c.Query("displayName")
	avatarURL := c.Query("avatarUrl")

	if userID == "" || displayName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "userId and displayName are required",
		})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}

	// Join meeting
	joinReq := &models.JoinMeetingRequest{
		MeetingID:   meetingID,
		UserID:      userID,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}

	joinResp, err := h.signalingService.JoinMeeting(c.Request.Context(), joinReq, conn)
	if err != nil {
		h.logger.Error("Failed to join meeting", zap.Error(err))
		errMsg, _ := json.Marshal(gin.H{
			"type": "error",
			"error": err.Error(),
		})
		conn.WriteMessage(websocket.TextMessage, errMsg)
		conn.Close()
		return
	}

	// Send join response
	joinRespData, _ := json.Marshal(gin.H{
		"type": "joined",
		"data": joinResp,
	})
	conn.WriteMessage(websocket.TextMessage, joinRespData)

	// Get the connection from the service
	connID := joinResp.Participant.ConnectionID
	sigConn := &services.SignalingConnection{
		ID:          connID,
		UserID:      userID,
		MeetingID:   meetingID,
		DisplayName: displayName,
		Conn:        conn,
		Send:        make(chan []byte, 256),
	}

	// Start writer goroutine
	go h.signalingService.RunConnectionWriter(sigConn)

	// Handle incoming messages
	h.handleSignalingMessages(sigConn)
}

// handleSignalingMessages reads and processes incoming WebSocket messages
func (h *MeetingHandler) handleSignalingMessages(conn *services.SignalingConnection) {
	defer h.signalingService.CloseConnection(conn)

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		var msg models.SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Error("Failed to unmarshal signaling message", zap.Error(err))
			continue
		}

		// Set user ID from connection
		msg.UserID = conn.UserID
		msg.MeetingID = conn.MeetingID

		if err := h.signalingService.HandleSignalingMessage(conn.ID, &msg); err != nil {
			h.logger.Error("Failed to handle signaling message",
				zap.Error(err),
				zap.String("type", string(msg.Type)),
			)

			// Send error back to client
			errResp := &models.SignalingMessage{
				Type:      models.SignalTypeError,
				MeetingID: conn.MeetingID,
				Payload: map[string]interface{}{
					"error": err.Error(),
				},
			}
			errData, _ := json.Marshal(errResp)
			conn.Send <- errData
		}
	}
}

// UpdateMediaState handles media state update requests
func (h *MeetingHandler) UpdateMediaState(c *gin.Context) {
	meetingID := c.Param("meetingId")
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	var payload models.MediaStatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// This would typically update the participant's media state
	// For now, we just acknowledge the request
	c.JSON(http.StatusOK, gin.H{
		"message": "Media state updated",
		"meetingId": meetingID,
		"userId": userID,
		"state": payload,
	})
}

// StartScreenShare handles screen share start requests
func (h *MeetingHandler) StartScreenShare(c *gin.Context) {
	meetingID := c.Param("meetingId")
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	var payload models.ScreenSharePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = models.ScreenSharePayload{
			Action:     "start",
			ShareType:  "screen",
			Resolution: "1080p",
			FrameRate:  30,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Screen share started",
		"meetingId": meetingID,
		"userId": userID,
		"config": payload,
	})
}

// StopScreenShare handles screen share stop requests
func (h *MeetingHandler) StopScreenShare(c *gin.Context) {
	meetingID := c.Param("meetingId")
	userID := c.GetHeader("X-User-ID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Screen share stopped",
		"meetingId": meetingID,
		"userId": userID,
	})
}
