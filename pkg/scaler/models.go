package scaler

import "time"

// MetricType defines the metric to evaluate for autoscaling.
type MetricType string

const (
	MetricTypeCPUUtilization     MetricType = "CPUUtilization"
	MetricTypeStorageUtilization MetricType = "StorageUtilization"
	MetricTypeCustomPromQL       MetricType = "CustomPromQL"
)

// MetricTarget defines desired metric target.
type MetricTarget struct {
	Type               MetricType
	AverageUtilization *int32
	CustomPromQL       string
	Threshold          *float64
}

// ScalingBehavior defines scale-up and scale-down dampening.
type ScalingBehavior struct {
	ScaleUp   *ScalingPolicy
	ScaleDown *ScalingPolicy
}

// ScalingPolicy defines stabilization and step limits.
type ScalingPolicy struct {
	StabilizationWindowSeconds *int32
	MaxStepReplicas            *int32
	CooldownSeconds            *int32
}

// SpannerOmniAdminConfig defines safety thresholds.
type SpannerOmniAdminConfig struct {
	DeploymentEndpoint string
	SafeScaleDown      bool
	RootServersPerZone int32
}

// TargetRef defines target statefulset.
type TargetRef struct {
	Name      string
	Namespace string
}

// Spec defines autoscaler parameters for evaluator.
type Spec struct {
	TargetRef        TargetRef
	MinReplicas      int32
	MaxReplicas      int32
	Metrics          []MetricTarget
	Behavior         *ScalingBehavior
	SpannerOmniAdmin *SpannerOmniAdminConfig
}

// Status defines observed autoscaler status for evaluator.
type Status struct {
	CurrentReplicas int32
	DesiredReplicas int32
	LastScaleTime   *time.Time
}
