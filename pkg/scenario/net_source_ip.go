package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetSourceIP scenario: Preserve client source IP (externalTrafficPolicy: Local).
type NetSourceIP struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewNetSourceIP(clientset *kubernetes.Clientset) *NetSourceIP {
	return &NetSourceIP{
		BaseScenario: BaseScenario{Namespace: "net-source-ip"},
		clientset:    clientset,
	}
}

func (s *NetSourceIP) GetMetadata() Metadata {
	return Metadata{
		ID:          "net-source-ip",
		Name:        "Network: The Vanishing Source IP",
		Description: "The backend sees all traffic coming from Node IPs instead of real client IPs. Fix it.",
		Difficulty:  DifficultyMedium,
		Category:    "Networking",
		Hints:       []string{"Traffic is being SNATed", "Check externalTrafficPolicy in Service spec"},
	}
}

func (s *NetSourceIP) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// NodePort Service with Default Policy (Cluster)
	_, err = s.clientset.CoreV1().Services(s.Namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "public-service"},
		Spec: corev1.ServiceSpec{
			Selector:              map[string]string{"app": "public"},
			Ports:                 []corev1.ServicePort{{Port: 80}},
			Type:                  corev1.ServiceTypeNodePort,
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetSourceIP) Validate(ctx context.Context) Result {
	svc, err := s.clientset.CoreV1().Services(s.Namespace).Get(ctx, "public-service", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
		return Result{Solved: true, Message: "Success! ExternalTrafficPolicy is set to Local."}
	}

	return Result{Solved: false, Message: "Policy is still set to Cluster (SNAT enabled)."}
}

func (s *NetSourceIP) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
