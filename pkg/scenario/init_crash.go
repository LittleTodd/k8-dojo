package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InitContainerCrash scenario: InitContainer fails to complete.
type InitContainerCrash struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewInitContainerCrash(clientset *kubernetes.Clientset) *InitContainerCrash {
	return &InitContainerCrash{
		BaseScenario: BaseScenario{Namespace: "init-crash"},
		clientset:    clientset,
	}
}

func (s *InitContainerCrash) GetMetadata() Metadata {
	return Metadata{
		ID:          "init-container-crash",
		Name:        "Lifecycle: Stuck Initializing",
		Description: "Pod Status says 'Init:CrashLoopBackOff'. The main container never starts.",
		Difficulty:  DifficultyEasy,
		Category:    "Lifecycle",
		Hints:       []string{"Use `kubectl logs -c init-myservice`", "The init container command is failing"},
	}
}

func (s *InitContainerCrash) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "init-check",
				Image:   "busybox",
				Command: []string{"sh", "-c", "exit 1"}, // Fails!
			}},
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:alpine",
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *InitContainerCrash) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Status.Phase == corev1.PodRunning {
		return Result{Solved: true, Message: "Success! Pod is running."}
	}
	return Result{Solved: false, Message: "Pod is not Running."}
}

func (s *InitContainerCrash) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
