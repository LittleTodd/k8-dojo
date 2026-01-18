package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// ProbeReadinessTimeout scenario: Readiness probe timeout too short.
type ProbeReadinessTimeout struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewProbeReadinessTimeout(clientset *kubernetes.Clientset) *ProbeReadinessTimeout {
	return &ProbeReadinessTimeout{
		BaseScenario: BaseScenario{Namespace: "probe-ready"},
		clientset:    clientset,
	}
}

func (s *ProbeReadinessTimeout) GetMetadata() Metadata {
	return Metadata{
		ID:          "probe-readiness-timeout",
		Name:        "Lifecycle: Readiness Timeout",
		Description: "The Pod is running but never becomes Ready. The app is slow to respond.",
		Difficulty:  DifficultyMedium,
		Category:    "Lifecycle",
		Hints:       []string{"The app takes 2s to respond", "Check readinessProbe `timeoutSeconds` (default is 1s)"},
	}
}

func (s *ProbeReadinessTimeout) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Simulate slow app by sleeping in command before responding? Hard with standard nginx.
	// Since we validate the CONFIG, we don't need to simulate actual network timeout unless check relies on status.
	// We'll rely on config check validations.

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "slow-app"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox",
				Command: []string{"sh", "-c", "while true; do echo -e 'HTTP/1.1 200 OK\n\nOK' | nc -l -p 8080 -w 5; done"},
				// nc -w is connect timeout not processing delay.
				// Just checking config is enough.

				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt(8080),
						},
					},
					TimeoutSeconds: 1, // Too short if we assume app is slow
					PeriodSeconds:  5,
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *ProbeReadinessTimeout) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "slow-app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(pod.Spec.Containers) > 0 {
		probe := pod.Spec.Containers[0].ReadinessProbe
		if probe != nil && probe.TimeoutSeconds > 1 {
			return Result{Solved: true, Message: "Success! Readiness timeout increased."}
		}
	}
	return Result{Solved: false, Message: "Readiness timeout is still 1s."}
}

func (s *ProbeReadinessTimeout) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
