package controllers

import (
	"context"
	"fmt"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/api/v1alpha1"
	"istio.io/pkg/log"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	roleDefaultName = "iter8-controller-role"

	roleBindingDefaultName = "iter8-controller-rolebinding"
)

func (r *Iter8Reconciler) rbacForIter8(iter8 *iter8v1alpha1.Iter8) error {

	err := r.roleForIter8(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateClusterRoleBindingForIter8(iter8)
	return err
}

func (r *Iter8Reconciler) roleForIter8(iter8 *iter8v1alpha1.Iter8) error {
	found := &rbacv1.ClusterRole{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: roleDefaultName}, found)
	if err != nil {
		// not present
		err = r.fromYaml("config/iter8/role.yaml", iter8)
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
		log.Info(fmt.Sprintf("ClusterRole '%s' already present", found.Name))
	}
	return nil
}

func (r *Iter8Reconciler) createOrUpdateClusterRoleBindingForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	rolebinding := r.clusterRoleBindingForIter8(iter8)

	// Get current state
	found := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: rolebinding.Name}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), rolebinding)
	}

	// If changed, update
	log.Info(fmt.Sprintf("ClusterRoleBinding '%s' already present", rolebinding.Name))
	// service.ResourceVersion = found.GetResourceVersion()
	// service.Spec = corev1.ServiceSpec{}
	// This causes errors; not sure why
	// return r.client.Update(context.TODO(), service)
	return nil
}

func (r *Iter8Reconciler) clusterRoleBindingForIter8(iter8 *iter8v1alpha1.Iter8) *rbacv1.ClusterRoleBinding {
	rolebinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingDefaultName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      controllerDefaultName,
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
