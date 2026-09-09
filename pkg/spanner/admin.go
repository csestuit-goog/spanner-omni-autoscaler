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
	DecommissionServer(ctx context.Context, zone, serverName, endpoint string) error
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
	if err != nil {
		return nil, fmt.Errorf("list servers failed: %w (output: %s)", err, string(out))
	}

	return parseServerList(string(out)), nil
}

// DecommissionServer instructs Spanner Omni to drain data and remove server from cluster
func (c *adminClient) DecommissionServer(ctx context.Context, zone, serverName, endpoint string) error {
	args := []string{"deployment", "servers", "delete", serverName}
	if zone != "" {
		args = append(args, fmt.Sprintf("--zone=%s", zone))
	}
	if endpoint != "" {
		args = append(args, fmt.Sprintf("--deployment-endpoint=%s", endpoint))
	}

	log.Printf("[Spanner Admin] Executing server decommission: %s %v", c.cliPath, args)
	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("decommission server %s failed: %w (output: %s)", serverName, err, string(out))
	}

	log.Printf("[Spanner Admin] Server %s decommission response: %s", serverName, string(out))
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
		log.Printf("[Spanner Admin] Warning: could not list servers via CLI: %v. Proceeding with K8s patch.", err)
		return nil
	}

	// Example server name in docs: zones/us-central1-a/servers/spanner-a-1.pod.spanner-ns:15000
	for _, s := range servers {
		if s.IsRoot {
			continue // Root servers must never be decommissioned
		}
		// Match statefulset name in server host
		if strings.Contains(s.Host, statefulSetName) {
			// Extract pod index
			var podIndex int32
			_, scanErr := fmt.Sscanf(s.Host, statefulSetName+"-%d.", &podIndex)
			if scanErr == nil && podIndex >= targetReplicaCount {
				log.Printf("[Spanner Admin] Draining and decommissioning candidate server %s (index %d >= target %d)", s.Name, podIndex, targetReplicaCount)
				if err := c.DecommissionServer(ctx, zone, s.Name, endpoint); err != nil {
					return fmt.Errorf("failed decommissioning %s: %w", s.Name, err)
				}
				// Allow brief rebalance period
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
