package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KernelOOMDisable scenario: Ensure QoS Guaranteed.
type KernelOOMDisable struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewKernelOOMDisable(clientset *kubernetes.Clientset) *KernelOOMDisable {
	return &KernelOOMDisable{
		BaseScenario: BaseScenario{Namespace: "kernel-oom"},
		clientset:    clientset,
	}
}

func (s *KernelOOMDisable) GetMetadata() Metadata {
	return Metadata{
		ID:          "kernel-oom-disable",
		Name:        "Kernel: OOM Survival",
		Description: "This critical pod must not be OOM Killed. Configure it as QoS Guaranteed (or simulate OOM prevention).",
		Difficulty:  DifficultyHard,
		Category:    "Kernel",
		Hints:       []string{"Set Limits == Requests", "Look for QoS Class 'Guaranteed'"},
	}
}

func (s *KernelOOMDisable) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Burstable Pod
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "critical-pod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:alpine",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceMemory: mustParse("64Mi")},
					// No limits = Burstable or BestEffort depending on others
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *KernelOOMDisable) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "critical-pod", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Status.QOSClass == corev1.PodQOSGuaranteed {
		return Result{Solved: true, Message: "Success! Pod is QoS Guaranteed."}
	}
	return Result{Solved: false, Message: "Pod QoS is " + string(pod.Status.QOSClass) + ", expected Guaranteed."}
}

func (s *KernelOOMDisable) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
