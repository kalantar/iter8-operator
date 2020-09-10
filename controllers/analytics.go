package controllers

import (
	"context"
	"fmt"
	"strconv"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/api/v1alpha1"
	"istio.io/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	analyticsDefaultName        = "iter8-analytics"
	analyticsDefaultConfigFile  = "config.yaml"
	analyticsDefaultServicePort = int32(8080)

	metricsBackendAuthType        = "authType"
	metricsBackendAuthTypeNone    = "none"
	metricsBackendAuthTypeBasic   = "basic"
	prometheusSecret              = "htpasswd"
	prometheusDefaultUsername     = "internal"
	prometheusSecretPasswordField = "rawPassword"

	analyticsDefaultBackendMetricsType = "prometheus"
	analyticsDefaultBackendMetricsURL  = "http://prometheus.istio-system:9090"
)

func (r *Iter8Reconciler) analyticsEngineForIter8(iter8 *iter8v1alpha1.Iter8) error {
	err := r.createOrUpdateConfigConfigMapForAnalytics(iter8)
	if err != nil {
		return err
	}

	err = r.createOrUpdateServiceForAnalytics(iter8)
	if err != nil {
		return err
	}
	err = r.createOrUpdateDeploymentForAnalytics(iter8)
	return err
}

func (r *Iter8Reconciler) createOrUpdateConfigConfigMapForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.configConfigMapForAnalytics(iter8)

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

func (r *Iter8Reconciler) getUsernamePassword(iter8 *iter8v1alpha1.Iter8) (string, string) {
	username := ""
	password := ""

	found := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: prometheusSecret, Namespace: iter8.Namespace}, found)
	if err != nil {
		log.Info("No secret found", "name", prometheusSecret)
		return username, password
	}
	if enc, ok := found.Data[prometheusSecretPasswordField]; ok {
		username = prometheusDefaultUsername
		return username, string(enc)
	}
	log.Info(fmt.Sprintf("No field '%s' in secret data", prometheusSecretPasswordField))
	return username, password
}

func (r *Iter8Reconciler) configConfigMapForAnalytics(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":     analyticsDefaultName,
		"app.kubernetes.io/instance": "iter8-analytics",
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)
	backendType := analyticsDefaultBackendMetricsType

	caFile := ""
	token := ""
	username := ""
	password := ""
	authType := *iter8v1alpha1.GetMetricsBackendAuthenticationType(iter8.Spec.AnalyticsEngine.MetricsBackend)
	if authType == metricsBackendAuthTypeBasic {
		username = *iter8v1alpha1.GetMetricsBackendUsername(iter8.Spec.AnalyticsEngine.MetricsBackend)
		password = *iter8v1alpha1.GetMetricsBackendPassword(iter8.Spec.AnalyticsEngine.MetricsBackend)
	}
	url := iter8v1alpha1.GetMetricsBackendURL(iter8.Spec.AnalyticsEngine.MetricsBackend, analyticsDefaultBackendMetricsURL)
	insecureSkipVerify := *iter8v1alpha1.GetMetricsBackendInsecureSkipVerify(iter8.Spec.AnalyticsEngine.MetricsBackend)

	// check for username, password stored in a secret; use if present
	// TODO?: do only if not alrady defined
	u, p := r.getUsernamePassword(iter8)
	if p != "" {
		authType = metricsBackendAuthTypeBasic
		username, password = u, p
		insecureSkipVerify = true
	}

	config := `
port: ` + strconv.FormatInt(int64(port), 10) + `
metricsBackend:
  type: ` + backendType + `
  url: ` + *url + `
  auth:
    insecure_skip_verify: ` + strconv.FormatBool(insecureSkipVerify) + `
    type: ` + authType + `
    ca_file: ` + caFile + `
    token: ` + token + `
    username: ` + username + `
    password: ` + password + `
`

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultName,
			Namespace: iter8.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"config.yaml": config,
		},
	}

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, cm, r.Scheme)
	return cm
}

func (r *Iter8Reconciler) createOrUpdateServiceForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	service := r.serviceForAnalytics(iter8)

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

func (r *Iter8Reconciler) serviceForAnalytics(iter8 *iter8v1alpha1.Iter8) *corev1.Service {
	labels := map[string]string{
		"app": analyticsDefaultName,
	}

	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultName,
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

func (r *Iter8Reconciler) createOrUpdateDeploymentForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	deployment := r.deploymentForIter8Analytics(iter8)

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

func (r *Iter8Reconciler) deploymentForIter8Analytics(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	labels := map[string]string{
		"app": analyticsDefaultName,
	}

	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.AnalyticsEngine.Deployment)
	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      analyticsDefaultName,
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
						Name:            analyticsDefaultName,
						Env: []corev1.EnvVar{{
							Name:  "METRICS_BACKEND_CONFIGFILE",
							Value: "config.yaml",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: port,
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-volume",
							MountPath: "/config.yaml",
							SubPath:   "config.yaml",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: analyticsDefaultName,
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
	controllerutil.SetControllerReference(iter8, deploy, r.Scheme)
	return deploy
}
