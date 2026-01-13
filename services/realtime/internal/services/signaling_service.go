// Package services provides business logic for the realtime meeting service.
package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SignalingService handles WebRTC signaling for meetings
type SignalingService struct {
	logger      *zap.Logger
	meetings    map[string]*MeetingRoom
	connections map[string]*SignalingConnection
	mu          sync.RWMutex
	rtcConfig   *models.RTCConfiguration
}

// MeetingRoom represents an active meeting room with participants
type MeetingRoom struct {
	Meeting      *models.Meeting
	Participants map[string]*SignalingConnection
	mu           sync.RWMutex
	createdAt    time.Time
}

// SignalingConnection represents a WebSocket connection for signaling
type SignalingConnection struct {
	ID          string
	UserID      string
	MeetingID   string
	DisplayName string
	Conn        *websocket.Conn
	Send        chan []byte
	mu          sync.Mutex
	closed      bool
}

// NewSignalingService creates a new signaling service
func NewSignalingService(logger *zap.Logger) *SignalingService {
	return &SignalingService{
		logger:      logger,
		meetings:    make(map[string]*MeetingRoom),
		connections: make(map[string]*SignalingConnection),
		rtcConfig:   models.DefaultRTCConfiguration(),
	}
}

// SetRTCConfiguration sets custom RTC configuration
func (s *SignalingService) SetRTCConfiguration(config *models.RTCConfiguration) {
	s.rtcConfig = config
}

// CreateMeeting creates a new meeting room
func (s *SignalingService) CreateMeeting(ctx context.Context, req *models.CreateMeetingRequest) (*models.Meeting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	envID, err := uuid.Parse(req.EnvironmentID)
	if err != nil {
		return nil, err
	}

	hostID, err := uuid.Parse(req.HostUserID)
	if err != nil {
		return nil, err
	}

	settings := models.DefaultMeetingSettings()
	if req.Settings != nil {
		settings = *req.Settings
	}

	meetingType := req.Type
	if meetingType == "" {
		meetingType = models.MeetingTypeVideo
	}

	meeting := &models.Meeting{
		ID:            uuid.New(),
		EnvironmentID: envID,
		HostUserID:    hostID,
		Title:         req.Title,
		Type:          meetingType,
		Status:        models.MeetingStatusScheduled,
		Settings:      settings,
		Participants:  []models.MeetingParticipant{},
		CreatedAt:     time.Now(),
	}

	room := &MeetingRoom{
		Meeting:      meeting,
		Participants: make(map[string]*SignalingConnection),
		createdAt:    time.Now(),
	}

	s.meetings[meeting.ID.String()] = room

	s.logger.Info("Meeting created",
		zap.String("meetingId", meeting.ID.String()),
		zap.String("title", meeting.Title),
		zap.String("type", string(meeting.Type)),
	)

	return meeting, nil
}

// GetMeeting retrieves a meeting by ID
func (s *SignalingService) GetMeeting(meetingID string) (*models.Meeting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, exists := s.meetings[meetingID]
	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Meeting not found",
		}
	}

	return room.Meeting, nil
}


// JoinMeeting handles a user joining a meeting
func (s *SignalingService) JoinMeeting(ctx context.Context, req *models.JoinMeetingRequest, conn *websocket.Conn) (*models.JoinMeetingResponse, error) {
	s.mu.Lock()
	room, exists := s.meetings[req.MeetingID]
	if !exists {
		s.mu.Unlock()
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Meeting not found",
		}
	}
	s.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	// Check if meeting is full
	if len(room.Participants) >= room.Meeting.Settings.MaxParticipants {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingFull,
			Message: "Meeting is full",
		}
	}

	// Check if meeting has ended
	if room.Meeting.Status == models.MeetingStatusEnded {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingEnded,
			Message: "Meeting has ended",
		}
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, err
	}

	// Determine participant role
	role := models.MeetingRoleParticipant
	if room.Meeting.HostUserID.String() == req.UserID {
		role = models.MeetingRoleHost
	}

	// Create connection
	connID := uuid.New().String()
	sigConn := &SignalingConnection{
		ID:          connID,
		UserID:      req.UserID,
		MeetingID:   req.MeetingID,
		DisplayName: req.DisplayName,
		Conn:        conn,
		Send:        make(chan []byte, 256),
	}

	// Create participant
	participant := models.MeetingParticipant{
		ID:            uuid.New(),
		MeetingID:     room.Meeting.ID,
		UserID:        userID,
		DisplayName:   req.DisplayName,
		AvatarURL:     req.AvatarURL,
		Role:          role,
		JoinedAt:      time.Now(),
		AudioEnabled:  !room.Meeting.Settings.MuteOnJoin,
		VideoEnabled:  room.Meeting.Settings.VideoOnJoin && room.Meeting.Type != models.MeetingTypeAudio,
		ScreenSharing: false,
		VideoQuality:  models.VideoQuality720p,
		ConnectionID:  connID,
	}

	// Add to room
	room.Participants[connID] = sigConn
	room.Meeting.Participants = append(room.Meeting.Participants, participant)

	// Update meeting status if first participant
	if room.Meeting.Status == models.MeetingStatusScheduled {
		room.Meeting.Status = models.MeetingStatusActive
		now := time.Now()
		room.Meeting.StartedAt = &now
	}

	// Store connection
	s.mu.Lock()
	s.connections[connID] = sigConn
	s.mu.Unlock()

	// Get current participants
	participants := make([]models.MeetingParticipant, 0, len(room.Meeting.Participants))
	for _, p := range room.Meeting.Participants {
		if p.LeftAt == nil {
			participants = append(participants, p)
		}
	}

	// Notify other participants
	s.broadcastToRoom(room, connID, &models.SignalingMessage{
		Type:      models.SignalTypeUserJoined,
		MeetingID: req.MeetingID,
		UserID:    req.UserID,
		Payload: map[string]interface{}{
			"participant": participant,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("User joined meeting",
		zap.String("meetingId", req.MeetingID),
		zap.String("userId", req.UserID),
		zap.String("displayName", req.DisplayName),
	)

	return &models.JoinMeetingResponse{
		Meeting:      room.Meeting,
		Participant:  &participant,
		RTCConfig:    s.rtcConfig,
		Participants: participants,
	}, nil
}

// LeaveMeeting handles a user leaving a meeting
func (s *SignalingService) LeaveMeeting(connID string) error {
	s.mu.Lock()
	conn, exists := s.connections[connID]
	if !exists {
		s.mu.Unlock()
		return nil
	}
	delete(s.connections, connID)

	room, roomExists := s.meetings[conn.MeetingID]
	s.mu.Unlock()

	if !roomExists {
		return nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	// Remove from room participants
	delete(room.Participants, connID)

	// Update participant left time
	now := time.Now()
	for i := range room.Meeting.Participants {
		if room.Meeting.Participants[i].ConnectionID == connID {
			room.Meeting.Participants[i].LeftAt = &now
			break
		}
	}

	// Notify other participants
	s.broadcastToRoom(room, connID, &models.SignalingMessage{
		Type:      models.SignalTypeUserLeft,
		MeetingID: conn.MeetingID,
		UserID:    conn.UserID,
		Payload: map[string]interface{}{
			"connectionId": connID,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// End meeting if no participants left
	if len(room.Participants) == 0 {
		room.Meeting.Status = models.MeetingStatusEnded
		room.Meeting.EndedAt = &now
	}

	s.logger.Info("User left meeting",
		zap.String("meetingId", conn.MeetingID),
		zap.String("userId", conn.UserID),
	)

	return nil
}

// HandleSignalingMessage processes incoming signaling messages
func (s *SignalingService) HandleSignalingMessage(connID string, msg *models.SignalingMessage) error {
	s.mu.RLock()
	conn, exists := s.connections[connID]
	if !exists {
		s.mu.RUnlock()
		return &models.MeetingError{
			Code:    models.ErrConnectionFailed,
			Message: "Connection not found",
		}
	}

	room, roomExists := s.meetings[conn.MeetingID]
	s.mu.RUnlock()

	if !roomExists {
		return &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Meeting not found",
		}
	}

	switch msg.Type {
	case models.SignalTypeOffer, models.SignalTypeAnswer, models.SignalTypeCandidate:
		return s.handleRTCSignal(room, connID, msg)
	case models.SignalTypeMute, models.SignalTypeUnmute:
		return s.handleMediaStateChange(room, connID, msg)
	case models.SignalTypeScreenShare, models.SignalTypeStopShare:
		return s.handleScreenShare(room, connID, msg)
	case models.SignalTypeQualityChange:
		return s.handleQualityChange(room, connID, msg)
	default:
		return &models.MeetingError{
			Code:    models.ErrInvalidSignal,
			Message: "Unknown signal type",
		}
	}
}

// handleRTCSignal handles WebRTC offer/answer/candidate signals
func (s *SignalingService) handleRTCSignal(room *MeetingRoom, senderConnID string, msg *models.SignalingMessage) error {
	room.mu.RLock()
	defer room.mu.RUnlock()

	// If target specified, send only to target
	if msg.TargetID != "" {
		targetConn, exists := room.Participants[msg.TargetID]
		if exists {
			s.sendToConnection(targetConn, msg)
		}
		return nil
	}

	// Broadcast to all other participants
	s.broadcastToRoom(room, senderConnID, msg)
	return nil
}

// handleMediaStateChange handles mute/unmute signals
func (s *SignalingService) handleMediaStateChange(room *MeetingRoom, connID string, msg *models.SignalingMessage) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	// Update participant state
	for i := range room.Meeting.Participants {
		if room.Meeting.Participants[i].ConnectionID == connID {
			if msg.Type == models.SignalTypeMute {
				room.Meeting.Participants[i].AudioEnabled = false
			} else {
				room.Meeting.Participants[i].AudioEnabled = true
			}
			break
		}
	}

	// Broadcast state change
	s.broadcastToRoom(room, connID, &models.SignalingMessage{
		Type:      models.SignalTypeMediaState,
		MeetingID: room.Meeting.ID.String(),
		UserID:    msg.UserID,
		Payload: map[string]interface{}{
			"audioEnabled": msg.Type == models.SignalTypeUnmute,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	return nil
}


// handleScreenShare handles screen share start/stop signals
func (s *SignalingService) handleScreenShare(room *MeetingRoom, connID string, msg *models.SignalingMessage) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	if !room.Meeting.Settings.AllowScreenShare {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Screen sharing is not allowed in this meeting",
		}
	}

	isStarting := msg.Type == models.SignalTypeScreenShare

	// Check if someone else is already sharing
	if isStarting {
		for _, p := range room.Meeting.Participants {
			if p.ScreenSharing && p.ConnectionID != connID {
				return &models.MeetingError{
					Code:    models.ErrScreenShareInUse,
					Message: "Another participant is already sharing their screen",
				}
			}
		}
	}

	// Update participant state
	for i := range room.Meeting.Participants {
		if room.Meeting.Participants[i].ConnectionID == connID {
			room.Meeting.Participants[i].ScreenSharing = isStarting
			break
		}
	}

	// Broadcast screen share state
	s.broadcastToRoom(room, connID, &models.SignalingMessage{
		Type:      msg.Type,
		MeetingID: room.Meeting.ID.String(),
		UserID:    msg.UserID,
		Payload:   msg.Payload,
		Timestamp: time.Now().UnixMilli(),
	})

	s.logger.Info("Screen share state changed",
		zap.String("meetingId", room.Meeting.ID.String()),
		zap.String("userId", msg.UserID),
		zap.Bool("sharing", isStarting),
	)

	return nil
}

// handleQualityChange handles video quality change signals
func (s *SignalingService) handleQualityChange(room *MeetingRoom, connID string, msg *models.SignalingMessage) error {
	room.mu.Lock()
	defer room.mu.Unlock()

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return &models.MeetingError{
			Code:    models.ErrInvalidSignal,
			Message: "Invalid quality change payload",
		}
	}

	quality, _ := payload["quality"].(string)

	// Update participant quality
	for i := range room.Meeting.Participants {
		if room.Meeting.Participants[i].ConnectionID == connID {
			room.Meeting.Participants[i].VideoQuality = models.VideoQuality(quality)
			break
		}
	}

	// Broadcast quality change
	s.broadcastToRoom(room, connID, &models.SignalingMessage{
		Type:      models.SignalTypeQualityChange,
		MeetingID: room.Meeting.ID.String(),
		UserID:    msg.UserID,
		Payload:   msg.Payload,
		Timestamp: time.Now().UnixMilli(),
	})

	return nil
}

// broadcastToRoom sends a message to all participants except the sender
func (s *SignalingService) broadcastToRoom(room *MeetingRoom, excludeConnID string, msg *models.SignalingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	for connID, conn := range room.Participants {
		if connID != excludeConnID {
			select {
			case conn.Send <- data:
			default:
				s.logger.Warn("Failed to send message to participant",
					zap.String("connectionId", connID),
				)
			}
		}
	}
}

// sendToConnection sends a message to a specific connection
func (s *SignalingService) sendToConnection(conn *SignalingConnection, msg *models.SignalingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	select {
	case conn.Send <- data:
	default:
		s.logger.Warn("Failed to send message to connection",
			zap.String("connectionId", conn.ID),
		)
	}
}

// EndMeeting ends a meeting
func (s *SignalingService) EndMeeting(meetingID string, userID string) error {
	s.mu.Lock()
	room, exists := s.meetings[meetingID]
	if !exists {
		s.mu.Unlock()
		return &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Meeting not found",
		}
	}
	s.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	// Check if user is host
	if room.Meeting.HostUserID.String() != userID {
		// Check if user is co-host
		isCoHost := false
		for _, p := range room.Meeting.Participants {
			if p.UserID.String() == userID && p.Role == models.MeetingRoleCoHost {
				isCoHost = true
				break
			}
		}
		if !isCoHost {
			return &models.MeetingError{
				Code:    models.ErrUnauthorized,
				Message: "Only host or co-host can end the meeting",
			}
		}
	}

	// Update meeting status
	now := time.Now()
	room.Meeting.Status = models.MeetingStatusEnded
	room.Meeting.EndedAt = &now

	// Notify all participants
	for _, conn := range room.Participants {
		s.sendToConnection(conn, &models.SignalingMessage{
			Type:      models.SignalTypeLeave,
			MeetingID: meetingID,
			Payload: map[string]interface{}{
				"reason": "meeting_ended",
			},
			Timestamp: time.Now().UnixMilli(),
		})
	}

	s.logger.Info("Meeting ended",
		zap.String("meetingId", meetingID),
		zap.String("endedBy", userID),
	)

	return nil
}

// GetMeetingParticipants returns the list of active participants
func (s *SignalingService) GetMeetingParticipants(meetingID string) ([]models.MeetingParticipant, error) {
	s.mu.RLock()
	room, exists := s.meetings[meetingID]
	s.mu.RUnlock()

	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Meeting not found",
		}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	participants := make([]models.MeetingParticipant, 0)
	for _, p := range room.Meeting.Participants {
		if p.LeftAt == nil {
			participants = append(participants, p)
		}
	}

	return participants, nil
}

// RunConnectionWriter handles writing messages to a WebSocket connection
func (s *SignalingService) RunConnectionWriter(conn *SignalingConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-conn.Send:
			if !ok {
				return
			}
			conn.mu.Lock()
			if conn.closed {
				conn.mu.Unlock()
				return
			}
			err := conn.Conn.WriteMessage(websocket.TextMessage, message)
			conn.mu.Unlock()
			if err != nil {
				s.logger.Error("Failed to write message", zap.Error(err))
				return
			}
		case <-ticker.C:
			conn.mu.Lock()
			if conn.closed {
				conn.mu.Unlock()
				return
			}
			err := conn.Conn.WriteMessage(websocket.PingMessage, nil)
			conn.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// CloseConnection closes a signaling connection
func (s *SignalingService) CloseConnection(conn *SignalingConnection) {
	conn.mu.Lock()
	if conn.closed {
		conn.mu.Unlock()
		return
	}
	conn.closed = true
	close(conn.Send)
	conn.Conn.Close()
	conn.mu.Unlock()

	s.LeaveMeeting(conn.ID)
}

// ListMeetings returns all active meetings for an environment
func (s *SignalingService) ListMeetings(environmentID string) ([]*models.Meeting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meetings := make([]*models.Meeting, 0)
	for _, room := range s.meetings {
		if room.Meeting.EnvironmentID.String() == environmentID &&
			room.Meeting.Status == models.MeetingStatusActive {
			meetings = append(meetings, room.Meeting)
		}
	}

	return meetings, nil
}
