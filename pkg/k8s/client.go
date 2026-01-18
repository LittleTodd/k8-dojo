// Package k8s provides Kubernetes client functionality.
package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset with helper methods.
type Client struct {
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

// NewClientFromKubeconfig creates a new Client from an in-memory kubeconfig string.
func NewClientFromKubeconfig(kubeconfig string) (*Client, error) {
	// Parse kubeconfig from string
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Create clientset
	// Increase rate limits to prevent "client-side throttling" logs and UI lag
	config.QPS = 50.0
	config.Burst = 100

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		Clientset: clientset,
		Config:    config,
	}, nil
}

// GetServerVersion returns the Kubernetes server version string.
func (c *Client) GetServerVersion() (string, error) {
	version, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get server version: %w", err)
	}
	return version.GitVersion, nil
}
