#!/usr/bin/env bash
# ==============================================================================
# Spanner Omni Autoscaler - Cluster Reconfiguration & Auto-Discovery Utility
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

OUTPUT_FILE="${1:-spanner-omni-cluster.values.yaml}"

echo -e "${BLUE}${BOLD}======================================================================${NC}"
echo -e "${BLUE}${BOLD}   Spanner Omni Autoscaler - Cluster Configuration Generator          ${NC}"
echo -e "${BLUE}${BOLD}======================================================================${NC}"

# Check kubectl
if ! command -v kubectl &>/dev/null; then
  echo -e "${RED}[ERROR] 'kubectl' command is required but not found in PATH.${NC}"
  exit 1
fi

echo -e "\n${BOLD}[1/4] Inspecting active Kubernetes context...${NC}"
CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "unknown")
echo -e "Current Kubernetes Context: ${GREEN}${CURRENT_CONTEXT}${NC}"

echo -e "\n${BOLD}[2/4] Auto-discovering Spanner Omni StatefulSets & Prometheus...${NC}"

python3 - << 'PY_DISCOVER'
import subprocess
import json
import sys

def run_cmd(cmd):
    try:
        return subprocess.check_output(cmd, stderr=subprocess.DEVNULL).decode()
    except Exception:
        return "{}"

# 1. Discover StatefulSets
sts_raw = run_cmd(['kubectl', 'get', 'statefulsets', '-A', '-o', 'json'])
try:
    sts_data = json.loads(sts_raw)
except Exception:
    sts_data = {'items': []}

spanner_targets = []
spanner_ns = "spanner-ns"

for item in sts_data.get('items', []):
    name = item['metadata']['name']
    ns = item['metadata']['namespace']
    labels = item['metadata'].get('labels', {})
    app = labels.get('app', '') or labels.get('app.kubernetes.io/name', '')
    
    if 'spanner' in name or 'spanner' in app or 'spanner' in ns:
        zone = labels.get('zone', '')
        if not zone:
            zone = item['spec']['template']['metadata'].get('labels', {}).get('zone', '')
        if not zone:
            zone = item['spec']['template']['spec'].get('nodeSelector', {}).get('topology.kubernetes.io/zone', '')
        
        replicas = item['spec'].get('replicas', 3)
        spanner_targets.append({
            'name': name,
            'namespace': ns,
            'zone': zone,
            'replicas': replicas
        })
        spanner_ns = ns

# 2. Discover Services
svc_raw = run_cmd(['kubectl', 'get', 'svc', '-A', '-o', 'json'])
try:
    svc_data = json.loads(svc_raw)
except Exception:
    svc_data = {'items': []}

prom_endpoint = "http://prometheus-service.monitoring.svc.cluster.local:9090"
spanner_endpoint = f"spanner.{spanner_ns}.svc.cluster.local:15000"

for item in svc_data.get('items', []):
    name = item['metadata']['name']
    ns = item['metadata']['namespace']
    
    if 'prometheus' in name and not 'alertmanager' in name:
        ports = item['spec'].get('ports', [])
        port = ports[0]['port'] if ports else 9090
        prom_endpoint = f"http://{name}.{ns}.svc.cluster.local:{port}"
        
    if 'spanner' in name and not 'console' in name:
        for p in item['spec'].get('ports', []):
            if p.get('port') == 15000:
                spanner_endpoint = f"{name}.{ns}.svc.cluster.local:15000"

output = {
    'namespace': spanner_ns,
    'endpoint': spanner_endpoint,
    'prometheus': prom_endpoint,
    'targets': spanner_targets
}

with open('/tmp/spanner_discovered.json', 'w') as f:
    json.dump(output, f)
PY_DISCOVER

DISCOVERED_FILE="/tmp/spanner_discovered.json"
DISCOVERED_NS=$(python3 -c "import json; data=json.load(open('${DISCOVERED_FILE}')); print(data['namespace'])")
DISCOVERED_ENDPOINT=$(python3 -c "import json; data=json.load(open('${DISCOVERED_FILE}')); print(data['endpoint'])")
DISCOVERED_PROM=$(python3 -c "import json; data=json.load(open('${DISCOVERED_FILE}')); print(data['prometheus'])")
TARGET_COUNT=$(python3 -c "import json; data=json.load(open('${DISCOVERED_FILE}')); print(len(data['targets']))")

echo -e "Discovered Spanner Namespace:    ${GREEN}${DISCOVERED_NS}${NC}"
echo -e "Discovered Deployment Endpoint:  ${GREEN}${DISCOVERED_ENDPOINT}${NC}"
echo -e "Discovered Prometheus Address:   ${GREEN}${DISCOVERED_PROM}${NC}"
echo -e "Discovered StatefulSet Targets:  ${GREEN}${TARGET_COUNT} target(s)${NC}"

echo -e "\n${BOLD}[3/4] Configure Cluster Settings (press ENTER to accept discovered default):${NC}"
read -rp "Target Spanner Namespace [${DISCOVERED_NS}]: " INPUT_NS || INPUT_NS=""
TARGET_NS="${INPUT_NS:-$DISCOVERED_NS}"

read -rp "Spanner Deployment Endpoint [${DISCOVERED_ENDPOINT}]: " INPUT_ENDPOINT || INPUT_ENDPOINT=""
TARGET_ENDPOINT="${INPUT_ENDPOINT:-$DISCOVERED_ENDPOINT}"

read -rp "Prometheus Address [${DISCOVERED_PROM}]: " INPUT_PROM || INPUT_PROM=""
TARGET_PROM="${INPUT_PROM:-$DISCOVERED_PROM}"

read -rp "Root Servers per Zone (protected Paxos boundary) [3]: " INPUT_ROOTS || INPUT_ROOTS=""
TARGET_ROOTS="${INPUT_ROOTS:-3}"

read -rp "Safe Scale-Down (Drain tablets before pod removal) [true]: " INPUT_SAFE || INPUT_SAFE=""
TARGET_SAFE="${INPUT_SAFE:-true}"

read -rp "CPU Target Utilization % [65]: " INPUT_CPU || INPUT_CPU=""
TARGET_CPU="${INPUT_CPU:-65}"

read -rp "Storage Target Utilization % [80]: " INPUT_STORAGE || INPUT_STORAGE=""
TARGET_STORAGE="${INPUT_STORAGE:-80}"

echo -e "\n${BOLD}[4/4] Generating ${OUTPUT_FILE}...${NC}"

python3 - << PY_WRITE
import json

with open('${DISCOVERED_FILE}') as f:
    data = json.load(f)

ns = "${TARGET_NS}"
endpoint = "${TARGET_ENDPOINT}"
prom = "${TARGET_PROM}"
roots = int("${TARGET_ROOTS}")
safe = "${TARGET_SAFE}".lower() == "true"
cpu = int("${TARGET_CPU}")
storage = int("${TARGET_STORAGE}")

targets = data.get('targets', [])
if not targets:
    targets = [
        {'name': 'spanner-a', 'zone': 'europe-west4-a', 'replicas': 3},
        {'name': 'spanner-b', 'zone': 'europe-west4-b', 'replicas': 3},
        {'name': 'spanner-c', 'zone': 'europe-west4-c', 'replicas': 3}
    ]

content = f"""# ==============================================================================
# Auto-generated Spanner Omni Autoscaler Configuration
# Generated context: ${CURRENT_CONTEXT}
# ==============================================================================
spannerOmni:
  namespace: "{ns}"
  deploymentEndpoint: "{endpoint}"
  prometheusAddress: "{prom}"
  rootServersPerZone: {roots}
  safeScaleDown: {str(safe).lower()}

targets:
"""

for t in targets:
    zone_str = f"  # Zone: {t.get('zone', 'default')}" if t.get('zone') else ""
    content += f"""  - name: "{t['name']}"{zone_str}
    minReplicas: {roots}
    maxReplicas: {max(roots * 3, 10)}
    cpuTargetPercent: {cpu}
    storageTargetPercent: {storage}

"""

content += """# Unified CronJob Model (Standard across GKE, EKS, AKS, Bare-Metal)
unifiedModel:
  enabled: true
  schedule: "*/2 * * * *"
  configMapName: "autoscaler-config"

# Grafana Dashboard ConfigMap provisioned into monitoring namespace
grafana:
  enabled: true
  namespace: "monitoring"
  extraLabels:
    grafana_dashboard: "1"
"""

with open("${OUTPUT_FILE}", "w") as out:
    out.write(content)

print("[✓] Written values to ${OUTPUT_FILE}")
PY_WRITE

echo -e "\n${GREEN}${BOLD}✓ Successfully generated ${OUTPUT_FILE}!${NC}"
echo -e "\nTo deploy or update the autoscaler pointing to this cluster:"
echo -e "  ${BOLD}helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler -f ${OUTPUT_FILE} -n spanner-autoscaler --create-namespace${NC}"
