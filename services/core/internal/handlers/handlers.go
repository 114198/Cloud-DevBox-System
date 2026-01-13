// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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

// Auth handlers (placeholders)
func Register(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func Login(c *gin.Context)        { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func RefreshToken(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

// User handlers (placeholders)
func GetCurrentUser(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func UpdateCurrentUser(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

// Environment handlers (placeholders)
func ListEnvironments(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"data": []interface{}{}}) }
func CreateEnvironment(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func GetEnvironment(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func UpdateEnvironment(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func DeleteEnvironment(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func StartEnvironment(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func StopEnvironment(c *gin.Context)   { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

// Template handlers (placeholders)
func ListTemplates(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": []interface{}{}}) }
func GetTemplate(c *gin.Context)   { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }

// Project handlers (placeholders)
func ListProjects(c *gin.Context)   { c.JSON(http.StatusOK, gin.H{"data": []interface{}{}}) }
func CreateProject(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func GetProject(c *gin.Context)     { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func UpdateProject(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func DeleteProject(c *gin.Context)  { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
