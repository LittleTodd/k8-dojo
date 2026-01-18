package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StorageZonalAffinity scenario: Pod/PV in different zones (Simulated).
type StorageZonalAffinity struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewStorageZonalAffinity(clientset *kubernetes.Clientset) *StorageZonalAffinity {
	return &StorageZonalAffinity{
		BaseScenario: BaseScenario{Namespace: "storage-zonal"},
		clientset:    clientset,
	}
}

func (s *StorageZonalAffinity) GetMetadata() Metadata {
	return Metadata{
		ID:          "storage-zonal-affinity",
		Name:        "Storage: Zonal Connectivity",
		Description: "Pod cannot mount the PV because they are in different zones. Fix the affinity.",
		Difficulty:  DifficultyHard,
		Category:    "Storage",
		Hints:       []string{"Check PV NodeAffinity", "Ensure Pod is scheduled in the same zone", "Kind usually only has one zone, this is a simulation"},
	}
}

func (s *StorageZonalAffinity) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create a PV simulating a specific zone
	scName := "manual"
	_, err = s.clientset.CoreV1().PersistentVolumes().Create(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "zone-pv"},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity:         corev1.ResourceList{corev1.ResourceStorage: mustParse("1Gi")},
			StorageClassName: scName,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/tmp/data"},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "topology.kubernetes.io/zone",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"us-east-1a"}, // Simulated Zone
						}},
					}},
				},
			},
		},
	}, metav1.CreateOptions{})

	// Create Pod that needs it (will fail scheduling if node doesn't have label)
	// User needs to label the node, OR update PV affinity to match node's actual label (e.g. none/default)

	_, err = s.clientset.CoreV1().PersistentVolumeClaims(s.Namespace).Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "zone-pvc"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			VolumeName:       "zone-pv",
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: mustParse("1Gi")},
			},
		},
	}, metav1.CreateOptions{})

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "zone-pod"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:alpine", VolumeMounts: []corev1.VolumeMount{{Name: "vol", MountPath: "/data"}}}},
			Volumes:    []corev1.Volume{{Name: "vol", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "zone-pvc"}}}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *StorageZonalAffinity) Validate(ctx context.Context) Result {
	pod, err := s.clientset.CoreV1().Pods(s.Namespace).Get(ctx, "zone-pod", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	if pod.Status.Phase == corev1.PodRunning {
		return Result{Solved: true, Message: "Success! Pod successfully mounted the Zonal PV."}
	}
	return Result{Solved: false, Message: "Pod is not Running."}
}

func (s *StorageZonalAffinity) Cleanup(ctx context.Context) error {
	_ = s.clientset.CoreV1().PersistentVolumes().Delete(ctx, "zone-pv", metav1.DeleteOptions{})
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
