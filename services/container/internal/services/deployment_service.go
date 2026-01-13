// Package services provides business logic for the container service.
package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// DeploymentService handles deployment operations
type DeploymentService struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	namespace  string
	registry   string

	// In-memory storage (in production, use database)
	deployments map[string]*models.Deployment
	versions    map[string][]*models.DeploymentVersion
	logs        map[string][]*models.DeploymentLog
	mu          sync.RWMutex

	// Dockerfile templates
	dockerfileTemplates map[string]*models.DockerfileTemplate
}

// DeploymentServiceConfig holds configuration for the deployment service
type DeploymentServiceConfig struct {
	Namespace string
	Registry  string
}

// DefaultDeploymentServiceConfig returns default configuration
func DefaultDeploymentServiceConfig() *DeploymentServiceConfig {
	return &DeploymentServiceConfig{
		Namespace: "devbox-deployments",
		Registry:  "registry.devbox.io",
	}
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	kubeClient kubernetes.Interface,
	logger *zap.Logger,
	config *DeploymentServiceConfig,
) *DeploymentService {
	svc := &DeploymentService{
		kubeClient:          kubeClient,
		logger:              logger.Named("deployment-service"),
		namespace:           config.Namespace,
		registry:            config.Registry,
		deployments:         make(map[string]*models.Deployment),
		versions:            make(map[string][]*models.DeploymentVersion),
		logs:                make(map[string][]*models.DeploymentLog),
		dockerfileTemplates: make(map[string]*models.DockerfileTemplate),
	}

	// Initialize Dockerfile templates
	svc.initDockerfileTemplates()

	return svc
}

// initDockerfileTemplates initializes the Dockerfile templates
func (s *DeploymentService) initDockerfileTemplates() {
	s.dockerfileTemplates = map[string]*models.DockerfileTemplate{
		"node": {
			Name:     "node",
			Language: "nodejs",
			Content:  nodeDockerfileTemplate,
			BuildArgs: map[string]string{
				"NODE_VERSION": "18",
			},
			Description: "Node.js application with multi-stage build",
		},
		"node-nextjs": {
			Name:      "node-nextjs",
			Language:  "nodejs",
			Framework: "nextjs",
			Content:   nextjsDockerfileTemplate,
			BuildArgs: map[string]string{
				"NODE_VERSION": "18",
			},
			Description: "Next.js application with optimized production build",
		},
		"go": {
			Name:     "go",
			Language: "go",
			Content:  goDockerfileTemplate,
			BuildArgs: map[string]string{
				"GO_VERSION": "1.21",
			},
			Description: "Go application with multi-stage build",
		},
		"python": {
			Name:     "python",
			Language: "python",
			Content:  pythonDockerfileTemplate,
			BuildArgs: map[string]string{
				"PYTHON_VERSION": "3.11",
			},
			Description: "Python application with pip dependencies",
		},
		"python-django": {
			Name:      "python-django",
			Language:  "python",
			Framework: "django",
			Content:   djangoDockerfileTemplate,
			BuildArgs: map[string]string{
				"PYTHON_VERSION": "3.11",
			},
			Description: "Django application with Gunicorn",
		},
		"java": {
			Name:     "java",
			Language: "java",
			Content:  javaDockerfileTemplate,
			BuildArgs: map[string]string{
				"JAVA_VERSION": "17",
			},
			Description: "Java application with Maven build",
		},
		"rust": {
			Name:     "rust",
			Language: "rust",
			Content:  rustDockerfileTemplate,
			BuildArgs: map[string]string{
				"RUST_VERSION": "1.74",
			},
			Description: "Rust application with cargo build",
		},
	}
}


// Dockerfile templates
const nodeDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Node.js application
ARG NODE_VERSION={{.NodeVersion}}

# Stage 1: Dependencies
FROM node:${NODE_VERSION}-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

# Stage 2: Build
FROM node:${NODE_VERSION}-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Stage 3: Production
FROM node:${NODE_VERSION}-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 appuser

COPY --from=deps /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json ./

USER appuser
EXPOSE {{.Port}}
CMD ["node", "dist/index.js"]
`

const nextjsDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Next.js application
ARG NODE_VERSION={{.NodeVersion}}

# Stage 1: Dependencies
FROM node:${NODE_VERSION}-alpine AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app
COPY package*.json ./
RUN npm ci

# Stage 2: Build
FROM node:${NODE_VERSION}-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Stage 3: Production
FROM node:${NODE_VERSION}-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE {{.Port}}
ENV PORT={{.Port}}
CMD ["node", "server.js"]
`

const goDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Go application
ARG GO_VERSION={{.GoVersion}}

# Stage 1: Build
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main .

# Stage 2: Production
FROM scratch AS runner
WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/main .

EXPOSE {{.Port}}
ENTRYPOINT ["/app/main"]
`

const pythonDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Python application
ARG PYTHON_VERSION={{.PythonVersion}}

# Stage 1: Build
FROM python:${PYTHON_VERSION}-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# Stage 2: Production
FROM python:${PYTHON_VERSION}-slim AS runner
WORKDIR /app

RUN useradd --create-home --shell /bin/bash appuser

COPY --from=builder /root/.local /home/appuser/.local
COPY . .

RUN chown -R appuser:appuser /app
USER appuser

ENV PATH=/home/appuser/.local/bin:$PATH
EXPOSE {{.Port}}
CMD ["python", "main.py"]
`

const djangoDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Django application
ARG PYTHON_VERSION={{.PythonVersion}}

# Stage 1: Build
FROM python:${PYTHON_VERSION}-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# Stage 2: Production
FROM python:${PYTHON_VERSION}-slim AS runner
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --shell /bin/bash appuser

COPY --from=builder /root/.local /home/appuser/.local
COPY . .

RUN chown -R appuser:appuser /app
USER appuser

ENV PATH=/home/appuser/.local/bin:$PATH
ENV DJANGO_SETTINGS_MODULE=config.settings.production

RUN python manage.py collectstatic --noinput

EXPOSE {{.Port}}
CMD ["gunicorn", "--bind", "0.0.0.0:{{.Port}}", "--workers", "4", "config.wsgi:application"]
`

const javaDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Java application
ARG JAVA_VERSION={{.JavaVersion}}

# Stage 1: Build
FROM maven:3.9-eclipse-temurin-${JAVA_VERSION} AS builder
WORKDIR /app

COPY pom.xml .
RUN mvn dependency:go-offline -B

COPY src ./src
RUN mvn package -DskipTests -B

# Stage 2: Production
FROM eclipse-temurin:${JAVA_VERSION}-jre-alpine AS runner
WORKDIR /app

RUN addgroup --system --gid 1001 javauser
RUN adduser --system --uid 1001 javauser

COPY --from=builder /app/target/*.jar app.jar

RUN chown -R javauser:javauser /app
USER javauser

EXPOSE {{.Port}}
ENTRYPOINT ["java", "-jar", "app.jar"]
`

const rustDockerfileTemplate = `# syntax=docker/dockerfile:1
# Multi-stage build for Rust application
ARG RUST_VERSION={{.RustVersion}}

# Stage 1: Build
FROM rust:${RUST_VERSION}-alpine AS builder
WORKDIR /app

RUN apk add --no-cache musl-dev

COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm -rf src

COPY src ./src
RUN touch src/main.rs
RUN cargo build --release

# Stage 2: Production
FROM alpine:latest AS runner
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/target/release/app .

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE {{.Port}}
ENTRYPOINT ["/app/app"]
`


// DockerfileTemplateData holds data for Dockerfile template rendering
type DockerfileTemplateData struct {
	NodeVersion   string
	GoVersion     string
	PythonVersion string
	JavaVersion   string
	RustVersion   string
	Port          int
}

// GenerateDockerfile generates a Dockerfile based on the request
func (s *DeploymentService) GenerateDockerfile(ctx context.Context, req *models.GenerateDockerfileRequest) (*models.GenerateDockerfileResponse, error) {
	logger := s.logger.With(zap.String("language", req.Language), zap.String("framework", req.Framework))
	logger.Info("Generating Dockerfile")

	// Find the appropriate template
	templateKey := req.Language
	if req.Framework != "" {
		templateKey = fmt.Sprintf("%s-%s", req.Language, req.Framework)
	}

	tmpl, ok := s.dockerfileTemplates[templateKey]
	if !ok {
		// Fall back to language-only template
		tmpl, ok = s.dockerfileTemplates[req.Language]
		if !ok {
			return nil, fmt.Errorf("no Dockerfile template found for language: %s", req.Language)
		}
	}

	// Prepare template data
	data := DockerfileTemplateData{
		NodeVersion:   getOrDefault(req.BuildArgs, "NODE_VERSION", "18"),
		GoVersion:     getOrDefault(req.BuildArgs, "GO_VERSION", "1.21"),
		PythonVersion: getOrDefault(req.BuildArgs, "PYTHON_VERSION", "3.11"),
		JavaVersion:   getOrDefault(req.BuildArgs, "JAVA_VERSION", "17"),
		RustVersion:   getOrDefault(req.BuildArgs, "RUST_VERSION", "1.74"),
		Port:          8080,
	}

	if req.Version != "" {
		switch req.Language {
		case "nodejs", "node":
			data.NodeVersion = req.Version
		case "go":
			data.GoVersion = req.Version
		case "python":
			data.PythonVersion = req.Version
		case "java":
			data.JavaVersion = req.Version
		case "rust":
			data.RustVersion = req.Version
		}
	}

	// Parse and execute template
	t, err := template.New("dockerfile").Parse(tmpl.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute Dockerfile template: %w", err)
	}

	outputPath := "Dockerfile"
	if req.OutputPath != "" {
		outputPath = req.OutputPath
	}

	return &models.GenerateDockerfileResponse{
		Dockerfile: buf.String(),
		Path:       outputPath,
	}, nil
}

// getOrDefault returns the value from the map or the default value
func getOrDefault(m map[string]string, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultValue
}

// Create creates a new deployment
func (s *DeploymentService) Create(ctx context.Context, userID string, req *models.CreateDeploymentRequest) (*models.Deployment, error) {
	logger := s.logger.With(zap.String("userId", userID), zap.String("name", req.Name))
	logger.Info("Creating deployment")

	id := uuid.New().String()
	version := fmt.Sprintf("v%d", time.Now().Unix())
	now := time.Now()

	deployment := &models.Deployment{
		ID:            id,
		EnvironmentID: req.EnvironmentID,
		UserID:        userID,
		Name:          req.Name,
		Version:       version,
		Phase:         models.DeploymentPhasePending,
		BuildConfig:   req.BuildConfig,
		DeployConfig:  req.DeployConfig,
		CreatedAt:     now,
		UpdatedAt:     now,
		Metrics:       &models.DeploymentMetrics{},
	}

	// Set default build config
	if deployment.BuildConfig == nil {
		deployment.BuildConfig = &models.BuildConfig{
			DockerfilePath: "Dockerfile",
			ContextPath:    ".",
			Platform:       "linux/amd64",
		}
	}

	// Set default deploy config
	if deployment.DeployConfig == nil {
		deployment.DeployConfig = &models.DeployConfig{
			Replicas:    1,
			ServiceType: "ClusterIP",
			Resources: &models.ResourceConfig{
				CPU:    "500m",
				Memory: "512Mi",
			},
			Strategy: &models.DeploymentStrategy{
				Type:           "RollingUpdate",
				MaxUnavailable: "25%",
				MaxSurge:       "25%",
			},
		}
	}

	// Store deployment
	s.mu.Lock()
	s.deployments[id] = deployment
	s.mu.Unlock()

	// Add initial log
	s.addLog(id, "pending", "info", "Deployment created", "")

	// Start async build and deploy process
	go s.runDeploymentPipeline(context.Background(), deployment)

	logger.Info("Deployment created", zap.String("id", id), zap.String("version", version))
	return deployment, nil
}

// runDeploymentPipeline runs the build and deploy pipeline
func (s *DeploymentService) runDeploymentPipeline(ctx context.Context, deployment *models.Deployment) {
	logger := s.logger.With(zap.String("deploymentId", deployment.ID))
	startTime := time.Now()

	// Phase 1: Build
	if err := s.buildImage(ctx, deployment); err != nil {
		logger.Error("Build failed", zap.Error(err))
		s.updateDeploymentPhase(deployment.ID, models.DeploymentPhaseFailed, err.Error())
		return
	}

	// Phase 2: Push
	if err := s.pushImage(ctx, deployment); err != nil {
		logger.Error("Push failed", zap.Error(err))
		s.updateDeploymentPhase(deployment.ID, models.DeploymentPhaseFailed, err.Error())
		return
	}

	// Phase 3: Deploy
	if err := s.deployToK8s(ctx, deployment); err != nil {
		logger.Error("Deploy failed", zap.Error(err))
		s.updateDeploymentPhase(deployment.ID, models.DeploymentPhaseFailed, err.Error())
		return
	}

	// Update total time
	s.mu.Lock()
	if deployment.Metrics != nil {
		deployment.Metrics.TotalTime = time.Since(startTime).Seconds()
	}
	completedAt := time.Now()
	deployment.CompletedAt = &completedAt
	s.mu.Unlock()

	// Create version record
	s.createVersionRecord(deployment)

	logger.Info("Deployment completed successfully",
		zap.String("version", deployment.Version),
		zap.Float64("totalTime", deployment.Metrics.TotalTime))
}


// buildImage builds the Docker image
func (s *DeploymentService) buildImage(ctx context.Context, deployment *models.Deployment) error {
	logger := s.logger.With(zap.String("deploymentId", deployment.ID))
	logger.Info("Building Docker image")

	s.updateDeploymentPhase(deployment.ID, models.DeploymentPhaseBuilding, "Building Docker image")
	s.addLog(deployment.ID, "building", "info", "Starting Docker build", "")

	buildStart := time.Now()

	// In production, this would use Docker SDK or BuildKit
	// For now, we simulate the build process
	imageName := fmt.Sprintf("%s/%s/%s", s.registry, deployment.UserID, deployment.Name)
	imageTag := deployment.Version

	// Simulate build time (in production, this would be actual build)
	// Build time depends on complexity, typically 1-5 minutes
	time.Sleep(100 * time.Millisecond) // Simulated for testing

	buildDuration := time.Since(buildStart).Seconds()

	// Update image info
	s.mu.Lock()
	deployment.ImageInfo = &models.ImageInfo{
		Name:          imageName,
		Tag:           imageTag,
		Registry:      s.registry,
		BuildDuration: buildDuration,
		CreatedAt:     time.Now(),
	}
	if deployment.Metrics != nil {
		deployment.Metrics.BuildTime = buildDuration
	}
	s.mu.Unlock()

	s.addLog(deployment.ID, "building", "info",
		fmt.Sprintf("Docker image built: %s:%s", imageName, imageTag),
		fmt.Sprintf("Build duration: %.2fs", buildDuration))

	logger.Info("Docker image built",
		zap.String("image", imageName),
		zap.String("tag", imageTag),
		zap.Float64("duration", buildDuration))

	return nil
}

// pushImage pushes the Docker image to registry
func (s *DeploymentService) pushImage(ctx context.Context, deployment *models.Deployment) error {
	logger := s.logger.With(zap.String("deploymentId", deployment.ID))
	logger.Info("Pushing Docker image")

	s.updateDeploymentPhase(deployment.ID, models.DeploymentPhasePushing, "Pushing Docker image to registry")
	s.addLog(deployment.ID, "pushing", "info", "Starting image push", "")

	pushStart := time.Now()

	// In production, this would use Docker SDK
	// For now, we simulate the push process
	time.Sleep(50 * time.Millisecond) // Simulated for testing

	pushDuration := time.Since(pushStart).Seconds()

	// Update image info
	s.mu.Lock()
	if deployment.ImageInfo != nil {
		deployment.ImageInfo.PushDuration = pushDuration
		deployment.ImageInfo.Digest = fmt.Sprintf("sha256:%s", uuid.New().String()[:12])
	}
	if deployment.Metrics != nil {
		deployment.Metrics.PushTime = pushDuration
	}
	s.mu.Unlock()

	s.addLog(deployment.ID, "pushing", "info",
		"Image pushed to registry",
		fmt.Sprintf("Push duration: %.2fs", pushDuration))

	logger.Info("Docker image pushed",
		zap.Float64("duration", pushDuration))

	return nil
}

// deployToK8s deploys the application to Kubernetes
func (s *DeploymentService) deployToK8s(ctx context.Context, deployment *models.Deployment) error {
	logger := s.logger.With(zap.String("deploymentId", deployment.ID))
	logger.Info("Deploying to Kubernetes")

	s.updateDeploymentPhase(deployment.ID, models.DeploymentPhaseDeploying, "Deploying to Kubernetes")
	s.addLog(deployment.ID, "deploying", "info", "Starting Kubernetes deployment", "")

	deployStart := time.Now()

	// Generate K8s resource names
	resourceName := fmt.Sprintf("%s-%s", deployment.Name, deployment.ID[:8])

	// Create or update Deployment
	k8sDeployment := s.generateK8sDeployment(deployment, resourceName)
	if s.kubeClient != nil {
		_, err := s.kubeClient.AppsV1().Deployments(s.namespace).Create(ctx, k8sDeployment, metav1.CreateOptions{})
		if err != nil {
			// Try update if create fails
			_, err = s.kubeClient.AppsV1().Deployments(s.namespace).Update(ctx, k8sDeployment, metav1.UpdateOptions{})
			if err != nil {
				s.addLog(deployment.ID, "deploying", "error", "Failed to create/update Deployment", err.Error())
				return fmt.Errorf("failed to create/update deployment: %w", err)
			}
		}
	}

	// Create or update Service
	service := s.generateK8sService(deployment, resourceName)
	if s.kubeClient != nil {
		_, err := s.kubeClient.CoreV1().Services(s.namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			_, err = s.kubeClient.CoreV1().Services(s.namespace).Update(ctx, service, metav1.UpdateOptions{})
			if err != nil {
				s.addLog(deployment.ID, "deploying", "error", "Failed to create/update Service", err.Error())
				return fmt.Errorf("failed to create/update service: %w", err)
			}
		}
	}

	// Create Ingress if configured
	var ingressName string
	if deployment.DeployConfig != nil && deployment.DeployConfig.Ingress != nil && deployment.DeployConfig.Ingress.Enabled {
		ingress := s.generateK8sIngress(deployment, resourceName)
		ingressName = ingress.Name
		if s.kubeClient != nil {
			_, err := s.kubeClient.NetworkingV1().Ingresses(s.namespace).Create(ctx, ingress, metav1.CreateOptions{})
			if err != nil {
				_, err = s.kubeClient.NetworkingV1().Ingresses(s.namespace).Update(ctx, ingress, metav1.UpdateOptions{})
				if err != nil {
					s.addLog(deployment.ID, "deploying", "warn", "Failed to create/update Ingress", err.Error())
					// Don't fail deployment for ingress issues
				}
			}
		}
	}

	deployDuration := time.Since(deployStart).Seconds()

	// Update K8s resource info
	s.mu.Lock()
	deployment.K8sResources = &models.K8sResourceInfo{
		DeploymentName: k8sDeployment.Name,
		ServiceName:    service.Name,
		IngressName:    ingressName,
		Namespace:      s.namespace,
		InternalURL:    fmt.Sprintf("http://%s.%s.svc.cluster.local", service.Name, s.namespace),
	}
	if deployment.DeployConfig != nil && deployment.DeployConfig.Ingress != nil {
		deployment.K8sResources.ExternalURL = fmt.Sprintf("https://%s", deployment.DeployConfig.Ingress.Host)
	}
	if deployment.Metrics != nil {
		deployment.Metrics.DeployTime = deployDuration
	}
	deployment.Phase = models.DeploymentPhaseRunning
	deployment.Message = "Deployment successful"
	deployment.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.addLog(deployment.ID, "deploying", "info",
		"Kubernetes deployment successful",
		fmt.Sprintf("Deploy duration: %.2fs", deployDuration))

	logger.Info("Kubernetes deployment completed",
		zap.String("deployment", k8sDeployment.Name),
		zap.String("service", service.Name),
		zap.Float64("duration", deployDuration))

	return nil
}


// generateK8sDeployment generates a Kubernetes Deployment
func (s *DeploymentService) generateK8sDeployment(deployment *models.Deployment, resourceName string) *appsv1.Deployment {
	replicas := int32(1)
	if deployment.DeployConfig != nil && deployment.DeployConfig.Replicas > 0 {
		replicas = deployment.DeployConfig.Replicas
	}

	imageName := fmt.Sprintf("%s:%s", deployment.ImageInfo.Name, deployment.ImageInfo.Tag)

	// Build container ports
	var containerPorts []corev1.ContainerPort
	if deployment.DeployConfig != nil {
		for _, p := range deployment.DeployConfig.Ports {
			containerPorts = append(containerPorts, corev1.ContainerPort{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				Protocol:      corev1.Protocol(strings.ToUpper(getOrDefault(map[string]string{"protocol": p.Protocol}, "protocol", "TCP"))),
			})
		}
	}
	if len(containerPorts) == 0 {
		containerPorts = []corev1.ContainerPort{{
			Name:          "http",
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		}}
	}

	// Build environment variables
	var envVars []corev1.EnvVar
	if deployment.DeployConfig != nil {
		for _, e := range deployment.DeployConfig.Environment {
			envVars = append(envVars, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{}
	if deployment.DeployConfig != nil && deployment.DeployConfig.Resources != nil {
		resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(deployment.DeployConfig.Resources.CPU),
				corev1.ResourceMemory: resource.MustParse(deployment.DeployConfig.Resources.Memory),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(deployment.DeployConfig.Resources.CPU),
				corev1.ResourceMemory: resource.MustParse(deployment.DeployConfig.Resources.Memory),
			},
		}
	}

	// Build health check probes
	var livenessProbe, readinessProbe *corev1.Probe
	if deployment.DeployConfig != nil && deployment.DeployConfig.HealthCheck != nil {
		hc := deployment.DeployConfig.HealthCheck
		probe := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: hc.Path,
					Port: intstr.FromInt(int(hc.Port)),
				},
			},
			InitialDelaySeconds: hc.InitialDelaySeconds,
			PeriodSeconds:       hc.PeriodSeconds,
			TimeoutSeconds:      hc.TimeoutSeconds,
			FailureThreshold:    hc.FailureThreshold,
			SuccessThreshold:    hc.SuccessThreshold,
		}
		livenessProbe = probe
		readinessProbe = probe.DeepCopy()
	}

	// Build deployment strategy
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
			MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
		},
	}
	if deployment.DeployConfig != nil && deployment.DeployConfig.Strategy != nil {
		if deployment.DeployConfig.Strategy.Type == "Recreate" {
			strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			}
		} else if deployment.DeployConfig.Strategy.MaxUnavailable != "" {
			strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.String, StrVal: deployment.DeployConfig.Strategy.MaxUnavailable}
			strategy.RollingUpdate.MaxSurge = &intstr.IntOrString{Type: intstr.String, StrVal: deployment.DeployConfig.Strategy.MaxSurge}
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":           deployment.Name,
				"deployment-id": deployment.ID,
				"version":       deployment.Version,
				"managed-by":    "devbox",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":           deployment.Name,
					"deployment-id": deployment.ID,
				},
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           deployment.Name,
						"deployment-id": deployment.ID,
						"version":       deployment.Version,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "app",
							Image:           imageName,
							Ports:           containerPorts,
							Env:             envVars,
							Resources:       resources,
							LivenessProbe:   livenessProbe,
							ReadinessProbe:  readinessProbe,
							ImagePullPolicy: corev1.PullAlways,
						},
					},
				},
			},
		},
	}
}

// generateK8sService generates a Kubernetes Service
func (s *DeploymentService) generateK8sService(deployment *models.Deployment, resourceName string) *corev1.Service {
	serviceType := corev1.ServiceTypeClusterIP
	if deployment.DeployConfig != nil {
		switch deployment.DeployConfig.ServiceType {
		case "NodePort":
			serviceType = corev1.ServiceTypeNodePort
		case "LoadBalancer":
			serviceType = corev1.ServiceTypeLoadBalancer
		}
	}

	// Build service ports
	var servicePorts []corev1.ServicePort
	if deployment.DeployConfig != nil {
		for _, p := range deployment.DeployConfig.Ports {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       p.Name,
				Port:       p.ContainerPort,
				TargetPort: intstr.FromInt(int(p.ContainerPort)),
				Protocol:   corev1.Protocol(strings.ToUpper(getOrDefault(map[string]string{"protocol": p.Protocol}, "protocol", "TCP"))),
			})
		}
	}
	if len(servicePorts) == 0 {
		servicePorts = []corev1.ServicePort{{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
			Protocol:   corev1.ProtocolTCP,
		}}
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app":           deployment.Name,
				"deployment-id": deployment.ID,
				"managed-by":    "devbox",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app":           deployment.Name,
				"deployment-id": deployment.ID,
			},
			Ports: servicePorts,
		},
	}
}

// generateK8sIngress generates a Kubernetes Ingress
func (s *DeploymentService) generateK8sIngress(deployment *models.Deployment, resourceName string) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	ingressClassName := "nginx"

	annotations := map[string]string{
		"kubernetes.io/ingress.class": "nginx",
	}
	if deployment.DeployConfig.Ingress.Annotations != nil {
		for k, v := range deployment.DeployConfig.Ingress.Annotations {
			annotations[k] = v
		}
	}

	path := "/"
	if deployment.DeployConfig.Ingress.Path != "" {
		path = deployment.DeployConfig.Ingress.Path
	}

	port := int32(80)
	if deployment.DeployConfig != nil && len(deployment.DeployConfig.Ports) > 0 {
		port = deployment.DeployConfig.Ports[0].ContainerPort
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        resourceName,
			Namespace:   s.namespace,
			Labels: map[string]string{
				"app":           deployment.Name,
				"deployment-id": deployment.ID,
				"managed-by":    "devbox",
			},
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: deployment.DeployConfig.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: resourceName,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Add TLS if enabled
	if deployment.DeployConfig.Ingress.TLS {
		secretName := deployment.DeployConfig.Ingress.TLSSecretName
		if secretName == "" {
			secretName = fmt.Sprintf("%s-tls", resourceName)
		}
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{deployment.DeployConfig.Ingress.Host},
				SecretName: secretName,
			},
		}
	}

	return ingress
}


// updateDeploymentPhase updates the deployment phase
func (s *DeploymentService) updateDeploymentPhase(id string, phase models.DeploymentPhase, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if deployment, ok := s.deployments[id]; ok {
		deployment.Phase = phase
		deployment.Message = message
		deployment.UpdatedAt = time.Now()

		if phase == models.DeploymentPhaseBuilding && deployment.StartedAt == nil {
			now := time.Now()
			deployment.StartedAt = &now
		}
	}
}

// addLog adds a log entry for a deployment
func (s *DeploymentService) addLog(deploymentID, phase, level, message, details string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := &models.DeploymentLog{
		ID:           uuid.New().String(),
		DeploymentID: deploymentID,
		Phase:        phase,
		Level:        level,
		Message:      message,
		Details:      details,
		Timestamp:    time.Now(),
	}

	s.logs[deploymentID] = append(s.logs[deploymentID], log)
}

// createVersionRecord creates a version record for the deployment
func (s *DeploymentService) createVersionRecord(deployment *models.Deployment) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mark previous versions as inactive
	for _, v := range s.versions[deployment.ID] {
		v.IsActive = false
	}

	now := time.Now()
	version := &models.DeploymentVersion{
		ID:           uuid.New().String(),
		DeploymentID: deployment.ID,
		Version:      deployment.Version,
		ImageInfo:    deployment.ImageInfo,
		DeployConfig: deployment.DeployConfig,
		Phase:        deployment.Phase,
		Message:      deployment.Message,
		CreatedAt:    deployment.CreatedAt,
		DeployedAt:   &now,
		IsActive:     true,
	}

	s.versions[deployment.ID] = append(s.versions[deployment.ID], version)
}

// Get retrieves a deployment by ID
func (s *DeploymentService) Get(ctx context.Context, userID, id string) (*models.Deployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployment, ok := s.deployments[id]
	if !ok {
		return nil, fmt.Errorf("deployment not found: %s", id)
	}

	if deployment.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return deployment, nil
}

// List lists deployments
func (s *DeploymentService) List(ctx context.Context, req *models.ListDeploymentsRequest) (*models.ListDeploymentsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []models.Deployment
	for _, d := range s.deployments {
		if req.UserID != "" && d.UserID != req.UserID {
			continue
		}
		if req.EnvironmentID != "" && d.EnvironmentID != req.EnvironmentID {
			continue
		}
		if req.Phase != "" && d.Phase != req.Phase {
			continue
		}
		filtered = append(filtered, *d)
	}

	// Pagination
	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return &models.ListDeploymentsResponse{
		Deployments: filtered[start:end],
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}, nil
}

// GetLogs retrieves logs for a deployment
func (s *DeploymentService) GetLogs(ctx context.Context, userID, deploymentID string) ([]*models.DeploymentLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployment, ok := s.deployments[deploymentID]
	if !ok {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return s.logs[deploymentID], nil
}

// GetHistory retrieves deployment history
func (s *DeploymentService) GetHistory(ctx context.Context, userID, deploymentID string) (*models.DeploymentHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployment, ok := s.deployments[deploymentID]
	if !ok {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	versions := s.versions[deploymentID]
	versionList := make([]models.DeploymentVersion, len(versions))
	for i, v := range versions {
		versionList[i] = *v
	}

	return &models.DeploymentHistory{
		DeploymentID: deploymentID,
		Versions:     versionList,
		Total:        len(versionList),
	}, nil
}

// Rollback rolls back a deployment to a previous version
func (s *DeploymentService) Rollback(ctx context.Context, userID, deploymentID string, req *models.RollbackRequest) (*models.Deployment, error) {
	logger := s.logger.With(zap.String("deploymentId", deploymentID), zap.String("targetVersion", req.TargetVersion))
	logger.Info("Rolling back deployment")

	rollbackStart := time.Now()

	s.mu.Lock()
	deployment, ok := s.deployments[deploymentID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.UserID != userID {
		s.mu.Unlock()
		return nil, fmt.Errorf("access denied")
	}

	// Find target version
	var targetVersion *models.DeploymentVersion
	for _, v := range s.versions[deploymentID] {
		if v.Version == req.TargetVersion {
			targetVersion = v
			break
		}
	}

	if targetVersion == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("target version not found: %s", req.TargetVersion)
	}

	// Store previous state for rollback record
	previousVersion := deployment.Version
	previousImageInfo := deployment.ImageInfo

	// Update deployment with target version
	deployment.Version = targetVersion.Version
	deployment.ImageInfo = targetVersion.ImageInfo
	deployment.DeployConfig = targetVersion.DeployConfig
	deployment.Phase = models.DeploymentPhaseDeploying
	deployment.Message = fmt.Sprintf("Rolling back to version %s", req.TargetVersion)
	deployment.UpdatedAt = time.Now()
	s.mu.Unlock()

	s.addLog(deploymentID, "rollback", "info",
		fmt.Sprintf("Rolling back from %s to %s", previousVersion, req.TargetVersion),
		req.Reason)

	// Perform K8s rollback
	resourceName := fmt.Sprintf("%s-%s", deployment.Name, deployment.ID[:8])
	k8sDeployment := s.generateK8sDeployment(deployment, resourceName)

	if s.kubeClient != nil {
		_, err := s.kubeClient.AppsV1().Deployments(s.namespace).Update(ctx, k8sDeployment, metav1.UpdateOptions{})
		if err != nil {
			s.mu.Lock()
			deployment.Phase = models.DeploymentPhaseFailed
			deployment.Message = fmt.Sprintf("Rollback failed: %s", err.Error())
			deployment.Version = previousVersion
			deployment.ImageInfo = previousImageInfo
			s.mu.Unlock()

			s.addLog(deploymentID, "rollback", "error", "Rollback failed", err.Error())
			return nil, fmt.Errorf("rollback failed: %w", err)
		}
	}

	rollbackDuration := time.Since(rollbackStart).Seconds()

	s.mu.Lock()
	deployment.Phase = models.DeploymentPhaseRolledBack
	deployment.Message = fmt.Sprintf("Rolled back to version %s in %.2fs", req.TargetVersion, rollbackDuration)
	deployment.UpdatedAt = time.Now()

	// Mark current version as rolled back
	for _, v := range s.versions[deploymentID] {
		if v.Version == req.TargetVersion {
			v.IsActive = true
			now := time.Now()
			v.RolledBackAt = &now
		} else {
			v.IsActive = false
		}
	}
	s.mu.Unlock()

	s.addLog(deploymentID, "rollback", "info",
		fmt.Sprintf("Rollback completed in %.2fs", rollbackDuration),
		"")

	logger.Info("Rollback completed",
		zap.String("targetVersion", req.TargetVersion),
		zap.Float64("duration", rollbackDuration))

	return deployment, nil
}

// Delete deletes a deployment
func (s *DeploymentService) Delete(ctx context.Context, userID, id string) error {
	s.mu.Lock()
	deployment, ok := s.deployments[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("deployment not found: %s", id)
	}

	if deployment.UserID != userID {
		s.mu.Unlock()
		return fmt.Errorf("access denied")
	}
	s.mu.Unlock()

	// Delete K8s resources
	resourceName := fmt.Sprintf("%s-%s", deployment.Name, deployment.ID[:8])

	if s.kubeClient != nil {
		// Delete Deployment
		_ = s.kubeClient.AppsV1().Deployments(s.namespace).Delete(ctx, resourceName, metav1.DeleteOptions{})
		// Delete Service
		_ = s.kubeClient.CoreV1().Services(s.namespace).Delete(ctx, resourceName, metav1.DeleteOptions{})
		// Delete Ingress
		_ = s.kubeClient.NetworkingV1().Ingresses(s.namespace).Delete(ctx, resourceName, metav1.DeleteOptions{})
	}

	// Remove from storage
	s.mu.Lock()
	delete(s.deployments, id)
	delete(s.versions, id)
	delete(s.logs, id)
	s.mu.Unlock()

	return nil
}

// GetDockerfileTemplates returns available Dockerfile templates
func (s *DeploymentService) GetDockerfileTemplates() []*models.DockerfileTemplate {
	templates := make([]*models.DockerfileTemplate, 0, len(s.dockerfileTemplates))
	for _, t := range s.dockerfileTemplates {
		templates = append(templates, t)
	}
	return templates
}

// StreamLogs streams logs for a deployment (for real-time log viewing)
func (s *DeploymentService) StreamLogs(ctx context.Context, userID, deploymentID string) (<-chan *models.DeploymentLog, error) {
	s.mu.RLock()
	deployment, ok := s.deployments[deploymentID]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.UserID != userID {
		s.mu.RUnlock()
		return nil, fmt.Errorf("access denied")
	}
	s.mu.RUnlock()

	logChan := make(chan *models.DeploymentLog, 100)

	// In production, this would use a proper pub/sub mechanism
	go func() {
		defer close(logChan)

		// Send existing logs
		s.mu.RLock()
		existingLogs := s.logs[deploymentID]
		s.mu.RUnlock()

		for _, log := range existingLogs {
			select {
			case logChan <- log:
			case <-ctx.Done():
				return
			}
		}

		// Poll for new logs (in production, use pub/sub)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		lastCount := len(existingLogs)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.RLock()
				currentLogs := s.logs[deploymentID]
				s.mu.RUnlock()

				if len(currentLogs) > lastCount {
					for i := lastCount; i < len(currentLogs); i++ {
						select {
						case logChan <- currentLogs[i]:
						case <-ctx.Done():
							return
						}
					}
					lastCount = len(currentLogs)
				}
			}
		}
	}()

	return logChan, nil
}
