// Package services provides business logic for the container service.
package services

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// NetworkService handles network isolation and policies
type NetworkService struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	namespace  string
}

// NetworkServiceConfig holds configuration for the network service
type NetworkServiceConfig struct {
	Namespace string
}

// NewNetworkService creates a new network service
func NewNetworkService(
	kubeClient kubernetes.Interface,
	logger *zap.Logger,
	config *NetworkServiceConfig,
) *NetworkService {
	return &NetworkService{
		kubeClient: kubeClient,
		logger:     logger.Named("network-service"),
		namespace:  config.Namespace,
	}
}

// CreateIsolatedNamespace creates a namespace with network isolation for a DevBox
func (s *NetworkService) CreateIsolatedNamespace(ctx context.Context, envID, userID string) error {
	logger := s.logger.With(zap.String("envId", envID), zap.String("userId", userID))
	logger.Info("Creating isolated namespace")

	nsName := fmt.Sprintf("devbox-%s", envID)

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"devbox.clouddevbox.io/owner":     userID,
				"devbox.clouddevbox.io/devbox":    envID,
				"devbox.clouddevbox.io/managed":   "true",
				"devbox.clouddevbox.io/isolation": "enabled",
			},
			Annotations: map[string]string{
				"devbox.clouddevbox.io/created-by": "network-service",
			},
		},
	}

	_, err := s.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create default deny-all network policy
	if err := s.createDefaultDenyPolicy(ctx, nsName); err != nil {
		return err
	}

	// Create allow egress policy for internet access
	if err := s.createEgressPolicy(ctx, nsName); err != nil {
		return err
	}

	// Create allow ingress policy for SSH and web ports
	if err := s.createIngressPolicy(ctx, nsName, envID); err != nil {
		return err
	}

	logger.Info("Isolated namespace created successfully")
	return nil
}

// createDefaultDenyPolicy creates a default deny-all network policy
func (s *NetworkService) createDefaultDenyPolicy(ctx context.Context, namespace string) error {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-all",
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
				"devbox.clouddevbox.io/policy":  "default-deny",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Apply to all pods in the namespace
			PodSelector: metav1.LabelSelector{},
			// Deny all ingress and egress by default
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Empty ingress and egress rules = deny all
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress:  []networkingv1.NetworkPolicyEgressRule{},
		},
	}

	_, err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create default deny policy: %w", err)
	}

	return nil
}

// createEgressPolicy creates a policy allowing egress to the internet
func (s *NetworkService) createEgressPolicy(ctx context.Context, namespace string) error {
	// Allow DNS resolution
	dnsPort := intstr.FromInt(53)
	
	// Allow HTTPS
	httpsPort := intstr.FromInt(443)
	httpPort := intstr.FromInt(80)

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-egress",
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
				"devbox.clouddevbox.io/policy":  "allow-egress",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				// Allow DNS (UDP and TCP)
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: protocolPtr(corev1.ProtocolUDP),
							Port:     &dnsPort,
						},
						{
							Protocol: protocolPtr(corev1.ProtocolTCP),
							Port:     &dnsPort,
						},
					},
				},
				// Allow HTTP/HTTPS to external
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: protocolPtr(corev1.ProtocolTCP),
							Port:     &httpPort,
						},
						{
							Protocol: protocolPtr(corev1.ProtocolTCP),
							Port:     &httpsPort,
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							// Allow to any IP except private ranges
							IPBlock: &networkingv1.IPBlock{
								CIDR: "0.0.0.0/0",
								Except: []string{
									"10.0.0.0/8",
									"172.16.0.0/12",
									"192.168.0.0/16",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create egress policy: %w", err)
	}

	return nil
}

// createIngressPolicy creates a policy allowing ingress for SSH and web ports
func (s *NetworkService) createIngressPolicy(ctx context.Context, namespace, envID string) error {
	sshPort := intstr.FromInt(22)

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-ingress",
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
				"devbox.clouddevbox.io/policy":  "allow-ingress",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"devbox.clouddevbox.io/devbox": envID,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				// Allow SSH from gateway namespace
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"devbox.clouddevbox.io/component": "gateway",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: protocolPtr(corev1.ProtocolTCP),
							Port:     &sshPort,
						},
					},
				},
				// Allow web traffic from ingress controller
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "ingress-nginx",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ingress policy: %w", err)
	}

	return nil
}

func protocolPtr(p corev1.Protocol) *corev1.Protocol {
	return &p
}

// AddPortAccess adds ingress access for a specific port
func (s *NetworkService) AddPortAccess(ctx context.Context, namespace, envID string, port int32, protocol string) error {
	policyName := fmt.Sprintf("allow-port-%d", port)
	portVal := intstr.FromInt(int(port))

	var proto corev1.Protocol
	if protocol == "UDP" {
		proto = corev1.ProtocolUDP
	} else {
		proto = corev1.ProtocolTCP
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed": "true",
				"devbox.clouddevbox.io/policy":  "custom-port",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"devbox.clouddevbox.io/devbox": envID,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &proto,
							Port:     &portVal,
						},
					},
				},
			},
		},
	}

	_, err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create port access policy: %w", err)
	}

	return nil
}

// RemovePortAccess removes ingress access for a specific port
func (s *NetworkService) RemovePortAccess(ctx context.Context, namespace string, port int32) error {
	policyName := fmt.Sprintf("allow-port-%d", port)

	err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, policyName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete port access policy: %w", err)
	}

	return nil
}

// AllowCrossNamespaceAccess allows access between two DevBox namespaces (for collaboration)
func (s *NetworkService) AllowCrossNamespaceAccess(ctx context.Context, sourceEnvID, targetEnvID string) error {
	sourceNS := fmt.Sprintf("devbox-%s", sourceEnvID)
	targetNS := fmt.Sprintf("devbox-%s", targetEnvID)

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("allow-from-%s", sourceEnvID),
			Namespace: targetNS,
			Labels: map[string]string{
				"devbox.clouddevbox.io/managed":     "true",
				"devbox.clouddevbox.io/policy":      "cross-namespace",
				"devbox.clouddevbox.io/source-env":  sourceEnvID,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"devbox.clouddevbox.io/devbox": sourceEnvID,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := s.kubeClient.NetworkingV1().NetworkPolicies(targetNS).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cross-namespace policy: %w", err)
	}

	// Create reverse policy
	reversePolicy := policy.DeepCopy()
	reversePolicy.Name = fmt.Sprintf("allow-from-%s", targetEnvID)
	reversePolicy.Namespace = sourceNS
	reversePolicy.Labels["devbox.clouddevbox.io/source-env"] = targetEnvID
	reversePolicy.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["devbox.clouddevbox.io/devbox"] = targetEnvID

	_, err = s.kubeClient.NetworkingV1().NetworkPolicies(sourceNS).Create(ctx, reversePolicy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create reverse cross-namespace policy: %w", err)
	}

	return nil
}

// RevokeCrossNamespaceAccess revokes access between two DevBox namespaces
func (s *NetworkService) RevokeCrossNamespaceAccess(ctx context.Context, sourceEnvID, targetEnvID string) error {
	sourceNS := fmt.Sprintf("devbox-%s", sourceEnvID)
	targetNS := fmt.Sprintf("devbox-%s", targetEnvID)

	// Delete policy in target namespace
	err := s.kubeClient.NetworkingV1().NetworkPolicies(targetNS).Delete(
		ctx,
		fmt.Sprintf("allow-from-%s", sourceEnvID),
		metav1.DeleteOptions{},
	)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete cross-namespace policy: %w", err)
	}

	// Delete reverse policy
	err = s.kubeClient.NetworkingV1().NetworkPolicies(sourceNS).Delete(
		ctx,
		fmt.Sprintf("allow-from-%s", targetEnvID),
		metav1.DeleteOptions{},
	)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete reverse cross-namespace policy: %w", err)
	}

	return nil
}

// DeleteNamespaceNetworkPolicies deletes all network policies in a namespace
func (s *NetworkService) DeleteNamespaceNetworkPolicies(ctx context.Context, namespace string) error {
	err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).DeleteCollection(
		ctx,
		metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: "devbox.clouddevbox.io/managed=true",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete network policies: %w", err)
	}

	return nil
}

// GetNetworkPolicies lists all network policies for a DevBox
func (s *NetworkService) GetNetworkPolicies(ctx context.Context, envID string) ([]NetworkPolicyInfo, error) {
	namespace := fmt.Sprintf("devbox-%s", envID)

	policies, err := s.kubeClient.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "devbox.clouddevbox.io/managed=true",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list network policies: %w", err)
	}

	var result []NetworkPolicyInfo
	for _, policy := range policies.Items {
		info := NetworkPolicyInfo{
			Name:        policy.Name,
			PolicyType:  policy.Labels["devbox.clouddevbox.io/policy"],
			IngressRules: len(policy.Spec.Ingress),
			EgressRules:  len(policy.Spec.Egress),
		}
		result = append(result, info)
	}

	return result, nil
}

// NetworkPolicyInfo represents information about a network policy
type NetworkPolicyInfo struct {
	Name         string `json:"name"`
	PolicyType   string `json:"policyType"`
	IngressRules int    `json:"ingressRules"`
	EgressRules  int    `json:"egressRules"`
}

// VerifyIsolation verifies that two environments are properly isolated
func (s *NetworkService) VerifyIsolation(ctx context.Context, envID1, envID2 string) (bool, error) {
	ns1 := fmt.Sprintf("devbox-%s", envID1)
	ns2 := fmt.Sprintf("devbox-%s", envID2)

	// Check if there are any cross-namespace policies
	policies1, err := s.kubeClient.NetworkingV1().NetworkPolicies(ns1).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("devbox.clouddevbox.io/source-env=%s", envID2),
	})
	if err != nil {
		return false, err
	}

	policies2, err := s.kubeClient.NetworkingV1().NetworkPolicies(ns2).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("devbox.clouddevbox.io/source-env=%s", envID1),
	})
	if err != nil {
		return false, err
	}

	// If no cross-namespace policies exist, environments are isolated
	return len(policies1.Items) == 0 && len(policies2.Items) == 0, nil
}
