package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SchedTaintToleration scenario: Pod pending due to NoSchedule taint.
type SchedTaintToleration struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSchedTaintToleration(clientset *kubernetes.Clientset) *SchedTaintToleration {
	return &SchedTaintToleration{
		BaseScenario: BaseScenario{Namespace: "sched-taint"},
		clientset:    clientset,
	}
}

func (s *SchedTaintToleration) GetMetadata() Metadata {
	return Metadata{
		ID:          "sched-taint-toleration",
		Name:        "Scheduling: Forbidden Node",
		Description: "Pod is Pending. describe shows '1 node(s) had untolerated taint {dedicated: db}'.",
		Difficulty:  DifficultyMedium,
		Category:    "Scheduling",
		Hints:       []string{"Add a `toleration` to the Pod", "Match key, value, and effect"},
	}
}

func (s *SchedTaintToleration) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Taint the node
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		// Clean existing to be safe
		// In real world, we append. Here assume single node kind.
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:    "dedicated",
			Value:  "db",
			Effect: corev1.TaintEffectNoSchedule,
		})
		_, _ = s.clientset.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{})
	}

	// Pod without toleration
	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "db-pod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SchedTaintToleration) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "db-pod", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Status.Phase == corev1.PodRunning {
		return Result{Solved: true, Message: "Success! Pod is running."}
	}
	return Result{Solved: false, Message: "Pod is Pending."}
}

func (s *SchedTaintToleration) Cleanup(ctx context.Context) error {
	// Remove taint
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		newTaints := []corev1.Taint{}
		for _, t := range node.Spec.Taints {
			if t.Key != "dedicated" {
				newTaints = append(newTaints, t)
			}
		}
		node.Spec.Taints = newTaints
		_, _ = s.clientset.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{})
	}
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
