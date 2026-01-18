package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StorageSubpathOverwrite scenario: Mount hides existing files.
type StorageSubpathOverwrite struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewStorageSubpathOverwrite(clientset *kubernetes.Clientset) *StorageSubpathOverwrite {
	return &StorageSubpathOverwrite{
		BaseScenario: BaseScenario{Namespace: "storage-subpath"},
		clientset:    clientset,
	}
}

func (s *StorageSubpathOverwrite) GetMetadata() Metadata {
	return Metadata{
		ID:          "storage-subpath-overwrite",
		Name:        "Storage: File Wipeout",
		Description: "Mounting a file to /etc/app/config.json hides the rest of /etc/app/. Fix it.",
		Difficulty:  DifficultyMedium,
		Category:    "Storage",
		Hints:       []string{"Accessing other files in directory fails", "Use `subPath` to mount a single file"},
	}
}

func (s *StorageSubpathOverwrite) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// ConfigMap
	_, err = s.clientset.CoreV1().ConfigMaps(s.Namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config"},
		Data:       map[string]string{"config.json": "{}"},
	}, metav1.CreateOptions{})

	// Pod mounting CM to /etc/app
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:alpine",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "config",
					MountPath: "/etc/nginx", // Overwrites entire nginx dir!
				}},
			}},
			Volumes: []corev1.Volume{{
				Name:         "config",
				VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *StorageSubpathOverwrite) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "app", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(pod.Spec.Containers) > 0 {
		for _, vm := range pod.Spec.Containers[0].VolumeMounts {
			if vm.Name == "config" {
				if vm.SubPath != "" {
					return Result{Solved: true, Message: "Success! subPath used."}
				}
			}
		}
	}
	return Result{Solved: false, Message: "Volume mount is still overwriting entire directory."}
}

func (s *StorageSubpathOverwrite) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
