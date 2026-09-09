// Package v1alpha1 contains API Schema definitions for the autoscaling v1alpha1 API group
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricType defines the metric to evaluate for autoscaling.
type MetricType string

const (
	MetricTypeCPUUtilization     MetricType = "CPUUtilization"
	MetricTypeStorageUtilization MetricType = "StorageUtilization"
	MetricTypeCustomPromQL       MetricType = "CustomPromQL"
)

// MetricTarget defines the desired threshold for a metric.
type MetricTarget struct {
	// Type of metric: CPUUtilization, StorageUtilization, or CustomPromQL
	Type MetricType `json:"type"`

	// AverageUtilization percentage (e.g. 65 for 65% CPU target)
	// +optional
	AverageUtilization *int32 `json:"averageUtilization,omitempty"`

	// CustomPromQL query. If set, this query is executed against Prometheus.
	// +optional
	CustomPromQL string `json:"customPromQL,omitempty"`

	// Threshold value for CustomPromQL (float64)
	// +optional
	Threshold *float64 `json:"threshold,omitempty"`
}

// ScalingBehavior defines scale-up and scale-down rate limits and stabilization windows.
type ScalingBehavior struct {
	// ScaleUp policy controls rate of scaling out
	// +optional
	ScaleUp *ScalingPolicy `json:"scaleUp,omitempty"`

	// ScaleDown policy controls rate of scaling in
	// +optional
	ScaleDown *ScalingPolicy `json:"scaleDown,omitempty"`
}

// ScalingPolicy defines stabilization and step rules.
type ScalingPolicy struct {
	// StabilizationWindowSeconds prevents rapid oscillations (flapping)
	// +optional
	StabilizationWindowSeconds *int32 `json:"stabilizationWindowSeconds,omitempty"`

	// MaxStepReplicas is maximum number of replicas to add or remove in one evaluation step
	// +optional
	MaxStepReplicas *int32 `json:"maxStepReplicas,omitempty"`

	// CooldownSeconds is the minimum duration after a scaling action before another action can occur
	// +optional
	CooldownSeconds *int32 `json:"cooldownSeconds,omitempty"`
}

// PrometheusConfig defines Prometheus connection settings.
type PrometheusConfig struct {
	// Address of the Prometheus server (e.g., http://prometheus-service.monitoring.svc.cluster.local:9090)
	Address string `json:"address"`

	// EvaluationIntervalSeconds frequency to poll Prometheus (default: 30)
	// +optional
	EvaluationIntervalSeconds *int32 `json:"evaluationIntervalSeconds,omitempty"`

	// TimeoutSeconds for Prometheus API queries (default: 10)
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// SpannerOmniAdminConfig defines the configuration for interacting with Spanner Omni cluster topology.
type SpannerOmniAdminConfig struct {
	// DeploymentEndpoint is the Spanner management endpoint (e.g. spanner-service:15000)
	// +optional
	DeploymentEndpoint string `json:"deploymentEndpoint,omitempty"`

	// SafeScaleDown if true, invokes Spanner Omni CLI/gRPC decommission API to drain data before reducing K8s replicas
	// +optional
	SafeScaleDown bool `json:"safeScaleDown"`

	// RootServersPerZone is the number of root servers per zone that should NEVER be decommissioned
	// +optional
	RootServersPerZone int32 `json:"rootServersPerZone"`
}

// TargetRef defines the target StatefulSet or deployment
type TargetRef struct {
	// Name of the target StatefulSet (e.g. spanner-a)
	Name string `json:"name"`
	// Namespace of target (defaults to CR namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SpannerOmniAutoscalerSpec defines the desired state of SpannerOmniAutoscaler
type SpannerOmniAutoscalerSpec struct {
	// TargetRef references the Spanner Omni StatefulSet
	TargetRef TargetRef `json:"targetRef"`

	// MinReplicas minimum non-root + root replicas count (must be >= RootServersPerZone)
	MinReplicas int32 `json:"minReplicas"`

	// MaxReplicas maximum replica count for this StatefulSet
	MaxReplicas int32 `json:"maxReplicas"`

	// Metrics list of metric thresholds to watch
	Metrics []MetricTarget `json:"metrics"`

	// Behavior optional tuning for scale up / scale down dampening
	// +optional
	Behavior *ScalingBehavior `json:"behavior,omitempty"`

	// Prometheus configuration
	Prometheus PrometheusConfig `json:"prometheus"`

	// SpannerOmniAdmin configuration
	// +optional
	SpannerOmniAdmin *SpannerOmniAdminConfig `json:"spannerOmniAdmin,omitempty"`
}

// AutoscalerCondition describes the state of autoscaler
type AutoscalerCondition struct {
	Type               string      `json:"type"`
	Status             string      `json:"status"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	Reason             string      `json:"reason"`
	Message            string      `json:"message"`
}

// SpannerOmniAutoscalerStatus defines the observed state of SpannerOmniAutoscaler
type SpannerOmniAutoscalerStatus struct {
	// CurrentReplicas is current number of replicas running
	CurrentReplicas int32 `json:"currentReplicas"`

	// DesiredReplicas is the calculated target replica count
	DesiredReplicas int32 `json:"desiredReplicas"`

	// LastScaleTime timestamp of the last scale action
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// CurrentMetrics contains current evaluated metric values
	// +optional
	CurrentMetrics map[string]string `json:"currentMetrics,omitempty"`

	// Conditions of the autoscaler
	// +optional
	Conditions []AutoscalerCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target",type="string",JSONPath=".spec.targetRef.name"
// +kubebuilder:printcolumn:name="Min",type="integer",JSONPath=".spec.minReplicas"
// +kubebuilder:printcolumn:name="Max",type="integer",JSONPath=".spec.maxReplicas"
// +kubebuilder:printcolumn:name="Current",type="integer",JSONPath=".status.currentReplicas"
// +kubebuilder:printcolumn:name="Desired",type="integer",JSONPath=".status.desiredReplicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SpannerOmniAutoscaler is the Schema for the spanneromniautoscalers API
type SpannerOmniAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpannerOmniAutoscalerSpec   `json:"spec,omitempty"`
	Status SpannerOmniAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpannerOmniAutoscalerList contains a list of SpannerOmniAutoscaler
type SpannerOmniAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpannerOmniAutoscaler `json:"items"`
}
