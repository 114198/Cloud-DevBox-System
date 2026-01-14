// Package services provides business logic for the realtime collaboration service.
package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// HistoryRetentionDays is the number of days to retain history
	HistoryRetentionDays = 30

	// MaxHistoryEntriesPerSession is the maximum number of history entries per session
	MaxHistoryEntriesPerSession = 10000

	// HistoryCleanupInterval is the interval for cleaning up old history
	HistoryCleanupInterval = 1 * time.Hour
)

// HistoryService manages collaboration history
type HistoryService struct {
	// History entries by session ID
	history map[string][]*models.CollaborationHistory

	// History index by session ID -> operation ID -> index
	historyIndex map[string]map[string]int

	// Optional persistence file path
	filePath string

	mu     sync.RWMutex
	logger *zap.Logger
}

// NewHistoryService creates a new history service
func NewHistoryService(logger *zap.Logger) *HistoryService {
	service := &HistoryService{
		history:      make(map[string][]*models.CollaborationHistory),
		historyIndex: make(map[string]map[string]int),
		filePath:     os.Getenv("COLLAB_HISTORY_PATH"),
		logger:       logger,
	}
	if service.filePath != "" {
		service.loadFromDisk()
	}
	return service
}

// RecordOperation records an operation in the history
func (s *HistoryService) RecordOperation(sessionID string, userID uuid.UUID, operationType string, filePath string, operationData interface{}) *models.CollaborationHistory {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &models.CollaborationHistory{
		ID:            uuid.New(),
		SessionID:     uuid.MustParse(sessionID),
		UserID:        userID,
		OperationType: operationType,
		FilePath:      filePath,
		OperationData: operationData,
		CreatedAt:     time.Now(),
	}

	if _, ok := s.history[sessionID]; !ok {
		s.history[sessionID] = make([]*models.CollaborationHistory, 0)
		s.historyIndex[sessionID] = make(map[string]int)
	}

	// Check if we need to trim history
	if len(s.history[sessionID]) >= MaxHistoryEntriesPerSession {
		// Remove oldest 10% of entries
		trimCount := MaxHistoryEntriesPerSession / 10
		s.history[sessionID] = s.history[sessionID][trimCount:]
		// Rebuild index
		s.rebuildIndex(sessionID)
	}

	index := len(s.history[sessionID])
	s.history[sessionID] = append(s.history[sessionID], entry)
	s.historyIndex[sessionID][entry.ID.String()] = index

	s.persistToDiskLocked()

	return entry
}

// rebuildIndex rebuilds the history index for a session
func (s *HistoryService) rebuildIndex(sessionID string) {
	s.historyIndex[sessionID] = make(map[string]int)
	for i, entry := range s.history[sessionID] {
		s.historyIndex[sessionID][entry.ID.String()] = i
	}
}

// GetHistory gets history entries for a session
func (s *HistoryService) GetHistory(sessionID string, limit int, offset int) []*models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok {
		return nil
	}

	// Apply offset and limit
	start := offset
	if start >= len(entries) {
		return nil
	}

	end := start + limit
	if end > len(entries) {
		end = len(entries)
	}

	// Return entries in reverse chronological order
	result := make([]*models.CollaborationHistory, end-start)
	for i := start; i < end; i++ {
		result[i-start] = entries[len(entries)-1-i]
	}

	return result
}

// GetHistoryByFile gets history entries for a specific file
func (s *HistoryService) GetHistoryByFile(sessionID, filePath string, limit int) []*models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok {
		return nil
	}

	var result []*models.CollaborationHistory
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		if entries[i].FilePath == filePath {
			result = append(result, entries[i])
		}
	}

	return result
}

// GetHistoryByUser gets history entries for a specific user
func (s *HistoryService) GetHistoryByUser(sessionID string, userID uuid.UUID, limit int) []*models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok {
		return nil
	}

	var result []*models.CollaborationHistory
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		if entries[i].UserID == userID {
			result = append(result, entries[i])
		}
	}

	return result
}

// GetHistoryByTimeRange gets history entries within a time range
func (s *HistoryService) GetHistoryByTimeRange(sessionID string, startTime, endTime time.Time, limit int) []*models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok {
		return nil
	}

	var result []*models.CollaborationHistory
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		entry := entries[i]
		if entry.CreatedAt.After(startTime) && entry.CreatedAt.Before(endTime) {
			result = append(result, entry)
		}
	}

	return result
}

// GetHistoryEntry gets a specific history entry by ID
func (s *HistoryService) GetHistoryEntry(sessionID, entryID string) *models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index, ok := s.historyIndex[sessionID]; ok {
		if idx, exists := index[entryID]; exists {
			return s.history[sessionID][idx]
		}
	}
	return nil
}

// GetHistoryCount gets the number of history entries for a session
func (s *HistoryService) GetHistoryCount(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entries, ok := s.history[sessionID]; ok {
		return len(entries)
	}
	return 0
}

// ClearHistory clears all history for a session
func (s *HistoryService) ClearHistory(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.history, sessionID)
	delete(s.historyIndex, sessionID)
}

// CleanupOldHistory removes history entries older than the retention period
func (s *HistoryService) CleanupOldHistory() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -HistoryRetentionDays)
	totalRemoved := 0

	for sessionID, entries := range s.history {
		var newEntries []*models.CollaborationHistory
		for _, entry := range entries {
			if entry.CreatedAt.After(cutoff) {
				newEntries = append(newEntries, entry)
			} else {
				totalRemoved++
			}
		}

		if len(newEntries) == 0 {
			delete(s.history, sessionID)
			delete(s.historyIndex, sessionID)
		} else if len(newEntries) != len(entries) {
			s.history[sessionID] = newEntries
			s.rebuildIndex(sessionID)
		}
	}

	if totalRemoved > 0 {
		s.logger.Info("Cleaned up old history entries",
			zap.Int("removedCount", totalRemoved),
		)
		s.persistToDiskLocked()
	}

	return totalRemoved
}

func (s *HistoryService) loadFromDisk() {
	if s.filePath == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logger.Warn("Failed to read history file", zap.Error(err))
		}
		return
	}

	var stored map[string][]*models.CollaborationHistory
	if err := json.Unmarshal(data, &stored); err != nil {
		s.logger.Warn("Failed to parse history file", zap.Error(err))
		return
	}

	s.history = stored
	for sessionID := range s.history {
		s.rebuildIndex(sessionID)
	}
}

func (s *HistoryService) persistToDiskLocked() {
	if s.filePath == "" {
		return
	}

	data, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		s.logger.Warn("Failed to serialize history", zap.Error(err))
		return
	}

	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		s.logger.Warn("Failed to create history directory", zap.Error(err))
		return
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		s.logger.Warn("Failed to write history file", zap.Error(err))
		return
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		s.logger.Warn("Failed to persist history file", zap.Error(err))
	}
}

// StartCleanupRoutine starts a background routine to clean up old history
func (s *HistoryService) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(HistoryCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.CleanupOldHistory()
		}
	}
}

// ExportHistory exports history for a session as a slice of entries
func (s *HistoryService) ExportHistory(sessionID string) []*models.CollaborationHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]*models.CollaborationHistory, len(entries))
	copy(result, entries)
	return result
}

// ImportHistory imports history entries for a session
func (s *HistoryService) ImportHistory(sessionID string, entries []*models.CollaborationHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history[sessionID] = make([]*models.CollaborationHistory, len(entries))
	copy(s.history[sessionID], entries)
	s.rebuildIndex(sessionID)
}

// GetSessionStats returns statistics about a session's history
type SessionHistoryStats struct {
	TotalOperations  int            `json:"totalOperations"`
	OperationsByType map[string]int `json:"operationsByType"`
	OperationsByUser map[string]int `json:"operationsByUser"`
	OperationsByFile map[string]int `json:"operationsByFile"`
	FirstOperation   *time.Time     `json:"firstOperation,omitempty"`
	LastOperation    *time.Time     `json:"lastOperation,omitempty"`
}

// GetSessionStats gets statistics about a session's history
func (s *HistoryService) GetSessionStats(sessionID string) *SessionHistoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, ok := s.history[sessionID]
	if !ok || len(entries) == 0 {
		return &SessionHistoryStats{
			OperationsByType: make(map[string]int),
			OperationsByUser: make(map[string]int),
			OperationsByFile: make(map[string]int),
		}
	}

	stats := &SessionHistoryStats{
		TotalOperations:  len(entries),
		OperationsByType: make(map[string]int),
		OperationsByUser: make(map[string]int),
		OperationsByFile: make(map[string]int),
	}

	for _, entry := range entries {
		stats.OperationsByType[entry.OperationType]++
		stats.OperationsByUser[entry.UserID.String()]++
		if entry.FilePath != "" {
			stats.OperationsByFile[entry.FilePath]++
		}
	}

	if len(entries) > 0 {
		first := entries[0].CreatedAt
		last := entries[len(entries)-1].CreatedAt
		stats.FirstOperation = &first
		stats.LastOperation = &last
	}

	return stats
}
