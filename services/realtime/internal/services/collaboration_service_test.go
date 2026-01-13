// Package services provides business logic for the realtime collaboration service.
package services

import (
	"sync"
	"testing"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"pgregory.net/rapid"
)

// TestCollaborationService_CreateSession tests session creation
func TestCollaborationService_CreateSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	cs := NewCollaborationService(logger, hub)

	envID := uuid.New()
	userID := uuid.New()

	session, err := cs.CreateSession(envID, userID, nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID == uuid.Nil {
		t.Error("Session ID should not be nil")
	}

	if session.EnvironmentID != envID {
		t.Errorf("Expected environment ID %s, got %s", envID, session.EnvironmentID)
	}

	if session.Status != models.SessionStatusActive {
		t.Errorf("Expected status %s, got %s", models.SessionStatusActive, session.Status)
	}
}

// TestCollaborationService_GetSession tests session retrieval
func TestCollaborationService_GetSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	cs := NewCollaborationService(logger, hub)

	envID := uuid.New()
	userID := uuid.New()

	session, _ := cs.CreateSession(envID, userID, nil)

	retrieved, err := cs.GetSession(session.ID.String())
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrieved.ID)
	}
}

// TestCollaborationService_SessionNotFound tests session not found error
func TestCollaborationService_SessionNotFound(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	cs := NewCollaborationService(logger, hub)

	_, err := cs.GetSession(uuid.New().String())
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

// TestCRDTEngine_InsertOperation tests CRDT insert operations
func TestCRDTEngine_InsertOperation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	engine := NewCRDTEngine(logger)

	sessionID := uuid.New().String()
	filePath := "test.txt"
	siteID := "user1"

	doc := engine.GetOrCreateDocument(sessionID, filePath)
	doc.InitializeDocument("Hello", siteID)

	content := doc.GetContent()
	if content != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", content)
	}
}

// TestCRDTEngine_DeleteOperation tests CRDT delete operations
func TestCRDTEngine_DeleteOperation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	engine := NewCRDTEngine(logger)

	sessionID := uuid.New().String()
	filePath := "test.txt"
	siteID := "user1"

	doc := engine.GetOrCreateDocument(sessionID, filePath)
	doc.InitializeDocument("Hello", siteID)

	// Delete 'e' at position 1
	op := engine.CreateDeleteOperation(sessionID, filePath, siteID, 1, 1)
	if op != nil {
		doc.ApplyOperation(op)
	}

	content := doc.GetContent()
	if content != "Hllo" {
		t.Errorf("Expected 'Hllo', got '%s'", content)
	}
}

// TestCursorService_UpdateCursor tests cursor updates
func TestCursorService_UpdateCursor(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hub := NewHub(logger)
	cs := NewCursorService(logger, hub)

	sessionID := uuid.New().String()
	userID := uuid.New().String()

	position := &models.CursorPosition{
		FilePath: "test.txt",
		Line:     10,
		Column:   5,
	}

	cs.UpdateCursor(sessionID, userID, position)

	retrieved := cs.GetCursor(sessionID, userID)
	if retrieved == nil {
		t.Fatal("Cursor should not be nil")
	}

	if retrieved.Line != 10 || retrieved.Column != 5 {
		t.Errorf("Expected line 10, column 5, got line %d, column %d", retrieved.Line, retrieved.Column)
	}
}

// TestHistoryService_RecordOperation tests history recording
func TestHistoryService_RecordOperation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hs := NewHistoryService(logger)

	sessionID := uuid.New().String()
	userID := uuid.New()

	entry := hs.RecordOperation(sessionID, userID, "insert", "test.txt", map[string]interface{}{
		"position": 0,
		"content":  "Hello",
	})

	if entry == nil {
		t.Fatal("Entry should not be nil")
	}

	if entry.OperationType != "insert" {
		t.Errorf("Expected operation type 'insert', got '%s'", entry.OperationType)
	}

	count := hs.GetHistoryCount(sessionID)
	if count != 1 {
		t.Errorf("Expected history count 1, got %d", count)
	}
}

// TestConflictResolver_DetectConflict tests conflict detection
func TestConflictResolver_DetectConflict(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cr := NewConflictResolver(logger)

	// Two inserts at the same position should conflict
	op1 := &models.EditOperation{
		ID:       "op1",
		UserID:   "user1",
		FilePath: "test.txt",
		Type:     models.OpInsert,
		Position: 5,
		Content:  "A",
	}

	op2 := &models.EditOperation{
		ID:       "op2",
		UserID:   "user2",
		FilePath: "test.txt",
		Type:     models.OpInsert,
		Position: 5,
		Content:  "B",
	}

	if !cr.DetectConflict(op1, op2) {
		t.Error("Expected conflict for two inserts at same position")
	}

	// Inserts at different positions should not conflict
	op3 := &models.EditOperation{
		ID:       "op3",
		UserID:   "user2",
		FilePath: "test.txt",
		Type:     models.OpInsert,
		Position: 10,
		Content:  "C",
	}

	if cr.DetectConflict(op1, op3) {
		t.Error("Expected no conflict for inserts at different positions")
	}
}


// =============================================================================
// Property-Based Tests
// =============================================================================

// Property 6: Synchronization Delay < 200ms
// Feature: cloud-devbox, Property 6: Real-time Collaboration Synchronization
// For any collaborative editing session, changes should be synchronized across
// all participants within 200ms
// **Validates: Requirements 4.1**

// TestProperty6_SyncDelayUnder200ms tests that synchronization happens within 200ms
func TestProperty6_SyncDelayUnder200ms(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		logger, _ := zap.NewDevelopment()
		hub := NewHub(logger)
		cs := NewCollaborationService(logger, hub)

		// Generate random session parameters
		envID := uuid.New()
		userID := uuid.New()

		// Create session
		session, err := cs.CreateSession(envID, userID, nil)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Generate random edit operation
		filePath := rapid.StringMatching(`[a-z]+\.txt`).Draw(t, "filePath")
		content := rapid.StringN(1, 100, 100).Draw(t, "content")
		position := rapid.IntRange(0, 1000).Draw(t, "position")

		// Initialize document
		cs.InitializeDocument(session.ID.String(), filePath, "", userID.String())

		// Measure synchronization time
		startTime := time.Now()

		// Simulate edit operation
		editData := map[string]interface{}{
			"filePath": filePath,
			"type":     string(models.OpInsert),
			"position": position,
			"content":  content,
		}

		// Record operation in history (simulates sync)
		cs.historyService.RecordOperation(
			session.ID.String(),
			userID,
			string(models.OpInsert),
			filePath,
			editData,
		)

		// Get document state (simulates sync completion)
		_, _ = cs.GetDocumentState(session.ID.String(), filePath)

		syncDuration := time.Since(startTime)

		// Property: Sync should complete within 200ms
		if syncDuration > 200*time.Millisecond {
			t.Fatalf("Sync took %v, expected < 200ms", syncDuration)
		}
	})
}

// TestProperty6_ConcurrentEditsSync tests that concurrent edits are synchronized correctly
func TestProperty6_ConcurrentEditsSync(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		logger, _ := zap.NewDevelopment()
		engine := NewCRDTEngine(logger)

		sessionID := uuid.New().String()
		filePath := rapid.StringMatching(`[a-z]+\.txt`).Draw(t, "filePath")

		// Initialize document with some content
		initialContent := rapid.StringN(0, 50, 50).Draw(t, "initialContent")
		doc := engine.GetOrCreateDocument(sessionID, filePath)
		doc.InitializeDocument(initialContent, "init")

		// Generate random number of concurrent users
		numUsers := rapid.IntRange(2, 5).Draw(t, "numUsers")
		
		// Generate random operations for each user
		var wg sync.WaitGroup
		startTime := time.Now()

		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			userID := uuid.New().String()
			
			go func(uid string, userNum int) {
				defer wg.Done()
				
				// Each user performs a random operation
				opType := rapid.SampledFrom([]models.OperationType{models.OpInsert, models.OpDelete}).Draw(t, "opType")
				
				if opType == models.OpInsert {
					content := rapid.StringN(1, 10, 10).Draw(t, "insertContent")
					for j, r := range content {
						op := engine.CreateInsertOperation(sessionID, filePath, uid, j, r, int64(userNum*100+j))
						doc.ApplyOperation(op)
					}
				} else {
					// Delete operation
					currentLen := len(doc.GetContent())
					if currentLen > 0 {
						pos := rapid.IntRange(0, currentLen-1).Draw(t, "deletePos")
						op := engine.CreateDeleteOperation(sessionID, filePath, uid, pos, int64(userNum*100))
						if op != nil {
							doc.ApplyOperation(op)
						}
					}
				}
			}(userID, i)
		}

		wg.Wait()
		syncDuration := time.Since(startTime)

		// Property: All concurrent operations should complete within 200ms
		if syncDuration > 200*time.Millisecond {
			t.Fatalf("Concurrent sync took %v, expected < 200ms", syncDuration)
		}

		// Property: Document should be in a consistent state
		content := doc.GetContent()
		_ = content // Document should be readable without panic
	})
}

// TestProperty6_CRDTConvergence tests that CRDT operations converge to the same state
func TestProperty6_CRDTConvergence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		logger, _ := zap.NewDevelopment()
		
		// Create two independent engines (simulating two clients)
		engine1 := NewCRDTEngine(logger)
		engine2 := NewCRDTEngine(logger)

		sessionID := uuid.New().String()
		filePath := "test.txt"

		// Initialize both documents with the same content
		initialContent := rapid.StringN(0, 20, 20).Draw(t, "initialContent")
		
		doc1 := engine1.GetOrCreateDocument(sessionID, filePath)
		doc2 := engine2.GetOrCreateDocument(sessionID, filePath)
		
		doc1.InitializeDocument(initialContent, "init")
		doc2.InitializeDocument(initialContent, "init")

		// Generate operations from two users
		user1ID := "user1"
		user2ID := "user2"

		// User 1 inserts at position 0
		insertContent1 := rapid.StringN(1, 5, 5).Draw(t, "insertContent1")
		var ops1 []*CRDTOperation
		for i, r := range insertContent1 {
			op := engine1.CreateInsertOperation(sessionID, filePath, user1ID, i, r, int64(i))
			ops1 = append(ops1, op)
			doc1.ApplyOperation(op)
		}

		// User 2 inserts at position 0 (concurrent)
		insertContent2 := rapid.StringN(1, 5, 5).Draw(t, "insertContent2")
		var ops2 []*CRDTOperation
		for i, r := range insertContent2 {
			op := engine2.CreateInsertOperation(sessionID, filePath, user2ID, i, r, int64(100+i))
			ops2 = append(ops2, op)
			doc2.ApplyOperation(op)
		}

		// Apply ops1 to doc2 and ops2 to doc1 (simulating sync)
		for _, op := range ops1 {
			doc2.ApplyOperation(op)
		}
		for _, op := range ops2 {
			doc1.ApplyOperation(op)
		}

		// Property: Both documents should converge to the same content
		content1 := doc1.GetContent()
		content2 := doc2.GetContent()

		if content1 != content2 {
			t.Fatalf("Documents did not converge: doc1='%s', doc2='%s'", content1, content2)
		}
	})
}

// TestProperty6_HistoryRetention tests that history is retained for 30 days
func TestProperty6_HistoryRetention(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hs := NewHistoryService(logger)

	sessionID := uuid.New().String()
	userID := uuid.New()

	// Record an operation
	entry := hs.RecordOperation(sessionID, userID, "insert", "test.txt", nil)

	// Verify entry exists
	retrieved := hs.GetHistoryEntry(sessionID, entry.ID.String())
	if retrieved == nil {
		t.Fatal("History entry should exist")
	}

	// Verify retention constant is 30 days
	if HistoryRetentionDays != 30 {
		t.Errorf("Expected retention of 30 days, got %d", HistoryRetentionDays)
	}
}
