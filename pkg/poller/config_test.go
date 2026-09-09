package poller

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigMapYAML(t *testing.T) {
	yamlContent := `---
- namespace: spanner-ns
  statefulSetName: spanner-a
  units: NODES
  minSize: 3
  maxSize: 15
  scalingMethod: LINEAR
  rootServersPerZone: 3
  deploymentEndpoint: "spanner-service:15000"
  safeScaleDown: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "autoscaler-config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	configs, err := ParseConfigMapYAML(configPath)
	if err != nil {
		t.Fatalf("failed loading config: %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	cfg := configs[0]
	if cfg.StatefulSetName != "spanner-a" {
		t.Fatalf("expected spanner-a, got %s", cfg.StatefulSetName)
	}
	if cfg.MinSize != 3 || cfg.MaxSize != 15 {
		t.Fatalf("expected minSize 3, maxSize 15, got %d and %d", cfg.MinSize, cfg.MaxSize)
	}
	if cfg.RootServersPerZone != 3 {
		t.Fatalf("expected root servers 3, got %d", cfg.RootServersPerZone)
	}
	if !cfg.SafeScaleDown {
		t.Fatalf("expected SafeScaleDown true")
	}
}
