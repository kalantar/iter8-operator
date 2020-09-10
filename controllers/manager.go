package controllers

import (
	"context"
	"fmt"
	"strings"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	"istio.io/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	controllerDefaultName = "iter8-controller"

	controllerDefaultServicePort = int32(443)

	controllerDefaultDeploymentGracePeriod = int64(10)

	metricsDefaultConfigMapName = "iter8config-metrics"
	istioTelemetryV1            = "v1"
	istioTelemetryV2            = "v2"

	notifiersDefaultConfigMapName = "iter8config-notifiers"
)

func (r *Iter8Reconciler) controllerForIter8(iter8 *iter8v1alpha1.Iter8) error {

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

func (r *Iter8Reconciler) createOrUpdateNotifierConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.notifierConfigMapForIter8(iter8)

	// Get current state
	found := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), cm)
	}

	// If changed, update
	log.Info(fmt.Sprintf("ConfigMap '%s' already present", cm.Name))
	// cm.ResourceVersion = found.GetResourceVersion()
	// return r.Client.Update(context.TODO(), cm)
	return nil
}

func (r *Iter8Reconciler) notifierConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      notifiersDefaultConfigMapName,
			Namespace: iter8.Namespace,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.Scheme)
	return cm
}

func (r *Iter8Reconciler) createOrUpdateMetricsConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.metricsConfigMapForIter8(iter8)

	// Get current state
	found := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), cm)
	}

	// If changed, update
	log.Info(fmt.Sprintf("ConfigMap '%s' already present", cm.Name))
	// cm.ResourceVersion = found.GetResourceVersion()
	// return r.Client.Update(context.TODO(), cm)
	return nil
}

func (r *Iter8Reconciler) metricsConfigMapForIter8(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	counterMetrics := iter8v1alpha1.GetCounterMetrics(iter8.Spec.Metrics)
	ratioMetrics := iter8v1alpha1.GetRatioMetrics(iter8.Spec.Metrics)
	istioTelemetryVersion := iter8v1alpha1.GetIstioTelemetryVersion(iter8.Spec.Metrics)

	if istioTelemetryV1 == istioTelemetryVersion {
		log.Info("istioTelemetry v1 identified")
		for _, metric := range *counterMetrics {
			metric.QueryTemplate = strings.Replace(metric.QueryTemplate, "istio_request_duration_milliseconds_sum", "istio_request_duration_seconds_sum", -1)
			metric.QueryTemplate = strings.Replace(metric.QueryTemplate, "envoy-stats", "istio-mesh", -1)
		}
	}

	counterMetricsYaml, err := yaml.Marshal(counterMetrics)
	if err != nil {
		counterMetricsYaml = make([]byte, 0)
	}

	ratioMetricsYaml, err := yaml.Marshal(ratioMetrics)
	if err != nil {
		ratioMetricsYaml = make([]byte, 0)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsDefaultConfigMapName,
			Namespace: iter8.Namespace,
		},
		Data: map[string]string{
			"counter_metrics.yaml": string(counterMetricsYaml),
			"ratio_metrics.yaml":   string(ratioMetricsYaml),
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.Scheme)
	return cm
}

func (r *Iter8Reconciler) createOrUpdateServiceAccount(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	serviceAccount := r.serviceAccountForIter8Controller(iter8)

	// Get current state
	found := &corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: serviceAccount.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), serviceAccount)
	}

	// If changed, update
	log.Info(fmt.Sprintf("ServiceAccount '%s' already present", serviceAccount.Name))
	// serviceAccount.ResourceVersion = found.GetResourceVersion()
	// return r.Client.Update(context.TODO(), serviceAccount)
	return nil
}

func (r *Iter8Reconciler) serviceAccountForIter8Controller(iter8 *iter8v1alpha1.Iter8) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerDefaultName,
			Namespace: iter8.Namespace,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, sa, r.Scheme)
	return sa
}

func (r *Iter8Reconciler) createOrUpdateServiceForController(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	service := r.serviceForIter8Controller(iter8)

	// Get current state
	found := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), service)
	}

	// If changed, update
	log.Info(fmt.Sprintf("Service '%s' already present", service.Name))
	// service.ResourceVersion = found.GetResourceVersion()
	// service.Spec = corev1.ServiceSpec{}
	// This causes errors; not sure why
	// return r.Client.Update(context.TODO(), service)
	return nil
}

func (r *Iter8Reconciler) serviceForIter8Controller(iter8 *iter8v1alpha1.Iter8) *corev1.Service {
	labels := map[string]string{
		"app": controllerDefaultName,
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.Controller.Service, controllerDefaultServicePort)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerDefaultName,
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
	controllerutil.SetControllerReference(iter8, svc, r.Scheme)
	return svc
}

func (r *Iter8Reconciler) createOrUpdateDeploymentForController(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	deployment := r.deploymentForIter8Controller(iter8)

	// Get current state
	found := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.Client.Create(context.TODO(), deployment)
	}

	// If changed, update
	log.Info(fmt.Sprintf("Deployment '%s' already present", deployment.Name))
	// deployment.ResourceVersion = found.GetResourceVersion()
	// return r.Client.Update(context.TODO(), deployment)
	return nil
}

func (r *Iter8Reconciler) deploymentForIter8Controller(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	labels := map[string]string{
		"app": controllerDefaultName,
	}

	serviceAccountName := controllerDefaultName
	gracePeriod := controllerDefaultDeploymentGracePeriod
	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.Controller.Deployment)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerDefaultName,
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
						Name:            controllerDefaultName,
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
	controllerutil.SetControllerReference(iter8, deploy, r.Scheme)
	return deploy

}
