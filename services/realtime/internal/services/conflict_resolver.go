// Package services provides business logic for the realtime collaboration service.
package services

import (
	"strings"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConflictResolver handles conflict detection and resolution
type ConflictResolver struct {
	// Pending conflicts by session ID -> conflict ID -> conflict
	conflicts map[string]map[string]*models.ConflictInfo

	// Resolution strategies
	strategies map[string]ResolutionStrategy

	mu     sync.RWMutex
	logger *zap.Logger
}

// ResolutionStrategy defines a conflict resolution strategy
type ResolutionStrategy interface {
	Resolve(conflict *models.ConflictInfo) (*models.ConflictResolution, error)
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(logger *zap.Logger) *ConflictResolver {
	cr := &ConflictResolver{
		conflicts:  make(map[string]map[string]*models.ConflictInfo),
		strategies: make(map[string]ResolutionStrategy),
		logger:     logger,
	}

	// Register default strategies
	cr.strategies["last_write_wins"] = &LastWriteWinsStrategy{}
	cr.strategies["first_write_wins"] = &FirstWriteWinsStrategy{}
	cr.strategies["three_way_merge"] = &ThreeWayMergeStrategy{}

	return cr
}

// DetectConflict checks if two operations conflict
func (r *ConflictResolver) DetectConflict(op1, op2 *models.EditOperation) bool {
	// Operations on different files don't conflict
	if op1.FilePath != op2.FilePath {
		return false
	}

	// Operations from the same user don't conflict
	if op1.UserID == op2.UserID {
		return false
	}

	// Check if operations overlap
	switch {
	case op1.Type == models.OpInsert && op2.Type == models.OpInsert:
		// Two inserts at the same position conflict
		return op1.Position == op2.Position
	case op1.Type == models.OpDelete && op2.Type == models.OpDelete:
		// Two deletes that overlap conflict
		return r.rangesOverlap(op1.Position, op1.Length, op2.Position, op2.Length)
	case op1.Type == models.OpInsert && op2.Type == models.OpDelete:
		// Insert within delete range conflicts
		return op1.Position >= op2.Position && op1.Position < op2.Position+op2.Length
	case op1.Type == models.OpDelete && op2.Type == models.OpInsert:
		// Delete range containing insert conflicts
		return op2.Position >= op1.Position && op2.Position < op1.Position+op1.Length
	}

	return false
}

// rangesOverlap checks if two ranges overlap
func (r *ConflictResolver) rangesOverlap(start1, len1, start2, len2 int) bool {
	end1 := start1 + len1
	end2 := start2 + len2
	return start1 < end2 && start2 < end1
}

// CreateConflict creates a new conflict record
func (r *ConflictResolver) CreateConflict(sessionID string, localOp, remoteOp *models.EditOperation, baseVersion int64) *models.ConflictInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	conflict := &models.ConflictInfo{
		ID:          uuid.New().String(),
		FilePath:    localOp.FilePath,
		BaseVersion: baseVersion,
		LocalOp:     localOp,
		RemoteOp:    remoteOp,
		CreatedAt:   time.Now().UnixMilli(),
	}

	if _, ok := r.conflicts[sessionID]; !ok {
		r.conflicts[sessionID] = make(map[string]*models.ConflictInfo)
	}
	r.conflicts[sessionID][conflict.ID] = conflict

	r.logger.Info("Conflict created",
		zap.String("conflictId", conflict.ID),
		zap.String("sessionId", sessionID),
		zap.String("filePath", conflict.FilePath),
	)

	return conflict
}

// GetConflict gets a conflict by ID
func (r *ConflictResolver) GetConflict(sessionID, conflictID string) *models.ConflictInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if conflicts, ok := r.conflicts[sessionID]; ok {
		return conflicts[conflictID]
	}
	return nil
}

// GetSessionConflicts gets all conflicts for a session
func (r *ConflictResolver) GetSessionConflicts(sessionID string) []*models.ConflictInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*models.ConflictInfo
	if conflicts, ok := r.conflicts[sessionID]; ok {
		for _, conflict := range conflicts {
			result = append(result, conflict)
		}
	}
	return result
}

// GetUnresolvedConflicts gets all unresolved conflicts for a session
func (r *ConflictResolver) GetUnresolvedConflicts(sessionID string) []*models.ConflictInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*models.ConflictInfo
	if conflicts, ok := r.conflicts[sessionID]; ok {
		for _, conflict := range conflicts {
			if conflict.Resolution == nil {
				result = append(result, conflict)
			}
		}
	}
	return result
}


// ResolveConflict resolves a conflict using the specified strategy
func (r *ConflictResolver) ResolveConflict(sessionID, conflictID, strategy, resolvedBy string, manualContent string) (*models.ConflictResolution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conflicts, ok := r.conflicts[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	conflict, ok := conflicts[conflictID]
	if !ok {
		return nil, ErrConflict
	}

	var resolution *models.ConflictResolution
	var err error

	switch strategy {
	case "accept_local":
		resolution = &models.ConflictResolution{
			ID:            uuid.New().String(),
			ConflictID:    conflictID,
			ResolvedBy:    resolvedBy,
			Strategy:      strategy,
			MergedContent: conflict.LocalOp.Content,
			ResolvedAt:    time.Now().UnixMilli(),
		}
	case "accept_remote":
		resolution = &models.ConflictResolution{
			ID:            uuid.New().String(),
			ConflictID:    conflictID,
			ResolvedBy:    resolvedBy,
			Strategy:      strategy,
			MergedContent: conflict.RemoteOp.Content,
			ResolvedAt:    time.Now().UnixMilli(),
		}
	case "manual":
		resolution = &models.ConflictResolution{
			ID:            uuid.New().String(),
			ConflictID:    conflictID,
			ResolvedBy:    resolvedBy,
			Strategy:      strategy,
			MergedContent: manualContent,
			ResolvedAt:    time.Now().UnixMilli(),
		}
	case "three_way_merge":
		if strat, ok := r.strategies[strategy]; ok {
			resolution, err = strat.Resolve(conflict)
			if err != nil {
				return nil, err
			}
			resolution.ResolvedBy = resolvedBy
		}
	default:
		return nil, ErrInvalidOperation
	}

	conflict.Resolution = resolution

	r.logger.Info("Conflict resolved",
		zap.String("conflictId", conflictID),
		zap.String("sessionId", sessionID),
		zap.String("strategy", strategy),
		zap.String("resolvedBy", resolvedBy),
	)

	return resolution, nil
}

// RemoveConflict removes a conflict
func (r *ConflictResolver) RemoveConflict(sessionID, conflictID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conflicts, ok := r.conflicts[sessionID]; ok {
		delete(conflicts, conflictID)
		if len(conflicts) == 0 {
			delete(r.conflicts, sessionID)
		}
	}
}

// CleanupSession removes all conflicts for a session
func (r *ConflictResolver) CleanupSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.conflicts, sessionID)
}

// LastWriteWinsStrategy resolves conflicts by accepting the most recent operation
type LastWriteWinsStrategy struct{}

func (s *LastWriteWinsStrategy) Resolve(conflict *models.ConflictInfo) (*models.ConflictResolution, error) {
	var content string
	if conflict.LocalOp.Timestamp > conflict.RemoteOp.Timestamp {
		content = conflict.LocalOp.Content
	} else {
		content = conflict.RemoteOp.Content
	}

	return &models.ConflictResolution{
		ID:            uuid.New().String(),
		ConflictID:    conflict.ID,
		Strategy:      "last_write_wins",
		MergedContent: content,
		ResolvedAt:    time.Now().UnixMilli(),
	}, nil
}

// FirstWriteWinsStrategy resolves conflicts by accepting the earliest operation
type FirstWriteWinsStrategy struct{}

func (s *FirstWriteWinsStrategy) Resolve(conflict *models.ConflictInfo) (*models.ConflictResolution, error) {
	var content string
	if conflict.LocalOp.Timestamp < conflict.RemoteOp.Timestamp {
		content = conflict.LocalOp.Content
	} else {
		content = conflict.RemoteOp.Content
	}

	return &models.ConflictResolution{
		ID:            uuid.New().String(),
		ConflictID:    conflict.ID,
		Strategy:      "first_write_wins",
		MergedContent: content,
		ResolvedAt:    time.Now().UnixMilli(),
	}, nil
}

// ThreeWayMergeStrategy implements three-way merge for conflict resolution
type ThreeWayMergeStrategy struct{}

func (s *ThreeWayMergeStrategy) Resolve(conflict *models.ConflictInfo) (*models.ConflictResolution, error) {
	// Simple three-way merge implementation
	// In a real implementation, this would use a proper diff3 algorithm
	
	localContent := conflict.LocalOp.Content
	remoteContent := conflict.RemoteOp.Content

	// If contents are identical, no conflict
	if localContent == remoteContent {
		return &models.ConflictResolution{
			ID:            uuid.New().String(),
			ConflictID:    conflict.ID,
			Strategy:      "three_way_merge",
			MergedContent: localContent,
			ResolvedAt:    time.Now().UnixMilli(),
		}, nil
	}

	// Simple merge: concatenate with conflict markers
	merged := s.mergeWithMarkers(localContent, remoteContent)

	return &models.ConflictResolution{
		ID:            uuid.New().String(),
		ConflictID:    conflict.ID,
		Strategy:      "three_way_merge",
		MergedContent: merged,
		ResolvedAt:    time.Now().UnixMilli(),
	}, nil
}

// mergeWithMarkers creates a merge with conflict markers
func (s *ThreeWayMergeStrategy) mergeWithMarkers(local, remote string) string {
	var builder strings.Builder
	builder.WriteString("<<<<<<< LOCAL\n")
	builder.WriteString(local)
	builder.WriteString("\n=======\n")
	builder.WriteString(remote)
	builder.WriteString("\n>>>>>>> REMOTE")
	return builder.String()
}
