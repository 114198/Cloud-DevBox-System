// Package services provides business logic for the realtime collaboration service.
package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"go.uber.org/zap"
)

// CursorService manages cursor and selection synchronization
type CursorService struct {
	// Cursors by session ID -> user ID -> cursor position
	cursors map[string]map[string]*models.CursorPosition

	// Selections by session ID -> user ID -> selection
	selections map[string]map[string]*models.Selection

	// User colors by session ID -> user ID -> color
	userColors map[string]map[string]string

	mu     sync.RWMutex
	logger *zap.Logger
	hub    *Hub
}

// NewCursorService creates a new cursor service
func NewCursorService(logger *zap.Logger, hub *Hub) *CursorService {
	return &CursorService{
		cursors:    make(map[string]map[string]*models.CursorPosition),
		selections: make(map[string]map[string]*models.Selection),
		userColors: make(map[string]map[string]string),
		logger:     logger,
		hub:        hub,
	}
}

// UpdateCursor updates a user's cursor position
func (s *CursorService) UpdateCursor(sessionID, userID string, position *models.CursorPosition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cursors[sessionID]; !ok {
		s.cursors[sessionID] = make(map[string]*models.CursorPosition)
	}
	s.cursors[sessionID][userID] = position
}

// GetCursor gets a user's cursor position
func (s *CursorService) GetCursor(sessionID, userID string) *models.CursorPosition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cursors, ok := s.cursors[sessionID]; ok {
		return cursors[userID]
	}
	return nil
}

// GetAllCursors gets all cursor positions for a session
func (s *CursorService) GetAllCursors(sessionID string) map[string]*models.CursorPosition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.CursorPosition)
	if cursors, ok := s.cursors[sessionID]; ok {
		for userID, cursor := range cursors {
			result[userID] = cursor
		}
	}
	return result
}

// RemoveCursor removes a user's cursor
func (s *CursorService) RemoveCursor(sessionID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cursors, ok := s.cursors[sessionID]; ok {
		delete(cursors, userID)
		if len(cursors) == 0 {
			delete(s.cursors, sessionID)
		}
	}
}

// UpdateSelection updates a user's selection
func (s *CursorService) UpdateSelection(sessionID, userID string, selection *models.Selection) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.selections[sessionID]; !ok {
		s.selections[sessionID] = make(map[string]*models.Selection)
	}
	s.selections[sessionID][userID] = selection
}

// GetSelection gets a user's selection
func (s *CursorService) GetSelection(sessionID, userID string) *models.Selection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if selections, ok := s.selections[sessionID]; ok {
		return selections[userID]
	}
	return nil
}

// GetAllSelections gets all selections for a session
func (s *CursorService) GetAllSelections(sessionID string) map[string]*models.Selection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.Selection)
	if selections, ok := s.selections[sessionID]; ok {
		for userID, selection := range selections {
			result[userID] = selection
		}
	}
	return result
}

// ClearSelection clears a user's selection
func (s *CursorService) ClearSelection(sessionID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if selections, ok := s.selections[sessionID]; ok {
		delete(selections, userID)
		if len(selections) == 0 {
			delete(s.selections, sessionID)
		}
	}
}


// SetUserColor sets a user's color for a session
func (s *CursorService) SetUserColor(sessionID, userID, color string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.userColors[sessionID]; !ok {
		s.userColors[sessionID] = make(map[string]string)
	}
	s.userColors[sessionID][userID] = color
}

// GetUserColor gets a user's color for a session
func (s *CursorService) GetUserColor(sessionID, userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if colors, ok := s.userColors[sessionID]; ok {
		if color, exists := colors[userID]; exists {
			return color
		}
	}
	return models.UserColors[0] // Default color
}

// GetAllUserColors gets all user colors for a session
func (s *CursorService) GetAllUserColors(sessionID string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	if colors, ok := s.userColors[sessionID]; ok {
		for userID, color := range colors {
			result[userID] = color
		}
	}
	return result
}

// CleanupSession removes all cursor and selection data for a session
func (s *CursorService) CleanupSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cursors, sessionID)
	delete(s.selections, sessionID)
	delete(s.userColors, sessionID)
}

// CleanupUser removes all cursor and selection data for a user in a session
func (s *CursorService) CleanupUser(sessionID, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cursors, ok := s.cursors[sessionID]; ok {
		delete(cursors, userID)
	}
	if selections, ok := s.selections[sessionID]; ok {
		delete(selections, userID)
	}
	if colors, ok := s.userColors[sessionID]; ok {
		delete(colors, userID)
	}
}

// BroadcastCursorUpdate broadcasts a cursor update to all clients in a session
func (s *CursorService) BroadcastCursorUpdate(sessionID, userID string, position *models.CursorPosition, excludeClientID string) {
	msg := models.WebSocketMessage{
		Type:      models.MessageTypeCursor,
		SessionID: sessionID,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
		Data: models.CursorUpdate{
			UserID:   userID,
			Position: *position,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to marshal cursor update", zap.Error(err))
		return
	}

	s.hub.Broadcast(sessionID, data, excludeClientID)
}

// BroadcastSelectionUpdate broadcasts a selection update to all clients in a session
func (s *CursorService) BroadcastSelectionUpdate(sessionID, userID string, selection *models.Selection, excludeClientID string) {
	msg := models.WebSocketMessage{
		Type:      models.MessageTypeSelection,
		SessionID: sessionID,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
		Data: models.SelectionUpdate{
			UserID:    userID,
			Selection: *selection,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		s.logger.Error("Failed to marshal selection update", zap.Error(err))
		return
	}

	s.hub.Broadcast(sessionID, data, excludeClientID)
}

// GetCursorState returns the complete cursor state for a session
type CursorState struct {
	Cursors    map[string]*models.CursorPosition `json:"cursors"`
	Selections map[string]*models.Selection      `json:"selections"`
	Colors     map[string]string                 `json:"colors"`
}

// GetCursorState gets the complete cursor state for a session
func (s *CursorService) GetCursorState(sessionID string) *CursorState {
	return &CursorState{
		Cursors:    s.GetAllCursors(sessionID),
		Selections: s.GetAllSelections(sessionID),
		Colors:     s.GetAllUserColors(sessionID),
	}
}

// GetFileCursors gets all cursors for a specific file in a session
func (s *CursorService) GetFileCursors(sessionID, filePath string) map[string]*models.CursorPosition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.CursorPosition)
	if cursors, ok := s.cursors[sessionID]; ok {
		for userID, cursor := range cursors {
			if cursor.FilePath == filePath {
				result[userID] = cursor
			}
		}
	}
	return result
}

// GetFileSelections gets all selections for a specific file in a session
func (s *CursorService) GetFileSelections(sessionID, filePath string) map[string]*models.Selection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.Selection)
	if selections, ok := s.selections[sessionID]; ok {
		for userID, selection := range selections {
			if selection.FilePath == filePath {
				result[userID] = selection
			}
		}
	}
	return result
}
