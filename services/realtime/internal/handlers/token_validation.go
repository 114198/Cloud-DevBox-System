// Package handlers provides HTTP/WebSocket request handlers for the realtime service.
package handlers

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func extractAccessToken(c *gin.Context) string {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[len("bearer "):])
		}
	}
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Access-Token"))
	}
	return token
}

func validateAccessToken(token string) error {
	required := strings.EqualFold(os.Getenv("REALTIME_REQUIRE_TOKEN"), "true")
	expectedToken := os.Getenv("REALTIME_API_TOKEN")
	if expectedToken != "" {
		required = true
	}
	if !required {
		return nil
	}
	if token == "" {
		return errors.New("token is required")
	}
	if expectedToken != "" && token != expectedToken {
		return errors.New("invalid token")
	}
	return nil
}
