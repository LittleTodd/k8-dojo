package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetPolDNSBlock scenario: NetworkPolicy blocking DNS.
type NetPolDNSBlock struct {
	BaseScenario
	clientset  *kubernetes.Clientset
	restConfig interface{} // Needed for exec (TODO: refactor to pass config)
}

// Note: We need rest.Config for Exec. For now, we'll verify via Policy check to avoid complexity of Exec in validation loop.
// Exec is expensive and slow.

func NewNetPolDNSBlock(clientset *kubernetes.Clientset) *NetPolDNSBlock {
	return &NetPolDNSBlock{
		BaseScenario: BaseScenario{Namespace: "netpol-dns-block"},
		clientset:    clientset,
	}
}

func (s *NetPolDNSBlock) GetMetadata() Metadata {
	return Metadata{
		ID:          "netpol-dns-block",
		Name:        "Network Security: The Silent Block",
		Description: "The app cannot resolve any domains. A restrictive NetworkPolicy is in place.",
		Difficulty:  DifficultyHard,
		Category:    "Networking",
		Hints:       []string{"Review the NetworkPolicy 'default-deny'", "DNS runs on UDP/TCP port 53", "CoreDNS is in kube-system"},
	}
}

func (s *NetPolDNSBlock) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Test Pod
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "blocked-pod",
			Labels: map[string]string{"app": "secure"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox",
				Command: []string{"sleep", "3600"},
			}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Deny All Egress Policy
	_, err = s.clientset.NetworkingV1().NetworkPolicies(s.Namespace).Create(ctx, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default-deny-egress"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{}, // Empty means deny all
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetPolDNSBlock) Validate(ctx context.Context) Result {
	// Check if any NetworkPolicy allows UDP 53
	pols, err := s.clientset.NetworkingV1().NetworkPolicies(s.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	allowsDNS := false
	for _, pol := range pols.Items {
		for _, egress := range pol.Spec.Egress {
			for _, port := range egress.Ports {
				if port.Port != nil && port.Port.IntVal == 53 {
					allowsDNS = true
				}
			}
			// Or check if it allows all (empty ports)
			if len(egress.Ports) == 0 && len(egress.To) > 0 {
				// Potentially allows all ports to some destination
				allowsDNS = true // Simplified check
			}
		}
	}

	if allowsDNS {
		return Result{Solved: true, Message: "Success! NetworkPolicy now allows DNS traffic."}
	}

	return Result{Solved: false, Message: "No NetworkPolicy rule found explicitly allowing Port 53."}
}

func (s *NetPolDNSBlock) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
