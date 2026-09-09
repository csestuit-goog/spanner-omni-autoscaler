package poller

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseConfigMapYAML parses the autoscaler ConfigMap YAML file.
func ParseConfigMapYAML(path string) ([]SpannerOmniConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var configs []SpannerOmniConfig
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("unmarshal YAML: %w", err)
	}

	for i := range configs {
		applyDefaults(&configs[i])
	}

	return configs, nil
}
