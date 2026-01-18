package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetServiceSelector scenario: Service selector typo.
type NetServiceSelector struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewNetServiceSelector(clientset *kubernetes.Clientset) *NetServiceSelector {
	return &NetServiceSelector{
		BaseScenario: BaseScenario{Namespace: "net-service-selector"},
		clientset:    clientset,
	}
}

func (s *NetServiceSelector) GetMetadata() Metadata {
	return Metadata{
		ID:          "net-service-selector",
		Name:        "Network 101: Service Discovery Failure",
		Description: "A Service is deployed but cannot find its Pods. Fix the connection.",
		Difficulty:  DifficultyEasy,
		Category:    "Networking",
		Hints:       []string{"Check the Service selector and Pod labels", "Use `kubectl get endpoints`"},
	}
}

func (s *NetServiceSelector) Setup(ctx context.Context) error {
	// Namespace
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Pods
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "web-pod",
			Labels: map[string]string{"app": "web"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "nginx", Image: "nginx:alpine"}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Broken Service
	_, err = s.clientset.CoreV1().Services(s.Namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-service"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web-server"}, // Typo! Should be "web"
			Ports:    []corev1.ServicePort{{Port: 80}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetServiceSelector) Validate(ctx context.Context) Result {
	ep, err := s.clientset.CoreV1().Endpoints(s.Namespace).Get(ctx, "web-service", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(ep.Subsets) > 0 && len(ep.Subsets[0].Addresses) > 0 {
		return Result{Solved: true, Message: "Success! Service found the Pods."}
	}

	return Result{Solved: false, Message: "Service has no endpoints."}
}

func (s *NetServiceSelector) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
