package scenario

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecRBACForbidden scenario: Role missing permissions.
type SecRBACForbidden struct {
	BaseScenario
	clientset *kubernetes.Clientset
}

func NewSecRBACForbidden(clientset *kubernetes.Clientset) *SecRBACForbidden {
	return &SecRBACForbidden{
		BaseScenario: BaseScenario{Namespace: "sec-rbac"},
		clientset:    clientset,
	}
}

func (s *SecRBACForbidden) GetMetadata() Metadata {
	return Metadata{
		ID:          "rbac-forbidden",
		Name:        "Security: Access Denied",
		Description: "The 'intern' service account cannot list pods. Fix the Role permissions.",
		Difficulty:  DifficultyMedium,
		Category:    "Security",
		Hints:       []string{"Use `kubectl get role`", "Edit the Role to add 'list' verb"},
	}
}

func (s *SecRBACForbidden) Setup(ctx context.Context) error {
	_, err := s.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: s.Namespace},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// ServiceAccount
	_, err = s.clientset.CoreV1().ServiceAccounts(s.Namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "intern"},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Role (missing list)
	_, err = s.clientset.RbacV1().Roles(s.Namespace).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-reader"},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "watch"}, // Missing "list"
		}},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Binding
	_, err = s.clientset.RbacV1().RoleBindings(s.Namespace).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "read-pods"},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "intern",
			Namespace: s.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "pod-reader",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}, metav1.CreateOptions{})

	return err
}

func (s *SecRBACForbidden) Validate(ctx context.Context) Result {
	role, err := s.clientset.RbacV1().Roles(s.Namespace).Get(ctx, "pod-reader", metav1.GetOptions{})
	if err != nil {
		return Result{Solved: false, Message: err.Error()}
	}

	for _, rule := range role.Rules {
		for _, v := range rule.Verbs {
			if v == "list" || v == "*" {
				return Result{Solved: true, Message: "Success! 'list' verb added."}
			}
		}
	}
	return Result{Solved: false, Message: "Role still missing 'list' verb."}
}

func (s *SecRBACForbidden) Cleanup(ctx context.Context) error {
	return s.clientset.CoreV1().Namespaces().Delete(ctx, s.Namespace, metav1.DeleteOptions{})
}
