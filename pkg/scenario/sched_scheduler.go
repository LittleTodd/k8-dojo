package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SchedMissingScheduler scenario: Pod Pending due to non-existent scheduler.
type SchedMissingScheduler struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSchedMissingScheduler(clientset *kubernetes.Clientset) *SchedMissingScheduler {
	return &SchedMissingScheduler{
		BaseScenario: BaseScenario{Namespace: "sched-missing"},
		clientset:    clientset,
	}
}

func (s *SchedMissingScheduler) GetMetadata() Metadata {
	return Metadata{
		ID:          "sched-missing-scheduler",
		Name:        "Scheduling: The Ghost Scheduler",
		Description: "Pod is stuck in Pending state forever. Investigate why.",
		Difficulty:  DifficultyEasy,
		Category:    "Scheduling",
		Hints:       []string{"Check `kubectl describe pod` events", "Look at `schedulerName` in Pod spec"},
	}
}

func (s *SchedMissingScheduler) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-pod"},
		Spec: corev1.PodSpec{
			SchedulerName: "ghost-scheduler", // Does not exist
			Containers:    []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SchedMissingScheduler) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "custom-pod", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Spec.SchedulerName == "default-scheduler" || pod.Spec.SchedulerName == "" {
		if pod.Status.Phase == corev1.PodRunning {
			return Result{Solved: true, Message: "Success! Pod is running."}
		}
		return Result{Solved: false, Message: "Scheduler fixed, waiting for Pod to start..."}
	}
	return Result{Solved: false, Message: "Pod still using invalid scheduler: " + pod.Spec.SchedulerName}
}

func (s *SchedMissingScheduler) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
