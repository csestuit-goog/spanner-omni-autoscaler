package scaler

import (
	"testing"
	"time"
)

func TestEvaluatorScaleUp(t *testing.T) {
	evaluator := NewEvaluator()

	targetCPU := int32(65)
	maxStep := int32(2)
	spec := &Spec{
		MinReplicas: 3,
		MaxReplicas: 10,
		Metrics: []MetricTarget{
			{
				Type:               MetricTypeCPUUtilization,
				AverageUtilization: &targetCPU,
			},
		},
		Behavior: &ScalingBehavior{
			ScaleUp: &ScalingPolicy{
				MaxStepReplicas: &maxStep,
			},
		},
	}

	status := &Status{
		CurrentReplicas: 3,
	}

	metricValues := map[string]float64{
		string(MetricTypeCPUUtilization): 90.0,
	}

	decision := evaluator.Evaluate(spec, status, 3, metricValues, nil, time.Now())
	if decision.Action != "SCALE_UP" {
		t.Fatalf("expected SCALE_UP, got %s", decision.Action)
	}
	if decision.TargetReplicas != 5 {
		t.Fatalf("expected 5 replicas, got %d", decision.TargetReplicas)
	}
}

func TestEvaluatorBlockedByTrueTimeAlert(t *testing.T) {
	evaluator := NewEvaluator()

	targetCPU := int32(65)
	spec := &Spec{
		MinReplicas: 3,
		MaxReplicas: 10,
		Metrics: []MetricTarget{
			{
				Type:               MetricTypeCPUUtilization,
				AverageUtilization: &targetCPU,
			},
		},
	}

	status := &Status{
		CurrentReplicas: 3,
	}

	// High CPU demands scale up, but TrueTime is unavailable
	metricValues := map[string]float64{
		string(MetricTypeCPUUtilization): 95.0,
	}
	alerts := &AlertInputs{
		TrueTimeUnavailable: true,
	}

	decision := evaluator.Evaluate(spec, status, 3, metricValues, alerts, time.Now())
	if decision.Action != "BLOCKED_BY_TRUETIME" {
		t.Fatalf("expected BLOCKED_BY_TRUETIME, got %s", decision.Action)
	}
	if decision.TargetReplicas != 3 {
		t.Fatalf("expected replica count to remain 3, got %d", decision.TargetReplicas)
	}
}

func TestEvaluatorScaleDownRespectsRootServers(t *testing.T) {
	evaluator := NewEvaluator()

	targetCPU := int32(65)
	spec := &Spec{
		MinReplicas: 1,
		MaxReplicas: 10,
		Metrics: []MetricTarget{
			{
				Type:               MetricTypeCPUUtilization,
				AverageUtilization: &targetCPU,
			},
		},
		SpannerOmniAdmin: &SpannerOmniAdminConfig{
			RootServersPerZone: 3,
		},
	}

	status := &Status{
		CurrentReplicas: 5,
	}

	metricValues := map[string]float64{
		string(MetricTypeCPUUtilization): 10.0,
	}

	decision := evaluator.Evaluate(spec, status, 5, metricValues, nil, time.Now())
	if decision.Action != "SCALE_DOWN" {
		t.Fatalf("expected SCALE_DOWN, got %s", decision.Action)
	}
	if decision.TargetReplicas != 3 {
		t.Fatalf("expected 3 replicas (protected rootServersPerZone), got %d", decision.TargetReplicas)
	}
}
