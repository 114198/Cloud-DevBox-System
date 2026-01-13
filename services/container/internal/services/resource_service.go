// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceService handles dynamic resource adjustments
type ResourceService struct {
	kubeClient   kubernetes.Interface
	quotaService *QuotaService
	logger       *zap.Logger
	namespace    string
}

// ResourceAdjustment represents a resource adjustment request
type ResourceAdjustment struct {
	EnvironmentID string `json:"environmentId"`
	CPU           string `json:"cpu,omitempty"`
	Memory        string `json:"memory,omitempty"`
	Storage       string `json:"storage,omitempty"`
}

// ResourceAdjustmentResult represents the result of a resource adjustment
type ResourceAdjustmentResult struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message,omitempty"`
	EffectiveTime time.Time `json:"effectiveTime"`
	OldResources  Resources `json:"oldResources"`
	NewResources  Resources `json:"newResources"`
}

// Resources represents resource allocation
type Resources struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}

// NewResourceService creates a new resource service
func NewResourceService(
	kubeClient kubernetes.Interface,
	quotaService *QuotaService,
	logger *zap.Logger,
	namespace string,
) *ResourceService {
	return &ResourceService{
		kubeClient:   kubeClient,
		quotaService: quotaService,
		logger:       logger.Named("resource-service"),
		namespace:    namespace,
	}
}

// AdjustResources adjusts resources for an environment
// This should complete within 30 seconds as per requirements
func (s *ResourceService) AdjustResources(ctx context.Context, userID string, adj *ResourceAdjustment) (*ResourceAdjustmentResult, error) {
	logger := s.logger.With(
		zap.String("environmentId", adj.EnvironmentID),
		zap.String("userId", userID),
	)
	logger.Info("Adjusting resources")

	startTime := time.Now()

	// Create a context with 30-second timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get current pod
	podNamespace := fmt.Sprintf("devbox-%s", adj.EnvironmentID)
	podName := fmt.Sprintf("%s-pod", adj.EnvironmentID)

	pod, err := s.kubeClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	// Get current resources
	oldResources := s.extractResources(pod)

	// Check quota for the adjustment
	if err := s.checkQuotaForAdjustment(ctx, userID, oldResources, adj); err != nil {
		return &ResourceAdjustmentResult{
			Success:      false,
			Message:      err.Error(),
			OldResources: oldResources,
		}, nil
	}

	// Apply resource changes
	newResources, err := s.applyResourceChanges(ctx, pod, adj)
	if err != nil {
		return &ResourceAdjustmentResult{
			Success:      false,
			Message:      err.Error(),
			OldResources: oldResources,
		}, nil
	}

	// Update quota usage
	s.updateQuotaUsage(ctx, userID, oldResources, *newResources)

	effectiveTime := time.Now()
	duration := effectiveTime.Sub(startTime)
	logger.Info("Resource adjustment completed",
		zap.Duration("duration", duration),
		zap.Bool("withinSLA", duration < 30*time.Second),
	)

	return &ResourceAdjustmentResult{
		Success:       true,
		Message:       fmt.Sprintf("Resources adjusted in %v", duration),
		EffectiveTime: effectiveTime,
		OldResources:  oldResources,
		NewResources:  *newResources,
	}, nil
}

// extractResources extracts current resources from a pod
func (s *ResourceService) extractResources(pod *corev1.Pod) Resources {
	resources := Resources{}

	if len(pod.Spec.Containers) > 0 {
		container := pod.Spec.Containers[0]
		if cpu := container.Resources.Limits.Cpu(); cpu != nil {
			resources.CPU = cpu.String()
		}
		if mem := container.Resources.Limits.Memory(); mem != nil {
			resources.Memory = mem.String()
		}
	}

	// Get storage from PVC
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {
			// In a real implementation, we'd query the PVC
			resources.Storage = "10Gi" // Default
		}
	}

	return resources
}

// checkQuotaForAdjustment checks if the adjustment is within quota
func (s *ResourceService) checkQuotaForAdjustment(ctx context.Context, userID string, old Resources, adj *ResourceAdjustment) error {
	// Calculate the delta
	request := &ResourceRequest{}

	if adj.CPU != "" {
		oldCPU := resource.MustParse(old.CPU)
		newCPU := resource.MustParse(adj.CPU)
		if newCPU.Cmp(oldCPU) > 0 {
			delta := newCPU.DeepCopy()
			delta.Sub(oldCPU)
			request.CPU = delta.String()
		}
	}

	if adj.Memory != "" {
		oldMem := resource.MustParse(old.Memory)
		newMem := resource.MustParse(adj.Memory)
		if newMem.Cmp(oldMem) > 0 {
			delta := newMem.DeepCopy()
			delta.Sub(oldMem)
			request.Memory = delta.String()
		}
	}

	if adj.Storage != "" {
		oldStorage := resource.MustParse(old.Storage)
		newStorage := resource.MustParse(adj.Storage)
		if newStorage.Cmp(oldStorage) > 0 {
			delta := newStorage.DeepCopy()
			delta.Sub(oldStorage)
			request.Storage = delta.String()
		}
	}

	return s.quotaService.CheckQuota(ctx, userID, request)
}

// applyResourceChanges applies resource changes to a pod
func (s *ResourceService) applyResourceChanges(ctx context.Context, pod *corev1.Pod, adj *ResourceAdjustment) (*Resources, error) {
	newResources := &Resources{
		CPU:     pod.Spec.Containers[0].Resources.Limits.Cpu().String(),
		Memory:  pod.Spec.Containers[0].Resources.Limits.Memory().String(),
		Storage: "10Gi",
	}

	// Update container resources
	if len(pod.Spec.Containers) > 0 {
		container := &pod.Spec.Containers[0]

		if adj.CPU != "" {
			cpuQuantity := resource.MustParse(adj.CPU)
			container.Resources.Requests[corev1.ResourceCPU] = cpuQuantity
			container.Resources.Limits[corev1.ResourceCPU] = cpuQuantity
			newResources.CPU = adj.CPU
		}

		if adj.Memory != "" {
			memQuantity := resource.MustParse(adj.Memory)
			container.Resources.Requests[corev1.ResourceMemory] = memQuantity
			container.Resources.Limits[corev1.ResourceMemory] = memQuantity
			newResources.Memory = adj.Memory
		}
	}

	// Note: In Kubernetes, you cannot directly update pod resources.
	// You need to either:
	// 1. Use VPA (Vertical Pod Autoscaler) for in-place updates
	// 2. Delete and recreate the pod
	// 3. Use a Deployment and update the spec
	
	// For this implementation, we'll update via patch
	// In production, consider using VPA or recreating the pod

	_, err := s.kubeClient.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		// If direct update fails, we need to recreate
		s.logger.Warn("Direct pod update failed, attempting recreate", zap.Error(err))
		return s.recreatePodWithNewResources(ctx, pod, adj)
	}

	if adj.Storage != "" {
		newResources.Storage = adj.Storage
		// Storage adjustment requires PVC resize
		if err := s.resizePVC(ctx, pod.Namespace, adj.EnvironmentID, adj.Storage); err != nil {
			s.logger.Warn("PVC resize failed", zap.Error(err))
		}
	}

	return newResources, nil
}

// recreatePodWithNewResources recreates a pod with new resources
func (s *ResourceService) recreatePodWithNewResources(ctx context.Context, pod *corev1.Pod, adj *ResourceAdjustment) (*Resources, error) {
	// Delete the old pod
	err := s.kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to delete old pod: %w", err)
	}

	// Wait for pod to be deleted
	time.Sleep(2 * time.Second)

	// Update resources in pod spec
	newResources := &Resources{
		CPU:     pod.Spec.Containers[0].Resources.Limits.Cpu().String(),
		Memory:  pod.Spec.Containers[0].Resources.Limits.Memory().String(),
		Storage: "10Gi",
	}

	if adj.CPU != "" {
		cpuQuantity := resource.MustParse(adj.CPU)
		pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = cpuQuantity
		pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = cpuQuantity
		newResources.CPU = adj.CPU
	}

	if adj.Memory != "" {
		memQuantity := resource.MustParse(adj.Memory)
		pod.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = memQuantity
		pod.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = memQuantity
		newResources.Memory = adj.Memory
	}

	// Clear metadata for recreation
	pod.ResourceVersion = ""
	pod.UID = ""
	pod.Status = corev1.PodStatus{}

	// Create new pod
	_, err = s.kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create new pod: %w", err)
	}

	return newResources, nil
}

// resizePVC resizes a PersistentVolumeClaim
func (s *ResourceService) resizePVC(ctx context.Context, namespace, envID, newSize string) error {
	pvcName := fmt.Sprintf("%s-storage", envID)

	pvc, err := s.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get PVC: %w", err)
	}

	// Check if storage class supports expansion
	// In production, verify the storage class has allowVolumeExpansion: true

	newQuantity := resource.MustParse(newSize)
	currentQuantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]

	// Can only expand, not shrink
	if newQuantity.Cmp(currentQuantity) <= 0 {
		return fmt.Errorf("cannot shrink PVC, current: %s, requested: %s", currentQuantity.String(), newSize)
	}

	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = newQuantity

	_, err = s.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update PVC: %w", err)
	}

	return nil
}

// updateQuotaUsage updates quota usage after resource adjustment
func (s *ResourceService) updateQuotaUsage(ctx context.Context, userID string, old, new Resources) {
	// Calculate deltas
	oldCPU := resource.MustParse(old.CPU)
	newCPU := resource.MustParse(new.CPU)
	cpuDelta := newCPU.DeepCopy()
	cpuDelta.Sub(oldCPU)

	oldMem := resource.MustParse(old.Memory)
	newMem := resource.MustParse(new.Memory)
	memDelta := newMem.DeepCopy()
	memDelta.Sub(oldMem)

	oldStorage := resource.MustParse(old.Storage)
	newStorage := resource.MustParse(new.Storage)
	storageDelta := newStorage.DeepCopy()
	storageDelta.Sub(oldStorage)

	delta := &ResourceDelta{
		Add:          true,
		CPUDelta:     cpuDelta.String(),
		MemoryDelta:  memDelta.String(),
		StorageDelta: storageDelta.String(),
	}

	if err := s.quotaService.UpdateUsage(ctx, userID, delta); err != nil {
		s.logger.Error("Failed to update quota usage", zap.Error(err))
	}
}

// GetResourceUsage gets current resource usage for an environment
func (s *ResourceService) GetResourceUsage(ctx context.Context, envID string) (*ResourceUsage, error) {
	podNamespace := fmt.Sprintf("devbox-%s", envID)
	podName := fmt.Sprintf("%s-pod", envID)

	// Get pod metrics (in production, use metrics-server)
	pod, err := s.kubeClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	usage := &ResourceUsage{
		CPU:     "0",
		Memory:  "0",
		Storage: "0",
	}

	// Get limits for percentage calculation
	if len(pod.Spec.Containers) > 0 {
		container := pod.Spec.Containers[0]
		if cpu := container.Resources.Limits.Cpu(); cpu != nil {
			usage.CPU = cpu.String()
		}
		if mem := container.Resources.Limits.Memory(); mem != nil {
			usage.Memory = mem.String()
		}
	}

	return usage, nil
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPU            string  `json:"cpu"`
	CPUPercent     float64 `json:"cpuPercent"`
	Memory         string  `json:"memory"`
	MemoryPercent  float64 `json:"memoryPercent"`
	Storage        string  `json:"storage"`
	StoragePercent float64 `json:"storagePercent"`
}
