package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LifeGracefulShutdown scenario: Missing preStop hook.
type LifeGracefulShutdown struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewLifeGracefulShutdown(clientset *kubernetes.Clientset) *LifeGracefulShutdown {
	return &LifeGracefulShutdown{
		BaseScenario: BaseScenario{Namespace: "life-graceful"},
		clientset:    clientset,
	}
}

func (s *LifeGracefulShutdown) GetMetadata() Metadata {
	return Metadata{
		ID:          "life-graceful-shutdown",
		Name:        "Lifecycle: Zero Downtime",
		Description: "Requests fail during rollout. Configure a graceful shutdown strategy.",
		Difficulty:  DifficultyMedium,
		Category:    "Lifecycle",
		Hints:       []string{"Add a preStop hook", "Sleep for a few seconds to allow traffic to drain"},
	}
}

func (s *LifeGracefulShutdown) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginx:alpine",
						// Missing Lifecycle Hook
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *LifeGracefulShutdown) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(dep.Spec.Template.Spec.Containers) > 0 {
		c := dep.Spec.Template.Spec.Containers[0]
		if c.Lifecycle != nil && c.Lifecycle.PreStop != nil {
			return Result{Solved: true, Message: "Success! preStop hook configured."}
		}
	}
	return Result{Solved: false, Message: "No preStop hook found in container spec."}
}

func (s *LifeGracefulShutdown) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
