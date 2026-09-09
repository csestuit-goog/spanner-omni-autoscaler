package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/api/v1alpha1"
	"github.com/GoogleCloudPlatform/spanner-omni-autoscaler/pkg/controller"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	gvr = schema.GroupVersionResource{
		Group:    "autoscaling.spanner.google.com",
		Version:  "v1alpha1",
		Resource: "spanneromniautoscalers",
	}
)

func main() {
	var kubeconfig string
	var syncPeriod time.Duration
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if empty)")
	flag.DurationVar(&syncPeriod, "sync-period", 30*time.Second, "Evaluation and reconciliation sync interval")
	flag.Parse()

	log.Println("Starting Spanner Omni Autoscaler Controller...")

	var config *rest.Config
	var err error
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Fatalf("Failed to build Kubernetes client config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	reconciler := controller.NewReconciler(clientset)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()

	log.Printf("Controller running with sync period %v. Watching SpannerOmniAutoscaler resources...", syncPeriod)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down controller gracefully...")
			return
		case <-ticker.C:
			runReconciliationLoop(ctx, dynamicClient, reconciler)
		}
	}
}

func runReconciliationLoop(ctx context.Context, dynamicClient dynamic.Interface, reconciler *controller.Reconciler) {
	list, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing SpannerOmniAutoscalers: %v", err)
		return
	}

	for _, item := range list.Items {
		// Convert unstructured to typed instance
		as := parseUnstructuredAutoscaler(&item)
		if err := reconciler.ReconcileAutoscaler(ctx, as); err != nil {
			log.Printf("Reconciliation error for %s/%s: %v", as.Namespace, as.Name, err)
		}
	}
}

func parseUnstructuredAutoscaler(u *unstructured.Unstructured) *v1alpha1.SpannerOmniAutoscaler {
	as := &v1alpha1.SpannerOmniAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      u.GetName(),
			Namespace: u.GetNamespace(),
		},
	}

	specMap, ok := u.Object["spec"].(map[string]interface{})
	if !ok {
		return as
	}

	if target, ok := specMap["targetRef"].(map[string]interface{}); ok {
		if name, ok := target["name"].(string); ok {
			as.Spec.TargetRef.Name = name
		}
		if ns, ok := target["namespace"].(string); ok {
			as.Spec.TargetRef.Namespace = ns
		}
	}

	if minR, ok := specMap["minReplicas"].(int64); ok {
		as.Spec.MinReplicas = int32(minR)
	}
	if maxR, ok := specMap["maxReplicas"].(int64); ok {
		as.Spec.MaxReplicas = int32(maxR)
	}

	if promMap, ok := specMap["prometheus"].(map[string]interface{}); ok {
		if addr, ok := promMap["address"].(string); ok {
			as.Spec.Prometheus.Address = addr
		}
	}

	if metricsList, ok := specMap["metrics"].([]interface{}); ok {
		for _, m := range metricsList {
			if mm, ok := m.(map[string]interface{}); ok {
				target := v1alpha1.MetricTarget{}
				if t, ok := mm["type"].(string); ok {
					target.Type = v1alpha1.MetricType(t)
				}
				if avg, ok := mm["averageUtilization"].(int64); ok {
					v := int32(avg)
					target.AverageUtilization = &v
				}
				if q, ok := mm["customPromQL"].(string); ok {
					target.CustomPromQL = q
				}
				if th, ok := mm["threshold"].(float64); ok {
					target.Threshold = &th
				}
				as.Spec.Metrics = append(as.Spec.Metrics, target)
			}
		}
	}

	if adminMap, ok := specMap["spannerOmniAdmin"].(map[string]interface{}); ok {
		adminConfig := &v1alpha1.SpannerOmniAdminConfig{}
		if ep, ok := adminMap["deploymentEndpoint"].(string); ok {
			adminConfig.DeploymentEndpoint = ep
		}
		if safe, ok := adminMap["safeScaleDown"].(bool); ok {
			adminConfig.SafeScaleDown = safe
		}
		if rootServers, ok := adminMap["rootServersPerZone"].(int64); ok {
			adminConfig.RootServersPerZone = int32(rootServers)
		}
		as.Spec.SpannerOmniAdmin = adminConfig
	}

	return as
}
