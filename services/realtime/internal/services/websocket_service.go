// Package services provides business logic for the realtime collaboration service.
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

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB

	// Buffer size for client channels
	sendBufferSize = 256
)

// Client represents a WebSocket client connection
type Client struct {
	ID            string
	UserID        string
	Username      string
	DisplayName   string
	AvatarURL     string
	SessionID     string
	EnvironmentID string
	Color         string
	Conn          *websocket.Conn
	Send          chan []byte
	Hub           *Hub
	LastPing      time.Time
	JoinedAt      time.Time
	mu            sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by session ID
	sessions map[string]map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to all clients in a session
	broadcast chan *BroadcastMessage

	// Logger
	logger *zap.Logger

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Collaboration service reference
	collabService *CollaborationService
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	SessionID string
	Message   []byte
	ExcludeID string // Client ID to exclude from broadcast
}

// NewHub creates a new Hub instance
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		sessions:   make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		logger:     logger,
	}
}

// SetCollaborationService sets the collaboration service reference
func (h *Hub) SetCollaborationService(cs *CollaborationService) {
	h.collabService = cs
}


// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Hub shutting down")
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.cleanupInactiveClients()
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.sessions[client.SessionID]; !ok {
		h.sessions[client.SessionID] = make(map[string]*Client)
	}
	h.sessions[client.SessionID][client.ID] = client

	h.logger.Info("Client registered",
		zap.String("clientId", client.ID),
		zap.String("userId", client.UserID),
		zap.String("sessionId", client.SessionID),
	)

	// Notify other clients about the new participant
	h.notifyUserJoined(client)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.sessions[client.SessionID]; ok {
		if _, exists := clients[client.ID]; exists {
			delete(clients, client.ID)
			close(client.Send)

			h.logger.Info("Client unregistered",
				zap.String("clientId", client.ID),
				zap.String("userId", client.UserID),
				zap.String("sessionId", client.SessionID),
			)

			// Notify other clients about the departure
			h.notifyUserLeft(client)

			// Clean up empty sessions
			if len(clients) == 0 {
				delete(h.sessions, client.SessionID)
			}
		}
	}
}

// broadcastMessage sends a message to all clients in a session
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	h.mu.RLock()
	clients, ok := h.sessions[msg.SessionID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for id, client := range clients {
		if msg.ExcludeID != "" && id == msg.ExcludeID {
			continue
		}
		select {
		case client.Send <- msg.Message:
		default:
			h.logger.Warn("Client send buffer full, dropping message",
				zap.String("clientId", client.ID),
			)
		}
	}
}

// notifyUserJoined notifies all clients in a session about a new user
func (h *Hub) notifyUserJoined(client *Client) {
	msg := models.WebSocketMessage{
		Type:          models.MessageTypeJoin,
		SessionID:     client.SessionID,
		EnvironmentID: client.EnvironmentID,
		UserID:        client.UserID,
		Timestamp:     time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"userId":      client.UserID,
			"username":    client.Username,
			"displayName": client.DisplayName,
			"avatarUrl":   client.AvatarURL,
			"color":       client.Color,
		},
	}

	data, _ := json.Marshal(msg)
	h.broadcast <- &BroadcastMessage{
		SessionID: client.SessionID,
		Message:   data,
		ExcludeID: client.ID,
	}
}

// notifyUserLeft notifies all clients in a session about a user leaving
func (h *Hub) notifyUserLeft(client *Client) {
	msg := models.WebSocketMessage{
		Type:          models.MessageTypeLeave,
		SessionID:     client.SessionID,
		EnvironmentID: client.EnvironmentID,
		UserID:        client.UserID,
		Timestamp:     time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"userId":   client.UserID,
			"username": client.Username,
		},
	}

	data, _ := json.Marshal(msg)
	h.broadcast <- &BroadcastMessage{
		SessionID: client.SessionID,
		Message:   data,
	}
}


// cleanupInactiveClients removes clients that haven't responded to pings
func (h *Hub) cleanupInactiveClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	timeout := time.Now().Add(-pongWait)
	for sessionID, clients := range h.sessions {
		for clientID, client := range clients {
			client.mu.RLock()
			lastPing := client.LastPing
			client.mu.RUnlock()

			if lastPing.Before(timeout) {
				h.logger.Warn("Removing inactive client",
					zap.String("clientId", clientID),
					zap.String("sessionId", sessionID),
				)
				delete(clients, clientID)
				close(client.Send)
			}
		}
		if len(clients) == 0 {
			delete(h.sessions, sessionID)
		}
	}
}

// GetSessionClients returns all clients in a session
func (h *Hub) GetSessionClients(sessionID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.sessions[sessionID]
	if !ok {
		return nil
	}

	result := make([]*Client, 0, len(clients))
	for _, client := range clients {
		result = append(result, client)
	}
	return result
}

// GetSessionParticipants returns participant info for all clients in a session
func (h *Hub) GetSessionParticipants(sessionID string) []models.SessionParticipant {
	clients := h.GetSessionClients(sessionID)
	participants := make([]models.SessionParticipant, 0, len(clients))

	for _, client := range clients {
		participants = append(participants, models.SessionParticipant{
			UserID:      uuid.MustParse(client.UserID),
			Username:    client.Username,
			DisplayName: client.DisplayName,
			AvatarURL:   client.AvatarURL,
			Color:       client.Color,
			JoinedAt:    client.JoinedAt,
			IsOnline:    true,
		})
	}
	return participants
}

// GetClientCount returns the number of clients in a session
func (h *Hub) GetClientCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.sessions[sessionID]; ok {
		return len(clients)
	}
	return 0
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, sessionID string, message []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.sessions[sessionID]; ok {
		if client, exists := clients[clientID]; exists {
			select {
			case client.Send <- message:
				return nil
			default:
				return ErrClientBufferFull
			}
		}
	}
	return ErrClientNotFound
}

// Broadcast sends a message to all clients in a session
func (h *Hub) Broadcast(sessionID string, message []byte, excludeClientID string) {
	h.broadcast <- &BroadcastMessage{
		SessionID: sessionID,
		Message:   message,
		ExcludeID: excludeClientID,
	}
}

// NewClient creates a new client instance
func NewClient(hub *Hub, conn *websocket.Conn, userID, username, displayName, avatarURL, sessionID, environmentID string, colorIndex int) *Client {
	return &Client{
		ID:            uuid.New().String(),
		UserID:        userID,
		Username:      username,
		DisplayName:   displayName,
		AvatarURL:     avatarURL,
		SessionID:     sessionID,
		EnvironmentID: environmentID,
		Color:         models.GetUserColor(colorIndex),
		Conn:          conn,
		Send:          make(chan []byte, sendBufferSize),
		Hub:           hub,
		LastPing:      time.Now(),
		JoinedAt:      time.Now(),
	}
}


// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.LastPing = time.Now()
		c.mu.Unlock()
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Error("WebSocket read error",
					zap.String("clientId", c.ID),
					zap.Error(err),
				)
			}
			break
		}

		// Parse and handle the message
		var msg models.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Hub.logger.Error("Failed to parse message",
				zap.String("clientId", c.ID),
				zap.Error(err),
			)
			continue
		}

		// Handle the message
		c.handleMessage(&msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(msg *models.WebSocketMessage) {
	msg.UserID = c.UserID
	msg.SessionID = c.SessionID
	msg.EnvironmentID = c.EnvironmentID

	switch msg.Type {
	case models.MessageTypePing:
		c.handlePing(msg)
	case models.MessageTypeEdit:
		c.handleEdit(msg)
	case models.MessageTypeCursor:
		c.handleCursor(msg)
	case models.MessageTypeSelection:
		c.handleSelection(msg)
	case models.MessageTypeSync:
		c.handleSync(msg)
	default:
		c.Hub.logger.Warn("Unknown message type",
			zap.String("type", string(msg.Type)),
			zap.String("clientId", c.ID),
		)
	}
}

// handlePing handles ping messages
func (c *Client) handlePing(msg *models.WebSocketMessage) {
	response := models.WebSocketMessage{
		Type:      models.MessageTypePong,
		SessionID: c.SessionID,
		UserID:    c.UserID,
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(response)
	c.Send <- data
}

// handleEdit handles edit operation messages
func (c *Client) handleEdit(msg *models.WebSocketMessage) {
	if c.Hub.collabService != nil {
		c.Hub.collabService.HandleEditOperation(c, msg)
	}
}

// handleCursor handles cursor position updates
func (c *Client) handleCursor(msg *models.WebSocketMessage) {
	if c.Hub.collabService != nil {
		c.Hub.collabService.HandleCursorUpdate(c, msg)
	}
}

// handleSelection handles selection updates
func (c *Client) handleSelection(msg *models.WebSocketMessage) {
	if c.Hub.collabService != nil {
		c.Hub.collabService.HandleSelectionUpdate(c, msg)
	}
}

// handleSync handles sync requests
func (c *Client) handleSync(msg *models.WebSocketMessage) {
	if c.Hub.collabService != nil {
		c.Hub.collabService.HandleSyncRequest(c, msg)
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg *models.WebSocketMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	select {
	case c.Send <- data:
		return nil
	default:
		return ErrClientBufferFull
	}
}
