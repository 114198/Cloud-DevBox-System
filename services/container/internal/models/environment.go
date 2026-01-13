// Package models defines the data models for the container service.
package models

import (
	"time"
)

// EnvironmentPhase represents the current phase of an environment
type EnvironmentPhase string

const (
	EnvironmentPhaseCreating  EnvironmentPhase = "Creating"
	EnvironmentPhaseRunning   EnvironmentPhase = "Running"
	EnvironmentPhaseStopped   EnvironmentPhase = "Stopped"
	EnvironmentPhaseFailed    EnvironmentPhase = "Failed"
	EnvironmentPhaseSuspended EnvironmentPhase = "Suspended"
	EnvironmentPhaseDeleting  EnvironmentPhase = "Deleting"
)

// Environment represents a development environment
type Environment struct {
	ID          string           `json:"id"`
	UserID      string           `json:"userId"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	TemplateID  string           `json:"templateId"`
	Phase       EnvironmentPhase `json:"phase"`
	Message     string           `json:"message,omitempty"`

	// Resource configuration
	Resources *ResourceConfig `json:"resources,omitempty"`

	// Runtime configuration
	Runtime *RuntimeConfig `json:"runtime,omitempty"`

	// Environment variables
	Environment []EnvVar `json:"environment,omitempty"`

	// Port mappings
	Ports []PortMapping `json:"ports,omitempty"`

	// Connection information
	Connection *ConnectionInfo `json:"connection,omitempty"`

	// Resource usage
	ResourceUsage *ResourceUsage `json:"resourceUsage,omitempty"`

	// Timestamps
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	StoppedAt        *time.Time `json:"stoppedAt,omitempty"`
	LastActivityTime *time.Time `json:"lastActivityTime,omitempty"`
}

// ResourceConfig defines resource allocation
type ResourceConfig struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}

// RuntimeConfig defines runtime configuration
type RuntimeConfig struct {
	Image      string   `json:"image,omitempty"`
	Command    []string `json:"command,omitempty"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"workingDir,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PortMapping defines a port mapping
type PortMapping struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	ExternalPort  int32  `json:"externalPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	URL           string `json:"url,omitempty"`
}

// ConnectionInfo contains connection details
type ConnectionInfo struct {
	PodIP      string        `json:"podIP,omitempty"`
	ExternalIP string        `json:"externalIP,omitempty"`
	SSHPort    int32         `json:"sshPort,omitempty"`
	SSHHost    string        `json:"sshHost,omitempty"`
	WebPorts   []PortMapping `json:"webPorts,omitempty"`
}

// ResourceUsage shows current resource usage
type ResourceUsage struct {
	CPU            string  `json:"cpu,omitempty"`
	CPUPercent     float64 `json:"cpuPercent,omitempty"`
	Memory         string  `json:"memory,omitempty"`
	MemoryPercent  float64 `json:"memoryPercent,omitempty"`
	Storage        string  `json:"storage,omitempty"`
	StoragePercent float64 `json:"storagePercent,omitempty"`
}

// CreateEnvironmentRequest represents a request to create an environment
type CreateEnvironmentRequest struct {
	Name        string          `json:"name" binding:"required,max=255"`
	Description string          `json:"description,omitempty"`
	TemplateID  string          `json:"templateId" binding:"required"`
	Resources   *ResourceConfig `json:"resources,omitempty"`
	Runtime     *RuntimeConfig  `json:"runtime,omitempty"`
	Environment []EnvVar        `json:"environment,omitempty"`
	Ports       []PortMapping   `json:"ports,omitempty"`
	SSHPublicKey string         `json:"sshPublicKey,omitempty"`
}

// UpdateEnvironmentRequest represents a request to update an environment
type UpdateEnvironmentRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Resources   *ResourceConfig `json:"resources,omitempty"`
	Environment []EnvVar        `json:"environment,omitempty"`
	Ports       []PortMapping   `json:"ports,omitempty"`
}

// ListEnvironmentsRequest represents a request to list environments
type ListEnvironmentsRequest struct {
	UserID     string           `form:"userId"`
	Phase      EnvironmentPhase `form:"phase"`
	TemplateID string           `form:"templateId"`
	Page       int              `form:"page,default=1"`
	PageSize   int              `form:"pageSize,default=20"`
}

// ListEnvironmentsResponse represents a response containing a list of environments
type ListEnvironmentsResponse struct {
	Environments []Environment `json:"environments"`
	Total        int64         `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
}

// BatchOperationRequest represents a batch operation request
type BatchOperationRequest struct {
	IDs       []string `json:"ids" binding:"required,min=1,max=50"`
	Operation string   `json:"operation" binding:"required,oneof=start stop restart delete"`
}

// BatchOperationResponse represents a batch operation response
type BatchOperationResponse struct {
	Results []BatchOperationResult `json:"results"`
	Success int                    `json:"success"`
	Failed  int                    `json:"failed"`
}

// BatchOperationResult represents the result of a single operation in a batch
type BatchOperationResult struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// EnvironmentStats represents environment statistics
type EnvironmentStats struct {
	Total     int64 `json:"total"`
	Running   int64 `json:"running"`
	Stopped   int64 `json:"stopped"`
	Failed    int64 `json:"failed"`
	Creating  int64 `json:"creating"`
	Suspended int64 `json:"suspended"`
}
