package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecPrivilegedPolicy scenario: Fix privileged pod.
type SecPrivilegedPolicy struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSecPrivilegedPolicy(clientset *kubernetes.Clientset) *SecPrivilegedPolicy {
	return &SecPrivilegedPolicy{
		BaseScenario: BaseScenario{Namespace: "sec-priv"},
		clientset:    clientset,
	}
}

func (s *SecPrivilegedPolicy) GetMetadata() Metadata {
	return Metadata{
		ID:          "sec-privileged-policy",
		Name:        "Security: The Privileged Container",
		Description: "A Deployment is running with `privileged: true`. Harden it by removing this flag.",
		Difficulty:  DifficultyEasy,
		Category:    "Security",
		Hints:       []string{"Edit the deployment", "Look for `privileged: true` in securityContext"},
	}
}

func (s *SecPrivilegedPolicy) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	privileged := true
	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "risky-app"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "risky"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "risky"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "nginx:alpine",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SecPrivilegedPolicy) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "risky-app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(dep.Spec.Template.Spec.Containers) > 0 {
		sc := dep.Spec.Template.Spec.Containers[0].SecurityContext
		if sc == nil || sc.Privileged == nil || *sc.Privileged == false {
			return Result{Solved: true, Message: "Success! Privileged flag removed."}
		}
	}
	return Result{Solved: false, Message: "Container is still privileged."}
}

func (s *SecPrivilegedPolicy) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
