package poller

// ScalingMethod defines the scaling algorithm (e.g. LINEAR, DIRECT, STEP).
type ScalingMethod string

const (
	ScalingMethodLinear ScalingMethod = "LINEAR"
	ScalingMethodDirect ScalingMethod = "DIRECT"
	ScalingMethodStep   ScalingMethod = "STEP"
)

// Units defines the scale unit: NODES (StatefulSet pod replicas)
type Units string

const (
	UnitsNodes Units = "NODES"
)

// MetricConfig defines threshold settings for each metric.
type MetricConfig struct {
	Name              string `json:"name"`
	TargetThreshold   int32  `json:"target_threshold,omitempty"`
	RegionalThreshold int32  `json:"regional_threshold,omitempty"`
	RegionalMargin    int32  `json:"regional_margin,omitempty"`
	CustomPromQL      string `json:"custom_promql,omitempty"`
}

// SpannerOmniConfig defines the target Spanner Omni deployment in the ConfigMap.
type SpannerOmniConfig struct {
	Namespace          string        `json:"namespace"`
	StatefulSetName    string        `json:"statefulSetName"`
	Location           string        `json:"location,omitempty"`
	Zone               string        `json:"zone,omitempty"`
	Units              Units         `json:"units"`
	MinSize            int32         `json:"minSize"`
	MaxSize            int32         `json:"maxSize"`
	ScalingMethod      ScalingMethod `json:"scalingMethod"`
	RootServersPerZone int32         `json:"rootServersPerZone"`
	DeploymentEndpoint string        `json:"deploymentEndpoint,omitempty"`
	SafeScaleDown      bool          `json:"safeScaleDown"`
	PrometheusAddress  string        `json:"prometheusAddress"`
	Metrics            []MetricConfig`json:"metrics"`
}

func applyDefaults(cfg *SpannerOmniConfig) {
	if cfg.Units == "" {
		cfg.Units = UnitsNodes
	}
	if cfg.ScalingMethod == "" {
		cfg.ScalingMethod = ScalingMethodLinear
	}
	if cfg.RootServersPerZone <= 0 {
		cfg.RootServersPerZone = 3 // Standard Spanner Omni default
	}
	if cfg.MinSize < cfg.RootServersPerZone {
		cfg.MinSize = cfg.RootServersPerZone
	}
	if cfg.PrometheusAddress == "" {
		cfg.PrometheusAddress = "http://prometheus-service.monitoring.svc.cluster.local:9090"
	}
	if len(cfg.Metrics) == 0 {
		cfg.Metrics = []MetricConfig{
			{
				Name:              "high_priority_cpu",
				RegionalThreshold: 65,
				RegionalMargin:    5,
			},
		}
	}
}
