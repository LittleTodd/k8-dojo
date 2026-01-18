package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetDNSNdots scenario: DNS latency due to ndots.
type NetDNSNdots struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewNetDNSNdots(clientset *kubernetes.Clientset) *NetDNSNdots {
	return &NetDNSNdots{
		BaseScenario: BaseScenario{Namespace: "net-dns-ndots"},
		clientset:    clientset,
	}
}

func (s *NetDNSNdots) GetMetadata() Metadata {
	return Metadata{
		ID:          "net-dns-ndots",
		Name:        "Network: DNS 5s Latency",
		Description: "External domain lookups have high latency. Optimize the DNS configuration for a Pod that mostly accesses external FQDNs.",
		Difficulty:  DifficultyHard,
		Category:    "Networking",
		Hints:       []string{"Default ndots is 5", "Check /etc/resolv.conf inside pod", "Set dnsConfig in Pod spec"},
	}
}

func (s *NetDNSNdots) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Pod with default configuration
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "legacy-app"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetDNSNdots) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "legacy-app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Spec.DNSConfig != nil {
		for _, opt := range pod.Spec.DNSConfig.Options {
			if opt.Name == "ndots" && opt.Value != nil {
				val := *opt.Value
				if val < "3" {
					return Result{Solved: true, Message: "Success! ndots reduced to optimized level."}
				}
			}
		}
	}

	return Result{Solved: false, Message: "ndots configuration not found or value too high."}
}

func (s *NetDNSNdots) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
