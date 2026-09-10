# Google Cloud Spanner Omni Autoscaler for Kubernetes

[![CI](https://github.com/cloud-gtm/spanner-omni-autoscaler/actions/workflows/ci.yml/badge.svg)](https://github.com/cloud-gtm/spanner-omni-autoscaler/actions)
[![Engine: Spanner Omni 2026.r2-beta](https://img.shields.io/badge/Engine-Spanner_Omni_2026.r2--beta-4285F4.svg)](https://cloud.google.com/spanner-omni)
[![Platforms: GKE | EKS | AKS](https://img.shields.io/badge/Platforms-GKE_%7C_EKS_%7C_AKS-34A853.svg)](https://cloud.google.com/kubernetes-engine)
[![Observability: Open-Standard PromQL](https://img.shields.io/badge/Observability-Prometheus_%7C_OTel-EA4335.svg)](https://cloud.google.com/spanner-omni/prometheus-alerts)
[![Architecture: Reference Design](https://img.shields.io/badge/Design-GTM_Reference_Architecture-FAB005.svg)](DESIGN.md)
[![IaC: Terraform & Helm](https://img.shields.io/badge/IaC-Terraform_%26_Helm-7B42BC.svg)](terraform/)
[![Safety: TrueTime Circuit Breaker](https://img.shields.io/badge/Safety-TrueTime_Circuit_Breaker-34A853.svg)](DESIGN.md)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

> 🚀 **Customer Demo & Field Walkthrough**: Presenting to Google Cloud customers or database architects? See the complete step-by-step presentation script in [**`DEMO_GUIDE.md`**](DEMO_GUIDE.md), inspect the full technical architecture and design in [**`DESIGN.md`**](DESIGN.md), and launch the interactive runner with `./quickstart.sh`!

An enterprise-grade, multi-cloud **Kubernetes-native Autoscaler**, **Helm Chart**, and **Terraform Suite** for **Google Cloud Spanner Omni** StatefulSets across any standard Kubernetes cluster (**Google Kubernetes Engine (GKE)**, **Amazon Elastic Kubernetes Service (AWS EKS)**, and **Azure Kubernetes Service (Azure AKS)**).

---

## 📋 Table of Contents
- [🌟 Executive Summary & Key Highlights](#-executive-summary--key-highlights)
- [🎯 Customer & Field Demo Walkthrough (`DEMO_GUIDE.md`)](DEMO_GUIDE.md)
- [🏛️ Architectural Design & System Topology (`DESIGN.md`)](DESIGN.md)
- [🏗️ Project Structure](#-project-structure)
- [🚀 Quickstart & Interactive Runner (`quickstart.sh`)](#-quickstart--interactive-runner-quickstartsh)
- [⚙️ Easy Configuration & Discovery (`reconfigure.sh`)](#️-easy-configuration--discovery-reconfiguresh)
- [📊 Open-Standard Observability (Prometheus & OTel)](#-open-standard-observability-prometheus--otel)
- [🚨 Integration with Official Prometheus Alerts](#-integration-with-official-prometheus-alerts)
- [📈 Integrated Grafana Dashboard](#-integrated-grafana-dashboard)
- [☁️ Multi-Cloud Deployment: AWS EKS, Azure AKS & GKE](#️-multi-cloud-deployment-aws-eks-azure-aks--gke)
- [🏗️ Terraform Infrastructure Suite (`terraform/README.md`)](terraform/README.md)
- [🔒 Security & Compliance](#-security--compliance)
- [🧪 Automated Test & Quality Guardrails](#-automated-test--quality-guardrails)
- [📄 License & Contributing](#-license--contributing)

---

## 🌟 Executive Summary & Key Highlights

This reference implementation addresses the core technical friction points of horizontally autoscaling distributed databases on Kubernetes:

1. **Multi-Cloud Universal Compatibility (GKE / EKS / AKS / Bare-Metal)**:
   - Deployed as a standard Helm chart and Kubernetes Custom Resource/CronJob.
   - Built with zero cloud-provider lock-in or proprietary APIs.
2. **Open-Standard Metrics (Avoids Proprietary Cloud Monitoring Lock-in)**:
   - Queries standard **PromQL HTTP API** (`/api/v1/query`) against internal Prometheus or OpenTelemetry (OTel) Collectors.
   - Evaluates native Spanner Omni indicators exported on port `:15012/metrics` (`spanner_cpu_utilization_by_priority_and_category`, `filesystem_size`).
3. **TrueTime Health & Safety Circuit Breaker**:
   - Spanner's serializable transactions and Paxos consensus rely strictly on bounded clock uncertainty.
   - Automatically freezes scaling actions (`BLOCKED_BY_TRUETIME`) if `TrueTimeUnavailable` or `ClockSlaViolation` alerts trigger.
4. **Root Server & Paxos Quorum Protection**:
   - In Spanner Omni, root servers maintain Raft/Paxos quorums. The autoscaler guarantees that `replicas >= rootServersPerZone` (default 3), ensuring root nodes are never decommissioned.
5. **Safe Scale-Down & Tablet Draining Protocol**:
   - StatefulSets terminate pods from highest to lowest index. The autoscaler decommissions the highest-index server in the Spanner Omni registry (`spanner deployment servers delete`) to migrate tablet splits *before* reducing pod counts.
6. **Dual Deployment Topologies (Adapted from Google Cloud Spanner GKE Guide)**:
   - **Unified CronJob Model (Recommended)**: Poller and Scaler execute as a lightweight Pod on a Cron schedule (`*/2 * * * *`). Zero idle resource usage.
   - **Decoupled Controller Model**: Runs continuously as a Kubernetes Custom Controller.
7. **Integrated Prometheus & Grafana Dashboard Stack**:
   - Pre-packaged dashboard tracking TrueTime drift, CPU targets (65%), storage thresholds (80%/90%), and real-time tablet splits and moves.

---

## 🏗️ Project Structure

```
spanner-omni-autoscaler/
├── cmd/
│   └── scaler/
│       └── main.go                 # 🚀 Unified Autoscaler CLI & CronJob entrypoint
├── pkg/
│   ├── poller/                     # ⏱️ PromQL evaluation & metric polling
│   ├── prometheus/                 # 📊 Prometheus client & fallback metric drivers
│   ├── scaler/                     # ⚙️ Scaling algorithm & TrueTime circuit breaker
│   └── spanner/                    # 🛡️ In-pod server decommission & tablet split evictor
├── helm/
│   └── spanner-omni-autoscaler/    # 📦 Production Helm chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           ├── cronjob.yaml        # Unified CronJob topology
│           ├── rbac.yaml           # Least-privilege RBAC
│           ├── prometheus/         # Optional bundled Prometheus server & alert rules
│           └── grafana/            # Optional bundled Grafana server & dashboards
├── helm-values-examples/           # ☁️ Multi-Cloud pre-configured values
│   ├── values-aws-eks.yaml         # AWS EKS override
│   ├── values-azure-aks.yaml       # Azure AKS override
│   └── values-gke-regional.yaml    # GKE Regional override
├── terraform/                      # 🏗️ Terraform Infrastructure Suite
│   ├── README.md                   # 📖 Terraform deployment guide
│   ├── modules/autoscaler/         # Core Helm-based autoscaler module
│   └── examples/                   # Ready-to-run examples (GKE, EKS, AKS)
├── grafana/
│   └── spanner-omni-autoscaler-dashboard.json # 📈 Production Grafana dashboard
├── scripts/
│   ├── build_image.sh              # Container build script
│   └── sql_shell.sh                # Interactive SQL shell with pre-loaded dataset
├── quickstart.sh                   # ⚡ Interactive deployment wizard
├── reconfigure.sh                  # 🔄 Zero-downtime cluster reconfiguration script
├── DEMO_GUIDE.md                   # 🎯 Complete customer presentation walkthrough
├── DESIGN.md                       # 🏛️ Architecture & technical design specification
├── CONTRIBUTING.md                 # 🤝 Contribution guidelines
├── SECURITY.md                     # 🔒 Security policy
└── LICENSE                         # 📄 Apache 2.0 License
```

---

## 🚀 Quickstart & Interactive Runner (`quickstart.sh`)

Launch the automated discovery and setup wizard:

```bash
chmod +x quickstart.sh
./quickstart.sh
```

The script will:
1. Auto-discover active Spanner Omni StatefulSets across all namespaces.
2. Probe for in-cluster Prometheus and Grafana instances.
3. Generate a tailored `my-spanner-values.yaml` file.
4. Deploy the autoscaler with a single confirmation.

---

## ⚙️ Easy Configuration & Discovery (`reconfigure.sh`)

Reconfigure the autoscaler to target an existing Spanner Omni cluster at any time:

```bash
./reconfigure.sh \
  --namespace spanner-ns \
  --statefulsets spanner-a,spanner-b,spanner-c \
  --prometheus-url http://prometheus-server.monitoring.svc.cluster.local:9090 \
  --target-cpu 65 \
  --min-replicas 3 \
  --max-replicas 15
```

---

## 📊 Open-Standard Observability (Prometheus & OTel)

Spanner Omni nodes export standard metrics on port `:15012/metrics`. The autoscaler evaluates these open PromQL expressions:

| Metric | PromQL Expression | Purpose |
| :--- | :--- | :--- |
| **CPU Utilization %** | `(sum(spanner_cpu_utilization_by_priority_and_category) * 100) / sum(spanner_available_milligcu)` | Primary scaling indicator |
| **TrueTime Availability** | `true_time_is_available` | Circuit breaker (< 1 freezes scaling) |
| **Clock SLA Violations** | `sla_tester_violation_count` | Circuit breaker (> 0 freezes scaling) |
| **Storage Utilization %** | `sum(filesystem_size{type="used"}) / sum(filesystem_size{type="total"}) * 100` | Storage capacity alert |

---

## 🚨 Integration with Official Prometheus Alerts

The autoscaler integrates directly with official [Google Cloud Spanner Omni Prometheus Alerts](https://cloud.google.com/spanner-omni/prometheus-alerts):
- **`TrueTimeUnavailable`**: Triggered when clock synchronization is lost. Scaling is blocked to prevent distributed transaction corruption.
- **`ClockSlaViolation`**: Triggered when TrueTime bounds are violated. Halts all horizontal mutations.
- **`SpannerHighCPUUtilization`**: Warns at > 65% CPU.
- **`SpannerStorageUtilizationCritical`**: Critical alert at > 90% disk usage.

---

## 📈 Integrated Grafana Dashboard

The bundled dashboard (`grafana/spanner-omni-autoscaler-dashboard.json`) displays:
- **TrueTime Circuit Breaker Status**: Visual badges indicating TrueTime availability and clock uncertainty in microseconds.
- **CPU Scaling Target Line**: Cluster CPU utilization with dynamic 65% target line.
- **Per-Zone Server Replicas**: Real-time replica count for each availability zone.
- **Tablet Movement & Split Rates**: Live tablet splits and Paxos migrations.

```bash
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

---

## ☁️ Multi-Cloud Deployment: AWS EKS, Azure AKS & GKE

Pre-configured Helm values are provided in `helm-values-examples/`:

### Amazon Elastic Kubernetes Service (AWS EKS)
```bash
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f ./helm-values-examples/values-aws-eks.yaml \
  --namespace spanner-autoscaler \
  --create-namespace
```

### Azure Kubernetes Service (Azure AKS)
```bash
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f ./helm-values-examples/values-azure-aks.yaml \
  --namespace spanner-autoscaler \
  --create-namespace
```

### Google Kubernetes Engine (GKE Regional)
```bash
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f ./helm-values-examples/values-gke-regional.yaml \
  --namespace spanner-autoscaler \
  --create-namespace
```

---

## 🏗️ Terraform Infrastructure Suite (`terraform/README.md`)

Deploy the autoscaler via Terraform:

```bash
cd terraform/examples/gke
terraform init
terraform apply -var="project_id=my-project" -var="cluster_name=spanner-cluster" -var="location=europe-west1"
```

See [**`terraform/README.md`**](terraform/README.md) for full module documentation and input variable references.

---

## 🔒 Security & Compliance

- **Restricted Pod Security Standard**: Runs non-root (`UID 65534`), read-only root filesystem, `capabilities: drop: ["ALL"]`, no privilege escalation.
- **Least-Privilege RBAC**: Scoped strictly to `statefulsets` (patch, get, list) and read-only `pods`/`pvcs`.
- **Zero Ingress Attack Surface**: Operates completely in-cluster via internal ServiceAccount tokens.
- **Open-Source Compliance**: Includes official Google [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [LICENSE](LICENSE) (Apache 2.0).

---

## 🧪 Automated Test & Quality Guardrails

Run unit tests and linting locally:

```bash
# Run unit test suite
go test -v ./pkg/scaler/... ./pkg/prometheus/... ./pkg/spanner/... ./pkg/poller/...

# Lint Helm chart
helm lint helm/spanner-omni-autoscaler/

# Validate Terraform formatting
cd terraform && terraform fmt -check -recursive
```

---

## 📄 License & Contributing

Licensed under the **Apache License, Version 2.0**. See [LICENSE](LICENSE) and [CONTRIBUTING.md](CONTRIBUTING.md) for details.
