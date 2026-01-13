// Package controller implements the DevBox Kubernetes controller.
package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-devbox/services/container/internal/k8s/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// ControllerName is the name of this controller
	ControllerName = "devbox-controller"

	// DevBoxFinalizer is the finalizer for DevBox resources
	DevBoxFinalizer = "devbox.clouddevbox.io/finalizer"

	// DefaultSSHPort is the default SSH port
	DefaultSSHPort = 22

	// DefaultInactivityTimeout is the default inactivity timeout
	DefaultInactivityTimeout = 72 * time.Hour

	// DefaultStoppedTimeout is the default stopped timeout before deletion
	DefaultStoppedTimeout = 7 * 24 * time.Hour

	// ReconcileInterval is the interval between reconciliation loops
	ReconcileInterval = 30 * time.Second
)

// DevBoxController manages DevBox resources
type DevBoxController struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	workqueue  workqueue.RateLimitingInterface
	scheme     *runtime.Scheme

	// Configuration
	namespace       string
	defaultImage    string
	sshPortRange    PortRange
	webPortRange    PortRange
	storageClass    string
	nodeSelector    map[string]string
	tolerations     []corev1.Toleration
}

// PortRange defines a range of ports
type PortRange struct {
	Start int32
	End   int32
}

// ControllerConfig holds controller configuration
type ControllerConfig struct {
	Namespace       string
	DefaultImage    string
	SSHPortStart    int32
	SSHPortEnd      int32
	WebPortStart    int32
	WebPortEnd      int32
	StorageClass    string
	NodeSelector    map[string]string
	Tolerations     []corev1.Toleration
}

// NewDevBoxController creates a new DevBox controller
func NewDevBoxController(
	kubeClient kubernetes.Interface,
	logger *zap.Logger,
	config *ControllerConfig,
) *DevBoxController {
	return &DevBoxController{
		kubeClient: kubeClient,
		logger:     logger.Named(ControllerName),
		workqueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		namespace:  config.Namespace,
		defaultImage: config.DefaultImage,
		sshPortRange: PortRange{
			Start: config.SSHPortStart,
			End:   config.SSHPortEnd,
		},
		webPortRange: PortRange{
			Start: config.WebPortStart,
			End:   config.WebPortEnd,
		},
		storageClass: config.StorageClass,
		nodeSelector: config.NodeSelector,
		tolerations:  config.Tolerations,
	}
}

// Reconcile handles the reconciliation of a DevBox resource
func (c *DevBoxController) Reconcile(ctx context.Context, devbox *types.DevBox) error {
	logger := c.logger.With(
		zap.String("name", devbox.Name),
		zap.String("namespace", devbox.Namespace),
	)

	logger.Info("Reconciling DevBox")

	// Handle deletion
	if !devbox.DeletionTimestamp.IsZero() {
		return c.handleDeletion(ctx, devbox)
	}

	// Ensure finalizer is set
	if !containsString(devbox.Finalizers, DevBoxFinalizer) {
		devbox.Finalizers = append(devbox.Finalizers, DevBoxFinalizer)
		// Update would happen here in a real controller
	}

	// Reconcile based on current phase
	switch devbox.Status.Phase {
	case "", types.DevBoxPhaseCreating:
		return c.reconcileCreating(ctx, devbox)
	case types.DevBoxPhaseRunning:
		return c.reconcileRunning(ctx, devbox)
	case types.DevBoxPhaseStopped:
		return c.reconcileStopped(ctx, devbox)
	case types.DevBoxPhaseSuspended:
		return c.reconcileSuspended(ctx, devbox)
	case types.DevBoxPhaseFailed:
		return c.reconcileFailed(ctx, devbox)
	default:
		logger.Warn("Unknown phase", zap.String("phase", string(devbox.Status.Phase)))
		return nil
	}
}

// reconcileCreating handles DevBox in Creating phase
func (c *DevBoxController) reconcileCreating(ctx context.Context, devbox *types.DevBox) error {
	logger := c.logger.With(zap.String("name", devbox.Name))
	logger.Info("Creating DevBox resources")

	// Create namespace for the DevBox (for isolation)
	if err := c.ensureNamespace(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "NamespaceCreationFailed", err.Error())
	}

	// Create PVC for storage
	if err := c.ensurePVC(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "PVCCreationFailed", err.Error())
	}

	// Create ConfigMap for SSH keys
	if err := c.ensureSSHConfigMap(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "SSHConfigFailed", err.Error())
	}

	// Create the Pod
	if err := c.ensurePod(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "PodCreationFailed", err.Error())
	}

	// Create Service for SSH access
	if err := c.ensureService(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "ServiceCreationFailed", err.Error())
	}

	// Create NetworkPolicy for isolation
	if err := c.ensureNetworkPolicy(ctx, devbox); err != nil {
		return c.setFailed(ctx, devbox, "NetworkPolicyFailed", err.Error())
	}

	// Update status to Running if Pod is ready
	return c.checkAndUpdateStatus(ctx, devbox)
}

// reconcileRunning handles DevBox in Running phase
func (c *DevBoxController) reconcileRunning(ctx context.Context, devbox *types.DevBox) error {
	logger := c.logger.With(zap.String("name", devbox.Name))

	// Check if Pod is still running
	pod, err := c.getPod(ctx, devbox)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Warn("Pod not found, recreating")
			devbox.Status.Phase = types.DevBoxPhaseCreating
			return c.reconcileCreating(ctx, devbox)
		}
		return err
	}

	// Update resource usage
	c.updateResourceUsage(ctx, devbox, pod)

	// Check for auto-stop due to inactivity
	if c.shouldAutoStop(devbox) {
		logger.Info("Auto-stopping due to inactivity")
		return c.stopDevBox(ctx, devbox)
	}

	return nil
}

// reconcileStopped handles DevBox in Stopped phase
func (c *DevBoxController) reconcileStopped(ctx context.Context, devbox *types.DevBox) error {
	logger := c.logger.With(zap.String("name", devbox.Name))

	// Check for auto-delete
	if c.shouldAutoDelete(devbox) {
		logger.Info("Auto-deleting stopped DevBox")
		return c.deleteDevBox(ctx, devbox)
	}

	return nil
}

// reconcileSuspended handles DevBox in Suspended phase
func (c *DevBoxController) reconcileSuspended(ctx context.Context, devbox *types.DevBox) error {
	// Similar to stopped, but may have different policies
	return c.reconcileStopped(ctx, devbox)
}

// reconcileFailed handles DevBox in Failed phase
func (c *DevBoxController) reconcileFailed(ctx context.Context, devbox *types.DevBox) error {
	// Log the failure and potentially clean up resources
	c.logger.Error("DevBox in failed state",
		zap.String("name", devbox.Name),
		zap.String("message", devbox.Status.Message),
	)
	return nil
}

// handleDeletion handles the deletion of a DevBox
func (c *DevBoxController) handleDeletion(ctx context.Context, devbox *types.DevBox) error {
	logger := c.logger.With(zap.String("name", devbox.Name))
	logger.Info("Handling DevBox deletion")

	// Delete all associated resources
	if err := c.deleteResources(ctx, devbox); err != nil {
		return err
	}

	// Remove finalizer
	devbox.Finalizers = removeString(devbox.Finalizers, DevBoxFinalizer)
	return nil
}

// ensureNamespace creates the namespace for the DevBox
func (c *DevBoxController) ensureNamespace(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/owner":    devbox.Spec.UserID,
				"devbox.clouddevbox.io/devbox":   devbox.Name,
				"devbox.clouddevbox.io/managed":  "true",
			},
		},
	}

	_, err := c.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	return nil
}

// ensurePVC creates the PersistentVolumeClaim for storage
func (c *DevBoxController) ensurePVC(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)
	pvcName := fmt.Sprintf("%s-storage", devbox.Name)

	storageSize := "10Gi"
	if devbox.Spec.Resources != nil && devbox.Spec.Resources.Storage != "" {
		storageSize = devbox.Spec.Resources.Storage
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/devbox": devbox.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}

	if c.storageClass != "" {
		pvc.Spec.StorageClassName = &c.storageClass
	}

	_, err := c.kubeClient.CoreV1().PersistentVolumeClaims(nsName).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	return nil
}

// ensureSSHConfigMap creates the ConfigMap for SSH configuration
func (c *DevBoxController) ensureSSHConfigMap(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)
	cmName := fmt.Sprintf("%s-ssh", devbox.Name)

	publicKey := ""
	if devbox.Spec.SSH != nil {
		publicKey = devbox.Spec.SSH.PublicKey
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/devbox": devbox.Name,
			},
		},
		Data: map[string]string{
			"authorized_keys": publicKey,
		},
	}

	_, err := c.kubeClient.CoreV1().ConfigMaps(nsName).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create SSH ConfigMap: %w", err)
	}

	return nil
}

// ensurePod creates the Pod for the DevBox
func (c *DevBoxController) ensurePod(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)
	podName := fmt.Sprintf("%s-pod", devbox.Name)
	pvcName := fmt.Sprintf("%s-storage", devbox.Name)
	sshCMName := fmt.Sprintf("%s-ssh", devbox.Name)

	// Determine image
	image := c.defaultImage
	if devbox.Spec.Runtime != nil && devbox.Spec.Runtime.Image != "" {
		image = devbox.Spec.Runtime.Image
	}

	// Resource limits
	cpuLimit := "1"
	memoryLimit := "2Gi"
	if devbox.Spec.Resources != nil {
		if devbox.Spec.Resources.CPU != "" {
			cpuLimit = devbox.Spec.Resources.CPU
		}
		if devbox.Spec.Resources.Memory != "" {
			memoryLimit = devbox.Spec.Resources.Memory
		}
	}

	// Build environment variables
	envVars := []corev1.EnvVar{
		{Name: "DEVBOX_ID", Value: devbox.Name},
		{Name: "DEVBOX_USER_ID", Value: devbox.Spec.UserID},
	}
	for _, env := range devbox.Spec.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}

	// Build container ports
	containerPorts := []corev1.ContainerPort{
		{Name: "ssh", ContainerPort: 22, Protocol: corev1.ProtocolTCP},
	}
	for _, port := range devbox.Spec.Ports {
		protocol := corev1.ProtocolTCP
		if port.Protocol == "UDP" {
			protocol = corev1.ProtocolUDP
		}
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          port.Name,
			ContainerPort: port.ContainerPort,
			Protocol:      protocol,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/devbox": devbox.Name,
				"devbox.clouddevbox.io/owner":  devbox.Spec.UserID,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "devbox",
					Image: image,
					Ports: containerPorts,
					Env:   envVars,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuLimit),
							corev1.ResourceMemory: resource.MustParse(memoryLimit),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuLimit),
							corev1.ResourceMemory: resource.MustParse(memoryLimit),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
						{
							Name:      "ssh-keys",
							MountPath: "/home/devbox/.ssh",
							ReadOnly:  true,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:  int64Ptr(1000),
						RunAsGroup: int64Ptr(1000),
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
				{
					Name: "ssh-keys",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: sshCMName,
							},
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
			NodeSelector:  c.nodeSelector,
			Tolerations:   c.tolerations,
		},
	}

	_, err := c.kubeClient.CoreV1().Pods(nsName).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Pod: %w", err)
	}

	return nil
}

// ensureService creates the Service for SSH access
func (c *DevBoxController) ensureService(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)
	svcName := fmt.Sprintf("%s-svc", devbox.Name)

	// Build service ports
	servicePorts := []corev1.ServicePort{
		{
			Name:       "ssh",
			Port:       22,
			TargetPort: intstr.FromInt(22),
			Protocol:   corev1.ProtocolTCP,
		},
	}
	for _, port := range devbox.Spec.Ports {
		protocol := corev1.ProtocolTCP
		if port.Protocol == "UDP" {
			protocol = corev1.ProtocolUDP
		}
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       port.Name,
			Port:       port.ContainerPort,
			TargetPort: intstr.FromInt(int(port.ContainerPort)),
			Protocol:   protocol,
		})
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/devbox": devbox.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"devbox.clouddevbox.io/devbox": devbox.Name,
			},
			Ports: servicePorts,
		},
	}

	_, err := c.kubeClient.CoreV1().Services(nsName).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Service: %w", err)
	}

	return nil
}

// ensureNetworkPolicy creates the NetworkPolicy for isolation
func (c *DevBoxController) ensureNetworkPolicy(ctx context.Context, devbox *types.DevBox) error {
	// NetworkPolicy implementation will be in the network isolation task
	return nil
}

// getPod retrieves the Pod for a DevBox
func (c *DevBoxController) getPod(ctx context.Context, devbox *types.DevBox) (*corev1.Pod, error) {
	nsName := c.getDevBoxNamespace(devbox)
	podName := fmt.Sprintf("%s-pod", devbox.Name)
	return c.kubeClient.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
}

// checkAndUpdateStatus checks Pod status and updates DevBox status
func (c *DevBoxController) checkAndUpdateStatus(ctx context.Context, devbox *types.DevBox) error {
	pod, err := c.getPod(ctx, devbox)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Pod not created yet
		}
		return err
	}

	// Update status based on Pod phase
	switch pod.Status.Phase {
	case corev1.PodRunning:
		devbox.Status.Phase = types.DevBoxPhaseRunning
		devbox.Status.PodName = pod.Name
		devbox.Status.PodIP = pod.Status.PodIP
		now := metav1.Now()
		devbox.Status.StartedAt = &now
		devbox.Status.LastActivityTime = &now

		// Get Service to find NodePort
		svc, err := c.getService(ctx, devbox)
		if err == nil {
			for _, port := range svc.Spec.Ports {
				if port.Name == "ssh" {
					devbox.Status.SSHPort = port.NodePort
				}
			}
		}

		c.setCondition(devbox, types.DevBoxConditionReady, metav1.ConditionTrue, "PodRunning", "DevBox is running")
		c.setCondition(devbox, types.DevBoxConditionPodReady, metav1.ConditionTrue, "PodRunning", "Pod is running")

	case corev1.PodPending:
		devbox.Status.Phase = types.DevBoxPhaseCreating
		c.setCondition(devbox, types.DevBoxConditionReady, metav1.ConditionFalse, "PodPending", "Pod is pending")

	case corev1.PodFailed:
		devbox.Status.Phase = types.DevBoxPhaseFailed
		devbox.Status.Message = "Pod failed to start"
		c.setCondition(devbox, types.DevBoxConditionReady, metav1.ConditionFalse, "PodFailed", "Pod failed")
	}

	return nil
}

// getService retrieves the Service for a DevBox
func (c *DevBoxController) getService(ctx context.Context, devbox *types.DevBox) (*corev1.Service, error) {
	nsName := c.getDevBoxNamespace(devbox)
	svcName := fmt.Sprintf("%s-svc", devbox.Name)
	return c.kubeClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
}

// updateResourceUsage updates the resource usage in status
func (c *DevBoxController) updateResourceUsage(ctx context.Context, devbox *types.DevBox, pod *corev1.Pod) {
	// In a real implementation, this would query metrics server
	// For now, we just set placeholder values
	if devbox.Status.ResourceUsage == nil {
		devbox.Status.ResourceUsage = &types.ResourceUsageStatus{}
	}
}

// shouldAutoStop checks if the DevBox should be auto-stopped
func (c *DevBoxController) shouldAutoStop(devbox *types.DevBox) bool {
	if devbox.Spec.AutoStop == nil || !devbox.Spec.AutoStop.Enabled {
		return false
	}

	if devbox.Status.LastActivityTime == nil {
		return false
	}

	timeout := DefaultInactivityTimeout
	if devbox.Spec.AutoStop.InactivityTimeout != "" {
		if parsed, err := time.ParseDuration(devbox.Spec.AutoStop.InactivityTimeout); err == nil {
			timeout = parsed
		}
	}

	return time.Since(devbox.Status.LastActivityTime.Time) > timeout
}

// shouldAutoDelete checks if the DevBox should be auto-deleted
func (c *DevBoxController) shouldAutoDelete(devbox *types.DevBox) bool {
	if devbox.Spec.AutoDelete == nil || !devbox.Spec.AutoDelete.Enabled {
		return false
	}

	if devbox.Status.StoppedAt == nil {
		return false
	}

	timeout := DefaultStoppedTimeout
	if devbox.Spec.AutoDelete.StoppedTimeout != "" {
		if parsed, err := time.ParseDuration(devbox.Spec.AutoDelete.StoppedTimeout); err == nil {
			timeout = parsed
		}
	}

	return time.Since(devbox.Status.StoppedAt.Time) > timeout
}

// stopDevBox stops a running DevBox
func (c *DevBoxController) stopDevBox(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)
	podName := fmt.Sprintf("%s-pod", devbox.Name)

	err := c.kubeClient.CoreV1().Pods(nsName).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Pod: %w", err)
	}

	devbox.Status.Phase = types.DevBoxPhaseStopped
	now := metav1.Now()
	devbox.Status.StoppedAt = &now
	devbox.Status.PodIP = ""
	devbox.Status.PodName = ""

	c.setCondition(devbox, types.DevBoxConditionReady, metav1.ConditionFalse, "Stopped", "DevBox is stopped")

	return nil
}

// deleteDevBox deletes a DevBox and all its resources
func (c *DevBoxController) deleteDevBox(ctx context.Context, devbox *types.DevBox) error {
	return c.deleteResources(ctx, devbox)
}

// deleteResources deletes all resources associated with a DevBox
func (c *DevBoxController) deleteResources(ctx context.Context, devbox *types.DevBox) error {
	nsName := c.getDevBoxNamespace(devbox)

	// Delete the entire namespace (which will delete all resources in it)
	err := c.kubeClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// setFailed sets the DevBox to failed state
func (c *DevBoxController) setFailed(ctx context.Context, devbox *types.DevBox, reason, message string) error {
	devbox.Status.Phase = types.DevBoxPhaseFailed
	devbox.Status.Message = message
	c.setCondition(devbox, types.DevBoxConditionReady, metav1.ConditionFalse, reason, message)
	return fmt.Errorf("%s: %s", reason, message)
}

// setCondition sets a condition on the DevBox
func (c *DevBoxController) setCondition(devbox *types.DevBox, condType types.DevBoxConditionType, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	condition := types.DevBoxCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	found := false
	for i, c := range devbox.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				devbox.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}
	if !found {
		devbox.Status.Conditions = append(devbox.Status.Conditions, condition)
	}
}

// getDevBoxNamespace returns the namespace for a DevBox
func (c *DevBoxController) getDevBoxNamespace(devbox *types.DevBox) string {
	return fmt.Sprintf("devbox-%s", devbox.Name)
}

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func int64Ptr(i int64) *int64 {
	return &i
}

// OnAdd handles DevBox add events
func (c *DevBoxController) OnAdd(obj interface{}, isInInitialList bool) {
	c.enqueue(obj)
}

// OnUpdate handles DevBox update events
func (c *DevBoxController) OnUpdate(oldObj, newObj interface{}) {
	c.enqueue(newObj)
}

// OnDelete handles DevBox delete events
func (c *DevBoxController) OnDelete(obj interface{}) {
	c.enqueue(obj)
}

func (c *DevBoxController) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error("Failed to get key for object", zap.Error(err))
		return
	}
	c.workqueue.Add(key)
}
