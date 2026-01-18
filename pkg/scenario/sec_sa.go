package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecSANoMount scenario: automountServiceAccountToken: false.
type SecSANoMount struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSecSANoMount(clientset *kubernetes.Clientset) *SecSANoMount {
	return &SecSANoMount{
		BaseScenario: BaseScenario{Namespace: "sec-sa"},
		clientset:    clientset,
	}
}

func (s *SecSANoMount) GetMetadata() Metadata {
	return Metadata{
		ID:          "sec-sa-nomount",
		Name:        "Security: No Certificates",
		Description: "The app needs to talk to K8s API but cannot find credentials.",
		Difficulty:  DifficultyEasy,
		Category:    "Security",
		Hints:       []string{"Auto-mounting of service account token is disabled", "Set `automountServiceAccountToken: true`"},
	}
}

func (s *SecSANoMount) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	nomount := false
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "dashboard"},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &nomount,
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:alpine",
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SecSANoMount) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "dashboard", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken == true {
		return Result{Solved: true, Message: "Success! Token is auto-mounted."}
	}
	// Also check if they just mounted a volume blindly manually? Unlikely.
	return Result{Solved: false, Message: "Automount is disabled."}
}

func (s *SecSANoMount) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
