package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IngressTLSMismatch scenario: Ingress references missing Secret.
type IngressTLSMismatch struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewIngressTLSMismatch(clientset *kubernetes.Clientset) *IngressTLSMismatch {
	return &IngressTLSMismatch{
		BaseScenario: BaseScenario{Namespace: "ingress-tls"},
		clientset:    clientset,
	}
}

func (s *IngressTLSMismatch) GetMetadata() Metadata {
	return Metadata{
		ID:          "ingress-tls-mismatch",
		Name:        "Ingress: TLS Secret Missing",
		Description: "Ingress is crashing or not loading certificate. Check the Secret reference.",
		Difficulty:  DifficultyMedium,
		Category:    "Networking",
		Hints:       []string{"Check `kubectl get secret`", "Compare with Ingress `tls` section", "Create the secret or fix the name"},
	}
}

func (s *IngressTLSMismatch) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Secret exists but with different name
	_, err = s.clientset.CoreV1().Secrets(s.Namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "connection-secure"}, // Different name
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("dummy"),
			"tls.key": []byte("dummy"),
		},
	}, metav1.CreateOptions{})

	// Ingress referencing "tls-secret"
	pathType := networkingv1.PathTypePrefix
	_, err = s.clientset.NetworkingV1().Ingresses(s.Namespace).Create(ctx, &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "secure-ingress"},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{"example.com"},
				SecretName: "tls-secret", // Missing
			}},
			Rules: []networkingv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend:  networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "svc", Port: networkingv1.ServiceBackendPort{Number: 80}}},
						}},
					},
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *IngressTLSMismatch) Validate(ctx context.Context) Result {
	ing, err := s.clientset.NetworkingV1().Ingresses(s.Namespace).Get(ctx, "secure-ingress", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	// User can either rename the secret to "tls-secret" OR update ingress to "connection-secure"
	secretName := ""
	if len(ing.Spec.TLS) > 0 {
		secretName = ing.Spec.TLS[0].SecretName
	}

	_, err = s.clientset.CoreV1().Secrets(s.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return Result{Solved: true, Message: "Success! Ingress TLS secret found."}
	}

	return Result{Solved: false, Message: "Referenced TLS secret '" + secretName + "' not found."}
}

func (s *IngressTLSMismatch) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
