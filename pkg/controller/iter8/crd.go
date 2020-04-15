package iter8

import (
	"context"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileIter8) crdsForIter8(iter8 *iter8v1alpha1.Iter8) error {
	found := &apiextensions.CustomResourceDefinition{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: "experiments.iter8.tools"}, found)
	if err != nil {
		// not present
		err = r.fromYaml("config/crds/iter8.tools_experiments.yaml", iter8)
		if err != nil {
			return err
		}
	} else {
		// // make sure has owner
		// owner := found.GetOwnerReferences()
		// if 0 == len(owner) {
		// 	log.Info("Updating owner on CustomResourceDefinition", "name", found.Name)
		// 	// Set Iter8 instance as the owner and controller
		// 	controllerutil.SetControllerReference(iter8, found, r.scheme)
		// 	err = r.client.Update(context.TODO(), found)
		// 	return err
		// }
	}
	log.Info("CustomResourceDefinition already present", "name", found.Name)
	return nil
}
