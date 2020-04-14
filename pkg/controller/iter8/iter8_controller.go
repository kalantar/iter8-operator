package iter8

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Deployment as an example
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
	reqLogger.Info("Reconcile", "instance", instance)

	// // Define a new Pod object
	// pod := newPodForCR(instance)

	// // Set Iter8 instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// Check if this Pod already exists
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "controller-manager", Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		err := r.controllerForIter8(instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.analyticsEngineForIter8(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Resources created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get controller")
		return reconcile.Result{}, err
	}

	// Do other things

	// Controller already exists - don't requeue
	reqLogger.Info("Skip reconcile: controller already exists", "Namespace", found.Namespace, "Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileIter8) controllerForIter8(iter8 *iter8v1alpha1.Iter8) error {
	fileR, err := ioutil.ReadFile("config/crds/iter8.tools_experiments.yaml")
	if err != nil {
		switch err {
		case os.ErrInvalid:
			log.Info("os.ErrInvalid")
		case os.ErrPermission:
			log.Info("os.ErrPermission")
		case os.ErrNotExist:
			log.Info("os.ErrNotExist")
		default:
			log.Error(err, "read failed 1")
		}
		log.Error(err, "read failed 2")
		return err
	}
	objects := r.crdForIter8(fileR)
	log.Info("controllerForIter8", "objects", objects)

	for _, obj := range objects {
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			return err
		}
	}

	serviceAccount := r.serviceAccountForIter8Controller(iter8)
	err = r.client.Create(context.TODO(), serviceAccount)
	if err != nil {
		return err
	}
	notifier := r.notifierConfigMapForIter8(iter8)
	err = r.client.Create(context.TODO(), notifier)
	if err != nil {
		return err
	}
	metrics := r.metricsConfigMapForIter8(iter8)
	err = r.client.Create(context.TODO(), metrics)
	if err != nil {
		return err
	}
	service := r.serviceForIter8Controller(iter8)
	err = r.client.Create(context.TODO(), service)
	if err != nil {
		return err
	}
	deployment := r.deploymentForIter8Controller(iter8)
	err = r.client.Create(context.TODO(), deployment)
	return err
}

func (r *ReconcileIter8) analyticsEngineForIter8(iter8 *iter8v1alpha1.Iter8) error {
	cm := r.configmapForIter8Analytics(iter8)
	err := r.client.Create(context.TODO(), cm)
	if err != nil {
		return err
	}

	service := r.serviceForIter8Analytics(iter8)
	err = r.client.Create(context.TODO(), service)
	if err != nil {
		return err
	}
	deployment := r.deploymentForIter8Analytics(iter8)
	err = r.client.Create(context.TODO(), deployment)
	return err
}

const (
	controllerDefaultServiceAccountName = "controller-manager"

	controllerDefaultServiceName = "controller-manager-service"
	controllerDefaultServiceType = "ClusterIP"
	controllerDefaultServicePort = int32(8443)

	controllerDefaultDeploymentName        = "controller-manager"
	controllerDefaultDeploymentGracePeriod = int64(10)

	metricsDefaultConfigMapName   = "iter8config-metrics"
	notifiersDefaultConfigMapName = "iter8config-notifiers"

	analyticsDefaultConfigMapName = "iter8-analytics"
	analyticsDefaultServiceName   = "iter8-analytics"
	analyticsDefaultServiceType   = "ClusterIP"
	analyticsDefaultServicePort   = int32(8080)

	analyticsDefaultDeploymentName    = "iter8-analytics"
	analyticsDefaultBackendMetricsURL = "http://prometheus.istio-system:9090"
)

// crdForIter8
// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
func (r *ReconcileIter8) crdForIter8(fileR []byte) []runtime.Object {
	acceptedK8sTypes := regexp.MustCompile(`(CustomResourceDefinition|Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object, 0, len(sepYamlfiles))

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
			retVal = append(retVal, obj)
		}

	}
	return retVal
}

func (r *ReconcileIter8) notifierConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      notifiersDefaultConfigMapName,
			Namespace: iter8.Namespace,
		},
		Data: map[string]string{
			"data": "",
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.scheme)
	return cm
}

func (r *ReconcileIter8) metricsConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	counterMetrics := iter8v1alpha1.GetCounterMetrics(iter8.Spec.Metrics)
	ratioMetrics := iter8v1alpha1.GetRatioMetrics(iter8.Spec.Metrics)

	queryTemplateCache := map[string]string{
		"iter8_sample_size": "sum(increase(istio_requests_total{source_workload_namespace!='knative-serving',reporter='source'}[$interval]$offset_str)) by ($entity_labels)",
	}

	queryTemplates := `iter8_sample_size: sum(increase(istio_requests_total{source_workload_namespace!='knative-serving',reporter='source'}[$interval]$offset_str)) by ($entity_labels)"`
	metrics := ``

	for _, metric := range *counterMetrics {
		name := metric.Name
		qt := metric.QueryTemplate
		queryTemplateCache[name] = qt
		queryTemplates += `
` + name + `: ` + qt
		metrics += `
- name: ` + name + `
  is_counter: True
  absent_value: "None"
  sample_size_query_template: iter8_sample_size`
	}

	for _, metric := range *ratioMetrics {
		name := metric.Name
		numerator := metric.Numerator
		denominator := metric.Denominator
		queryTemplates += `
` + name + `: (` + queryTemplateCache[numerator] + `)/(` + queryTemplateCache[denominator] + `)`
		metrics += `
- name: ` + name + `
  is_counter: False
  absent_value: "None"
  sample_size_query_template: iter8_sample_size`
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsDefaultConfigMapName,
			Namespace: iter8.Namespace,
		},
		Data: map[string]string{
			"query_templates": queryTemplates,
			"metrics":         metrics,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.scheme)
	return cm
}

func (r *ReconcileIter8) serviceAccountForIter8Controller(iter8 *iter8v1alpha1.Iter8) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "controller-manager",
			Namespace: iter8.Namespace,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, sa, r.scheme)
	return sa
}

func (r *ReconcileIter8) serviceForIter8Controller(iter8 *iter8v1alpha1.Iter8) *corev1.Service {
	labels := map[string]string{
		"control-plane": "controller-manager",
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.Controller.Service, controllerDefaultServicePort)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerDefaultServiceName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port: port,
			}},
		},
	}

	svcType := iter8v1alpha1.GetServiceType(iter8.Spec.Controller.Service)
	if nil != svcType {
		svc.Spec.Type = *svcType
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, svc, r.scheme)
	return svc
}

func (r *ReconcileIter8) deploymentForIter8Controller(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	// reqLogger := log.WithValues("Request.Namespace", iter8.Namespace, "Request.Name", iter8.Name)

	labels := map[string]string{
		"app": "controller-manager",
	}

	serviceAccountName := controllerDefaultServiceAccountName
	gracePeriod := controllerDefaultDeploymentGracePeriod
	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.Controller.Deployment)
	port := iter8v1alpha1.GetServicePort(iter8.Spec.Controller.Service, controllerDefaultServicePort)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerDefaultDeploymentName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: &gracePeriod,
					Containers: []corev1.Container{{
						Image:           iter8.Spec.Controller.Deployment.Image,
						ImagePullPolicy: iter8v1alpha1.GetImagePullPolicy(iter8.Spec.Controller.Deployment),
						Name:            controllerDefaultDeploymentName,
						Command:         []string{"/manager"},
						Env: []corev1.EnvVar{{
							Name:  "POD_NAMESPACE",
							Value: iter8.Spec.Namespace,
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: port,
						}},
					}},
				},
			},
		},
	}

	rsrc := iter8.Spec.Controller.Deployment.Resources
	if nil != rsrc {
		deploy.Spec.Template.Spec.Containers[0].Resources = *rsrc
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, deploy, r.scheme)
	return deploy

}

func (r *ReconcileIter8) configmapForIter8Analytics(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":     "iter8-analytics",
		"app.kubernetes.io/instance": "iter8-analytics",
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)
	config := `
port: ` + strconv.FormatInt(int64(port), 10) + `
prometheus:`

	for i := 0; i < iter8v1alpha1.GetNumMetricsBackends(iter8.Spec.AnalyticsEngine.MetricsBackends); i++ {
		authType := *iter8v1alpha1.GetMetricsBackendAuthenticationType(iter8.Spec.AnalyticsEngine.MetricsBackends, i)
		insecureSkipVerify := *iter8v1alpha1.GetMetricsBackendInsecureSkipVerify(iter8.Spec.AnalyticsEngine.MetricsBackends, i)
		caFile := ""
		token := ""
		username := ""
		password := ""
		if authType == "basic" {
			username = *iter8v1alpha1.GetMetricsBackendUsername(iter8.Spec.AnalyticsEngine.MetricsBackends, i)
			password = *iter8v1alpha1.GetMetricsBackendPassword(iter8.Spec.AnalyticsEngine.MetricsBackends, i)
		}
		url := iter8v1alpha1.GetMetricsBackendURL(iter8.Spec.AnalyticsEngine.MetricsBackends, i, analyticsDefaultBackendMetricsURL)

		config += `
  - url: ` + *url + `
	auth:
	  insecure_skip_verify: ` + strconv.FormatBool(insecureSkipVerify) + `
	  type: ` + authType + `
	  ca_file: ` + caFile + `
	  token: ` + token + `
	  username: ` + username + `
	  password: ` + password + `
`
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultConfigMapName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"config.yaml": config,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.scheme)
	return cm
}

func (r *ReconcileIter8) serviceForIter8Analytics(iter8 *iter8v1alpha1.Iter8) *corev1.Service {
	labels := map[string]string{
		"app.kubernetes.io/name":     "iter8-analytics",
		"app.kubernetes.io/instance": "iter8-analytics",
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultServiceName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port: port,
			}},
		},
	}

	svcType := iter8v1alpha1.GetServiceType(iter8.Spec.AnalyticsEngine.Service)
	if nil != svcType {
		svc.Spec.Type = *svcType
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, svc, r.scheme)
	return svc
}

func (r *ReconcileIter8) deploymentForIter8Analytics(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	// reqLogger := log.WithValues("Request.Namespace", iter8.Namespace, "Request.Name", iter8.Name)

	labels := map[string]string{
		"app.kubernetes.io/name":     "iter8-analytics",
		"app.kubernetes.io/instance": "iter8-analytics",
	}

	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.AnalyticsEngine.Deployment)
	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)
	backendURL := iter8v1alpha1.GetMetricsBackendURL(iter8.Spec.AnalyticsEngine.MetricsBackends, 0, analyticsDefaultBackendMetricsURL)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultDeploymentName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           iter8.Spec.AnalyticsEngine.Deployment.Image,
						ImagePullPolicy: iter8v1alpha1.GetImagePullPolicy(iter8.Spec.AnalyticsEngine.Deployment),
						Name:            analyticsDefaultDeploymentName,
						Env: []corev1.EnvVar{{
							Name:  "ITER8_ANALYTICS_SERVER_PORT",
							Value: strconv.FormatInt(int64(port), 10),
						}, {
							Name:  "ITER8_ANALYTICS_METRICS_BACKEND_URL",
							Value: *backendURL,
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: port,
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-volume",
							MountPath: "/python_code/src/config.yaml",
							SubPath:   "config.yaml",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: analyticsDefaultConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}

	rsrc := iter8.Spec.AnalyticsEngine.Deployment.Resources
	if nil != rsrc {
		deploy.Spec.Template.Spec.Containers[0].Resources = *rsrc
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, deploy, r.scheme)
	return deploy
}
