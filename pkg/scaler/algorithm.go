package scaler

import (
	"log"
	"math"
	"time"
)

// Decision contains the scaling recommendation and reason.
type Decision struct {
	CurrentReplicas int32
	TargetReplicas  int32
	Action          string // "SCALE_UP", "SCALE_DOWN", "NONE", "BLOCKED_BY_TRUETIME"
	Reason          string
	MetricValues    map[string]string
}

// AlertInputs reflects active alerts evaluated directly from official rules:
// https://cloud.google.com/spanner-omni/prometheus-alerts
type AlertInputs struct {
	TrueTimeUnavailable        bool
	ClockSlaViolation          bool
	SpannerHighCPUUtilization  bool
	StorageUtilizationWarning bool
	StorageUtilizationCritical bool
	StoragePerVCPUTooHigh     bool
}

// Evaluator decides whether to scale in or out based on current metrics and rules.
type Evaluator struct{}

// NewEvaluator creates a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate calculates target replicas given current replicas, metric targets, alerts, and behavior limits.
func (e *Evaluator) Evaluate(
	spec *Spec,
	status *Status,
	currentReplicas int32,
	metricValues map[string]float64,
	alerts *AlertInputs,
	now time.Time,
) Decision {
	// =========================================================================
	// 1. TrueTime Alert Integration (Critical Safety Guard)
	// If TrueTime is unavailable or Clock SLA is violated, scaling actions must
	// be blocked or frozen to protect distributed commit transaction consistency.
	// =========================================================================
	if alerts != nil {
		if alerts.TrueTimeUnavailable {
			log.Printf("[Evaluator] WARNING: TrueTimeUnavailable alert is ACTIVE (true_time_is_available < 1). Halting scaling operations for safety.")
			return Decision{
				CurrentReplicas: currentReplicas,
				TargetReplicas:  currentReplicas,
				Action:          "BLOCKED_BY_TRUETIME",
				Reason:          "TrueTime is unavailable; scaling frozen to protect database consistency",
			}
		}
		if alerts.ClockSlaViolation {
			log.Printf("[Evaluator] WARNING: ClockSlaViolation alert is ACTIVE (sla_tester_violation_count > 0). Halting scaling operations.")
			return Decision{
				CurrentReplicas: currentReplicas,
				TargetReplicas:  currentReplicas,
				Action:          "BLOCKED_BY_TRUETIME",
				Reason:          "Clock SLA violation detected; scaling frozen until clocks synchronize",
			}
		}
	}

	var maxTargetReplicas int32 = currentReplicas
	actionReason := "All metrics within desired target bounds"
	metricStrings := make(map[string]string)

	// =========================================================================
	// 2. Metrics & Alerts Evaluation
	// =========================================================================
	for _, target := range spec.Metrics {
		switch target.Type {
		case MetricTypeCPUUtilization:
			if target.AverageUtilization == nil || *target.AverageUtilization <= 0 {
				continue
			}
			curVal, exists := metricValues[string(target.Type)]
			if !exists {
				continue
			}
			metricStrings["CPUUtilization"] = string(target.Type)

			// Standard autoscaling formula:
			// desiredReplicas = ceil(currentReplicas * (currentMetricValue / targetMetricValue))
			ratio := curVal / float64(*target.AverageUtilization)
			// Apply a 10% tolerance band to prevent thrashing
			if ratio > 1.10 || ratio < 0.90 {
				desired := int32(math.Ceil(float64(currentReplicas) * ratio))
				if desired > maxTargetReplicas {
					maxTargetReplicas = desired
					actionReason = "CPU utilization above target (" + formatFloat(curVal) + "% > " + formatInt(*target.AverageUtilization) + "%) [SpannerHighCPUUtilization]"
				} else if desired < maxTargetReplicas && ratio < 0.90 {
					maxTargetReplicas = desired
					actionReason = "CPU utilization below target (" + formatFloat(curVal) + "% < " + formatInt(*target.AverageUtilization) + "%)"
				}
			}

		case MetricTypeStorageUtilization:
			if target.AverageUtilization == nil || *target.AverageUtilization <= 0 {
				continue
			}
			curVal, exists := metricValues[string(target.Type)]
			if !exists {
				continue
			}
			ratio := curVal / float64(*target.AverageUtilization)
			if ratio > 1.10 {
				desired := int32(math.Ceil(float64(currentReplicas) * ratio))
				if desired > maxTargetReplicas {
					maxTargetReplicas = desired
					actionReason = "Storage utilization above target (" + formatFloat(curVal) + "% > " + formatInt(*target.AverageUtilization) + "%) [SpannerStorageUtilization]"
				}
			}

		case MetricTypeCustomPromQL:
			if target.Threshold == nil || *target.Threshold <= 0 {
				continue
			}
			curVal, exists := metricValues[target.CustomPromQL]
			if !exists {
				continue
			}
			ratio := curVal / *target.Threshold
			if ratio > 1.10 {
				desired := int32(math.Ceil(float64(currentReplicas) * ratio))
				if desired > maxTargetReplicas {
					maxTargetReplicas = desired
					actionReason = "Custom PromQL metric exceeded threshold"
				}
			}
		}
	}

	// Immediate step-up if Critical Storage alert is firing
	if alerts != nil && alerts.StorageUtilizationCritical {
		if maxTargetReplicas <= currentReplicas {
			maxTargetReplicas = currentReplicas + 1
			actionReason = "SpannerStorageUtilizationCritical alert active (> 90%); preemptive scale-out initiated"
		}
	}

	// Clamp to MinReplicas and MaxReplicas
	if maxTargetReplicas < spec.MinReplicas {
		maxTargetReplicas = spec.MinReplicas
	}
	if maxTargetReplicas > spec.MaxReplicas {
		maxTargetReplicas = spec.MaxReplicas
	}

	// Ensure RootServersPerZone is protected:
	if spec.SpannerOmniAdmin != nil && spec.SpannerOmniAdmin.RootServersPerZone > 0 {
		if maxTargetReplicas < spec.SpannerOmniAdmin.RootServersPerZone {
			maxTargetReplicas = spec.SpannerOmniAdmin.RootServersPerZone
			actionReason = "Clamped to protected rootServersPerZone limit"
		}
	}

	// Apply Scaling Behavior (step limits and cooldowns)
	if maxTargetReplicas > currentReplicas {
		// Scale UP
		if spec.Behavior != nil && spec.Behavior.ScaleUp != nil {
			policy := spec.Behavior.ScaleUp
			if policy.CooldownSeconds != nil && status.LastScaleTime != nil {
				cooldown := time.Duration(*policy.CooldownSeconds) * time.Second
				if now.Sub(*status.LastScaleTime) < cooldown {
					return Decision{
						CurrentReplicas: currentReplicas,
						TargetReplicas:  currentReplicas,
						Action:          "NONE",
						Reason:          "Scale-up in cooldown window",
					}
				}
			}
			if policy.MaxStepReplicas != nil && *policy.MaxStepReplicas > 0 {
				step := *policy.MaxStepReplicas
				if (maxTargetReplicas - currentReplicas) > step {
					maxTargetReplicas = currentReplicas + step
					actionReason += " (capped by maxStepReplicas)"
				}
			}
		}
		return Decision{
			CurrentReplicas: currentReplicas,
			TargetReplicas:  maxTargetReplicas,
			Action:          "SCALE_UP",
			Reason:          actionReason,
		}
	} else if maxTargetReplicas < currentReplicas {
		// Scale DOWN
		if spec.Behavior != nil && spec.Behavior.ScaleDown != nil {
			policy := spec.Behavior.ScaleDown
			if policy.CooldownSeconds != nil && status.LastScaleTime != nil {
				cooldown := time.Duration(*policy.CooldownSeconds) * time.Second
				if now.Sub(*status.LastScaleTime) < cooldown {
					return Decision{
						CurrentReplicas: currentReplicas,
						TargetReplicas:  currentReplicas,
						Action:          "NONE",
						Reason:          "Scale-down in cooldown window",
					}
				}
			}
			if policy.MaxStepReplicas != nil && *policy.MaxStepReplicas > 0 {
				step := *policy.MaxStepReplicas
				if (currentReplicas - maxTargetReplicas) > step {
					maxTargetReplicas = currentReplicas - step
					actionReason += " (capped by maxStepReplicas)"
				}
			}
		}
		return Decision{
			CurrentReplicas: currentReplicas,
			TargetReplicas:  maxTargetReplicas,
			Action:          "SCALE_DOWN",
			Reason:          actionReason,
		}
	}

	return Decision{
		CurrentReplicas: currentReplicas,
		TargetReplicas:  currentReplicas,
		Action:          "NONE",
		Reason:          "Replicas match target requirement",
	}
}

func formatFloat(v float64) string {
	return floatToString(v)
}

func floatToString(v float64) string {
	n := math.Round(v*10) / 10
	intPart := int64(n)
	fracPart := int64(math.Abs((n - float64(intPart)) * 10))
	return formatInt(int32(intPart)) + "." + formatInt(int32(fracPart))
}

func formatInt(v int32) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	var res []byte
	for v > 0 {
		res = append([]byte{byte('0' + (v % 10))}, res...)
		v /= 10
	}
	if neg {
		res = append([]byte{'-'}, res...)
	}
	return string(res)
}
