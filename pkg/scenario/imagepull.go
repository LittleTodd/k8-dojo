// Package scenario provides troubleshooting scenarios for k8s-dojo.
package scenario

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ImagePullBackOff is a scenario where a deployment has an invalid image tag.
type ImagePullBackOff struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

// NewImagePullBackOff creates a new ImagePullBackOff scenario.
func NewImagePullBackOff(clientset *kubernetes.Clientset) *ImagePullBackOff {
	return &ImagePullBackOff{
		BaseScenario: BaseScenario{
			Namespace: "dojo-level-1",
		},
		clientset: clientset,
	}
}

// GetMetadata returns the scenario's metadata.
func (s *ImagePullBackOff) GetMetadata() Metadata {
	return Metadata{
		ID:          "image-pull-backoff",
		Name:        "Level 1: Image Pull Error",
		Description: "The web-server Deployment is failing to start. Investigate and fix the issue.",
		Difficulty:  DifficultyEasy,
		Category:    "Pods & Containers",
		Hints: []string{
			"Check the Pod status using: kubectl get pods -n " + s.Namespace,
			"Look at the Pod events: kubectl describe pod -n " + s.Namespace,
			"The image tag might be incorrect...",
		},
		TimeLimit: 10 * time.Minute,
	}
}

// Setup creates the faulty deployment in the cluster.
func (s *ImagePullBackOff) Setup(ctx context.Context) error {
	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "k8s-dojo",
			},
		},
	}

	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create deployment with wrong image tag
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-server",
			Namespace: s.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "web-server",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "web-server",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:wrongtag", // This is the bug!
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// Validate checks if the user has fixed the deployment.
func (s *ImagePullBackOff) Validate(ctx context.Context) Result {
	// Get pods in the namespace
	pods, err := s.clientset.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=web-server",
	})
	if err != nil {
		return Result{Solved: false, Message: fmt.Sprintf("Error checking pods: %v", err)}
	}

	if len(pods.Items) == 0 {
		return Result{Solved: false, Message: "No pods found. Deployment may have been deleted."}
	}

	// Check if any pod is running
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Verify all containers are ready
			allReady := true
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					allReady = false
					break
				}
			}
			if allReady {
				return Result{Solved: true, Message: "🎉 Congratulations! The web-server is now running!"}
			}
		}
	}

	// Check for ImagePullBackOff status
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
					return Result{Solved: false, Message: "Pod is stuck in " + reason + ". Keep investigating!"}
				}
			}
		}
	}

	return Result{Solved: false, Message: "Pod is not yet running. Keep trying!"}
}

// Cleanup removes all resources created by this scenario.
func (s *ImagePullBackOff) Cleanup(ctx context.Context) error {
	// Delete the namespace (this will cascade delete all resources)
	err := s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}
	return nil
}
