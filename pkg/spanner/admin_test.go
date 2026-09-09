package spanner

import (
	"testing"
)

func TestParseServerList(t *testing.T) {
	output := `NAME                                                          HOST                        PORT_BASE  ROOT  STATE
zones/us-central1-a/servers/spanner-a-0.pod.spanner-ns:15000  spanner-a-0.pod.spanner-ns  15000      true  -
zones/us-central1-a/servers/spanner-a-1.pod.spanner-ns:15000  spanner-a-1.pod.spanner-ns  15000      -     -
zones/us-central1-a/servers/spanner-a-2.pod.spanner-ns:15000  spanner-a-2.pod.spanner-ns  15000      -     -`

	servers := parseServerList(output)
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	if !servers[0].IsRoot {
		t.Fatalf("expected server 0 to be root server")
	}

	if servers[1].IsRoot {
		t.Fatalf("expected server 1 to be non-root server")
	}

	if servers[1].Host != "spanner-a-1.pod.spanner-ns" {
		t.Fatalf("expected host spanner-a-1.pod.spanner-ns, got %s", servers[1].Host)
	}
}
