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

package v1alpha1

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Iter8Spec defines the desired state of Iter8
type Iter8Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespace is namespace in which should be deployed. Defaults to istio-system.
	Namespace string `json:"namespace,omitempty"`
	// Controller is specification of controller
	Controller ControllerSpec `json:"controller"`
	// AnalyticsEngine is specification of analytics engine used by controller
	AnalyticsEngine AnalyticsEngineSpec `json:"analyticsEngine"`
	// Metrics is list of system defined metrics
	Metrics MetricsSpec `json:"metrics"`
}

// Iter8Status defines the observed state of Iter8
type Iter8Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// ControllerSpec describes the deployment of the iter8 controller
type ControllerSpec struct {
	// Service details of service
	Service *ServiceSpec `json:"service,omitempty"`
	// Deployment details of deployment
	Deployment DeploymentSpec `json:"deployment"`
}

// AnalyticsEngineSpec describes the deployment of the iter8 analytics engine
type AnalyticsEngineSpec struct {
	// Service details of service
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`
	// Deployment details of deployment
	Deployment DeploymentSpec `json:"deployment"`
	// MetricsBackends list of metrics backends. Default is single prometheus service with basic authentication in a default location.
	// +optional
	MetricsBackend *MetricsBackendSpec `json:"metricsBackend,omitempty"`
}

// ServiceSpec describes the service to be deployed
type ServiceSpec struct {
	// Port on which service will listen, default is 8080
	Port *int32 `json:"port,omitempty"`
}

// DeploymentSpec describes the deployment of the service
type DeploymentSpec struct {
	// ReplicaCount is number of replicas. Defaults to 1.
	// +optional
	ReplicaCount *int32 `json:"replicaCount,omitempty"`
	// Image is Docker image name. More info: https://kubernetes.io/docs/concepts/containers/images
	Image string `json:"image"`
	// ImagePullPolicy is the image pull policy. One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	//+kubebuilder:validation:Enum={Always,Never,IfNotPresent}
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Resources resource requirements for container
	// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#resourcerequirements-v1-core
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MetricsBackendSpec describes a backend from which metrics are collected
type MetricsBackendSpec struct {
	// Type of metrics backend. Defaults to Prometheus.
	// +optional
	//+kubebuilder:validation:Enum={prometheus}
	Type *string `json:"type,omitempty"`
	// URL of metrics backend. Defaults to http://prometheus.istio-system:9090
	// +optional
	URL            *string                           `json:"url,omitempty"`
	Authentication *MetricsBackendAuthenticationSpec `json:"authentication,omitempty"`
}

// MetricsBackendAuthenticationSpec is specification for authentication
type MetricsBackendAuthenticationSpec struct {
	InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`
	// Type is type of authentication. Defaults to "none".
	// +optional
	//+kubebuilder:validation:Enum={none,basic}
	Type *string `json:"type,omitempty"`
	// Username is username when authenticationType is "basic"
	// +optional
	Username *string `json:"username,omitempty"`
	// Password is password when authenticationType is "basic"
	// +optional
	Password *string `json:"password,omitempty"`
}

// MetricsSpec list of available metrics
type MetricsSpec struct {
	// CounterMetrics
	CounterMetrics *[]CounterMetricSpec `json:"counter,omitempty" yaml:"counter,omitempty"`
	// RatioMetrics
	RatioMetrics *[]RatioMetricSpec `json:"ratio,omitempty" yaml:"ratio,omitempty"`
}

// CounterMetricSpec defines a counter type metric
type CounterMetricSpec struct {
	// Name
	Name string `json:"name" yaml:"name"`
	// QueryTemplate
	QueryTemplate string `json:"query_template" yaml:"query_template"`
	// PreferredDirection
	// +optional
	//+kubebuilder:validation=Enum{lower,higher}
	PreferredDirection *string `json:"preferred_direction,omitempty" yaml:"preferred_direction,omitempty"`
	// Units
	//+kubebuilder:validation:Enum={msec,sec}
	Units *string `json:"units,omitempty" yaml:"units,omitempty"`
}

// RatioMetricSpec defines a ratio type metric
type RatioMetricSpec struct {
	// name of metric
	Name string `json:"name" yaml:"name"`

	// Counter metric used in numerator
	Numerator string `json:"numerator" yaml:"numerator"`

	// Counter metric used in denominator
	Denominator string `json:"denominator" yaml:"denominator"`

	// Preferred direction of the metric value
	// +optional
	//+kubebuilder:validation=Enum{lower,higher}
	PreferredDirection *string `json:"preferred_direction,omitempty" yaml:"preferred_direction,omitempty"`

	// Boolean flag indicating if the value of this metric is always in the range 0 to 1
	// +optional
	ZeroToOne *bool `json:"zero_to_one,omitempty" yaml:"zero_to_one,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Iter8 is the Schema for the iter8s API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Iter8 is the Schema for the iter8s API
type Iter8 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Iter8Spec   `json:"spec,omitempty"`
	Status Iter8Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Iter8List contains a list of Iter8
type Iter8List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Iter8 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Iter8{}, &Iter8List{})
}

// GetReplicaCount returns specified replica count or default
func GetReplicaCount(deploy DeploymentSpec) int32 {
	replicaCount := int32(1)

	rc := deploy.ReplicaCount
	if nil != rc {
		replicaCount = *rc
	}

	return replicaCount
}

// GetImagePullPolicy returns specified pull policy or default
func GetImagePullPolicy(deploy DeploymentSpec) corev1.PullPolicy {
	// Default pull policy is IfNotPresent
	pullPolicy := corev1.PullIfNotPresent

	// Unless image has tag :default in which case it is Always
	if strings.HasSuffix(deploy.Image, ":default") {
		pullPolicy = corev1.PullAlways
	}
	// if no tag then is implicit :default
	if strings.Contains(deploy.Image, ":") {
		pullPolicy = corev1.PullAlways
	}

	// override if defined in resource
	pp := deploy.ImagePullPolicy
	if nil != pp {
		pullPolicy = *pp
	}

	return pullPolicy
}

// GetServicePort returns specified replica count or default
func GetServicePort(svc *ServiceSpec, defaultPort int32) int32 {
	port := defaultPort

	// ServiceSpec is optional; test for it before using it
	if nil == svc {
		return port
	}

	// Port is optional; test for it before using it
	p := svc.Port
	if nil != p {
		port = *p
	}

	return port
}

// GetMetricsBackendURL returns url of the metrics backend
func GetMetricsBackendURL(mbes *MetricsBackendSpec, defaultURL string) *string {
	if nil == mbes {
		return &defaultURL
	}
	result := mbes.URL
	if nil == result {
		return &defaultURL
	}
	return result
}

// GetMetricsBackendUsername returns username of the metrics backend
func GetMetricsBackendUsername(mbes *MetricsBackendSpec) *string {
	defaultValue := ""

	if nil == mbes {
		return &defaultValue
	}
	auth := mbes.Authentication
	if nil == auth {
		return &defaultValue
	}
	result := auth.Username
	if nil == result {
		return &defaultValue
	}
	return result
}

// GetMetricsBackendPassword returns password of the metrics backend
func GetMetricsBackendPassword(mbes *MetricsBackendSpec) *string {
	defaultValue := ""

	if nil == mbes {
		return &defaultValue
	}
	auth := mbes.Authentication
	if nil == auth {
		return &defaultValue
	}
	result := auth.Password
	if nil == result {
		return &defaultValue
	}
	return result
}

// GetMetricsBackendAuthenticationType returns required authentication method for the metrics backend
func GetMetricsBackendAuthenticationType(mbes *MetricsBackendSpec) *string {
	defaultValue := "none"

	if nil == mbes {
		return &defaultValue
	}
	auth := mbes.Authentication
	if nil == auth {
		return &defaultValue
	}
	result := auth.Type
	if nil == result {
		return &defaultValue
	}
	return result
}

// GetMetricsBackendInsecureSkipVerify returns whether or not to skip verification for selected metrics backend
func GetMetricsBackendInsecureSkipVerify(mbes *MetricsBackendSpec) *bool {
	defaultValue := false

	if nil == mbes {
		return &defaultValue
	}
	auth := mbes.Authentication
	if nil == auth {
		return &defaultValue
	}
	result := auth.InsecureSkipVerify
	if nil == result {
		return &defaultValue
	}
	return result
}

// GetCounterMetrics returns counter metrics if any
func GetCounterMetrics(metrics MetricsSpec) *[]CounterMetricSpec {
	defaultValue := make([]CounterMetricSpec, 0)

	if nil == metrics.CounterMetrics {
		return &defaultValue
	}

	return metrics.CounterMetrics
}

// GetRatioMetrics returns ratio metrics if any
func GetRatioMetrics(metrics MetricsSpec) *[]RatioMetricSpec {
	defaultValue := make([]RatioMetricSpec, 0)

	if nil == metrics.RatioMetrics {
		return &defaultValue
	}

	return metrics.RatioMetrics
}

// GetCounterMetricUnits returns units of a counter type metric
func GetCounterMetricUnits(metric CounterMetricSpec) string {
	defaultValue := "secs"

	value := metric.Units
	if nil == value {
		return defaultValue
	}

	return *value
}
