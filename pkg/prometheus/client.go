package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MetricResult holds an evaluated numeric value from Prometheus or OTel Collector.
type MetricResult struct {
	Value     float64
	Timestamp time.Time
}

// SpannerOmniAlertStatus reflects alert states defined in:
// https://cloud.google.com/spanner-omni/prometheus-alerts
type SpannerOmniAlertStatus struct {
	// TrueTime Alerts
	TrueTimeUnavailable bool // expr: true_time_is_available < 1
	ClockSlaViolation   bool // expr: sla_tester_violation_count > 0

	// CPU Alert
	HighCPUUtilization bool // expr: (sum(spanner_cpu_utilization_by_priority_and_category)*100)/sum(spanner_available_milligcu) > 65

	// Storage Alerts
	StorageUtilizationWarning  bool // expr: (used / total) > 0.80
	StorageUtilizationCritical bool // expr: (used / total) > 0.90
	StoragePerVCPUTooHigh      bool // expr: (used / cpu_total) > 512000 (500 GiB/vCPU)
}

// Client interacts with open-standard metrics backends (Prometheus HTTP API / OpenTelemetry Collector PromQL endpoint).
type Client interface {
	Query(ctx context.Context, promQL string) (float64, error)
	GetCPUUtilization(ctx context.Context, namespace, statefulSetName string) (float64, error)
	GetStorageUtilization(ctx context.Context, namespace, statefulSetName string) (float64, error)
	GetSpannerAlerts(ctx context.Context, namespace, statefulSetName string) (*SpannerOmniAlertStatus, error)
}

type openStandardMetricsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Open-Standard Prometheus / OTel Collector HTTP Client.
func NewClient(address string, timeout time.Duration) Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &openStandardMetricsClient{
		baseURL: address,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type queryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"` // [timestamp, "value_str"]
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Query executes a standard PromQL query against Prometheus or OpenTelemetry Collector Prometheus Receiver.
func (c *openStandardMetricsClient) Query(ctx context.Context, promQL string) (float64, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/query", c.baseURL))
	if err != nil {
		return 0, fmt.Errorf("invalid metrics endpoint url: %w", err)
	}

	q := u.Query()
	q.Set("query", promQL)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query metrics endpoint (%s): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("metrics endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var qr queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return 0, fmt.Errorf("decode metrics response: %w", err)
	}

	if qr.Status != "success" {
		return 0, fmt.Errorf("metrics engine error: %s: %s", qr.ErrorType, qr.Error)
	}

	if len(qr.Data.Result) == 0 {
		return 0, fmt.Errorf("no metric data returned for query: %s", promQL)
	}

	valTuple := qr.Data.Result[0].Value
	if len(valTuple) < 2 {
		return 0, fmt.Errorf("invalid value format in metrics result")
	}

	valStr, ok := valTuple[1].(string)
	if !ok {
		return 0, fmt.Errorf("expected string value in result: %v", valTuple[1])
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float value %q: %w", valStr, err)
	}

	return val, nil
}

// GetCPUUtilization computes CPU utilization % based on official Spanner Omni alert expression:
// (sum(spanner_cpu_utilization_by_priority_and_category) * 100) / sum(spanner_available_milligcu)
func (c *openStandardMetricsClient) GetCPUUtilization(ctx context.Context, namespace, statefulSetName string) (float64, error) {
	query := fmt.Sprintf(`(sum(spanner_cpu_utilization_by_priority_and_category{namespace="%s",spanner_server=~"%s-.*"}) * 100) / (sum(spanner_available_milligcu{namespace="%s",spanner_server=~"%s-.*"}) > 0)`,
		namespace, statefulSetName, namespace, statefulSetName)

	val, err := c.Query(ctx, query)
	if err == nil {
		return val, nil
	}

	log.Printf("[Metrics] Native spanner_cpu_utilization query failed (%v), trying cAdvisor fallback...", err)

	fallbackQuery := fmt.Sprintf(`100 * (sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s-[0-9]+",container="spanner"}[2m])) / sum(kube_pod_container_resource_requests{namespace="%s",pod=~"%s-[0-9]+",resource="cpu"}))`,
		namespace, statefulSetName, namespace, statefulSetName)

	val, fallbackErr := c.Query(ctx, fallbackQuery)
	if fallbackErr == nil {
		return val, nil
	}

	return 0, fmt.Errorf("unable to evaluate CPU: %v / %v", err, fallbackErr)
}

// GetStorageUtilization computes storage utilization % based on official Spanner Omni alert expression:
// sum by (spanner_server)(filesystem_size{type="used"}) / sum by (spanner_server)(filesystem_size{type="total"})
func (c *openStandardMetricsClient) GetStorageUtilization(ctx context.Context, namespace, statefulSetName string) (float64, error) {
	query := fmt.Sprintf(`(sum(filesystem_size{type="used",namespace="%s",spanner_server=~"%s-.*"}) / sum(filesystem_size{type="total",namespace="%s",spanner_server=~"%s-.*"})) * 100`,
		namespace, statefulSetName, namespace, statefulSetName)

	val, err := c.Query(ctx, query)
	if err == nil {
		return val, nil
	}

	log.Printf("[Metrics] Native filesystem_size query failed (%v), trying kubelet fallback...", err)

	fallbackQuery := fmt.Sprintf(`100 * (sum(kubelet_volume_stats_used_bytes{namespace="%s",persistentvolumeclaim=~"data-volume-%s-.*"}) / sum(kubelet_volume_stats_capacity_bytes{namespace="%s",persistentvolumeclaim=~"data-volume-%s-.*"}))`,
		namespace, statefulSetName, namespace, statefulSetName)

	val, fallbackErr := c.Query(ctx, fallbackQuery)
	if fallbackErr == nil {
		return val, nil
	}

	return 0, fmt.Errorf("unable to evaluate Storage: %v / %v", err, fallbackErr)
}

// GetSpannerAlerts queries all official Prometheus alerts from:
// https://cloud.google.com/spanner-omni/prometheus-alerts
func (c *openStandardMetricsClient) GetSpannerAlerts(ctx context.Context, namespace, statefulSetName string) (*SpannerOmniAlertStatus, error) {
	alerts := &SpannerOmniAlertStatus{}

	// 1. TrueTime Alerts: TrueTimeUnavailable (true_time_is_available < 1)
	ttAvailQuery := fmt.Sprintf(`min(true_time_is_available{namespace="%s",spanner_server=~"%s-.*"})`, namespace, statefulSetName)
	if val, err := c.Query(ctx, ttAvailQuery); err == nil && val < 1.0 {
		alerts.TrueTimeUnavailable = true
	}

	// 2. TrueTime Alerts: ClockSlaViolation (sla_tester_violation_count > 0)
	clockSlaQuery := fmt.Sprintf(`max(sla_tester_violation_count{namespace="%s",spanner_server=~"%s-.*"})`, namespace, statefulSetName)
	if val, err := c.Query(ctx, clockSlaQuery); err == nil && val > 0 {
		alerts.ClockSlaViolation = true
	}

	// 3. CPU Alert: SpannerHighCPUUtilization (> 65%)
	if cpu, err := c.GetCPUUtilization(ctx, namespace, statefulSetName); err == nil && cpu > 65.0 {
		alerts.HighCPUUtilization = true
	}

	// 4. Storage Alerts: SpannerStorageUtilizationWarning (> 80%), Critical (> 90%)
	if storage, err := c.GetStorageUtilization(ctx, namespace, statefulSetName); err == nil {
		if storage > 80.0 {
			alerts.StorageUtilizationWarning = true
		}
		if storage > 90.0 {
			alerts.StorageUtilizationCritical = true
		}
	}

	// 5. Storage Per vCPU Alert: SpannerStoragePerVCPUTooHigh (> 512000 KiB = 500 GiB / vCPU)
	storagePerVCPUQuery := fmt.Sprintf(`(sum(filesystem_size{type="used",namespace="%s",spanner_server=~"%s-.*"}) / sum(spanner_box_vm_cpu_total{namespace="%s",spanner_server=~"%s-.*"}))`,
		namespace, statefulSetName, namespace, statefulSetName)
	if val, err := c.Query(ctx, storagePerVCPUQuery); err == nil && val > 512000.0 {
		alerts.StoragePerVCPUTooHigh = true
	}

	return alerts, nil
}
