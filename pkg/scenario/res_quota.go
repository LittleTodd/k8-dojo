package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceQuotaExceeded scenario: Quota blocks pod creation.
type ResourceQuotaExceeded struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewResourceQuotaExceeded(clientset *kubernetes.Clientset) *ResourceQuotaExceeded {
	return &ResourceQuotaExceeded{
		BaseScenario: BaseScenario{Namespace: "res-quota"},
		clientset:    clientset,
	}
}

func (s *ResourceQuotaExceeded) GetMetadata() Metadata {
	return Metadata{
		ID:          "resource-quota-exceeded",
		Name:        "Resources: Quota Limit Reached",
		Description: "Cannot create new Pod. Namespace quota exceeded.",
		Difficulty:  DifficultyMedium,
		Category:    "Resources",
		Hints:       []string{"Check `kubectl get resourcequota`", "Increase the quota or delete unused pods"},
	}
}

func (s *ResourceQuotaExceeded) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create strict Quota
	_, err = s.clientset.CoreV1().ResourceQuotas(s.Namespace).Create(ctx, &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "compute-quota"},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods: mustParse("1"), // Only 1 pod allowed
			},
		},
	}, metav1.CreateOptions{})

	// Create 1 pod to consume quota
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "hog"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}}},
	}, metav1.CreateOptions{})

	// Attempt to create second pod? It will fail API side.
	// Scenario: User tries to deploy "web" but it fails.
	// We can't actually create the failed state easily in "Setup" unless we use a higher level abstraction that ignores error?
	// But in DOJO, we set up the environment. The USER tries to create the pod?
	// OR: We create a Deployment, which creates ReplicaSet, which fails to create Pod.
	// This is better. Pod events directly show failure.

	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "blocked-dep"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "blocked"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "blocked"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}}},
			},
		},
	}, metav1.CreateOptions{})

	return nil // Ignore deployment creation error if any, though it succeeds, only pods fail
}

func (s *ResourceQuotaExceeded) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "blocked-dep", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if dep.Status.AvailableReplicas > 0 {
		return Result{Solved: true, Message: "Success! Deployment has available replicas."}
	}
	return Result{Solved: false, Message: "Deployment has 0 available replicas."}
}

func (s *ResourceQuotaExceeded) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
