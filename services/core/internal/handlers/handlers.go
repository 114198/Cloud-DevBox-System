// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
func Register(c *gin.Context) { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func Login(c *gin.Context)    { c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"}) }
func RefreshToken(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// User handlers (placeholders)
func GetCurrentUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
func UpdateCurrentUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// Environment handlers (proxy to container service)
func ListEnvironments(c *gin.Context) {
	proxyToContainer(c, http.MethodGet, "/api/v1/environments", nil)
}

func CreateEnvironment(c *gin.Context) {
	proxyBody(c, http.MethodPost, "/api/v1/environments")
}

func GetEnvironment(c *gin.Context) {
	proxyToContainer(c, http.MethodGet, fmt.Sprintf("/api/v1/environments/%s", c.Param("id")), nil)
}

func UpdateEnvironment(c *gin.Context) {
	proxyBody(c, http.MethodPut, fmt.Sprintf("/api/v1/environments/%s", c.Param("id")))
}

func DeleteEnvironment(c *gin.Context) {
	proxyToContainer(c, http.MethodDelete, fmt.Sprintf("/api/v1/environments/%s", c.Param("id")), nil)
}

func StartEnvironment(c *gin.Context) {
	proxyToContainer(c, http.MethodPost, fmt.Sprintf("/api/v1/environments/%s/start", c.Param("id")), nil)
}

func StopEnvironment(c *gin.Context) {
	proxyToContainer(c, http.MethodPost, fmt.Sprintf("/api/v1/environments/%s/stop", c.Param("id")), nil)
}

func proxyBody(c *gin.Context, method, path string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	proxyToContainer(c, method, path, body)
}

func proxyToContainer(c *gin.Context, method, path string, body []byte) {
	baseURL := os.Getenv("CONTAINER_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}

	target, err := url.JoinPath(baseURL, path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid container service url"})
		return
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, target, reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}
	req.Header.Set("Content-Type", c.GetHeader("Content-Type"))
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	if c.Request.URL.RawQuery != "" {
		req.URL.RawQuery = c.Request.URL.RawQuery
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "container service unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read container response"})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// Template handlers
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Runtime     map[string]interface{} `json:"runtime,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

var (
	templateStore   = make(map[string]*Template)
	templateStoreMu sync.RWMutex
)

func ListTemplates(c *gin.Context) {
	templateStoreMu.RLock()
	defer templateStoreMu.RUnlock()

	templates := make([]*Template, 0, len(templateStore))
	for _, template := range templateStore {
		templates = append(templates, template)
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

func CreateTemplate(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		DisplayName string                 `json:"displayName" binding:"required"`
		Description string                 `json:"description"`
		Category    string                 `json:"category" binding:"required"`
		Version     string                 `json:"version"`
		Runtime     map[string]interface{} `json:"runtime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	template := &Template{
		ID:          uuid.New().String(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Category:    req.Category,
		Version:     req.Version,
		Runtime:     req.Runtime,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if template.Version == "" {
		template.Version = "v1"
	}

	templateStoreMu.Lock()
	templateStore[template.ID] = template
	templateStoreMu.Unlock()

	c.JSON(http.StatusCreated, template)
}

func GetTemplate(c *gin.Context) {
	id := c.Param("id")

	templateStoreMu.RLock()
	template, ok := templateStore[id]
	templateStoreMu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

func UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string                 `json:"name"`
		DisplayName string                 `json:"displayName"`
		Description string                 `json:"description"`
		Category    string                 `json:"category"`
		Version     string                 `json:"version"`
		Runtime     map[string]interface{} `json:"runtime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	templateStoreMu.Lock()
	template, ok := templateStore[id]
	if !ok {
		templateStoreMu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.DisplayName != "" {
		template.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	if req.Version != "" {
		template.Version = req.Version
	}
	if req.Runtime != nil {
		template.Runtime = req.Runtime
	}
	template.UpdatedAt = time.Now()
	templateStoreMu.Unlock()

	c.JSON(http.StatusOK, template)
}

func DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	templateStoreMu.Lock()
	_, ok := templateStore[id]
	if ok {
		delete(templateStore, id)
	}
	templateStoreMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Project handlers
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

var (
	projectStore   = make(map[string]*Project)
	projectStoreMu sync.RWMutex
)

func ListProjects(c *gin.Context) {
	projectStoreMu.RLock()
	defer projectStoreMu.RUnlock()

	projects := make([]*Project, 0, len(projectStore))
	for _, project := range projectStore {
		projects = append(projects, project)
	}
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func CreateProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	project := &Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	projectStoreMu.Lock()
	projectStore[project.ID] = project
	projectStoreMu.Unlock()

	c.JSON(http.StatusCreated, project)
}

func GetProject(c *gin.Context) {
	id := c.Param("id")

	projectStoreMu.RLock()
	project, ok := projectStore[id]
	projectStoreMu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func UpdateProject(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectStoreMu.Lock()
	project, ok := projectStore[id]
	if !ok {
		projectStoreMu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	project.UpdatedAt = time.Now()
	projectStoreMu.Unlock()

	c.JSON(http.StatusOK, project)
}

func DeleteProject(c *gin.Context) {
	id := c.Param("id")

	projectStoreMu.Lock()
	_, ok := projectStore[id]
	if ok {
		delete(projectStore, id)
	}
	projectStoreMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
