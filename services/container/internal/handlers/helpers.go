// Package handlers provides HTTP request handlers for the container service.
package handlers

import "github.com/gin-gonic/gin"

// ErrorResponse represents a common error payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// getUserID extracts the user ID from the request context.
func getUserID(c *gin.Context) string {
	// In production, this would come from JWT token
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}
	return userID
}
