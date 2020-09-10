/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/deprecated/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/api/v1alpha1"
)

// Iter8Reconciler reconciles a Iter8 object
type Iter8Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=iter8.tools,resources=experiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iter8.tools,resources=experiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=destinationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations;validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=iter8.iter8.tools,resources=iter8s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iter8.iter8.tools,resources=iter8s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete;bind
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *Iter8Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("iter8", req.NamespacedName)

	// your logic here
	// Fetch the Iter8 instance
	instance := &iter8v1alpha1.Iter8{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("Iter8 resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get Iter8")
		return ctrl.Result{}, err
	}
	// r.Log.Info("Reconcile", "Iter8 instance", instance)

	// Add finalizer if not already present
	if !contains(instance.GetFinalizers(), finalizer) {
		r.Log.Info("finalize adding finalizer", "finalizer", finalizer)
		instance.SetFinalizers(append(instance.GetFinalizers(), finalizer))
		if err = r.Client.Update(context.TODO(), instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check whether object has been deleted
	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalize(instance)
	}

	err = r.crdsForIter8(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.rbacForIter8(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.controllerForIter8(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.analyticsEngineForIter8(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Do other things
	r.Log.Info("Reconcile ending with nil")

	return ctrl.Result{}, nil
}

func (r *Iter8Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iter8v1alpha1.Iter8{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}

// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func (r *Iter8Reconciler) fromYaml(fileName string, iter8 *iter8v1alpha1.Iter8) error {
	fileR, err := ioutil.ReadFile(fileName)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Error reading YAML file: %s", fileName))
		return err
	}

	acceptedK8sTypes := regexp.MustCompile(`(CustomResourceDefinition|Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	objects := make([]runtime.Object, 0, len(sepYamlfiles))

	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(f), nil, nil)

		if err != nil {
			r.Log.Info(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			r.Log.Info(fmt.Sprintf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind))
		} else {
			objects = append(objects, obj)
		}

	}

	for _, obj := range objects {
		// TODO: how can I do this?
		// Set Iter8 instance as the owner and controller
		// controllerutil.SetControllerReference(iter8, metav1.Object(obj), r.Scheme)
		err = r.Client.Create(context.TODO(), obj)
		if err != nil {
			return err
		}
	}

	return nil
}

const (
	finalizer = "tools.iter8.iter8-op"
)

func (r *Iter8Reconciler) finalize(iter8 *iter8v1alpha1.Iter8) error {
	r.Log.Info("finalizing")

	// if being deleted
	if iter8.GetDeletionTimestamp() != nil {
		if contains(iter8.GetFinalizers(), finalizer) {

			// Delete ClusterRoleBinding, ClusterRole, and CustomResourceDefinition
			r.Log.Info("finalize deleting ClusterRoleBinding", "name", roleBindingDefaultName)
			rolebinding := &rbacv1.ClusterRoleBinding{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{Name: roleBindingDefaultName}, rolebinding)
			if err == nil {
				err = r.Client.Delete(context.TODO(), rolebinding)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					r.Log.Error(err, "Unable to delete ClusterRoleBinding")
				}
			}

			r.Log.Info("finalize deleting ClusterRole", "name", roleDefaultName)
			role := &rbacv1.ClusterRole{}
			r.Client.Get(context.TODO(), types.NamespacedName{Name: roleDefaultName}, role)
			if err == nil {
				err = r.Client.Delete(context.TODO(), role)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					r.Log.Error(err, "Unable to delete ClusterRole")
				}
			}

			r.Log.Info("finalize deleting CustomResourceDefinition", "name", "experiments.iter8.tools")
			crd := &apiextensions.CustomResourceDefinition{}
			r.Client.Get(context.TODO(), types.NamespacedName{Name: "experiments.iter8.tools"}, crd)
			if err == nil {
				err = r.Client.Delete(context.TODO(), crd)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					r.Log.Error(err, "Unable to delete CustomResourceDefintion")
				}
			}
		}

		r.Log.Info("finalize removing finalizer", "finalizer", finalizer)
		iter8.SetFinalizers(remove(iter8.GetFinalizers(), finalizer))
		return r.Client.Update(context.TODO(), iter8)
	}

	// not being deleted
	return nil
}

// utility function determines if a str is in an array
func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// utility function removes a str from an array
func remove(arr []string, str string) []string {
	retval := make([]string, 0)
	for _, a := range arr {
		if a != str {
			retval = append(retval, a)
		}
	}
	return retval
}
