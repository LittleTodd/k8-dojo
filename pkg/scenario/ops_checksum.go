package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OpsConfigChecksum scenario: Config checksum annotation.
type OpsConfigChecksum struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewOpsConfigChecksum(clientset *kubernetes.Clientset) *OpsConfigChecksum {
	return &OpsConfigChecksum{
		BaseScenario: BaseScenario{Namespace: "ops-checksum"},
		clientset:    clientset,
	}
}

func (s *OpsConfigChecksum) GetMetadata() Metadata {
	return Metadata{
		ID:          "ops-config-checksum",
		Name:        "Ops: GitOps Trigger",
		Description: "The Deployment must restart when ConfigMap changes. Add a checksum annotation.",
		Difficulty:  DifficultyMedium,
		Category:    "Operations",
		Hints:       []string{"Add an annotation to the Pod template", "Key typically contains 'checksum' or 'sha256'"},
	}
}

func (s *OpsConfigChecksum) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "gitops-app"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "gitops"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "gitops"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *OpsConfigChecksum) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "gitops-app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	for k := range dep.Spec.Template.Annotations {
		if k == "checksum/config" || (len(k) > 8 && k[:8] == "checksum") {
			return Result{Solved: true, Message: "Success! Checksum annotation found."}
		}
	}
	return Result{Solved: false, Message: "No checksum annotation found in Pod template."}
}

func (s *OpsConfigChecksum) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
