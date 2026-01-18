// Package scenario provides the scenario registry.
package scenario

import (
	"k8s.io/client-go/kubernetes"
)

// Registry holds all available scenarios.
type Registry struct {
	scenarios []Scenario
}

// NewRegistry creates a new scenario registry with all available scenarios.
func NewRegistry(clientset *kubernetes.Clientset) *Registry {
	return &Registry{
		scenarios: []Scenario{
			// Networking
			NewNetServiceSelector(clientset),
			NewNetGrpcBalance(clientset),
			NewNetSourceIP(clientset),
			NewNetDNSNdots(clientset),
			NewNetPolDNSBlock(clientset),

			// Lifecycle
			NewImagePullBackOff(clientset),
			NewLifeCrashConfig(clientset),
			NewLifeGracefulShutdown(clientset),

			// Scheduling
			NewSchedNodeAffinity(clientset),
			NewSchedMissingScheduler(clientset),

			// Security
			NewSecRBACForbidden(clientset),
			NewSecPrivilegedPolicy(clientset),
			NewSecImageDigest(clientset),

			// Storage
			NewStoragePVCPending(clientset),
			NewStorageZonalAffinity(clientset),

			// Ops & Kernel
			NewKernelOOMDisable(clientset),
			NewOpsConfigChecksum(clientset),

			// Batch 3
			NewNetTargetPortMismatch(clientset),
			NewIngressPathError(clientset),
			NewIngressTLSMismatch(clientset),

			NewProbeLivenessFail(clientset),
			NewProbeReadinessTimeout(clientset),
			NewInitContainerCrash(clientset),
			NewPodFinalizerStuck(clientset),

			NewSchedTaintToleration(clientset),

			NewSecFSGroupDenied(clientset),
			NewSecSANoMount(clientset),

			NewStorageSubpathOverwrite(clientset),

			NewResourceQuotaExceeded(clientset),
			NewResourceLimitRange(clientset),
		},
	}
}

// List returns all available scenarios.
func (r *Registry) List() []Scenario {
	return r.scenarios
}

// Get returns a scenario by its ID.
func (r *Registry) Get(id string) Scenario {
	for _, s := range r.scenarios {
		if s.GetMetadata().ID == id {
			return s
		}
	}
	return nil
}

// Count returns the number of available scenarios.
func (r *Registry) Count() int {
	return len(r.scenarios)
}
