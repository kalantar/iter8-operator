package iter8

import (
	"context"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	roleDefaultName = "manager-role"

	roleBindingDefaultName = "manager-rolebinding"
)

func (r *ReconcileIter8) rbacForIter8(iter8 *iter8v1alpha1.Iter8) error {

	err := r.roleForIter8(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateClusterRoleBindingForIter8(iter8)
	return err
}

func (r *ReconcileIter8) roleForIter8(iter8 *iter8v1alpha1.Iter8) error {
	found := &rbacv1.ClusterRole{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: "manager-role"}, found)
	if err != nil {
		// not present
		err = r.fromYaml("config/rbac/manager_role.yaml", iter8)
		if err != nil {
			return err
		}
	} else {
		// // make sure has owner
		// owner := found.GetOwnerReferences()
		// if 0 == len(owner) {
		// 	log.Info("Updating owner on ClusterRole", "name", found.Name)
		// 	// Set Iter8 instance as the owner and controller
		// 	log.Info("Updating owner on ClusterRole", "found original", found.GetOwnerReferences())
		// 	err = controllerutil.SetControllerReference(iter8, found, r.scheme)
		// 	if err != nil {
		// 		log.Error(err, "Can't set owner")
		// 	}
		// 	log.Info("Updating owner on ClusterRole", "found modified", found.GetOwnerReferences())
		// 	err = r.client.Update(context.TODO(), found)
		// 	return err
		// }
	}
	log.Info("ClusterRole already present", "name", found.Name)
	return nil
}

func (r *ReconcileIter8) createOrUpdateClusterRoleBindingForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	rolebinding := r.clusterRoleBindingForIter8(iter8)

	// Get current state
	found := &rbacv1.ClusterRoleBinding{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: rolebinding.Name}, found)
	if err != nil {
		return r.client.Create(context.TODO(), rolebinding)
	}

	// If changed, update
	log.Info("Service already present", "name", rolebinding.Name)
	// service.ResourceVersion = found.GetResourceVersion()
	// service.Spec = corev1.ServiceSpec{}
	// This causes errors; not sure why
	// return r.client.Update(context.TODO(), service)
	return nil
}

func (r *ReconcileIter8) clusterRoleBindingForIter8(iter8 *iter8v1alpha1.Iter8) *rbacv1.ClusterRoleBinding {
	rolebinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingDefaultName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      controllerDefaultServiceAccountName,
			Namespace: iter8.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     roleDefaultName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// This doesn't work for cluster-scoped objects; they can't be owned by a namespace-scoped thing
	// Finalizers will be used to delete this instead
	// // Set Iter8 instance as the owner and controller
	// controllerutil.SetControllerReference(iter8, sa, r.scheme)
	return rolebinding
}
