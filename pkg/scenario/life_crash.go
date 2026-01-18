package scenario

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LifeCrashConfig scenario: CrashLoop due to missing ConfigMap.
type LifeCrashConfig struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewLifeCrashConfig(clientset *kubernetes.Clientset) *LifeCrashConfig {
	return &LifeCrashConfig{
		BaseScenario: BaseScenario{Namespace: "life-crash-config"},
		clientset:    clientset,
	}
}

func (s *LifeCrashConfig) GetMetadata() Metadata {
	return Metadata{
		ID:          "crashloop-missing-config",
		Name:        "Lifecycle: The CrashLoop Mystery",
		Description: "Pod is crash-looping. The logs mention a missing configuration.",
		Difficulty:  DifficultyEasy,
		Category:    "Lifecycle",
		Hints:       []string{"Use `kubectl logs`", "Check envFrom or volumeMounts", "The ConfigMap 'app-config' is missing"},
	}
}

func (s *LifeCrashConfig) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	replicas := int32(1)
	_, err = s.clientset.AppsV1().Deployments(s.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "crash"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "crash"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "app",
						Image:   "busybox",
						Command: []string{"sh", "-c", "if [ ! -f /config/settings.properties ]; then echo 'CRITICAL: Config not found' && exit 1; fi; sleep 3600"},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/config",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
							},
						},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *LifeCrashConfig) Validate(ctx context.Context) Result {
	pods, err := s.clientset.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=crash"})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return Result{Solved: true, Message: "Success! Application is running."}
		}
	}
	return Result{Solved: false, Message: "Pod is not running yet."}
}

func (s *LifeCrashConfig) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
