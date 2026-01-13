// Package services provides business logic for the realtime collaboration service.
package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CRDTEngine implements a CRDT-based collaborative editing engine
// using a simplified RGA (Replicated Growable Array) algorithm
type CRDTEngine struct {
	documents map[string]*CRDTDocument
	mu        sync.RWMutex
	logger    *zap.Logger
}

// CRDTDocument represents a collaborative document with CRDT state
type CRDTDocument struct {
	FilePath    string
	SessionID   string
	Content     []CRDTChar
	Version     int64
	LastUpdated time.Time
	mu          sync.RWMutex
}

// CRDTChar represents a character in the CRDT document
type CRDTChar struct {
	ID        CRDTCharID `json:"id"`
	Value     rune       `json:"value"`
	Deleted   bool       `json:"deleted"`
	Timestamp int64      `json:"timestamp"`
}

// CRDTCharID uniquely identifies a character in the CRDT
type CRDTCharID struct {
	SiteID    string `json:"siteId"`
	Clock     int64  `json:"clock"`
	Offset    int    `json:"offset"`
}

// CRDTOperation represents an operation in the CRDT
type CRDTOperation struct {
	ID        string            `json:"id"`
	Type      models.OperationType `json:"type"`
	CharID    CRDTCharID        `json:"charId"`
	Value     rune              `json:"value,omitempty"`
	AfterID   *CRDTCharID       `json:"afterId,omitempty"`
	SiteID    string            `json:"siteId"`
	Clock     int64             `json:"clock"`
	Timestamp int64             `json:"timestamp"`
}

// NewCRDTEngine creates a new CRDT engine
func NewCRDTEngine(logger *zap.Logger) *CRDTEngine {
	return &CRDTEngine{
		documents: make(map[string]*CRDTDocument),
		logger:    logger,
	}
}

// GetOrCreateDocument gets or creates a document for the given session and file
func (e *CRDTEngine) GetOrCreateDocument(sessionID, filePath string) *CRDTDocument {
	key := sessionID + ":" + filePath
	
	e.mu.Lock()
	defer e.mu.Unlock()

	if doc, ok := e.documents[key]; ok {
		return doc
	}

	doc := &CRDTDocument{
		FilePath:    filePath,
		SessionID:   sessionID,
		Content:     make([]CRDTChar, 0),
		Version:     0,
		LastUpdated: time.Now(),
	}
	e.documents[key] = doc
	return doc
}

// GetDocument gets a document by session and file path
func (e *CRDTEngine) GetDocument(sessionID, filePath string) (*CRDTDocument, bool) {
	key := sessionID + ":" + filePath
	
	e.mu.RLock()
	defer e.mu.RUnlock()

	doc, ok := e.documents[key]
	return doc, ok
}

// RemoveDocument removes a document from the engine
func (e *CRDTEngine) RemoveDocument(sessionID, filePath string) {
	key := sessionID + ":" + filePath
	
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.documents, key)
}

// RemoveSessionDocuments removes all documents for a session
func (e *CRDTEngine) RemoveSessionDocuments(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for key := range e.documents {
		if len(key) > len(sessionID) && key[:len(sessionID)+1] == sessionID+":" {
			delete(e.documents, key)
		}
	}
}


// InitializeDocument initializes a document with content
func (d *CRDTDocument) InitializeDocument(content string, siteID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.Content = make([]CRDTChar, 0, len(content))
	clock := int64(0)

	for i, r := range content {
		d.Content = append(d.Content, CRDTChar{
			ID: CRDTCharID{
				SiteID: siteID,
				Clock:  clock,
				Offset: i,
			},
			Value:     r,
			Deleted:   false,
			Timestamp: time.Now().UnixMilli(),
		})
		clock++
	}
	d.Version++
	d.LastUpdated = time.Now()
}

// ApplyOperation applies a CRDT operation to the document
func (d *CRDTDocument) ApplyOperation(op *CRDTOperation) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch op.Type {
	case models.OpInsert:
		return d.applyInsert(op)
	case models.OpDelete:
		return d.applyDelete(op)
	default:
		return ErrInvalidOperation
	}
}

// applyInsert applies an insert operation
func (d *CRDTDocument) applyInsert(op *CRDTOperation) error {
	newChar := CRDTChar{
		ID:        op.CharID,
		Value:     op.Value,
		Deleted:   false,
		Timestamp: op.Timestamp,
	}

	// Find insertion position
	insertPos := 0
	if op.AfterID != nil {
		for i, char := range d.Content {
			if char.ID == *op.AfterID {
				insertPos = i + 1
				break
			}
		}
	}

	// Handle concurrent inserts at the same position using site ID ordering
	for insertPos < len(d.Content) {
		existingChar := d.Content[insertPos]
		// If existing char has same afterID, compare by site ID and clock
		if op.AfterID != nil && existingChar.ID.SiteID > op.CharID.SiteID {
			break
		}
		if op.AfterID != nil && existingChar.ID.SiteID == op.CharID.SiteID && existingChar.ID.Clock > op.CharID.Clock {
			break
		}
		insertPos++
	}

	// Insert the character
	d.Content = append(d.Content[:insertPos], append([]CRDTChar{newChar}, d.Content[insertPos:]...)...)
	d.Version++
	d.LastUpdated = time.Now()

	return nil
}

// applyDelete applies a delete operation (tombstone)
func (d *CRDTDocument) applyDelete(op *CRDTOperation) error {
	for i := range d.Content {
		if d.Content[i].ID == op.CharID {
			d.Content[i].Deleted = true
			d.Version++
			d.LastUpdated = time.Now()
			return nil
		}
	}
	// Character not found - might have been deleted already, which is fine
	return nil
}

// GetContent returns the visible content of the document
func (d *CRDTDocument) GetContent() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []rune
	for _, char := range d.Content {
		if !char.Deleted {
			result = append(result, char.Value)
		}
	}
	return string(result)
}

// GetState returns the current document state
func (d *CRDTDocument) GetState() *models.DocumentState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return &models.DocumentState{
		FilePath:    d.FilePath,
		Content:     d.GetContentUnsafe(),
		Version:     d.Version,
		LastUpdated: d.LastUpdated.UnixMilli(),
	}
}

// GetContentUnsafe returns content without locking (caller must hold lock)
func (d *CRDTDocument) GetContentUnsafe() string {
	var result []rune
	for _, char := range d.Content {
		if !char.Deleted {
			result = append(result, char.Value)
		}
	}
	return string(result)
}

// GetVersion returns the current version
func (d *CRDTDocument) GetVersion() int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Version
}


// CreateInsertOperation creates an insert operation
func (e *CRDTEngine) CreateInsertOperation(sessionID, filePath, siteID string, position int, value rune, clock int64) *CRDTOperation {
	doc := e.GetOrCreateDocument(sessionID, filePath)
	doc.mu.RLock()
	defer doc.mu.RUnlock()

	var afterID *CRDTCharID
	if position > 0 && position <= len(doc.Content) {
		// Find the visible character at position-1
		visiblePos := 0
		for i := range doc.Content {
			if !doc.Content[i].Deleted {
				if visiblePos == position-1 {
					id := doc.Content[i].ID
					afterID = &id
					break
				}
				visiblePos++
			}
		}
	}

	return &CRDTOperation{
		ID:   uuid.New().String(),
		Type: models.OpInsert,
		CharID: CRDTCharID{
			SiteID: siteID,
			Clock:  clock,
			Offset: position,
		},
		Value:     value,
		AfterID:   afterID,
		SiteID:    siteID,
		Clock:     clock,
		Timestamp: time.Now().UnixMilli(),
	}
}

// CreateDeleteOperation creates a delete operation
func (e *CRDTEngine) CreateDeleteOperation(sessionID, filePath, siteID string, position int, clock int64) *CRDTOperation {
	doc := e.GetOrCreateDocument(sessionID, filePath)
	doc.mu.RLock()
	defer doc.mu.RUnlock()

	// Find the character at the visible position
	visiblePos := 0
	for i := range doc.Content {
		if !doc.Content[i].Deleted {
			if visiblePos == position {
				return &CRDTOperation{
					ID:        uuid.New().String(),
					Type:      models.OpDelete,
					CharID:    doc.Content[i].ID,
					SiteID:    siteID,
					Clock:     clock,
					Timestamp: time.Now().UnixMilli(),
				}
			}
			visiblePos++
		}
	}

	return nil
}

// TransformOperation transforms an operation against another concurrent operation
// This implements the Operational Transformation for conflict resolution
func (e *CRDTEngine) TransformOperation(op1, op2 *CRDTOperation) (*CRDTOperation, *CRDTOperation) {
	// In CRDT, operations are commutative, so we don't need traditional OT
	// The CRDT structure handles concurrent operations automatically
	// This method is provided for compatibility with OT-based systems
	return op1, op2
}

// SerializeOperation serializes an operation to JSON
func (e *CRDTEngine) SerializeOperation(op *CRDTOperation) ([]byte, error) {
	return json.Marshal(op)
}

// DeserializeOperation deserializes an operation from JSON
func (e *CRDTEngine) DeserializeOperation(data []byte) (*CRDTOperation, error) {
	var op CRDTOperation
	if err := json.Unmarshal(data, &op); err != nil {
		return nil, err
	}
	return &op, nil
}

// GetDocumentCount returns the number of documents in the engine
func (e *CRDTEngine) GetDocumentCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.documents)
}

// GetSessionDocuments returns all documents for a session
func (e *CRDTEngine) GetSessionDocuments(sessionID string) []*CRDTDocument {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var docs []*CRDTDocument
	prefix := sessionID + ":"
	for key, doc := range e.documents {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			docs = append(docs, doc)
		}
	}
	return docs
}
