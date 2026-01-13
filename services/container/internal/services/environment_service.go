// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-devbox/services/container/internal/k8s/controller"
	"github.com/cloud-devbox/services/container/internal/k8s/types"
	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RateLimitError represents a rate limit exceeded error (HTTP 429)
type RateLimitError struct {
	Message      string
	Limit        int
	Window       time.Duration
	RetryAfter   time.Duration
	CurrentCount int
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// EnvironmentService handles environment operations
type EnvironmentService struct {
	kubeClient   kubernetes.Interface
	controller   *controller.DevBoxController
	logger       *zap.Logger
	namespace    string
	stateMachine *models.StateMachine

	// In-memory cache for environments (in production, use Redis/DB)
	environments map[string]*models.Environment
	mu           sync.RWMutex

	// Rate limiting
	creationRateLimiter *RateLimiter
}

// RateLimiter implements a sliding window rate limiter
type RateLimiter struct {
	mu     sync.Mutex
	counts map[string][]time.Time
	limit  int
	window time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		counts: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed      bool
	CurrentCount int
	Limit        int
	Window       time.Duration
	RetryAfter   time.Duration
}

// Check checks if an operation is allowed and returns detailed result
func (r *RateLimiter) Check(key string) *RateLimitResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Clean up old entries
	if times, ok := r.counts[key]; ok {
		var valid []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		r.counts[key] = valid
	}

	result := &RateLimitResult{
		CurrentCount: len(r.counts[key]),
		Limit:        r.limit,
		Window:       r.window,
	}

	// Check if under limit
	if len(r.counts[key]) >= r.limit {
		result.Allowed = false
		// Calculate retry after based on oldest entry
		if len(r.counts[key]) > 0 {
			oldestEntry := r.counts[key][0]
			result.RetryAfter = oldestEntry.Add(r.window).Sub(now)
			if result.RetryAfter < 0 {
				result.RetryAfter = 0
			}
		}
		return result
	}

	result.Allowed = true
	return result
}

// Record records a request for the given key
func (r *RateLimiter) Record(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[key] = append(r.counts[key], time.Now())
}

// Allow checks if an operation is allowed for the given key (legacy method)
func (r *RateLimiter) Allow(key string) bool {
	result := r.Check(key)
	if result.Allowed {
		r.Record(key)
	}
	return result.Allowed
}

// GetCount returns the current count for a key
func (r *RateLimiter) GetCount(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	if times, ok := r.counts[key]; ok {
		count := 0
		for _, t := range times {
			if t.After(cutoff) {
				count++
			}
		}
		return count
	}
	return 0
}

// EnvironmentServiceConfig holds configuration for the environment service
type EnvironmentServiceConfig struct {
	Namespace          string
	DefaultImage       string
	CreationRateLimit  int
	CreationRateWindow time.Duration
}

// DefaultEnvironmentServiceConfig returns default configuration
func DefaultEnvironmentServiceConfig() *EnvironmentServiceConfig {
	return &EnvironmentServiceConfig{
		Namespace:          "devbox",
		DefaultImage:       "ubuntu:22.04",
		CreationRateLimit:  20,
		CreationRateWindow: time.Hour,
	}
}

// NewEnvironmentService creates a new environment service
func NewEnvironmentService(
	kubeClient kubernetes.Interface,
	logger *zap.Logger,
	config *EnvironmentServiceConfig,
) *EnvironmentService {
	controllerConfig := &controller.ControllerConfig{
		Namespace:    config.Namespace,
		DefaultImage: config.DefaultImage,
		SSHPortStart: 30000,
		SSHPortEnd:   32767,
		WebPortStart: 32768,
		WebPortEnd:   35000,
	}

	ctrl := controller.NewDevBoxController(kubeClient, logger, controllerConfig)

	return &EnvironmentService{
		kubeClient:          kubeClient,
		controller:          ctrl,
		logger:              logger.Named("environment-service"),
		namespace:           config.Namespace,
		stateMachine:        models.NewStateMachine(),
		environments:        make(map[string]*models.Environment),
		creationRateLimiter: NewRateLimiter(config.CreationRateLimit, config.CreationRateWindow),
	}
}

// Create creates a new environment
func (s *EnvironmentService) Create(ctx context.Context, userID string, req *models.CreateEnvironmentRequest) (*models.Environment, error) {
	logger := s.logger.With(zap.String("userId", userID), zap.String("name", req.Name))
	logger.Info("Creating environment")

	// Check rate limit with detailed result
	rateLimitResult := s.creationRateLimiter.Check(userID)
	if !rateLimitResult.Allowed {
		logger.Warn("Rate limit exceeded",
			zap.Int("currentCount", rateLimitResult.CurrentCount),
			zap.Int("limit", rateLimitResult.Limit),
			zap.Duration("retryAfter", rateLimitResult.RetryAfter))

		return nil, &RateLimitError{
			Message:      fmt.Sprintf("rate limit exceeded: maximum %d environments per hour, please try again later", rateLimitResult.Limit),
			Limit:        rateLimitResult.Limit,
			Window:       rateLimitResult.Window,
			RetryAfter:   rateLimitResult.RetryAfter,
			CurrentCount: rateLimitResult.CurrentCount,
		}
	}

	// Record the request after validation passes
	s.creationRateLimiter.Record(userID)

	// Generate ID
	id := uuid.New().String()

	// Create environment model
	now := time.Now()
	env := &models.Environment{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		TemplateID:  req.TemplateID,
		Phase:       models.EnvironmentPhaseCreating,
		Resources:   req.Resources,
		Runtime:     req.Runtime,
		Environment: req.Environment,
		Ports:       req.Ports,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set default resources if not provided
	if env.Resources == nil {
		env.Resources = &models.ResourceConfig{
			CPU:     "1",
			Memory:  "2Gi",
			Storage: "10Gi",
		}
	}

	// Store in cache
	s.mu.Lock()
	s.environments[id] = env
	s.mu.Unlock()

	// Create DevBox CR
	devbox := s.environmentToDevBox(env, req.SSHPublicKey)
	if err := s.controller.Reconcile(ctx, devbox); err != nil {
		logger.Error("Failed to create DevBox", zap.Error(err))
		s.mu.Lock()
		s.stateMachine.Transition(env, models.EnvironmentPhaseFailed)
		env.Message = err.Error()
		s.mu.Unlock()
		return env, nil // Return the environment with failed status
	}

	// Update environment from DevBox status
	s.updateEnvironmentFromDevBox(env, devbox)

	logger.Info("Environment created", zap.String("id", id))
	return env, nil
}

// Get retrieves an environment by ID
func (s *EnvironmentService) Get(ctx context.Context, userID, id string) (*models.Environment, error) {
	s.mu.RLock()
	env, ok := s.environments[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	// Check ownership
	if env.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return env, nil
}

// List lists environments for a user
func (s *EnvironmentService) List(ctx context.Context, req *models.ListEnvironmentsRequest) (*models.ListEnvironmentsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []models.Environment
	for _, env := range s.environments {
		// Filter by user
		if req.UserID != "" && env.UserID != req.UserID {
			continue
		}
		// Filter by phase
		if req.Phase != "" && env.Phase != req.Phase {
			continue
		}
		// Filter by template
		if req.TemplateID != "" && env.TemplateID != req.TemplateID {
			continue
		}
		filtered = append(filtered, *env)
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

	return &models.ListEnvironmentsResponse{
		Environments: filtered[start:end],
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}

// Update updates an environment
func (s *EnvironmentService) Update(ctx context.Context, userID, id string, req *models.UpdateEnvironmentRequest) (*models.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Update fields
	if req.Name != nil {
		env.Name = *req.Name
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.Resources != nil {
		env.Resources = req.Resources
	}
	if req.Environment != nil {
		env.Environment = req.Environment
	}
	if req.Ports != nil {
		env.Ports = req.Ports
	}
	env.UpdatedAt = time.Now()

	return env, nil
}

// Delete deletes an environment
func (s *EnvironmentService) Delete(ctx context.Context, userID, id string) error {
	s.mu.Lock()
	env, ok := s.environments[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		s.mu.Unlock()
		return fmt.Errorf("access denied")
	}

	env.Phase = models.EnvironmentPhaseDeleting
	s.mu.Unlock()

	// Delete DevBox CR
	devbox := s.environmentToDevBox(env, "")
	devbox.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	if err := s.controller.Reconcile(ctx, devbox); err != nil {
		s.logger.Error("Failed to delete DevBox", zap.Error(err))
	}

	// Remove from cache
	s.mu.Lock()
	delete(s.environments, id)
	s.mu.Unlock()

	return nil
}

// Start starts a stopped environment
func (s *EnvironmentService) Start(ctx context.Context, userID, id string) (*models.Environment, error) {
	s.mu.Lock()
	env, ok := s.environments[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		s.mu.Unlock()
		return nil, fmt.Errorf("access denied")
	}

	// Check if transition is valid using state machine
	if !s.stateMachine.CanTransition(env.Phase, models.EnvironmentPhaseCreating) {
		s.mu.Unlock()
		return nil, fmt.Errorf("cannot start environment in %s state", env.Phase)
	}

	// Transition to Creating state
	if err := s.stateMachine.Transition(env, models.EnvironmentPhaseCreating); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()

	// Reconcile to start
	devbox := s.environmentToDevBox(env, "")
	if err := s.controller.Reconcile(ctx, devbox); err != nil {
		s.logger.Error("Failed to start DevBox", zap.Error(err))
		s.mu.Lock()
		s.stateMachine.Transition(env, models.EnvironmentPhaseFailed)
		env.Message = err.Error()
		s.mu.Unlock()
	}

	s.updateEnvironmentFromDevBox(env, devbox)
	return env, nil
}

// Stop stops a running environment
func (s *EnvironmentService) Stop(ctx context.Context, userID, id string) (*models.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Check if transition is valid using state machine
	if !s.stateMachine.CanTransition(env.Phase, models.EnvironmentPhaseStopped) {
		return nil, fmt.Errorf("cannot stop environment in %s state", env.Phase)
	}

	// Transition to Stopped state
	if err := s.stateMachine.Transition(env, models.EnvironmentPhaseStopped); err != nil {
		return nil, err
	}

	return env, nil
}

// Suspend suspends a running environment (due to inactivity)
func (s *EnvironmentService) Suspend(ctx context.Context, userID, id string) (*models.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Check if transition is valid using state machine
	if !s.stateMachine.CanTransition(env.Phase, models.EnvironmentPhaseSuspended) {
		return nil, fmt.Errorf("cannot suspend environment in %s state", env.Phase)
	}

	// Transition to Suspended state
	if err := s.stateMachine.Transition(env, models.EnvironmentPhaseSuspended); err != nil {
		return nil, err
	}

	env.Message = "Environment suspended due to inactivity"
	return env, nil
}

// Archive archives a suspended environment
func (s *EnvironmentService) Archive(ctx context.Context, userID, id string) (*models.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	if env.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Check if transition is valid using state machine
	if !s.stateMachine.CanTransition(env.Phase, models.EnvironmentPhaseArchived) {
		return nil, fmt.Errorf("cannot archive environment in %s state", env.Phase)
	}

	// Transition to Archived state
	if err := s.stateMachine.Transition(env, models.EnvironmentPhaseArchived); err != nil {
		return nil, err
	}

	env.Message = "Environment archived"
	return env, nil
}

// Restart restarts an environment
func (s *EnvironmentService) Restart(ctx context.Context, userID, id string) (*models.Environment, error) {
	// Stop then start
	if _, err := s.Stop(ctx, userID, id); err != nil {
		// If not running, try to start anyway
		s.logger.Debug("Stop failed, attempting start", zap.Error(err))
	}
	return s.Start(ctx, userID, id)
}

// BatchOperation performs a batch operation on multiple environments
func (s *EnvironmentService) BatchOperation(ctx context.Context, userID string, req *models.BatchOperationRequest) (*models.BatchOperationResponse, error) {
	results := make([]models.BatchOperationResult, len(req.IDs))
	success := 0
	failed := 0

	for i, id := range req.IDs {
		var err error
		switch req.Operation {
		case "start":
			_, err = s.Start(ctx, userID, id)
		case "stop":
			_, err = s.Stop(ctx, userID, id)
		case "restart":
			_, err = s.Restart(ctx, userID, id)
		case "delete":
			err = s.Delete(ctx, userID, id)
		case "suspend":
			_, err = s.Suspend(ctx, userID, id)
		case "archive":
			_, err = s.Archive(ctx, userID, id)
		default:
			err = fmt.Errorf("unknown operation: %s", req.Operation)
		}

		results[i] = models.BatchOperationResult{
			ID:      id,
			Success: err == nil,
		}
		if err != nil {
			results[i].Error = err.Error()
			failed++
		} else {
			success++
		}
	}

	return &models.BatchOperationResponse{
		Results: results,
		Success: success,
		Failed:  failed,
	}, nil
}

// GetStats returns environment statistics for a user
func (s *EnvironmentService) GetStats(ctx context.Context, userID string) (*models.EnvironmentStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &models.EnvironmentStats{}
	for _, env := range s.environments {
		if env.UserID != userID {
			continue
		}
		stats.Total++
		switch env.Phase {
		case models.EnvironmentPhaseRunning:
			stats.Running++
		case models.EnvironmentPhaseStopped:
			stats.Stopped++
		case models.EnvironmentPhaseFailed:
			stats.Failed++
		case models.EnvironmentPhaseCreating:
			stats.Creating++
		case models.EnvironmentPhaseSuspended:
			stats.Suspended++
		}
	}
	return stats, nil
}

// environmentToDevBox converts an Environment to a DevBox CR
func (s *EnvironmentService) environmentToDevBox(env *models.Environment, sshPublicKey string) *types.DevBox {
	devbox := &types.DevBox{}
	devbox.Name = env.ID
	devbox.Namespace = s.namespace

	devbox.Spec = types.DevBoxSpec{
		TemplateID:  env.TemplateID,
		UserID:      env.UserID,
		Name:        env.Name,
		Description: env.Description,
	}

	if env.Resources != nil {
		devbox.Spec.Resources = &types.ResourceSpec{
			CPU:     env.Resources.CPU,
			Memory:  env.Resources.Memory,
			Storage: env.Resources.Storage,
		}
	}

	if env.Runtime != nil {
		devbox.Spec.Runtime = &types.RuntimeSpec{
			Image:      env.Runtime.Image,
			Command:    env.Runtime.Command,
			Args:       env.Runtime.Args,
			WorkingDir: env.Runtime.WorkingDir,
		}
	}

	for _, e := range env.Environment {
		devbox.Spec.Environment = append(devbox.Spec.Environment, types.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	for _, p := range env.Ports {
		devbox.Spec.Ports = append(devbox.Spec.Ports, types.PortSpec{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
		})
	}

	if sshPublicKey != "" {
		devbox.Spec.SSH = &types.SSHSpec{
			Enabled:   true,
			Port:      22,
			PublicKey: sshPublicKey,
		}
	}

	devbox.Spec.AutoStop = &types.AutoStopSpec{
		Enabled:           true,
		InactivityTimeout: "72h",
	}

	devbox.Spec.AutoDelete = &types.AutoDeleteSpec{
		Enabled:        true,
		StoppedTimeout: "168h",
	}

	return devbox
}

// updateEnvironmentFromDevBox updates an Environment from DevBox status
func (s *EnvironmentService) updateEnvironmentFromDevBox(env *models.Environment, devbox *types.DevBox) {
	// Update phase
	switch devbox.Status.Phase {
	case types.DevBoxPhaseCreating:
		env.Phase = models.EnvironmentPhaseCreating
	case types.DevBoxPhaseRunning:
		env.Phase = models.EnvironmentPhaseRunning
	case types.DevBoxPhaseStopped:
		env.Phase = models.EnvironmentPhaseStopped
	case types.DevBoxPhaseFailed:
		env.Phase = models.EnvironmentPhaseFailed
	case types.DevBoxPhaseSuspended:
		env.Phase = models.EnvironmentPhaseSuspended
	case types.DevBoxPhaseDeleting:
		env.Phase = models.EnvironmentPhaseDeleting
	}

	env.Message = devbox.Status.Message

	// Update connection info
	if devbox.Status.PodIP != "" || devbox.Status.SSHPort != 0 {
		env.Connection = &models.ConnectionInfo{
			PodIP:      devbox.Status.PodIP,
			ExternalIP: devbox.Status.ExternalIP,
			SSHPort:    devbox.Status.SSHPort,
		}

		for _, wp := range devbox.Status.WebPorts {
			env.Connection.WebPorts = append(env.Connection.WebPorts, models.PortMapping{
				Name:         wp.Name,
				ExternalPort: wp.ExternalPort,
				URL:          wp.URL,
			})
		}
	}

	// Update resource usage
	if devbox.Status.ResourceUsage != nil {
		env.ResourceUsage = &models.ResourceUsage{
			CPU:     devbox.Status.ResourceUsage.CPU,
			Memory:  devbox.Status.ResourceUsage.Memory,
			Storage: devbox.Status.ResourceUsage.Storage,
		}
	}

	// Update timestamps
	if devbox.Status.StartedAt != nil {
		t := devbox.Status.StartedAt.Time
		env.StartedAt = &t
	}
	if devbox.Status.StoppedAt != nil {
		t := devbox.Status.StoppedAt.Time
		env.StoppedAt = &t
	}
	if devbox.Status.LastActivityTime != nil {
		t := devbox.Status.LastActivityTime.Time
		env.LastActivityTime = &t
	}
}

// UpdateActivity updates the last activity time for an environment
func (s *EnvironmentService) UpdateActivity(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, ok := s.environments[id]
	if !ok {
		return fmt.Errorf("environment not found: %s", id)
	}

	now := time.Now()
	env.LastActivityTime = &now
	return nil
}

// GetStateMachine returns the state machine for external use
func (s *EnvironmentService) GetStateMachine() *models.StateMachine {
	return s.stateMachine
}

// GetRateLimitStatus returns the current rate limit status for a user
func (s *EnvironmentService) GetRateLimitStatus(userID string) *RateLimitResult {
	return s.creationRateLimiter.Check(userID)
}

// GetCreationRateLimiter returns the rate limiter for testing
func (s *EnvironmentService) GetCreationRateLimiter() *RateLimiter {
	return s.creationRateLimiter
}
