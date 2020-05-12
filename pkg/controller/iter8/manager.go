package iter8

import (
	"context"
	"strings"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	controllerDefaultServiceAccountName = "controller-manager"

	controllerDefaultServiceName = "controller-manager-service"
	controllerDefaultServicePort = int32(443)

	controllerDefaultDeploymentName        = "controller-manager"
	controllerDefaultDeploymentGracePeriod = int64(10)

	metricsDefaultConfigMapName = "iter8config-metrics"
	istioTelemetryV1            = "v1"
	istioTelemetryV2            = "v2"

	notifiersDefaultConfigMapName = "iter8config-notifiers"
)

func (r *ReconcileIter8) controllerForIter8(iter8 *iter8v1alpha1.Iter8) error {

	err := r.createOrUpdateServiceAccount(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateNotifierConfigMapForIter8(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateMetricsConfigMapForIter8(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateServiceForController(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateDeploymentForController(iter8)
	return err
}

func (r *ReconcileIter8) createOrUpdateNotifierConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.notifierConfigMapForIter8(iter8)

	// Get current state
	found := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), cm)
	}

	// If changed, update
	log.Info("ConfigMap already present", "name", cm.Name)
	// log.Info("ConfigMap already present", "name", cm.Name, "resource", found.Data)
	// cm.ResourceVersion = found.GetResourceVersion()
	// return r.client.Update(context.TODO(), cm)
	return nil
}

func (r *ReconcileIter8) notifierConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      notifiersDefaultConfigMapName,
			Namespace: iter8.Namespace,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.scheme)
	return cm
}

func (r *ReconcileIter8) createOrUpdateMetricsConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.metricsConfigMapForIter8(iter8)

	// Get current state
	found := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), cm)
	}

	// If changed, update
	log.Info("ConfigMap already present", "name", cm.Name)
	// log.Info("ConfigMap already present", "name", cm.Name, "resource", found.Data)
	// cm.ResourceVersion = found.GetResourceVersion()
	// return r.client.Update(context.TODO(), cm)
	return nil
}

// For mapping, see:
// https://github.com/iter8-tools/iter8-controller/issues/98#issuecomment-613084721
func (r *ReconcileIter8) metricsConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	counterMetrics := iter8v1alpha1.GetCounterMetrics(iter8.Spec.Metrics)
	ratioMetrics := iter8v1alpha1.GetRatioMetrics(iter8.Spec.Metrics)

	queryTemplateCache := map[string]string{}

	queryTemplates := ``
	metrics := ``

	for _, metric := range *counterMetrics {
		name := metric.Name
		qt := strings.Replace(metric.QueryTemplate, "version_labels", "entity_labels", -1)
		if istioTelemetryV1 == iter8v1alpha1.GetIstioTelemetryVersion(iter8.Spec.Metrics) {
			qt = strings.Replace(qt, "istio_request_duration_milliseconds_sum", "istio_request_duration_seconds_sum", -1)
		}
		queryTemplateCache[name] = qt
		queryTemplates += `
` + name + `: "` + qt + `"`
		metrics += `
- name: ` + name + `
  is_counter: True
  sample_size_query_template: iter8_request_count`
	}

	for _, metric := range *ratioMetrics {
		name := metric.Name
		if name == "iter8_mean_latency" {
			name = "iter8_latency"
		}
		numerator := metric.Numerator
		denominator := metric.Denominator
		queryTemplates += `
` + name + `: "(` + queryTemplateCache[numerator] + `) / (` + queryTemplateCache[denominator] + `)"`
		metrics += `
- name: ` + name + `
  is_counter: False
  absent_value: "None"
  sample_size_query_template: iter8_request_count`
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

func (r *ReconcileIter8) createOrUpdateServiceAccount(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	serviceAccount := r.serviceAccountForIter8Controller(iter8)

	// Get current state
	found := &corev1.ServiceAccount{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccount.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), serviceAccount)
	}

	// If changed, update
	// log.Info("ServiceAccount already present", "name", serviceAccount.Name)
	// serviceAccount.ResourceVersion = found.GetResourceVersion()
	// return r.client.Update(context.TODO(), serviceAccount)
	return nil
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

func (r *ReconcileIter8) createOrUpdateServiceForController(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	service := r.serviceForIter8Controller(iter8)

	// Get current state
	found := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), service)
	}

	// If changed, update
	log.Info("Service already present", "name", service.Name)
	// service.ResourceVersion = found.GetResourceVersion()
	// service.Spec = corev1.ServiceSpec{}
	// This causes errors; not sure why
	// return r.client.Update(context.TODO(), service)
	return nil
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

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, svc, r.scheme)
	return svc
}

func (r *ReconcileIter8) createOrUpdateDeploymentForController(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	deployment := r.deploymentForIter8Controller(iter8)

	// Get current state
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), deployment)
	}

	// If changed, update
	log.Info("Deployment already present", "name", deployment.Name)
	// deployment.ResourceVersion = found.GetResourceVersion()
	// return r.client.Update(context.TODO(), deployment)
	return nil
}

func (r *ReconcileIter8) deploymentForIter8Controller(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	// reqLogger := log.WithValues("Request.Namespace", iter8.Namespace, "Request.Name", iter8.Name)

	labels := map[string]string{
		"app": "controller-manager",
	}

	serviceAccountName := controllerDefaultServiceAccountName
	gracePeriod := controllerDefaultDeploymentGracePeriod
	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.Controller.Deployment)

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
							Name: "POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
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
