// Package handlers provides HTTP/WebSocket request handlers for the realtime service.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthCheck handles health check requests
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "0.1.0",
	})
}

// CollaborationWebSocket handles real-time collaboration connections
func CollaborationWebSocket(c *gin.Context) {
	environmentID := c.Param("environmentId")
	token := extractAccessToken(c)

	if err := validateAccessToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer conn.Close()

	// Send welcome message
	conn.WriteJSON(gin.H{
		"type":          "connected",
		"environmentId": environmentID,
	})

	// Handle messages
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if err := conn.WriteMessage(messageType, payload); err != nil {
			break
		}
	}
}

// TerminalWebSocket handles terminal connections
func TerminalWebSocket(c *gin.Context) {
	environmentID := c.Param("environmentId")
	token := extractAccessToken(c)

	if err := validateAccessToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer conn.Close()

	// Send welcome message
	conn.WriteJSON(gin.H{
		"type":          "connected",
		"environmentId": environmentID,
	})

	// Handle messages
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if err := conn.WriteMessage(messageType, payload); err != nil {
			break
		}
	}
}
