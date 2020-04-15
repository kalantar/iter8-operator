package iter8

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_iter8")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Iter8 Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileIter8{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("controller-manager", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Iter8
	err = c.Watch(&source.Kind{Type: &iter8v1alpha1.Iter8{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Iter8
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iter8v1alpha1.Iter8{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iter8v1alpha1.Iter8{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iter8v1alpha1.Iter8{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iter8v1alpha1.Iter8{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileIter8 implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileIter8{}

// ReconcileIter8 reconciles a Iter8 object
type ReconcileIter8 struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Iter8 object and makes changes based on the state read
// and what is in the Iter8.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIter8) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Iter8")

	// Fetch the Iter8 instance
	instance := &iter8v1alpha1.Iter8{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Iter8 resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Iter8")
		return reconcile.Result{}, err
	}
	// reqLogger.Info("Reconcile", "Iter8 instance", instance)

	// Add finalizer if not already present
	if !contains(instance.GetFinalizers(), finalizer) {
		log.Info("finalize adding finalizer", "finalizer", finalizer)
		instance.SetFinalizers(append(instance.GetFinalizers(), finalizer))
		if err = r.client.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check whether object has been deleted
	if instance.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.finalize(instance)
	}

	err = r.crdsForIter8(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.rbacForIter8(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.controllerForIter8(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.analyticsEngineForIter8(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Do other things
	reqLogger.Info("Reconcile ending with nil")
	return reconcile.Result{}, nil
}

// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func (r *ReconcileIter8) fromYaml(fileName string, iter8 *iter8v1alpha1.Iter8) error {
	fileR, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error reading YAML file: %s", fileName))
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
			log.Info(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.Info(fmt.Sprintf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind))
		} else {
			objects = append(objects, obj)
		}

	}

	for _, obj := range objects {
		// TODO: how can I do this?
		// Set Iter8 instance as the owner and controller
		// controllerutil.SetControllerReference(iter8, metav1.Object(obj), r.scheme)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			return err
		}
	}

	return nil
}

const (
	finalizer = "tools.iter8.iter8-op"
)

func (r *ReconcileIter8) finalize(iter8 *iter8v1alpha1.Iter8) error {
	log.Info("finalizing")

	// if being deleted
	if iter8.GetDeletionTimestamp() != nil {
		if contains(iter8.GetFinalizers(), finalizer) {

			// Delete ClusterRoleBinding, ClusterRole, and CustomResourceDefinition
			log.Info("finalize deleting ClusterRoleBinding", "name", "manager-rolebinding")
			rolebinding := &rbacv1.ClusterRoleBinding{}
			err := r.client.Get(context.TODO(), types.NamespacedName{Name: "manager-rolebinding"}, rolebinding)
			if err == nil {
				err = r.client.Delete(context.TODO(), rolebinding)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					return err
				}
			}

			log.Info("finalize deleting ClusterRole", "name", "manager-role")
			role := &rbacv1.ClusterRole{}
			r.client.Get(context.TODO(), types.NamespacedName{Name: "manager-role"}, role)
			if err == nil {
				err = r.client.Delete(context.TODO(), role)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					return err
				}
			}

			log.Info("finalize deleting CustomResourceDefinition", "name", "experiments.iter8.tools")
			crd := &apiextensions.CustomResourceDefinition{}
			r.client.Get(context.TODO(), types.NamespacedName{Name: "experiments.iter8.tools"}, crd)
			if err == nil {
				err = r.client.Delete(context.TODO(), crd)
				if err != nil {
					return err
				}
			} else {
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}

		log.Info("finalize removing finalizer", "finalizer", finalizer)
		iter8.SetFinalizers(remove(iter8.GetFinalizers(), finalizer))
		return r.client.Update(context.TODO(), iter8)
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
