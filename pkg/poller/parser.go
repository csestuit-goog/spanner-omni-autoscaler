package poller

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseConfigMapYAML is a standalone zero-external-dependency parser for standard GKE ConfigMap autoscaler files.
func ParseConfigMapYAML(path string) ([]SpannerOmniConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var configs []SpannerOmniConfig
	var current *SpannerOmniConfig

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if current != nil {
				applyDefaults(current)
				configs = append(configs, *current)
			}
			current = &SpannerOmniConfig{}
			line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		}

		if current == nil {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"")

			switch key {
			case "namespace":
				current.Namespace = val
			case "statefulSetName", "instanceId":
				current.StatefulSetName = val
			case "units":
				current.Units = Units(val)
			case "minSize":
				v, _ := strconv.Atoi(val)
				current.MinSize = int32(v)
			case "maxSize":
				v, _ := strconv.Atoi(val)
				current.MaxSize = int32(v)
			case "scalingMethod":
				current.ScalingMethod = ScalingMethod(val)
			case "rootServersPerZone":
				v, _ := strconv.Atoi(val)
				current.RootServersPerZone = int32(v)
			case "deploymentEndpoint":
				current.DeploymentEndpoint = val
			case "safeScaleDown":
				current.SafeScaleDown = (val == "true")
			case "prometheusAddress":
				current.PrometheusAddress = val
			}
		}
	}

	if current != nil {
		applyDefaults(current)
		configs = append(configs, *current)
	}

	return configs, nil
}
