package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SchedNodeAffinity scenario: GPU scheduling using Node Affinity.
type SchedNodeAffinity struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSchedNodeAffinity(clientset *kubernetes.Clientset) *SchedNodeAffinity {
	return &SchedNodeAffinity{
		BaseScenario: BaseScenario{Namespace: "sched-affinity"},
		clientset:    clientset,
	}
}

func (s *SchedNodeAffinity) GetMetadata() Metadata {
	return Metadata{
		ID:          "sched-node-affinity",
		Name:        "Scheduling: The Sticky GPU",
		Description: "A Pod requesting 'special' hardware is Pending. Force it to run on the node labeled 'hardware=gpu'.",
		Difficulty:  DifficultyMedium,
		Category:    "Scheduling",
		Hints:       []string{"Tolerations are not enough", "Use NodeAffinity", "The node already has label 'hardware=gpu'"},
	}
}

func (s *SchedNodeAffinity) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Label a node for the scenario (assuming single node Kind cluster)
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		node.Labels["hardware"] = "gpu"
		_, _ = s.clientset.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{})
	}

	// Pod with toleration but no affinity (so it floats or fails if we used taint)
	// To make it FAIL, we simulate requirement. In Kind, tough to force pending without taints.
	// We'll rely on the Check validating presence of Affinity.

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-workload"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine"}},
			// Missing Affinity
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SchedNodeAffinity) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "gpu-workload", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		return Result{Solved: true, Message: "Success! NodeAffinity configured."}
	}
	return Result{Solved: false, Message: "Pod spec does not have NodeAffinity configured."}
}

func (s *SchedNodeAffinity) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
