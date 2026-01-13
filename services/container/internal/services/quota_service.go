// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// QuotaService manages resource quotas for users and environments
type QuotaService struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	namespace  string

	// User quotas cache
	userQuotas map[string]*UserQuota
	mu         sync.RWMutex

	// Default quotas
	defaultQuota *UserQuota
}

// UserQuota defines resource limits for a user
type UserQuota struct {
	UserID string `json:"userId"`

	// Maximum number of environments
	MaxEnvironments int `json:"maxEnvironments"`

	// Maximum total CPU (in cores)
	MaxCPU string `json:"maxCpu"`

	// Maximum total memory
	MaxMemory string `json:"maxMemory"`

	// Maximum total storage
	MaxStorage string `json:"maxStorage"`

	// Maximum running time per month (in hours)
	MaxRunningHours int `json:"maxRunningHours"`

	// Current usage
	CurrentEnvironments int    `json:"currentEnvironments"`
	CurrentCPU          string `json:"currentCpu"`
	CurrentMemory       string `json:"currentMemory"`
	CurrentStorage      string `json:"currentStorage"`
	CurrentRunningHours int    `json:"currentRunningHours"`

	// Timestamps
	UpdatedAt time.Time `json:"updatedAt"`
}

// QuotaServiceConfig holds configuration for the quota service
type QuotaServiceConfig struct {
	Namespace           string
	DefaultMaxEnvs      int
	DefaultMaxCPU       string
	DefaultMaxMemory    string
	DefaultMaxStorage   string
	DefaultMaxRunHours  int
}

// DefaultQuotaServiceConfig returns default configuration
func DefaultQuotaServiceConfig() *QuotaServiceConfig {
	return &QuotaServiceConfig{
		Namespace:          "devbox",
		DefaultMaxEnvs:     10,
		DefaultMaxCPU:      "8",
		DefaultMaxMemory:   "16Gi",
		DefaultMaxStorage:  "100Gi",
		DefaultMaxRunHours: 720, // 30 days * 24 hours
	}
}

// NewQuotaService creates a new quota service
func NewQuotaService(
	kubeClient kubernetes.Interface,
	logger *zap.Logger,
	config *QuotaServiceConfig,
) *QuotaService {
	return &QuotaService{
		kubeClient: kubeClient,
		logger:     logger.Named("quota-service"),
		namespace:  config.Namespace,
		userQuotas: make(map[string]*UserQuota),
		defaultQuota: &UserQuota{
			MaxEnvironments: config.DefaultMaxEnvs,
			MaxCPU:          config.DefaultMaxCPU,
			MaxMemory:       config.DefaultMaxMemory,
			MaxStorage:      config.DefaultMaxStorage,
			MaxRunningHours: config.DefaultMaxRunHours,
		},
	}
}

// GetUserQuota retrieves the quota for a user
func (s *QuotaService) GetUserQuota(ctx context.Context, userID string) (*UserQuota, error) {
	s.mu.RLock()
	quota, ok := s.userQuotas[userID]
	s.mu.RUnlock()

	if ok {
		return quota, nil
	}

	// Return default quota for new users
	return &UserQuota{
		UserID:          userID,
		MaxEnvironments: s.defaultQuota.MaxEnvironments,
		MaxCPU:          s.defaultQuota.MaxCPU,
		MaxMemory:       s.defaultQuota.MaxMemory,
		MaxStorage:      s.defaultQuota.MaxStorage,
		MaxRunningHours: s.defaultQuota.MaxRunningHours,
		UpdatedAt:       time.Now(),
	}, nil
}

// SetUserQuota sets the quota for a user
func (s *QuotaService) SetUserQuota(ctx context.Context, quota *UserQuota) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	quota.UpdatedAt = time.Now()
	s.userQuotas[quota.UserID] = quota
	return nil
}

// CheckQuota checks if a resource request is within quota
func (s *QuotaService) CheckQuota(ctx context.Context, userID string, request *ResourceRequest) error {
	quota, err := s.GetUserQuota(ctx, userID)
	if err != nil {
		return err
	}

	// Check environment count
	if request.NewEnvironment && quota.CurrentEnvironments >= quota.MaxEnvironments {
		return fmt.Errorf("quota exceeded: maximum %d environments allowed", quota.MaxEnvironments)
	}

	// Check CPU
	if request.CPU != "" {
		requestedCPU := resource.MustParse(request.CPU)
		currentCPU := resource.MustParse(quota.CurrentCPU)
		maxCPU := resource.MustParse(quota.MaxCPU)

		totalCPU := currentCPU.DeepCopy()
		totalCPU.Add(requestedCPU)

		if totalCPU.Cmp(maxCPU) > 0 {
			return fmt.Errorf("quota exceeded: CPU request %s would exceed limit %s", request.CPU, quota.MaxCPU)
		}
	}

	// Check memory
	if request.Memory != "" {
		requestedMem := resource.MustParse(request.Memory)
		currentMem := resource.MustParse(quota.CurrentMemory)
		maxMem := resource.MustParse(quota.MaxMemory)

		totalMem := currentMem.DeepCopy()
		totalMem.Add(requestedMem)

		if totalMem.Cmp(maxMem) > 0 {
			return fmt.Errorf("quota exceeded: memory request %s would exceed limit %s", request.Memory, quota.MaxMemory)
		}
	}

	// Check storage
	if request.Storage != "" {
		requestedStorage := resource.MustParse(request.Storage)
		currentStorage := resource.MustParse(quota.CurrentStorage)
		maxStorage := resource.MustParse(quota.MaxStorage)

		totalStorage := currentStorage.DeepCopy()
		totalStorage.Add(requestedStorage)

		if totalStorage.Cmp(maxStorage) > 0 {
			return fmt.Errorf("quota exceeded: storage request %s would exceed limit %s", request.Storage, quota.MaxStorage)
		}
	}

	return nil
}

// ResourceRequest represents a resource request for quota checking
type ResourceRequest struct {
	NewEnvironment bool
	CPU            string
	Memory         string
	Storage        string
}

// UpdateUsage updates the current usage for a user
func (s *QuotaService) UpdateUsage(ctx context.Context, userID string, delta *ResourceDelta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	quota, ok := s.userQuotas[userID]
	if !ok {
		quota = &UserQuota{
			UserID:          userID,
			MaxEnvironments: s.defaultQuota.MaxEnvironments,
			MaxCPU:          s.defaultQuota.MaxCPU,
			MaxMemory:       s.defaultQuota.MaxMemory,
			MaxStorage:      s.defaultQuota.MaxStorage,
			MaxRunningHours: s.defaultQuota.MaxRunningHours,
			CurrentCPU:      "0",
			CurrentMemory:   "0",
			CurrentStorage:  "0",
		}
		s.userQuotas[userID] = quota
	}

	// Update environment count
	quota.CurrentEnvironments += delta.EnvironmentDelta

	// Update CPU
	if delta.CPUDelta != "" {
		currentCPU := resource.MustParse(quota.CurrentCPU)
		deltaCPU := resource.MustParse(delta.CPUDelta)
		if delta.Add {
			currentCPU.Add(deltaCPU)
		} else {
			currentCPU.Sub(deltaCPU)
		}
		quota.CurrentCPU = currentCPU.String()
	}

	// Update memory
	if delta.MemoryDelta != "" {
		currentMem := resource.MustParse(quota.CurrentMemory)
		deltaMem := resource.MustParse(delta.MemoryDelta)
		if delta.Add {
			currentMem.Add(deltaMem)
		} else {
			currentMem.Sub(deltaMem)
		}
		quota.CurrentMemory = currentMem.String()
	}

	// Update storage
	if delta.StorageDelta != "" {
		currentStorage := resource.MustParse(quota.CurrentStorage)
		deltaStorage := resource.MustParse(delta.StorageDelta)
		if delta.Add {
			currentStorage.Add(deltaStorage)
		} else {
			currentStorage.Sub(deltaStorage)
		}
		quota.CurrentStorage = currentStorage.String()
	}

	quota.UpdatedAt = time.Now()
	return nil
}

// ResourceDelta represents a change in resource usage
type ResourceDelta struct {
	Add              bool
	EnvironmentDelta int
	CPUDelta         string
	MemoryDelta      string
	StorageDelta     string
}

// CreateResourceQuota creates a Kubernetes ResourceQuota for a namespace
func (s *QuotaService) CreateResourceQuota(ctx context.Context, namespace string, quota *UserQuota) error {
	if s.kubeClient == nil {
		return nil // Skip if no Kubernetes client
	}

	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "devbox-quota",
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    resource.MustParse(quota.MaxCPU),
				corev1.ResourceLimitsCPU:      resource.MustParse(quota.MaxCPU),
				corev1.ResourceRequestsMemory: resource.MustParse(quota.MaxMemory),
				corev1.ResourceLimitsMemory:   resource.MustParse(quota.MaxMemory),
				corev1.ResourceRequestsStorage: resource.MustParse(quota.MaxStorage),
				corev1.ResourcePods:           resource.MustParse(fmt.Sprintf("%d", quota.MaxEnvironments)),
			},
		},
	}

	_, err := s.kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ResourceQuota: %w", err)
	}

	return nil
}

// UpdateResourceQuota updates a Kubernetes ResourceQuota
func (s *QuotaService) UpdateResourceQuota(ctx context.Context, namespace string, quota *UserQuota) error {
	if s.kubeClient == nil {
		return nil // Skip if no Kubernetes client
	}

	resourceQuota, err := s.kubeClient.CoreV1().ResourceQuotas(namespace).Get(ctx, "devbox-quota", metav1.GetOptions{})
	if err != nil {
		// Create if not exists
		return s.CreateResourceQuota(ctx, namespace, quota)
	}

	resourceQuota.Spec.Hard = corev1.ResourceList{
		corev1.ResourceRequestsCPU:    resource.MustParse(quota.MaxCPU),
		corev1.ResourceLimitsCPU:      resource.MustParse(quota.MaxCPU),
		corev1.ResourceRequestsMemory: resource.MustParse(quota.MaxMemory),
		corev1.ResourceLimitsMemory:   resource.MustParse(quota.MaxMemory),
		corev1.ResourceRequestsStorage: resource.MustParse(quota.MaxStorage),
		corev1.ResourcePods:           resource.MustParse(fmt.Sprintf("%d", quota.MaxEnvironments)),
	}

	_, err = s.kubeClient.CoreV1().ResourceQuotas(namespace).Update(ctx, resourceQuota, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ResourceQuota: %w", err)
	}

	return nil
}

// CreateLimitRange creates a Kubernetes LimitRange for a namespace
func (s *QuotaService) CreateLimitRange(ctx context.Context, namespace string) error {
	if s.kubeClient == nil {
		return nil // Skip if no Kubernetes client
	}

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "devbox-limits",
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
			},
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("8Gi"),
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
				{
					Type: corev1.LimitTypePersistentVolumeClaim,
					Max: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("50Gi"),
					},
					Min: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}

	_, err := s.kubeClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create LimitRange: %w", err)
	}

	return nil
}
