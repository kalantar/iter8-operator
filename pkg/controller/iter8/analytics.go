package iter8

import (
	"context"
	"strconv"

	iter8v1alpha1 "github.com/iter8-tools/iter8-operator/pkg/apis/iter8/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	analyticsDefaultConfigMapName = "iter8-analytics"
	analyticsDefaultServiceName   = "iter8-analytics"
	analyticsDefaultServicePort   = int32(8080)

	metricsBackendAuthType        = "authType"
	metricsBackendAuthTypeNone    = "none"
	metricsBackendAuthTypeBasic   = "basic"
	prometheusSecret              = "htpasswd"
	prometheusDefaultUsername     = "internal"
	prometheusSecretPasswordField = "rawPassword"

	analyticsDefaultDeploymentName     = "iter8-analytics"
	analyticsDefaultBackendMetricsType = "prometheus"
	analyticsDefaultBackendMetricsURL  = "http://prometheus.istio-system:9090"
)

func (r *ReconcileIter8) analyticsEngineForIter8(iter8 *iter8v1alpha1.Iter8) error {
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

func (r *ReconcileIter8) createOrUpdateConfigConfigMapForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	cm := r.configConfigMapForAnalytics(iter8)

	// Get current state
	found := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: iter8.Namespace}, found)
	if err != nil {
		return r.client.Create(context.TODO(), cm)
	}

	// If changed, update
	log.Info("ConfigMap already present", "name", cm.Name)
	// cm.ResourceVersion = found.GetResourceVersion()
	// return r.client.Update(context.TODO(), cm)
	return nil
}

func (r *ReconcileIter8) getUsernamePassword(iter8 *iter8v1alpha1.Iter8) (string, string) {
	username := ""
	password := ""

	found := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: prometheusSecret, Namespace: iter8.Namespace}, found)
	if err != nil {
		log.Info("No secret found", "name", prometheusSecret)
		return username, password
	}
	if enc, ok := found.Data[prometheusSecretPasswordField]; ok {
		username = prometheusDefaultUsername
		return username, string(enc)
	}
	log.Info("No such field in secret data", "name", prometheusSecretPasswordField)
	return username, password
}

func (r *ReconcileIter8) configConfigMapForAnalytics(iter8 *iter8v1alpha1.Iter8) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":     "iter8-analytics",
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

func (r *ReconcileIter8) createOrUpdateServiceForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	service := r.serviceForAnalytics(iter8)

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

func (r *ReconcileIter8) serviceForAnalytics(iter8 *iter8v1alpha1.Iter8) *corev1.Service {
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

	// Set Iter8 instance as the owner and controller
	controllerutil.SetControllerReference(iter8, svc, r.scheme)
	return svc
}

func (r *ReconcileIter8) createOrUpdateDeploymentForAnalytics(iter8 *iter8v1alpha1.Iter8) error {
	// Desired state
	deployment := r.deploymentForIter8Analytics(iter8)

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

func (r *ReconcileIter8) deploymentForIter8Analytics(iter8 *iter8v1alpha1.Iter8) *appsv1.Deployment {
	// reqLogger := log.WithValues("Request.Namespace", iter8.Namespace, "Request.Name", iter8.Name)

	labels := map[string]string{
		"app.kubernetes.io/name":     "iter8-analytics",
		"app.kubernetes.io/instance": "iter8-analytics",
	}

	replicaCount := iter8v1alpha1.GetReplicaCount(iter8.Spec.AnalyticsEngine.Deployment)
	port := iter8v1alpha1.GetServicePort(iter8.Spec.AnalyticsEngine.Service, analyticsDefaultServicePort)
	backendURL := iter8v1alpha1.GetMetricsBackendURL(iter8.Spec.AnalyticsEngine.MetricsBackend, analyticsDefaultBackendMetricsURL)

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
