// Package models defines the data models for the container service.
package models

import (
	"time"
)

// DeploymentPhase represents the current phase of a deployment
type DeploymentPhase string

const (
	DeploymentPhasePending   DeploymentPhase = "Pending"
	DeploymentPhaseBuilding  DeploymentPhase = "Building"
	DeploymentPhasePushing   DeploymentPhase = "Pushing"
	DeploymentPhaseDeploying DeploymentPhase = "Deploying"
	DeploymentPhaseRunning   DeploymentPhase = "Running"
	DeploymentPhaseFailed    DeploymentPhase = "Failed"
	DeploymentPhaseRolledBack DeploymentPhase = "RolledBack"
)

// Deployment represents a deployment of an application
type Deployment struct {
	ID            string          `json:"id"`
	EnvironmentID string          `json:"environmentId"`
	UserID        string          `json:"userId"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Phase         DeploymentPhase `json:"phase"`
	Message       string          `json:"message,omitempty"`

	// Build configuration
	BuildConfig *BuildConfig `json:"buildConfig,omitempty"`

	// Deploy configuration
	DeployConfig *DeployConfig `json:"deployConfig,omitempty"`

	// Image information
	ImageInfo *ImageInfo `json:"imageInfo,omitempty"`

	// K8s resources
	K8sResources *K8sResourceInfo `json:"k8sResources,omitempty"`

	// Deployment metrics
	Metrics *DeploymentMetrics `json:"metrics,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// BuildConfig defines the build configuration
type BuildConfig struct {
	// Dockerfile path relative to source root
	DockerfilePath string `json:"dockerfilePath,omitempty"`

	// Build context path
	ContextPath string `json:"contextPath,omitempty"`

	// Build arguments
	BuildArgs map[string]string `json:"buildArgs,omitempty"`

	// Target stage for multi-stage builds
	Target string `json:"target,omitempty"`

	// No cache flag
	NoCache bool `json:"noCache,omitempty"`

	// Platform (e.g., linux/amd64, linux/arm64)
	Platform string `json:"platform,omitempty"`

	// Source type (git, upload, environment)
	SourceType string `json:"sourceType,omitempty"`

	// Git repository URL
	GitURL string `json:"gitUrl,omitempty"`

	// Git branch
	GitBranch string `json:"gitBranch,omitempty"`

	// Git commit SHA
	GitCommit string `json:"gitCommit,omitempty"`
}

// DeployConfig defines the deployment configuration
type DeployConfig struct {
	// Number of replicas
	Replicas int32 `json:"replicas,omitempty"`

	// Resource limits
	Resources *ResourceConfig `json:"resources,omitempty"`

	// Environment variables
	Environment []EnvVar `json:"environment,omitempty"`

	// Port mappings
	Ports []PortMapping `json:"ports,omitempty"`

	// Health check configuration
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`

	// Rolling update strategy
	Strategy *DeploymentStrategy `json:"strategy,omitempty"`

	// Service type (ClusterIP, NodePort, LoadBalancer)
	ServiceType string `json:"serviceType,omitempty"`

	// Ingress configuration
	Ingress *IngressConfig `json:"ingress,omitempty"`

	// Auto-scaling configuration
	AutoScaling *AutoScalingConfig `json:"autoScaling,omitempty"`
}


// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	// HTTP path for health check
	Path string `json:"path,omitempty"`

	// Port for health check
	Port int32 `json:"port,omitempty"`

	// Initial delay seconds
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`

	// Period seconds
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`

	// Timeout seconds
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// Failure threshold
	FailureThreshold int32 `json:"failureThreshold,omitempty"`

	// Success threshold
	SuccessThreshold int32 `json:"successThreshold,omitempty"`
}

// DeploymentStrategy defines the deployment strategy
type DeploymentStrategy struct {
	// Type (RollingUpdate, Recreate)
	Type string `json:"type,omitempty"`

	// Max unavailable during rolling update
	MaxUnavailable string `json:"maxUnavailable,omitempty"`

	// Max surge during rolling update
	MaxSurge string `json:"maxSurge,omitempty"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	// Enable ingress
	Enabled bool `json:"enabled,omitempty"`

	// Host name
	Host string `json:"host,omitempty"`

	// Path
	Path string `json:"path,omitempty"`

	// TLS enabled
	TLS bool `json:"tls,omitempty"`

	// TLS secret name
	TLSSecretName string `json:"tlsSecretName,omitempty"`

	// Annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AutoScalingConfig defines auto-scaling configuration
type AutoScalingConfig struct {
	// Enable auto-scaling
	Enabled bool `json:"enabled,omitempty"`

	// Minimum replicas
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// Maximum replicas
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// Target CPU utilization percentage
	TargetCPUUtilization int32 `json:"targetCPUUtilization,omitempty"`

	// Target memory utilization percentage
	TargetMemoryUtilization int32 `json:"targetMemoryUtilization,omitempty"`

	// CPU threshold for scaling (percentage)
	CPUThreshold float64 `json:"cpuThreshold,omitempty"`

	// Memory threshold for scaling (percentage)
	MemoryThreshold float64 `json:"memoryThreshold,omitempty"`

	// Cooldown period after scale up
	ScaleUpCooldown time.Duration `json:"scaleUpCooldown,omitempty"`

	// Cooldown period after scale down
	ScaleDownCooldown time.Duration `json:"scaleDownCooldown,omitempty"`
}

// ImageInfo contains information about the built image
type ImageInfo struct {
	// Full image name with registry
	Name string `json:"name"`

	// Image tag
	Tag string `json:"tag"`

	// Image digest
	Digest string `json:"digest,omitempty"`

	// Image size in bytes
	Size int64 `json:"size,omitempty"`

	// Build duration in seconds
	BuildDuration float64 `json:"buildDuration,omitempty"`

	// Push duration in seconds
	PushDuration float64 `json:"pushDuration,omitempty"`

	// Registry URL
	Registry string `json:"registry,omitempty"`

	// Created at
	CreatedAt time.Time `json:"createdAt"`
}

// K8sResourceInfo contains information about K8s resources
type K8sResourceInfo struct {
	// Deployment name
	DeploymentName string `json:"deploymentName,omitempty"`

	// Service name
	ServiceName string `json:"serviceName,omitempty"`

	// Ingress name
	IngressName string `json:"ingressName,omitempty"`

	// HPA name
	HPAName string `json:"hpaName,omitempty"`

	// Namespace
	Namespace string `json:"namespace,omitempty"`

	// External URL
	ExternalURL string `json:"externalUrl,omitempty"`

	// Internal URL
	InternalURL string `json:"internalUrl,omitempty"`
}

// DeploymentMetrics contains deployment metrics
type DeploymentMetrics struct {
	// Total build time in seconds
	BuildTime float64 `json:"buildTime,omitempty"`

	// Total push time in seconds
	PushTime float64 `json:"pushTime,omitempty"`

	// Total deploy time in seconds
	DeployTime float64 `json:"deployTime,omitempty"`

	// Total time from start to running
	TotalTime float64 `json:"totalTime,omitempty"`

	// Number of retries
	Retries int `json:"retries,omitempty"`
}


// DeploymentVersion represents a version in deployment history
type DeploymentVersion struct {
	ID            string          `json:"id"`
	DeploymentID  string          `json:"deploymentId"`
	Version       string          `json:"version"`
	ImageInfo     *ImageInfo      `json:"imageInfo,omitempty"`
	DeployConfig  *DeployConfig   `json:"deployConfig,omitempty"`
	Phase         DeploymentPhase `json:"phase"`
	Message       string          `json:"message,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	DeployedAt    *time.Time      `json:"deployedAt,omitempty"`
	RolledBackAt  *time.Time      `json:"rolledBackAt,omitempty"`
	IsActive      bool            `json:"isActive"`
}

// DeploymentLog represents a log entry during deployment
type DeploymentLog struct {
	ID           string    `json:"id"`
	DeploymentID string    `json:"deploymentId"`
	Phase        string    `json:"phase"`
	Level        string    `json:"level"` // info, warn, error
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Details      string    `json:"details,omitempty"`
}

// CreateDeploymentRequest represents a request to create a deployment
type CreateDeploymentRequest struct {
	EnvironmentID string        `json:"environmentId" binding:"required"`
	Name          string        `json:"name" binding:"required,max=255"`
	BuildConfig   *BuildConfig  `json:"buildConfig,omitempty"`
	DeployConfig  *DeployConfig `json:"deployConfig,omitempty"`
}

// RollbackRequest represents a request to rollback a deployment
type RollbackRequest struct {
	TargetVersion string `json:"targetVersion" binding:"required"`
	Reason        string `json:"reason,omitempty"`
}

// ListDeploymentsRequest represents a request to list deployments
type ListDeploymentsRequest struct {
	EnvironmentID string          `form:"environmentId"`
	UserID        string          `form:"userId"`
	Phase         DeploymentPhase `form:"phase"`
	Page          int             `form:"page,default=1"`
	PageSize      int             `form:"pageSize,default=20"`
}

// ListDeploymentsResponse represents a response containing a list of deployments
type ListDeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
	Total       int64        `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"pageSize"`
}

// DeploymentHistory represents the deployment history
type DeploymentHistory struct {
	DeploymentID string              `json:"deploymentId"`
	Versions     []DeploymentVersion `json:"versions"`
	Total        int                 `json:"total"`
}

// DockerfileTemplate represents a Dockerfile template
type DockerfileTemplate struct {
	Name        string            `json:"name"`
	Language    string            `json:"language"`
	Framework   string            `json:"framework,omitempty"`
	Content     string            `json:"content"`
	BuildArgs   map[string]string `json:"buildArgs,omitempty"`
	Description string            `json:"description,omitempty"`
}

// GenerateDockerfileRequest represents a request to generate a Dockerfile
type GenerateDockerfileRequest struct {
	Language    string            `json:"language" binding:"required"`
	Framework   string            `json:"framework,omitempty"`
	Version     string            `json:"version,omitempty"`
	BuildArgs   map[string]string `json:"buildArgs,omitempty"`
	MultiStage  bool              `json:"multiStage,omitempty"`
	OutputPath  string            `json:"outputPath,omitempty"`
}

// GenerateDockerfileResponse represents a response with generated Dockerfile
type GenerateDockerfileResponse struct {
	Dockerfile string `json:"dockerfile"`
	Path       string `json:"path,omitempty"`
}
