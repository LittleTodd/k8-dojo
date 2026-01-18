package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecFSGroupDenied scenario: User cannot write to volume.
type SecFSGroupDenied struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSecFSGroupDenied(clientset *kubernetes.Clientset) *SecFSGroupDenied {
	return &SecFSGroupDenied{
		BaseScenario: BaseScenario{Namespace: "sec-fsgroup"},
		clientset:    clientset,
	}
}

func (s *SecFSGroupDenied) GetMetadata() Metadata {
	return Metadata{
		ID:          "sec-fsgroup-denied",
		Name:        "Security: Permission Denied",
		Description: "Container running as user 1000 cannot write to the mounted volume.",
		Difficulty:  DifficultyMedium,
		Category:    "Security",
		Hints:       []string{"Volume is owned by root", "Use `securityContext.fsGroup` to change volume ownership"},
	}
}

func (s *SecFSGroupDenied) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Pod with runAsUser 1000, emptyDir (root by default)
	runAsUser := int64(1000)
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "writer"},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &runAsUser,
				// Missing FSGroup
			},
			Containers: []corev1.Container{{
				Name:    "app",
				Image:   "busybox",
				Command: []string{"sh", "-c", "echo 'hello' > /data/file && sleep 3600"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name:         "data",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SecFSGroupDenied) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "writer", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.FSGroup != nil {
		if *pod.Spec.SecurityContext.FSGroup == 1000 {
			return Result{Solved: true, Message: "Success! FSGroup configured."}
		}
	}
	return Result{Solved: false, Message: "FSGroup missing or incorrect."}
}

func (s *SecFSGroupDenied) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
