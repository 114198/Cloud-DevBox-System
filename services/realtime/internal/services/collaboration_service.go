// Package services provides business logic for the realtime collaboration service.
package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CollaborationService orchestrates all collaboration functionality
type CollaborationService struct {
	hub              *Hub
	crdtEngine       *CRDTEngine
	cursorService    *CursorService
	conflictResolver *ConflictResolver
	historyService   *HistoryService

	// Sessions by session ID
	sessions map[string]*models.CollaborationSession

	// User clocks for CRDT operations
	userClocks map[string]int64

	mu     sync.RWMutex
	logger *zap.Logger
}

// NewCollaborationService creates a new collaboration service
func NewCollaborationService(logger *zap.Logger, hub *Hub) *CollaborationService {
	cs := &CollaborationService{
		hub:              hub,
		crdtEngine:       NewCRDTEngine(logger),
		cursorService:    NewCursorService(logger, hub),
		conflictResolver: NewConflictResolver(logger),
		historyService:   NewHistoryService(logger),
		sessions:         make(map[string]*models.CollaborationSession),
		userClocks:       make(map[string]int64),
		logger:           logger,
	}

	// Set the collaboration service reference in the hub
	hub.SetCollaborationService(cs)

	return cs
}

// Start starts the collaboration service
func (s *CollaborationService) Start(ctx context.Context) {
	// Start the hub
	go s.hub.Run(ctx)

	// Start history cleanup routine
	go s.historyService.StartCleanupRoutine(ctx)

	s.logger.Info("Collaboration service started")
}

// CreateSession creates a new collaboration session
func (s *CollaborationService) CreateSession(environmentID, createdBy uuid.UUID, settings *models.SessionSettings) (*models.CollaborationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if settings == nil {
		defaultSettings := models.DefaultSessionSettings()
		settings = &defaultSettings
	}

	session := &models.CollaborationSession{
		ID:            uuid.New(),
		EnvironmentID: environmentID,
		CreatedBy:     createdBy,
		Status:        models.SessionStatusActive,
		Settings:      *settings,
		Participants:  make([]models.SessionParticipant, 0),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.sessions[session.ID.String()] = session

	s.logger.Info("Collaboration session created",
		zap.String("sessionId", session.ID.String()),
		zap.String("environmentId", environmentID.String()),
	)

	return session, nil
}

// GetSession gets a session by ID
func (s *CollaborationService) GetSession(sessionID string) (*models.CollaborationSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// GetOrCreateSessionForEnvironment gets or creates a session for an environment
func (s *CollaborationService) GetOrCreateSessionForEnvironment(environmentID, userID uuid.UUID) (*models.CollaborationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if a session already exists for this environment
	for _, session := range s.sessions {
		if session.EnvironmentID == environmentID && session.Status == models.SessionStatusActive {
			return session, nil
		}
	}

	// Create a new session
	session := &models.CollaborationSession{
		ID:            uuid.New(),
		EnvironmentID: environmentID,
		CreatedBy:     userID,
		Status:        models.SessionStatusActive,
		Settings:      models.DefaultSessionSettings(),
		Participants:  make([]models.SessionParticipant, 0),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	s.sessions[session.ID.String()] = session

	return session, nil
}


// JoinSession handles a user joining a collaboration session
func (s *CollaborationService) JoinSession(sessionID string, client *Client) (*models.JoinResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	if session.Status != models.SessionStatusActive {
		return nil, ErrSessionNotFound
	}

	// Check if session is full
	if len(session.Participants) >= session.Settings.MaxParticipants {
		return nil, ErrSessionFull
	}

	// Add participant
	participant := models.SessionParticipant{
		ID:          uuid.New(),
		SessionID:   session.ID,
		UserID:      uuid.MustParse(client.UserID),
		Username:    client.Username,
		DisplayName: client.DisplayName,
		AvatarURL:   client.AvatarURL,
		Role:        models.RoleParticipant,
		Color:       client.Color,
		JoinedAt:    time.Now(),
		IsOnline:    true,
	}

	// First participant becomes host
	if len(session.Participants) == 0 {
		participant.Role = models.RoleHost
	}

	session.Participants = append(session.Participants, participant)
	session.UpdatedAt = time.Now()

	// Set user color in cursor service
	s.cursorService.SetUserColor(sessionID, client.UserID, client.Color)

	// Record join in history
	s.historyService.RecordOperation(
		sessionID,
		participant.UserID,
		"join",
		"",
		map[string]interface{}{
			"username":    client.Username,
			"displayName": client.DisplayName,
		},
	)

	// Register client with hub
	s.hub.register <- client

	response := &models.JoinResponse{
		SessionID:    sessionID,
		UserID:       client.UserID,
		Color:        client.Color,
		Participants: session.Participants,
	}

	return response, nil
}

// LeaveSession handles a user leaving a collaboration session
func (s *CollaborationService) LeaveSession(sessionID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	// Find and update participant
	for i := range session.Participants {
		if session.Participants[i].UserID.String() == userID {
			now := time.Now()
			session.Participants[i].LeftAt = &now
			session.Participants[i].IsOnline = false
			break
		}
	}

	session.UpdatedAt = time.Now()

	// Cleanup cursor and selection data
	s.cursorService.CleanupUser(sessionID, userID)

	// Record leave in history
	s.historyService.RecordOperation(
		sessionID,
		uuid.MustParse(userID),
		"leave",
		"",
		nil,
	)

	// Check if session should be ended (no active participants)
	activeCount := 0
	for _, p := range session.Participants {
		if p.IsOnline {
			activeCount++
		}
	}

	if activeCount == 0 {
		session.Status = models.SessionStatusEnded
	}

	return nil
}

// EndSession ends a collaboration session
func (s *CollaborationService) EndSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.Status = models.SessionStatusEnded
	session.UpdatedAt = time.Now()

	// Cleanup all resources
	s.crdtEngine.RemoveSessionDocuments(sessionID)
	s.cursorService.CleanupSession(sessionID)
	s.conflictResolver.CleanupSession(sessionID)

	return nil
}

// HandleEditOperation handles an edit operation from a client
func (s *CollaborationService) HandleEditOperation(client *Client, msg *models.WebSocketMessage) {
	// Parse edit operation from message data
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		s.sendError(client, "invalid_data", "Invalid edit operation data")
		return
	}

	filePath, _ := data["filePath"].(string)
	opType, _ := data["type"].(string)
	position, _ := data["position"].(float64)
	content, _ := data["content"].(string)
	length, _ := data["length"].(float64)

	// Get or create document
	doc := s.crdtEngine.GetOrCreateDocument(client.SessionID, filePath)

	// Get user clock
	s.mu.Lock()
	clockKey := client.SessionID + ":" + client.UserID
	clock := s.userClocks[clockKey]
	s.userClocks[clockKey] = clock + 1
	s.mu.Unlock()

	// Create and apply operation based on type
	var crdtOp *CRDTOperation
	switch models.OperationType(opType) {
	case models.OpInsert:
		for i, r := range content {
			crdtOp = s.crdtEngine.CreateInsertOperation(
				client.SessionID,
				filePath,
				client.UserID,
				int(position)+i,
				r,
				clock+int64(i),
			)
			doc.ApplyOperation(crdtOp)
		}
	case models.OpDelete:
		for i := 0; i < int(length); i++ {
			crdtOp = s.crdtEngine.CreateDeleteOperation(
				client.SessionID,
				filePath,
				client.UserID,
				int(position),
				clock+int64(i),
			)
			if crdtOp != nil {
				doc.ApplyOperation(crdtOp)
			}
		}
	}

	// Record in history
	s.historyService.RecordOperation(
		client.SessionID,
		uuid.MustParse(client.UserID),
		opType,
		filePath,
		data,
	)

	// Broadcast to other clients
	broadcastMsg := models.WebSocketMessage{
		Type:          models.MessageTypeEdit,
		SessionID:     client.SessionID,
		EnvironmentID: client.EnvironmentID,
		UserID:        client.UserID,
		Timestamp:     time.Now().UnixMilli(),
		Sequence:      msg.Sequence,
		Data:          data,
	}

	broadcastData, _ := json.Marshal(broadcastMsg)
	s.hub.Broadcast(client.SessionID, broadcastData, client.ID)

	// Send acknowledgment
	s.sendAck(client, msg.AckID, doc.GetVersion())
}


// HandleCursorUpdate handles a cursor position update from a client
func (s *CollaborationService) HandleCursorUpdate(client *Client, msg *models.WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	position := &models.CursorPosition{
		FilePath: getString(data, "filePath"),
		Line:     getInt(data, "line"),
		Column:   getInt(data, "column"),
		Offset:   getInt(data, "offset"),
	}

	// Update cursor in service
	s.cursorService.UpdateCursor(client.SessionID, client.UserID, position)

	// Broadcast to other clients
	s.cursorService.BroadcastCursorUpdate(client.SessionID, client.UserID, position, client.ID)
}

// HandleSelectionUpdate handles a selection update from a client
func (s *CollaborationService) HandleSelectionUpdate(client *Client, msg *models.WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	selection := &models.Selection{
		FilePath:    getString(data, "filePath"),
		StartLine:   getInt(data, "startLine"),
		StartCol:    getInt(data, "startColumn"),
		EndLine:     getInt(data, "endLine"),
		EndCol:      getInt(data, "endColumn"),
		StartOffset: getInt(data, "startOffset"),
		EndOffset:   getInt(data, "endOffset"),
	}

	// Update selection in service
	s.cursorService.UpdateSelection(client.SessionID, client.UserID, selection)

	// Broadcast to other clients
	s.cursorService.BroadcastSelectionUpdate(client.SessionID, client.UserID, selection, client.ID)
}

// HandleSyncRequest handles a sync request from a client
func (s *CollaborationService) HandleSyncRequest(client *Client, msg *models.WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		s.sendError(client, "invalid_data", "Invalid sync request data")
		return
	}

	filePath := getString(data, "filePath")

	// Get document state
	doc, exists := s.crdtEngine.GetDocument(client.SessionID, filePath)
	if !exists {
		s.sendError(client, "document_not_found", "Document not found")
		return
	}

	// Get cursor state
	cursorState := s.cursorService.GetCursorState(client.SessionID)

	// Send sync response
	response := models.WebSocketMessage{
		Type:          models.MessageTypeSync,
		SessionID:     client.SessionID,
		EnvironmentID: client.EnvironmentID,
		UserID:        client.UserID,
		Timestamp:     time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"document": doc.GetState(),
			"cursors":  cursorState,
		},
	}

	responseData, _ := json.Marshal(response)
	client.Send <- responseData
}

// GetDocumentState gets the current state of a document
func (s *CollaborationService) GetDocumentState(sessionID, filePath string) (*models.DocumentState, error) {
	doc, exists := s.crdtEngine.GetDocument(sessionID, filePath)
	if !exists {
		return nil, ErrDocumentNotFound
	}
	return doc.GetState(), nil
}

// InitializeDocument initializes a document with content
func (s *CollaborationService) InitializeDocument(sessionID, filePath, content, siteID string) {
	doc := s.crdtEngine.GetOrCreateDocument(sessionID, filePath)
	doc.InitializeDocument(content, siteID)
}

// GetSessionParticipants gets all participants in a session
func (s *CollaborationService) GetSessionParticipants(sessionID string) ([]models.SessionParticipant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session.Participants, nil
}

// GetSessionHistory gets history for a session
func (s *CollaborationService) GetSessionHistory(sessionID string, limit, offset int) []*models.CollaborationHistory {
	return s.historyService.GetHistory(sessionID, limit, offset)
}

// GetSessionStats gets statistics for a session
func (s *CollaborationService) GetSessionStats(sessionID string) *SessionHistoryStats {
	return s.historyService.GetSessionStats(sessionID)
}

// sendError sends an error message to a client
func (s *CollaborationService) sendError(client *Client, code, message string) {
	msg := models.WebSocketMessage{
		Type:      models.MessageTypeError,
		SessionID: client.SessionID,
		UserID:    client.UserID,
		Timestamp: time.Now().UnixMilli(),
		Data: models.ErrorResponse{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(msg)
	client.Send <- data
}

// sendAck sends an acknowledgment message to a client
func (s *CollaborationService) sendAck(client *Client, ackID string, version int64) {
	msg := models.WebSocketMessage{
		Type:      models.MessageTypeAck,
		SessionID: client.SessionID,
		UserID:    client.UserID,
		Timestamp: time.Now().UnixMilli(),
		AckID:     ackID,
		Data: map[string]interface{}{
			"version": version,
		},
	}
	data, _ := json.Marshal(msg)
	client.Send <- data
}

// Helper functions for type conversion
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if v, ok := data[key].(float64); ok {
		return int(v)
	}
	return 0
}
