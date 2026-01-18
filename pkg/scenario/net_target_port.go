package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// NetTargetPortMismatch scenario: Service targetPort doesn't match container port.
type NetTargetPortMismatch struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewNetTargetPortMismatch(clientset *kubernetes.Clientset) *NetTargetPortMismatch {
	return &NetTargetPortMismatch{
		BaseScenario: BaseScenario{Namespace: "net-target-port"},
		clientset:    clientset,
	}
}

func (s *NetTargetPortMismatch) GetMetadata() Metadata {
	return Metadata{
		ID:          "net-target-port-mismatch",
		Name:        "Network: The Unreachable Port",
		Description: "Service is refusing connections. Check the port mapping.",
		Difficulty:  DifficultyEasy,
		Category:    "Networking",
		Hints:       []string{"Check the Service `targetPort`", "Check the Container `ports`", "They must match"},
	}
}

func (s *NetTargetPortMismatch) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Pod listening on 80
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "web-app",
			Labels: map[string]string{"app": "web"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "nginx",
				Image: "nginx:alpine",
				Ports: []corev1.ContainerPort{{ContainerPort: 80}},
			}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Service targeting 8080 (Mismatch)
	_, err = s.clientset.CoreV1().Services(s.Namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-service"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports: []corev1.ServicePort{{
				Port:       80,
				TargetPort: intstr.FromInt(8080), // Wrong! Should be 80
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *NetTargetPortMismatch) Validate(ctx context.Context) Result {
	svc, err := s.clientset.CoreV1().Services(s.Namespace).Get(ctx, "web-service", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if len(svc.Spec.Ports) > 0 {
		tgt := svc.Spec.Ports[0].TargetPort
		if tgt.IntVal == 80 || tgt.StrVal == "80" {
			return Result{Solved: true, Message: "Success! TargetPort matches container port."}
		}
	}

	return Result{Solved: false, Message: "Service targetPort is still incorrect."}
}

func (s *NetTargetPortMismatch) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
