package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceLimitRange scenario: Pod request prohibited by LimitRange.
type ResourceLimitRange struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewResourceLimitRange(clientset *kubernetes.Clientset) *ResourceLimitRange {
	return &ResourceLimitRange{
		BaseScenario: BaseScenario{Namespace: "res-limit"},
		clientset:    clientset,
	}
}

func (s *ResourceLimitRange) GetMetadata() Metadata {
	return Metadata{
		ID:          "resource-limit-range",
		Name:        "Resources: LimitRange Block",
		Description: "Your Pod is rejected: 'Forbidden: maximum cpu usage per Container is 500m'.",
		Difficulty:  DifficultyEasy,
		Category:    "Resources",
		Hints:       []string{"Check `kubectl get limitrange`", "Reduce the CPU request in your Pod/Deployment"},
	}
}

func (s *ResourceLimitRange) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// LimitRange Max 500m
	_, err = s.clientset.CoreV1().LimitRanges(s.Namespace).Create(ctx, &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: "cpu-limit"},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{{
				Type: corev1.LimitTypeContainer,
				Max:  corev1.ResourceList{corev1.ResourceCPU: mustParse("500m")},
			}},
		},
	}, metav1.CreateOptions{})

	// We cannot create a violation directy (API rejects).
	// So we create a Deployment with violation.
	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "gaint-backend"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "giant"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "giant"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "nginx:alpine",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: mustParse("1")}, // Exceeds 500m
						},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return nil
}

func (s *ResourceLimitRange) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "gaint-backend", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if dep.Status.AvailableReplicas > 0 {
		return Result{Solved: true, Message: "Success! Pod fits within limits."}
	}
	return Result{Solved: false, Message: "Deployment cannot create pods due to LimitRange."}
}

func (s *ResourceLimitRange) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
