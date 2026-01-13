// Package services provides business logic for the container service.
package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// setupTestNetworkService creates a test network service with a fake Kubernetes client
func setupTestNetworkService(t *testing.T) *NetworkService {
	logger, _ := zap.NewDevelopment()
	fakeClient := fake.NewSimpleClientset()
	config := &NetworkServiceConfig{
		Namespace: "test-devbox",
	}
	return NewNetworkService(fakeClient, logger, config)
}

func TestNetworkService_CreateIsolatedNamespace(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	t.Run("creates namespace with isolation", func(t *testing.T) {
		err := service.CreateIsolatedNamespace(ctx, "env-123", "user-456")
		require.NoError(t, err)

		// Verify namespace was created
		ns, err := service.kubeClient.CoreV1().Namespaces().Get(ctx, "devbox-env-123", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "user-456", ns.Labels["devbox.clouddevbox.io/owner"])
		assert.Equal(t, "env-123", ns.Labels["devbox.clouddevbox.io/devbox"])
		assert.Equal(t, "enabled", ns.Labels["devbox.clouddevbox.io/isolation"])
	})

	t.Run("creates network policies", func(t *testing.T) {
		err := service.CreateIsolatedNamespace(ctx, "env-789", "user-abc")
		require.NoError(t, err)

		// Verify network policies were created
		policies, err := service.kubeClient.NetworkingV1().NetworkPolicies("devbox-env-789").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(policies.Items), 2, "Should have at least 2 network policies")
	})
}

func TestNetworkService_AddPortAccess(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// First create the namespace
	err := service.CreateIsolatedNamespace(ctx, "port-test-env", "user-123")
	require.NoError(t, err)

	t.Run("adds port access policy", func(t *testing.T) {
		err := service.AddPortAccess(ctx, "devbox-port-test-env", "port-test-env", 8080, "TCP")
		require.NoError(t, err)

		// Verify policy was created
		policy, err := service.kubeClient.NetworkingV1().NetworkPolicies("devbox-port-test-env").Get(ctx, "allow-port-8080", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "custom-port", policy.Labels["devbox.clouddevbox.io/policy"])
	})
}

func TestNetworkService_RemovePortAccess(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// Create namespace and add port
	err := service.CreateIsolatedNamespace(ctx, "remove-port-env", "user-123")
	require.NoError(t, err)
	err = service.AddPortAccess(ctx, "devbox-remove-port-env", "remove-port-env", 3000, "TCP")
	require.NoError(t, err)

	t.Run("removes port access policy", func(t *testing.T) {
		err := service.RemovePortAccess(ctx, "devbox-remove-port-env", 3000)
		require.NoError(t, err)

		// Verify policy was deleted
		_, err = service.kubeClient.NetworkingV1().NetworkPolicies("devbox-remove-port-env").Get(ctx, "allow-port-3000", metav1.GetOptions{})
		assert.Error(t, err, "Policy should be deleted")
	})
}

func TestNetworkService_CrossNamespaceAccess(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// Create two namespaces
	err := service.CreateIsolatedNamespace(ctx, "collab-env-1", "user-1")
	require.NoError(t, err)
	err = service.CreateIsolatedNamespace(ctx, "collab-env-2", "user-2")
	require.NoError(t, err)

	t.Run("allows cross-namespace access", func(t *testing.T) {
		err := service.AllowCrossNamespaceAccess(ctx, "collab-env-1", "collab-env-2")
		require.NoError(t, err)

		// Verify policies were created in both namespaces
		policy1, err := service.kubeClient.NetworkingV1().NetworkPolicies("devbox-collab-env-2").Get(ctx, "allow-from-collab-env-1", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "cross-namespace", policy1.Labels["devbox.clouddevbox.io/policy"])

		policy2, err := service.kubeClient.NetworkingV1().NetworkPolicies("devbox-collab-env-1").Get(ctx, "allow-from-collab-env-2", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "cross-namespace", policy2.Labels["devbox.clouddevbox.io/policy"])
	})

	t.Run("revokes cross-namespace access", func(t *testing.T) {
		err := service.RevokeCrossNamespaceAccess(ctx, "collab-env-1", "collab-env-2")
		require.NoError(t, err)

		// Verify policies were deleted
		_, err = service.kubeClient.NetworkingV1().NetworkPolicies("devbox-collab-env-2").Get(ctx, "allow-from-collab-env-1", metav1.GetOptions{})
		assert.Error(t, err, "Policy should be deleted")

		_, err = service.kubeClient.NetworkingV1().NetworkPolicies("devbox-collab-env-1").Get(ctx, "allow-from-collab-env-2", metav1.GetOptions{})
		assert.Error(t, err, "Policy should be deleted")
	})
}

// **Feature: cloud-devbox, Property 3: 资源和网络完全隔离 (Network Isolation)**
// **Validates: Requirements 7.4**
func TestProperty3_NetworkIsolation(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// Run 100 iterations as per property testing requirements
	const iterations = 100

	// Create multiple isolated environments
	for i := 0; i < iterations; i++ {
		envID := "isolation-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		userID := "user-" + string(rune('0'+i%10))

		err := service.CreateIsolatedNamespace(ctx, envID, userID)
		require.NoError(t, err)
	}

	// Property: Each pair of environments should be isolated by default
	for i := 0; i < iterations-1; i++ {
		envID1 := "isolation-test-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		envID2 := "isolation-test-" + string(rune('a'+(i+1)%26)) + string(rune('0'+(i+1)/26))

		// Verify isolation between consecutive environments
		isolated, err := service.VerifyIsolation(ctx, envID1, envID2)
		require.NoError(t, err)
		assert.True(t, isolated, "Environments %s and %s should be isolated", envID1, envID2)
	}

	t.Logf("Property 3 (Network): Verified network isolation for %d environments", iterations)
}

// TestNetworkService_GetNetworkPolicies tests listing network policies
func TestNetworkService_GetNetworkPolicies(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// Create namespace with policies
	err := service.CreateIsolatedNamespace(ctx, "list-policies-env", "user-123")
	require.NoError(t, err)

	// Add custom port
	err = service.AddPortAccess(ctx, "devbox-list-policies-env", "list-policies-env", 5000, "TCP")
	require.NoError(t, err)

	t.Run("lists all policies", func(t *testing.T) {
		policies, err := service.GetNetworkPolicies(ctx, "list-policies-env")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(policies), 3, "Should have at least 3 policies")

		// Verify policy types
		policyTypes := make(map[string]bool)
		for _, p := range policies {
			policyTypes[p.PolicyType] = true
		}
		assert.True(t, policyTypes["default-deny"] || policyTypes["allow-egress"] || policyTypes["allow-ingress"],
			"Should have standard policy types")
	})
}

// TestNetworkService_DeleteNamespaceNetworkPolicies tests deleting all policies
func TestNetworkService_DeleteNamespaceNetworkPolicies(t *testing.T) {
	service := setupTestNetworkService(t)
	ctx := context.Background()

	// Create namespace with policies
	err := service.CreateIsolatedNamespace(ctx, "delete-policies-env", "user-123")
	require.NoError(t, err)

	t.Run("deletes all managed policies", func(t *testing.T) {
		err := service.DeleteNamespaceNetworkPolicies(ctx, "devbox-delete-policies-env")
		require.NoError(t, err)

		// Verify policies were deleted
		policies, err := service.kubeClient.NetworkingV1().NetworkPolicies("devbox-delete-policies-env").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, policies.Items, "All policies should be deleted")
	})
}
