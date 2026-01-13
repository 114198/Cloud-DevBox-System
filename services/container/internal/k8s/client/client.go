// Package client provides Kubernetes client utilities.
package client

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds Kubernetes client configuration
type Config struct {
	// KubeConfig is the path to the kubeconfig file
	// If empty, in-cluster config will be used
	KubeConfig string

	// Namespace is the default namespace to use
	Namespace string

	// QPS is the maximum queries per second to the API server
	QPS float32

	// Burst is the maximum burst for throttle
	Burst int
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		KubeConfig: "",
		Namespace:  "devbox",
		QPS:        100,
		Burst:      200,
	}
}

// NewClient creates a new Kubernetes client
func NewClient(cfg *Config) (kubernetes.Interface, error) {
	var restConfig *rest.Config
	var err error

	if cfg.KubeConfig != "" {
		// Use kubeconfig file
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
	} else if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		// Use KUBECONFIG environment variable
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if home := homeDir(); home != "" {
		// Try default kubeconfig location
		kubeconfig := filepath.Join(home, ".kube", "config")
		if _, statErr := os.Stat(kubeconfig); statErr == nil {
			restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		} else {
			// Fall back to in-cluster config
			restConfig, err = rest.InClusterConfig()
		}
	} else {
		// Use in-cluster config
		restConfig, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
	}

	// Apply rate limiting
	restConfig.QPS = cfg.QPS
	restConfig.Burst = cfg.Burst

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}

// homeDir returns the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

// GetRestConfig returns the REST config for the Kubernetes client
func GetRestConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to default kubeconfig
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return nil, fmt.Errorf("unable to find Kubernetes configuration")
}
