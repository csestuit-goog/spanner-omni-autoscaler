#!/usr/bin/env bash
# ==============================================================================
# Spanner Omni Autoscaler - Interactive Quickstart & Operational Runner
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

log_info() { echo -e "${BLUE}${BOLD}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}${BOLD}[SUCCESS]${NC} $*"; }
log_warn() { echo -e "${YELLOW}${BOLD}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}${BOLD}[ERROR]${NC} $*"; }

print_banner() {
  cat << 'BANNER'
  ____                                      ___                  _ 
 / ___| _ __   __ _ _ __  _ __   ___ _ __  / _ \ _ __ ___  _ __ (_)
 \___ \| '_ \ / _` | '_ \| '_ \ / _ \ '__|| | | | '_ ` _ \| '_ \| |
  ___) | |_) | (_| | | | | | | |  __/ |   | |_| | | | | | | | | | |
 |____/| .__/ \__,_|_| |_|_| |_|\___|_|    \___/|_| |_| |_|_| |_|_|
       |_|            Autoscaler for Kubernetes
BANNER
  echo -e "  ${BOLD}Multi-Cloud Elastic Autoscaling for Google Cloud Spanner Omni${NC}\n"
}

check_prerequisites() {
  log_info "Checking local tools and prerequisites..."
  for cmd in kubectl helm; do
    if ! command -v "$cmd" &>/dev/null; then
      log_warn "Command '$cmd' not found in PATH. Please install $cmd."
    fi
  done
  log_success "Environment check completed."
}

show_menu() {
  echo -e "\n${BOLD}Select an action:${NC}"
  echo "  1) Deploy Autoscaler via Helm (Unified CronJob Model)"
  echo "  2) Deploy Autoscaler via Terraform (GKE / EKS / AKS)"
  echo "  3) Check Autoscaler Status & Pod Logs"
  echo "  4) Forward Grafana Dashboard (:3000)"
  echo "  5) Run Local Unit Tests"
  echo "  6) Exit"
  echo ""
  read -rp "Enter choice [1-6]: " choice

  case "$choice" in
    1) deploy_helm ;;
    2) deploy_terraform ;;
    3) check_status ;;
    4) forward_grafana ;;
    5) run_tests ;;
    6) exit 0 ;;
    *) log_error "Invalid option."; show_menu ;;
  esac
}

deploy_helm() {
  log_info "Deploying Spanner Omni Autoscaler via Helm..."
  helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
    -f my-spanner-values.yaml \
    --namespace spanner-autoscaler \
    --create-namespace
  log_success "Helm release installed. Checking resources..."
  kubectl get cronjobs,pods -n spanner-autoscaler
}

deploy_terraform() {
  echo -e "\nSelect cloud platform target:"
  echo "  1) Google Kubernetes Engine (GKE)"
  echo "  2) Amazon Elastic Kubernetes Service (EKS)"
  echo "  3) Azure Kubernetes Service (AKS)"
  read -rp "Enter platform [1-3]: " plat
  case "$plat" in
    1) TF_DIR="terraform/examples/gke" ;;
    2) TF_DIR="terraform/examples/eks" ;;
    3) TF_DIR="terraform/examples/aks" ;;
    *) log_error "Invalid choice"; return ;;
  esac

  log_info "Initializing Terraform in $TF_DIR..."
  (cd "$TF_DIR" && terraform init && terraform plan)
}

check_status() {
  log_info "Fetching Autoscaler CronJobs & Pods..."
  kubectl get cronjobs,pods,configmaps -n spanner-autoscaler || true
  log_info "Fetching Spanner Omni StatefulSets..."
  kubectl get statefulsets -n spanner-ns || true
}

forward_grafana() {
  log_info "Starting port-forward to Grafana on http://localhost:3000..."
  log_info "Press Ctrl+C to stop port-forwarding."
  kubectl port-forward svc/grafana 3000:80 -n monitoring
}

run_tests() {
  log_info "Running Go unit tests..."
  go test -v ./pkg/scaler/... ./pkg/prometheus/... ./pkg/spanner/... ./pkg/poller/...
}

print_banner
check_prerequisites
show_menu
