package spanner

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// ServerInfo represents an active Spanner Omni server node.
type ServerInfo struct {
	Name     string
	Host     string
	PortBase int
	IsRoot   bool
	State    string
}

// AdminClient defines operations to manage Spanner Omni cluster nodes.
type AdminClient interface {
	ListServers(ctx context.Context, zone, endpoint string) ([]ServerInfo, error)
	DecommissionServer(ctx context.Context, zone, serverName, endpoint, namespace string) error
	DrainAndDecommissionHighestIndex(ctx context.Context, zone, statefulSetName, namespace, endpoint string, targetReplicaCount int32) error
}

type adminClient struct {
	cliPath string
}

// NewAdminClient creates a Spanner Omni admin client.
func NewAdminClient() AdminClient {
	return &adminClient{
		cliPath: "spanner",
	}
}

// ListServers calls 'spanner deployment servers list'
// Tries local binary first; falls back to 'kubectl exec' into root pod if local binary not in container.
func (c *adminClient) ListServers(ctx context.Context, zone, endpoint string) ([]ServerInfo, error) {
	args := []string{"deployment", "servers", "list"}
	if zone != "" {
		args = append(args, fmt.Sprintf("--zone=%s", zone))
	}
	if endpoint != "" {
		args = append(args, fmt.Sprintf("--deployment-endpoint=%s", endpoint))
	}

	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return parseServerList(string(out)), nil
	}

	// Fallback via kubectl exec into spanner-a-0 (matching spanner-omni-demo-regional operations)
	k8sArgs := []string{
		"exec", "spanner-a-0", "-n", "spanner-ns", "-c", "spanner", "--",
		"/google/spanner/bin/spanner", "deployment", "servers", "list",
	}
	if zone != "" {
		k8sArgs = append(k8sArgs, fmt.Sprintf("--zone=%s", zone))
	}
	if endpoint != "" {
		k8sArgs = append(k8sArgs, fmt.Sprintf("--deployment-endpoint=%s", endpoint))
	} else {
		k8sArgs = append(k8sArgs, "--deployment-endpoint=dns:///spanner:15000")
	}

	kCmd := exec.CommandContext(ctx, "kubectl", k8sArgs...)
	kOut, kErr := kCmd.CombinedOutput()
	if kErr != nil {
		return nil, fmt.Errorf("list servers failed locally (%v) and via kubectl exec (%v: %s)", err, kErr, string(kOut))
	}

	return parseServerList(string(kOut)), nil
}

// DecommissionServer instructs Spanner Omni to drain data and remove server from cluster
func (c *adminClient) DecommissionServer(ctx context.Context, zone, serverName, endpoint, namespace string) error {
	if namespace == "" {
		namespace = "spanner-ns"
	}

	// The spanner deployment servers delete command expects server name as host:port or ID
	// If full resource name like 'zones/europe-west4-a/servers/spanner-a-3.pod.spanner-ns:15000' is passed,
	// extract the host:port portion or relative server ID.
	targetServerId := serverName
	if idx := strings.LastIndex(serverName, "/servers/"); idx != -1 {
		targetServerId = serverName[idx+len("/servers/"):]
	}

	args := []string{"deployment", "servers", "delete", targetServerId, "--quiet"}
	if zone != "" {
		args = append(args, fmt.Sprintf("--zone=%s", zone))
	}
	if endpoint != "" {
		args = append(args, fmt.Sprintf("--deployment-endpoint=%s", endpoint))
	}

	log.Printf("[Spanner Admin] Attempting direct server decommission: %s %v", c.cliPath, args)
	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		log.Printf("[Spanner Admin] Server %s decommission response: %s", targetServerId, string(out))
		return nil
	}

	// Robust in-cluster fallback via kubectl exec into root pod spanner-a-0
	log.Printf("[Spanner Admin] Direct CLI failed (%v). Falling back to kubectl exec into %s/spanner-a-0...", err, namespace)
	k8sArgs := []string{
		"exec", "spanner-a-0", "-n", namespace, "-c", "spanner", "--",
		"/google/spanner/bin/spanner", "deployment", "servers", "delete", targetServerId, "--quiet",
	}
	if zone != "" {
		k8sArgs = append(k8sArgs, fmt.Sprintf("--zone=%s", zone))
	}
	if endpoint != "" {
		k8sArgs = append(k8sArgs, fmt.Sprintf("--deployment-endpoint=%s", endpoint))
	} else {
		k8sArgs = append(k8sArgs, "--deployment-endpoint=dns:///spanner:15000")
	}

	kCmd := exec.CommandContext(ctx, "kubectl", k8sArgs...)
	kOut, kErr := kCmd.CombinedOutput()
	if kErr != nil {
		return fmt.Errorf("decommission server %s failed: %w (output: %s)", targetServerId, kErr, string(kOut))
	}

	log.Printf("[Spanner Admin] Server %s decommission via kubectl exec succeeded: %s", targetServerId, string(kOut))
	return nil
}

// DrainAndDecommissionHighestIndex coordinates safe removal of pods when scaling down from current to target
func (c *adminClient) DrainAndDecommissionHighestIndex(
	ctx context.Context,
	zone, statefulSetName, namespace, endpoint string,
	targetReplicaCount int32,
) error {
	servers, err := c.ListServers(ctx, zone, endpoint)
	if err != nil {
		log.Printf("[Spanner Admin] Warning: could not list servers via CLI or pod exec: %v. Proceeding with K8s patch.", err)
		return nil
	}

	// Example server name in regional GKE demo: zones/europe-west4-a/servers/spanner-a-4.pod.spanner-ns:15000
	for _, s := range servers {
		if s.IsRoot {
			continue // Root servers must never be decommissioned
		}
		if strings.Contains(s.Host, statefulSetName) {
			var podIndex int32
			_, scanErr := fmt.Sscanf(s.Host, statefulSetName+"-%d.", &podIndex)
			if scanErr == nil && podIndex >= targetReplicaCount {
				log.Printf("[Spanner Admin] Draining and decommissioning candidate server %s (index %d >= target %d)", s.Name, podIndex, targetReplicaCount)
				if err := c.DecommissionServer(ctx, zone, s.Name, endpoint, namespace); err != nil {
					return fmt.Errorf("failed decommissioning %s: %w", s.Name, err)
				}
				time.Sleep(3 * time.Second)
			}
		}
	}
	return nil
}

func parseServerList(output string) []ServerInfo {
	lines := strings.Split(output, "\n")
	var result []ServerInfo
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) >= 4 && fields[0] != "NAME" {
			info := ServerInfo{
				Name: fields[0],
				Host: fields[1],
			}
			if len(fields) > 3 && fields[3] == "true" {
				info.IsRoot = true
			}
			result = append(result, info)
		}
	}
	return result
}
