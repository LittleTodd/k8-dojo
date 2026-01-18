package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodFinalizerStuck scenario: Pod stuck in Terminating.
type PodFinalizerStuck struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewPodFinalizerStuck(clientset *kubernetes.Clientset) *PodFinalizerStuck {
	return &PodFinalizerStuck{
		BaseScenario: BaseScenario{Namespace: "pod-stuck"},
		clientset:    clientset,
	}
}

func (s *PodFinalizerStuck) GetMetadata() Metadata {
	return Metadata{
		ID:          "pod-finalizer-stuck",
		Name:        "Lifecycle: The Undying Pod",
		Description: "A Pod is stuck in 'Terminating' state and won't go away. Force delete doesn't help.",
		Difficulty:  DifficultyMedium,
		Category:    "Lifecycle",
		Hints:       []string{"Check `metadata.finalizers`", "Remove the finalizer to release the pod"},
	}
}

func (s *PodFinalizerStuck) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "zombie",
			Finalizers: []string{"example.com/lock"}, // Custom finalizer that no controller handles
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Trigger deletion to enter Terminating state
	_ = s.clientset.CoreV1().Pods(s.Namespace).Delete(ctx, "zombie", metav1.DeleteOptions{})

	// Pod is now stuck.
	return nil
}

func (s *PodFinalizerStuck) Validate(ctx context.Context) Result {
	_, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "zombie", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: true, Message: "Success! Pod is gone."}
	}

	return Result{Solved: false, Message: "Pod stuck in Terminating."}
}

func (s *PodFinalizerStuck) Cleanup(ctx context.Context) error {
	// Force cleanup
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "zombie", metav1.GetOptions{})
	if err == nil {
		pod.Finalizers = nil
		_, _ = s.clientset.CoreV1().Pods(s.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	}
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
