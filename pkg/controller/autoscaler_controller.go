package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/api/v1alpha1"
	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/pkg/prometheus"
	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/pkg/scaler"
	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/pkg/spanner"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Reconciler executes autoscaling logic loop for SpannerOmniAutoscaler resources.
type Reconciler struct {
	kubeClient   kubernetes.Interface
	evaluator    *scaler.Evaluator
	spannerAdmin spanner.AdminClient
}

// NewReconciler creates a Reconciler.
func NewReconciler(kubeClient kubernetes.Interface) *Reconciler {
	return &Reconciler{
		kubeClient:   kubeClient,
		evaluator:    scaler.NewEvaluator(),
		spannerAdmin: spanner.NewAdminClient(),
	}
}

// ReconcileAutoscaler performs a single reconciliation cycle on a SpannerOmniAutoscaler instance.
func (r *Reconciler) ReconcileAutoscaler(ctx context.Context, as *v1alpha1.SpannerOmniAutoscaler) error {
	targetNamespace := as.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = as.Namespace
	}
	targetName := as.Spec.TargetRef.Name

	log.Printf("[Autoscaler %s/%s] Evaluating target StatefulSet %s/%s", as.Namespace, as.Name, targetNamespace, targetName)

	// 1. Fetch target StatefulSet
	sts, err := r.kubeClient.AppsV1().StatefulSets(targetNamespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch target StatefulSet %s/%s: %w", targetNamespace, targetName, err)
	}

	currentReplicas := int32(1)
	if sts.Spec.Replicas != nil {
		currentReplicas = *sts.Spec.Replicas
	}
	as.Status.CurrentReplicas = currentReplicas

	// 2. Query Prometheus for metrics
	promClient := prometheus.NewClient(as.Spec.Prometheus.Address, 10*time.Second)
	metricValues := make(map[string]float64)
	metricStrings := make(map[string]string)

	for _, metric := range as.Spec.Metrics {
		switch metric.Type {
		case v1alpha1.MetricTypeCPUUtilization:
			val, err := promClient.GetCPUUtilization(ctx, targetNamespace, targetName)
			if err != nil {
				log.Printf("[Autoscaler %s/%s] Warning: unable to retrieve CPU utilization from Prometheus: %v", as.Namespace, as.Name, err)
			} else {
				metricValues[string(metric.Type)] = val
				metricStrings["CPUUtilization"] = fmt.Sprintf("%.2f%%", val)
				log.Printf("[Autoscaler %s/%s] Observed CPU Utilization: %.2f%%", as.Namespace, as.Name, val)
			}

		case v1alpha1.MetricTypeStorageUtilization:
			val, err := promClient.GetStorageUtilization(ctx, targetNamespace, targetName)
			if err != nil {
				log.Printf("[Autoscaler %s/%s] Warning: unable to retrieve storage utilization from Prometheus: %v", as.Namespace, as.Name, err)
			} else {
				metricValues[string(metric.Type)] = val
				metricStrings["StorageUtilization"] = fmt.Sprintf("%.2f%%", val)
				log.Printf("[Autoscaler %s/%s] Observed Storage Utilization: %.2f%%", as.Namespace, as.Name, val)
			}

		case v1alpha1.MetricTypeCustomPromQL:
			if metric.CustomPromQL != "" {
				val, err := promClient.Query(ctx, metric.CustomPromQL)
				if err != nil {
					log.Printf("[Autoscaler %s/%s] Warning: PromQL query error (%s): %v", as.Namespace, as.Name, metric.CustomPromQL, err)
				} else {
					metricValues[metric.CustomPromQL] = val
					metricStrings["CustomPromQL"] = fmt.Sprintf("%.2f", val)
					log.Printf("[Autoscaler %s/%s] Observed Custom PromQL: %.2f", as.Namespace, as.Name, val)
				}
			}
		}
	}
	as.Status.CurrentMetrics = metricStrings

	// 3. Evaluate scaling decision
	decision := r.evaluator.Evaluate(&as.Spec, &as.Status, currentReplicas, metricValues, time.Now())
	as.Status.DesiredReplicas = decision.TargetReplicas

	log.Printf("[Autoscaler %s/%s] Evaluator Decision: Action=%s, Current=%d, Desired=%d, Reason=%s",
		as.Namespace, as.Name, decision.Action, decision.CurrentReplicas, decision.TargetReplicas, decision.Reason)

	if decision.Action == "NONE" || decision.TargetReplicas == currentReplicas {
		return nil
	}

	// 4. Safe scale down workflow (Spanner Omni topology data relocation before K8s replica decrease)
	if decision.Action == "SCALE_DOWN" && as.Spec.SpannerOmniAdmin != nil && as.Spec.SpannerOmniAdmin.SafeScaleDown {
		zone := sts.Labels["zone"]
		endpoint := as.Spec.SpannerOmniAdmin.DeploymentEndpoint
		log.Printf("[Autoscaler %s/%s] SafeScaleDown active. Draining Spanner Omni servers with index >= %d in zone %s...",
			as.Namespace, as.Name, decision.TargetReplicas, zone)

		err := r.spannerAdmin.DrainAndDecommissionHighestIndex(ctx, zone, targetName, targetNamespace, endpoint, decision.TargetReplicas)
		if err != nil {
			log.Printf("[Autoscaler %s/%s] Error during Spanner server decommission: %v", as.Namespace, as.Name, err)
			// Return error or continue based on policy; here we log to protect data
			return fmt.Errorf("spanner server decommission failed: %w", err)
		}
	}

	// 5. Patch StatefulSet replicas using in-cluster ServiceAccount permissions
	patchData := fmt.Sprintf(`{"spec":{"replicas":%d}}`, decision.TargetReplicas)
	log.Printf("[Autoscaler %s/%s] Patching StatefulSet %s/%s replicas to %d", as.Namespace, as.Name, targetNamespace, targetName, decision.TargetReplicas)

	_, err = r.kubeClient.AppsV1().StatefulSets(targetNamespace).Patch(
		ctx,
		targetName,
		types.StrategicMergePatchType,
		[]byte(patchData),
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch StatefulSet replicas: %w", err)
	}

	now := metav1.Now()
	as.Status.LastScaleTime = &now
	as.Status.CurrentReplicas = decision.TargetReplicas

	// 6. Optional PVC cleanup guidance for scale down
	if decision.Action == "SCALE_DOWN" {
		log.Printf("[Autoscaler %s/%s] Successfully scaled down to %d. Note: Associated PVCs are retained by StatefulSet policy for data safety.",
			as.Namespace, as.Name, decision.TargetReplicas)
	}

	return nil
}
