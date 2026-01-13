// Package models provides data models for the realtime collaboration service.
package models

import (
	"time"

	"github.com/google/uuid"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Connection messages
	MessageTypeConnect    MessageType = "connect"
	MessageTypeDisconnect MessageType = "disconnect"
	MessageTypePing       MessageType = "ping"
	MessageTypePong       MessageType = "pong"

	// Collaboration messages
	MessageTypeJoin       MessageType = "join"
	MessageTypeLeave      MessageType = "leave"
	MessageTypeEdit       MessageType = "edit"
	MessageTypeCursor     MessageType = "cursor"
	MessageTypeSelection  MessageType = "selection"
	MessageTypeSync       MessageType = "sync"
	MessageTypeAck        MessageType = "ack"
	MessageTypeError      MessageType = "error"
	MessageTypeUserList   MessageType = "user_list"
	MessageTypeHistory    MessageType = "history"
	MessageTypeConflict   MessageType = "conflict"
	MessageTypeResolution MessageType = "resolution"
)

// ParticipantRole represents the role of a collaboration participant
type ParticipantRole string

const (
	RoleHost        ParticipantRole = "host"
	RoleParticipant ParticipantRole = "participant"
	RoleViewer      ParticipantRole = "viewer"
)

// SessionStatus represents the status of a collaboration session
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusEnded   SessionStatus = "ended"
	SessionStatusExpired SessionStatus = "expired"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type          MessageType    `json:"type"`
	SessionID     string         `json:"sessionId,omitempty"`
	EnvironmentID string         `json:"environmentId,omitempty"`
	UserID        string         `json:"userId,omitempty"`
	Timestamp     int64          `json:"timestamp"`
	Data          interface{}    `json:"data,omitempty"`
	Sequence      int64          `json:"sequence,omitempty"`
	AckID         string         `json:"ackId,omitempty"`
}

// CollaborationSession represents an active collaboration session
type CollaborationSession struct {
	ID            uuid.UUID              `json:"id"`
	EnvironmentID uuid.UUID              `json:"environmentId"`
	CreatedBy     uuid.UUID              `json:"createdBy"`
	Status        SessionStatus          `json:"status"`
	Settings      SessionSettings        `json:"settings"`
	Participants  []SessionParticipant   `json:"participants"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	ExpiresAt     *time.Time             `json:"expiresAt,omitempty"`
}

// SessionSettings represents collaboration session settings
type SessionSettings struct {
	MaxParticipants int  `json:"maxParticipants"`
	AllowEditing    bool `json:"allowEditing"`
	AllowCursors    bool `json:"allowCursors"`
	AutoSave        bool `json:"autoSave"`
	AutoSaveInterval int `json:"autoSaveInterval"` // seconds
}

// DefaultSessionSettings returns default session settings
func DefaultSessionSettings() SessionSettings {
	return SessionSettings{
		MaxParticipants:  10,
		AllowEditing:     true,
		AllowCursors:     true,
		AutoSave:         true,
		AutoSaveInterval: 30,
	}
}


// SessionParticipant represents a participant in a collaboration session
type SessionParticipant struct {
	ID             uuid.UUID       `json:"id"`
	SessionID      uuid.UUID       `json:"sessionId"`
	UserID         uuid.UUID       `json:"userId"`
	Username       string          `json:"username"`
	DisplayName    string          `json:"displayName"`
	AvatarURL      string          `json:"avatarUrl,omitempty"`
	Role           ParticipantRole `json:"role"`
	CursorPosition *CursorPosition `json:"cursorPosition,omitempty"`
	Selection      *Selection      `json:"selection,omitempty"`
	Color          string          `json:"color"`
	JoinedAt       time.Time       `json:"joinedAt"`
	LeftAt         *time.Time      `json:"leftAt,omitempty"`
	IsOnline       bool            `json:"isOnline"`
}

// CursorPosition represents a user's cursor position in a file
type CursorPosition struct {
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Offset   int    `json:"offset,omitempty"`
}

// Selection represents a user's text selection in a file
type Selection struct {
	FilePath   string `json:"filePath"`
	StartLine  int    `json:"startLine"`
	StartCol   int    `json:"startColumn"`
	EndLine    int    `json:"endLine"`
	EndCol     int    `json:"endColumn"`
	StartOffset int   `json:"startOffset,omitempty"`
	EndOffset   int   `json:"endOffset,omitempty"`
}

// JoinRequest represents a request to join a collaboration session
type JoinRequest struct {
	EnvironmentID string `json:"environmentId"`
	UserID        string `json:"userId"`
	Username      string `json:"username"`
	DisplayName   string `json:"displayName"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	Token         string `json:"token"`
}

// JoinResponse represents the response to a join request
type JoinResponse struct {
	SessionID    string               `json:"sessionId"`
	UserID       string               `json:"userId"`
	Color        string               `json:"color"`
	Participants []SessionParticipant `json:"participants"`
	Document     *DocumentState       `json:"document,omitempty"`
}

// DocumentState represents the current state of a collaborative document
type DocumentState struct {
	FilePath    string `json:"filePath"`
	Content     string `json:"content"`
	Version     int64  `json:"version"`
	LastUpdated int64  `json:"lastUpdated"`
}

// EditOperation represents an edit operation in a collaborative document
type EditOperation struct {
	ID        string        `json:"id"`
	UserID    string        `json:"userId"`
	FilePath  string        `json:"filePath"`
	Type      OperationType `json:"type"`
	Position  int           `json:"position"`
	Content   string        `json:"content,omitempty"`
	Length    int           `json:"length,omitempty"`
	Version   int64         `json:"version"`
	Timestamp int64         `json:"timestamp"`
}

// OperationType represents the type of edit operation
type OperationType string

const (
	OpInsert OperationType = "insert"
	OpDelete OperationType = "delete"
	OpRetain OperationType = "retain"
)

// CursorUpdate represents a cursor position update
type CursorUpdate struct {
	UserID   string         `json:"userId"`
	Position CursorPosition `json:"position"`
}

// SelectionUpdate represents a selection update
type SelectionUpdate struct {
	UserID    string    `json:"userId"`
	Selection Selection `json:"selection"`
}


// CollaborationHistory represents a history entry for collaboration operations
type CollaborationHistory struct {
	ID            uuid.UUID     `json:"id"`
	SessionID     uuid.UUID     `json:"sessionId"`
	UserID        uuid.UUID     `json:"userId"`
	OperationType string        `json:"operationType"`
	FilePath      string        `json:"filePath,omitempty"`
	OperationData interface{}   `json:"operationData"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// ConflictInfo represents information about a conflict
type ConflictInfo struct {
	ID           string          `json:"id"`
	FilePath     string          `json:"filePath"`
	BaseVersion  int64           `json:"baseVersion"`
	LocalOp      *EditOperation  `json:"localOp"`
	RemoteOp     *EditOperation  `json:"remoteOp"`
	Resolution   *ConflictResolution `json:"resolution,omitempty"`
	CreatedAt    int64           `json:"createdAt"`
}

// ConflictResolution represents a resolution to a conflict
type ConflictResolution struct {
	ID           string        `json:"id"`
	ConflictID   string        `json:"conflictId"`
	ResolvedBy   string        `json:"resolvedBy"`
	Strategy     string        `json:"strategy"` // "accept_local", "accept_remote", "merge", "manual"
	MergedContent string       `json:"mergedContent,omitempty"`
	ResolvedAt   int64         `json:"resolvedAt"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// UserColors provides a list of colors for user identification
var UserColors = []string{
	"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
	"#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
	"#BB8FCE", "#85C1E9", "#F8B500", "#00CED1",
}

// GetUserColor returns a color for a user based on their index
func GetUserColor(index int) string {
	return UserColors[index%len(UserColors)]
}
