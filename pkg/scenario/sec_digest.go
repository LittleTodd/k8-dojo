package scenario

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecImageDigest scenario: Enforce image digest.
type SecImageDigest struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSecImageDigest(clientset *kubernetes.Clientset) *SecImageDigest {
	return &SecImageDigest{
		BaseScenario: BaseScenario{Namespace: "sec-digest"},
		clientset:    clientset,
	}
}

func (s *SecImageDigest) GetMetadata() Metadata {
	return Metadata{
		ID:          "sec-image-digest",
		Name:        "Security: Supply Chain Integrity",
		Description: "The Deployment uses a mutable tag `nginx:latest`. Update it to use an immutable SHA256 digest.",
		Difficulty:  DifficultyMedium,
		Category:    "Security",
		Hints:       []string{"Find the digest for nginx:latest", "Update image field to use name@sha256:..."},
	}
}

func (s *SecImageDigest) Setup(ctx context.Context) error {
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
						Image: "nginx:latest",
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SecImageDigest) Validate(ctx context.Context) Result {
	dep, err := s.clientset.AppsV1().Deployments(s.Namespace).Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(dep.Spec.Template.Spec.Containers) > 0 {
		image := dep.Spec.Template.Spec.Containers[0].Image
		if strings.Contains(image, "@sha256:") {
			return Result{Solved: true, Message: "Success! Image is pinned by digest."}
		}
	}
	return Result{Solved: false, Message: "Image is still using a tag, not a digest."}
}

func (s *SecImageDigest) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
