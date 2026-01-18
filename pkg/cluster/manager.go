// Package cluster provides Kubernetes cluster management using Kind.
package cluster

import (
	"fmt"

	"sigs.k8s.io/kind/pkg/cluster"
)

const (
	// ClusterName is the name of the Kind cluster used by k8s-dojo.
	ClusterName = "k8s-dojo"
)

// Manager handles Kind cluster lifecycle operations.
type Manager struct {
	provider *cluster.Provider
}

// NewManager creates a new cluster Manager.
func NewManager() *Manager {
	return &Manager{
		provider: cluster.NewProvider(),
	}
}

// ClusterExists checks if the k8s-dojo cluster already exists.
func (m *Manager) ClusterExists() (bool, error) {
	clusters, err := m.provider.List()
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, c := range clusters {
		if c == ClusterName {
			return true, nil
		}
	}
	return false, nil
}

// EnsureCluster creates the cluster if it doesn't exist, using the specified version.
// Returns the kubeconfig as a string (in-memory, not written to disk).
func (m *Manager) EnsureCluster(version SupportedVersion) (string, error) {
	exists, err := m.ClusterExists()
	if err != nil {
		return "", err
	}

	if !exists {
		fmt.Printf("Creating cluster %s with Kubernetes %s...\n", ClusterName, version.Version)
		err = m.provider.Create(
			ClusterName,
			cluster.CreateWithNodeImage(version.NodeImage),
			cluster.CreateWithWaitForReady(0), // Wait indefinitely for cluster to be ready
		)
		if err != nil {
			return "", fmt.Errorf("failed to create cluster: %w", err)
		}
		fmt.Println("Cluster created successfully!")
	} else {
		fmt.Printf("Cluster %s already exists.\n", ClusterName)
	}

	// Get kubeconfig (in-memory)
	kubeconfig, err := m.provider.KubeConfig(ClusterName, false)
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

// DeleteCluster removes the k8s-dojo cluster.
func (m *Manager) DeleteCluster() error {
	exists, err := m.ClusterExists()
	if err != nil {
		return err
	}

	if !exists {
		return nil // Nothing to delete
	}

	fmt.Printf("Deleting cluster %s...\n", ClusterName)
	return m.provider.Delete(ClusterName, "")
}
