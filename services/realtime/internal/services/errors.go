// Package services provides business logic for the realtime collaboration service.
package services

import "errors"

var (
	// ErrClientNotFound is returned when a client is not found
	ErrClientNotFound = errors.New("client not found")

	// ErrClientBufferFull is returned when a client's send buffer is full
	ErrClientBufferFull = errors.New("client send buffer full")

	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionFull is returned when a session has reached max participants
	ErrSessionFull = errors.New("session is full")

	// ErrUnauthorized is returned when a user is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidOperation is returned when an operation is invalid
	ErrInvalidOperation = errors.New("invalid operation")

	// ErrConflict is returned when there's a conflict in operations
	ErrConflict = errors.New("operation conflict")

	// ErrDocumentNotFound is returned when a document is not found
	ErrDocumentNotFound = errors.New("document not found")

	// ErrVersionMismatch is returned when document versions don't match
	ErrVersionMismatch = errors.New("version mismatch")

	// ErrInvalidMessage is returned when a message is invalid
	ErrInvalidMessage = errors.New("invalid message")
)
