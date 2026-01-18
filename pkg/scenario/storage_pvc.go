package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StoragePVCPending scenario: PVC Pending due to wrong StorageClass.
type StoragePVCPending struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewStoragePVCPending(clientset *kubernetes.Clientset) *StoragePVCPending {
	return &StoragePVCPending{
		BaseScenario: BaseScenario{Namespace: "storage-pvc"},
		clientset:    clientset,
	}
}

func (s *StoragePVCPending) GetMetadata() Metadata {
	return Metadata{
		ID:          "storage-pvc-pending",
		Name:        "Storage: PVC Stuck Pending",
		Description: "A PersistentVolumeClaim is stuck in Pending state. The Pod is also pending.",
		Difficulty:  DifficultyEasy,
		Category:    "Storage",
		Hints:       []string{"Describe the PVC", "Check storageClassName", "The cluster uses 'standard' class"},
	}
}

func (s *StoragePVCPending) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	scName := "premium-ssd" // Does not exist
	_, err = s.clientset.CoreV1().PersistentVolumeClaims(s.Namespace).Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data-pvc"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: mustParse("1Gi"),
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	_, err = s.clientset.CoreV1().Pods(s.Namespace).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "db"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "db",
				Image: "postgres:alpine",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/var/lib/postgresql/data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "data-pvc",
					},
				},
			}},
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *StoragePVCPending) Validate(ctx context.Context) Result {
	pvc, err := s.clientset.CoreV1().PersistentVolumeClaims(s.Namespace).Get(ctx, "data-pvc", metav1.GetOptions{})
	if err != nil {
		// If user deleted and recreated, it might be missing briefly, or found.
		// If not found, check if they made a new one? Assuming same name.
		return Result{Solved: false, Message: err.Error()}
	}

	if pvc.Status.Phase == corev1.ClaimBound {
		return Result{Solved: true, Message: "Success! PVC is Bound."}
	}
	return Result{Solved: false, Message: "PVC is still Pending."}
}

func (s *StoragePVCPending) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
