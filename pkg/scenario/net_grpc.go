package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetGrpcBalance scenario: Standard Service for gRPC (needs Headless).
type NetGrpcBalance struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewNetGrpcBalance(clientset *kubernetes.Clientset) *NetGrpcBalance {
	return &NetGrpcBalance{
		BaseScenario: BaseScenario{Namespace: "net-grpc-balance"},
		clientset:    clientset,
	}
}

func (s *NetGrpcBalance) GetMetadata() Metadata {
	return Metadata{
		ID:          "net-grpc-balance",
		Name:        "Network: gRPC Load Balancing",
		Description: "gRPC traffic is unevenly distributed. Implement Client-side Load Balancing by converting the Service to Headless.",
		Difficulty:  DifficultyMedium,
		Category:    "Networking",
		Hints:       []string{"gRPC over HTTP/2 reuses connections", "Standard ClusterIP does L4 balancing", "Set clusterIP to None"},
	}
}

func (s *NetGrpcBalance) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Standard Service (wrong for gRPC LB)
	_, err = s.clientset.CoreV1().Services(s.Namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "grpc-service"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "grpc-server"},
			Ports:    []corev1.ServicePort{{Port: 50051}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetGrpcBalance) Validate(ctx context.Context) Result {
	svc, err := s.clientset.CoreV1().Services(s.Namespace).Get(ctx, "grpc-service", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if svc.Spec.ClusterIP == "None" {
		return Result{Solved: true, Message: "Success! Service is now Headless (ClusterIP: None)."}
	}

	return Result{Solved: false, Message: "Service is still using a Virtual IP (ClusterIP)."}
}

func (s *NetGrpcBalance) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
