package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// ProbeLivenessFail scenario: Liveness probe check fails.
type ProbeLivenessFail struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewProbeLivenessFail(clientset *kubernetes.Clientset) *ProbeLivenessFail {
	return &ProbeLivenessFail{
		BaseScenario: BaseScenario{Namespace: "probe-fail"},
		clientset:    clientset,
	}
}

func (s *ProbeLivenessFail) GetMetadata() Metadata {
	return Metadata{
		ID:          "probe-liveness-fail",
		Name:        "Lifecycle: Liveness Failure",
		Description: "The Pod keeps restarting. Investigating the Liveness Probe configuration.",
		Difficulty:  DifficultyEasy,
		Category:    "Lifecycle",
		Hints:       []string{"Check `kubectl describe pod` events", "Verify the livenessProbe port"},
	}
}

func (s *ProbeLivenessFail) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "unstable-app"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "nginx",
				Image: "nginx:alpine",
				Ports: []corev1.ContainerPort{{ContainerPort: 80}},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt(8080), // Wrong port
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *ProbeLivenessFail) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "unstable-app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	// Check if configured correctly
	if len(pod.Spec.Containers) > 0 {
		probe := pod.Spec.Containers[0].LivenessProbe
		if probe != nil && probe.HTTPGet != nil {
			if probe.HTTPGet.Port.IntVal == 80 || probe.HTTPGet.Port.StrVal == "80" {
				return Result{Solved: true, Message: "Success! Liveness probe port corrected."}
			}
		}
	}
	return Result{Solved: false, Message: "Liveness probe matches incorrect port."}
}

func (s *ProbeLivenessFail) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
