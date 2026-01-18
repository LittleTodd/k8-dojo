package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IngressPathError scenario: Mismatched Ingress path.
type IngressPathError struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewIngressPathError(clientset *kubernetes.Clientset) *IngressPathError {
	return &IngressPathError{
		BaseScenario: BaseScenario{Namespace: "ingress-path"},
		clientset:    clientset,
	}
}

func (s *IngressPathError) GetMetadata() Metadata {
	return Metadata{
		ID:          "ingress-path-error",
		Name:        "Ingress: 404 Not Found",
		Description: "Requests to /app return 404. Verify the Ingress path configuration.",
		Difficulty:  DifficultyMedium,
		Category:    "Networking",
		Hints:       []string{"Check the Ingress `path`", "Ensure the application handles that path or use Rewrite"},
	}
}

func (s *IngressPathError) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Service
	_, err = s.clientset.CoreV1().Services(s.Namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "app-svc"},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Port: 80}},
			Selector: map[string]string{"app": "web"},
		},
	}, metav1.CreateOptions{})

	// Ingress with wrong path (simulated scenario where user expects / to work or app expects /)
	// Let's say app listens on /, but Ingress sends /api without rewrite, or Ingress has /api but user curls /
	// Simplified: Ingress path is /wrong, but we want /app
	pathType := networkingv1.PathTypePrefix
	_, err = s.clientset.NetworkingV1().Ingresses(s.Namespace).Create(ctx, &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "app-ingress"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/wrong-path",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "app-svc",
									Port: networkingv1.ServiceBackendPort{Number: 80},
								},
							},
						}},
					},
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *IngressPathError) Validate(ctx context.Context) Result {
	ing, err := s.clientset.NetworkingV1().Ingresses(s.Namespace).Get(ctx, "app-ingress", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(ing.Spec.Rules) > 0 {
		paths := ing.Spec.Rules[0].HTTP.Paths
		if len(paths) > 0 && paths[0].Path == "/app" {
			return Result{Solved: true, Message: "Success! Ingress path updated to /app."}
		}
	}
	return Result{Solved: false, Message: "Ingress path is still incorrect (Target: /app)."}
}

func (s *IngressPathError) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
