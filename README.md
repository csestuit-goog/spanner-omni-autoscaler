# Google Cloud Spanner Omni Autoscaler for Kubernetes

[![CI](https://github.com/csestuit-goog/spanner-omni-autoscaler/actions/workflows/ci.yml/badge.svg)](https://github.com/csestuit-goog/spanner-omni-autoscaler/actions)
[![Engine: Spanner Omni 2026.r2-beta](https://img.shields.io/badge/Engine-Spanner_Omni_2026.r2--beta-4285F4.svg)](https://cloud.google.com/spanner-omni)
[![Platforms: GKE | EKS | AKS](https://img.shields.io/badge/Platforms-GKE_%7C_EKS_%7C_AKS-34A853.svg)](https://cloud.google.com/kubernetes-engine)
[![Observability: Open--Standard PromQL](https://img.shields.io/badge/Observability-Prometheus_%7C_OTel-EA4335.svg)](https://cloud.google.com/spanner-omni/prometheus-alerts)
[![IaC: Terraform & Helm](https://img.shields.io/badge/IaC-Terraform_%26_Helm-7B42BC.svg)](terraform/)
[![Safety: TrueTime Circuit Breaker](https://img.shields.io/badge/Safety-TrueTime_Circuit_Breaker-FAB005.svg)](DESIGN.md)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Design: Reference Architecture](https://img.shields.io/badge/Design-Production_Architecture-green.svg)](DESIGN.md)

> 🚀 **Field Engineering & Customer Walkthrough**: Presenting to customers or enterprise database architects? See the complete step-by-step presentation script in [**`DEMO_GUIDE.md`**](DEMO_GUIDE.md), inspect the formal engineering architecture in [**`DESIGN.md`**](DESIGN.md), and launch the interactive runner with `./quickstart.sh`!

An enterprise-grade, automated **Kubernetes-native Autoscaler** and **Helm Chart** for **Google Cloud Spanner Omni** StatefulSets across any standard Kubernetes cluster (**GKE**, **Amazon EKS**, and **Azure AKS**).

---

## 📋 Table of Contents
- [🌟 Executive Summary & Key Highlights](#-executive-summary--key-highlights)
- [🎯 Customer & Field Demo Walkthrough (`DEMO_GUIDE.md`)](DEMO_GUIDE.md)
- [🏛️ Architectural Design & System Topology (`DESIGN.md`)](DESIGN.md)
- [🏗️ Project Structure](#-project-structure)
- [🚀 Quickstart & Interactive Runner (`quickstart.sh`)](#-quickstart--interactive-runner-quickstartsh)
- [⚙️ Easy Configuration (Point to Your Instance)](#️-easy-configuration-point-to-your-instance)
- [📊 Open-Standard Observability (Prometheus & OTel)](#-open-standard-observability-prometheus--otel)
- [🚨 Integration with Official Prometheus Alerts](#-integration-with-official-prometheus-alerts)
- [📈 Integrated Grafana Dashboard](#-integrated-grafana-dashboard)
- [🏗️ Terraform & Helm Deployment Guide](#️-terraform--helm-deployment-guide)
- [🔒 Security & Compliance](#-security--compliance)
- [📄 License & Contributing](#-license--contributing)

---

## 🌟 Executive Summary & Key Highlights

This reference implementation solves the core challenges of horizontally autoscaling distributed databases on Kubernetes:

1. **Multi-Cloud Universal Compatibility (GKE / EKS / AKS / Bare-Metal)**:
   - Deployed as a standard Helm chart and Kubernetes Custom Resource/CronJob.
   - Built with zero cloud-provider lock-in or proprietary APIs.
2. **Open-Standard Metrics (Avoids Cloud Monitoring / CloudWatch / Azure Monitor)**:
   - Queries standard **PromQL HTTP API** (`/api/v1/query`) against internal Prometheus or OpenTelemetry (OTel) Collectors.
   - Evaluates native Spanner Omni indicators exported on port `:15012/metrics` (`spanner_cpu_utilization_by_priority_and_category`, `filesystem_size`).
3. **TrueTime Health & Safety Circuit Breaker**:
   - Spanner's serializable transactions and Paxos consensus rely strictly on TrueTime.
   - Automatically freezes scaling actions (`BLOCKED_BY_TRUETIME`) if `TrueTimeUnavailable` or `ClockSlaViolation` alerts trigger.
4. **Root Server Protection**:
   - In Spanner Omni, root servers maintain Raft/Paxos quorums. The autoscaler guarantees that `replicas >= rootServersPerZone` (default 3), ensuring root nodes are never decommissioned.
5. **Safe Scale-Down & Split Draining**:
   - StatefulSets terminate pods from highest to lowest index. The autoscaler decommissions the highest-index server in the Spanner Omni registry (`spanner deployment servers delete`) to migrate tablet splits *before* reducing pod counts.
6. **Dual Deployment Topologies (Adapted from Google Cloud Spanner GKE Guide)**:
   - **Unified CronJob Model (Recommended)**: Poller and Scaler execute as a lightweight Pod on a Cron schedule (`*/2 * * * *`). Zero idle resource usage.
   - **Decoupled Controller Model**: Runs continuously as a Kubernetes Custom Controller.
7. **Integrated Grafana Dashboard**:
   - Pre-packaged dashboard JSON tracking TrueTime drift, CPU targets (65%), storage thresholds (80%/90%), and real-time tablet splits and moves.

---

## 🏗️ Project Structure

```
spanner-omni-autoscaler/
├── README.md                           # 📖 Comprehensive documentation & usage guide
├── DESIGN.md                           # 🏛️ Google-standard architectural design document
├── DEMO_GUIDE.md                       # 🎯 Step-by-step customer demo & field engineering script
├── CONTRIBUTING.md                     # 🛠️ Developer setup & contribution guidelines
├── SECURITY.md                         # 🔒 Security policy & vulnerability reporting
├── LICENSE                             # 📄 Apache 2.0 License
├── quickstart.sh                       # 🚀 Universal interactive CLI & cluster runner
├── my-spanner-values.yaml              # ⚙️ Simple user override values file
├── Dockerfile                          # 🐳 Multi-stage distroless container build
│
├── api/v1alpha1/                       # 🧩 Kubernetes Custom Resource Definitions
│   └── spanneromniautoscaler_types.go  # Golang struct definitions
│
├── cmd/controller/                     # 🚀 Controller Manager Entrypoint
│   └── main.go                         # Multi-mode CLI runner (CronJob / Controller)
│
├── pkg/                                # 📦 Core Autoscaler Logic & Modules
│   ├── controller/                     # K8s reconciliation & StatefulSet patching
│   ├── scaler/                         # Scaling evaluation engine & TrueTime circuit breaker
│   ├── prometheus/                     # Open-Standard PromQL / OTel HTTP client
│   ├── poller/                         # Zero-dependency ConfigMap parser (GKE model)
│   └── spanner/                        # Spanner Omni node registry & safe decommission
│
├── helm/spanner-omni-autoscaler/       # 📦 Production Helm Chart
│   ├── Chart.yaml                      # Chart definition
│   ├── values.yaml                     # Default configuration values
│   └── templates/                      # Kubernetes manifests
│       ├── cronjob/                    # Unified CronJob & ConfigMap templates
│       ├── grafana/                    # Auto-provisioned Grafana Dashboard ConfigMap
│       ├── deployment.yaml             # Decoupled Controller Deployment
│       ├── rbac.yaml                   # Least-privilege RBAC
│       └── serviceaccount.yaml         # In-cluster ServiceAccount
│
├── terraform/                          # 🏗️ Terraform Infrastructure-as-Code
│   ├── modules/autoscaler/             # Reusable Helm-backed autoscaler module
│   └── examples/                       # Production cloud examples
│       ├── gke/main.tf                 # Google Kubernetes Engine deployment
│       ├── eks/main.tf                 # Amazon Elastic Kubernetes Service deployment
│       └── aks/main.tf                 # Azure Kubernetes Service deployment
│
├── grafana/dashboards/                 # 📊 Integrated Grafana Dashboards
│   └── spanner-omni-autoscaler-dashboard.json
│
└── .github/workflows/
    └── ci.yml                          # 🤖 GitHub Actions CI Pipeline
```

---

## 🚀 Quickstart & Interactive Runner (`quickstart.sh`)

Launch the interactive CLI runner:

```bash
chmod +x ./quickstart.sh ./reconfigure.sh
./quickstart.sh
```

Features available directly in the interactive menu:
1. **Auto-Discover & Reconfigure for Existing Cluster**: Automatically scans your Kubernetes cluster, discovers Spanner Omni StatefulSets, zones, and Prometheus endpoints, and generates a ready-to-deploy values file.
2. **Deploy Autoscaler via Helm (Unified Model)**.
3. **Deploy Autoscaler via Terraform (GKE / EKS / AKS)**.
4. **Check Autoscaler CronJob & StatefulSet Status**.
5. **Launch Port-Forward to Grafana (:3000)**.
6. **Run Local Unit Test Suite**.

---

## ⚙️ Easily Pointing to Any Existing Spanner Omni Cluster

### Method 1: Automatic Discovery & Reconfiguration (Recommended)

Run the included cluster reconfigurator:

```bash
./reconfigure.sh [output-values.yaml]
```

This utility:
- Automatically detects the active `kubectl` context.
- Discovers all Spanner Omni StatefulSets (`spanner-a`, `spanner-b`, `spanner-c`, etc.) across any namespace.
- Resolves topology zones (`europe-west4-a`, `us-central1-a`, etc.) from pod templates and node selectors.
- Auto-detects in-cluster Prometheus and Spanner service endpoints (`spanner.spanner-ns.svc.cluster.local:15000`).
- Prompts for confirmation/overrides and outputs a tailored values file (`spanner-omni-cluster.values.yaml`).

### Method 2: Manual Values File Customization

Customize [`my-spanner-values.yaml`](my-spanner-values.yaml) to point to your Spanner Omni instance:

```yaml
spannerOmni:
  namespace: "spanner-ns"
  deploymentEndpoint: "spanner.spanner-ns.svc.cluster.local:15000"
  prometheusAddress: "http://prometheus-service.monitoring.svc.cluster.local:9090"
  rootServersPerZone: 3
  safeScaleDown: true

targets:
  - name: "spanner-a"
    minReplicas: 3
    maxReplicas: 10
    cpuTargetPercent: 65
    storageTargetPercent: 80

  - name: "spanner-b"
    minReplicas: 3
    maxReplicas: 10
    cpuTargetPercent: 65
    storageTargetPercent: 80
```

Deploy or update the autoscaler pointing to your cluster:
```bash
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f my-spanner-values.yaml \
  --namespace spanner-autoscaler \
  --create-namespace
```
  --create-namespace
```

---

## 📊 Open-Standard Observability (Prometheus & OTel)

Strictly avoids cloud-provider proprietary metrics APIs (Google Cloud Monitoring, AWS CloudWatch, Azure Monitor) in favor of the **Prometheus HTTP API (`/api/v1/query`)** and **OpenTelemetry (OTel) Collectors**.

### Core PromQL Queries Evaluated:

| Indicator | Spanner Omni Native PromQL Query | Container Fallback PromQL Query |
|---|---|---|
| **CPU Utilization %** | `(sum(spanner_cpu_utilization_by_priority_and_category{namespace="<ns>", spanner_server=~"<target>-.*"}) * 100) / sum(spanner_available_milligcu{namespace="<ns>", spanner_server=~"<target>-.*"})` | `100 * (sum(rate(container_cpu_usage_seconds_total{namespace="<ns>", pod=~"<target>-[0-9]+", container="spanner"}[2m])) / sum(kube_pod_container_resource_requests{namespace="<ns>", pod=~"<target>-[0-9]+", resource="cpu"}))` |
| **Storage Utilization %** | `(sum(filesystem_size{type="used", namespace="<ns>", spanner_server=~"<target>-.*"}) / sum(filesystem_size{type="total", namespace="<ns>", spanner_server=~"<target>-.*"})) * 100` | `100 * (sum(kubelet_volume_stats_used_bytes{namespace="<ns>", persistentvolumeclaim=~"data-volume-<target>-.*"}) / sum(kubelet_volume_stats_capacity_bytes{namespace="<ns>", persistentvolumeclaim=~"data-volume-<target>-.*"}))` |

---

## 🚨 Integration with Official Prometheus Alerts

Conforms to **[Use Prometheus alerts to monitor Spanner Omni](https://cloud.google.com/spanner-omni/prometheus-alerts)**:

| Alert Rule | Threshold | Autoscaler Action & System Behavior |
|---|---|---|
| `TrueTimeUnavailable` | `true_time_is_available < 1` | **Safety Circuit Breaker (`BLOCKED_BY_TRUETIME`)**: Freezes all scaling actions to protect distributed consensus. |
| `ClockSlaViolation` | `sla_tester_violation_count > 0` | **Safety Circuit Breaker (`BLOCKED_BY_TRUETIME`)**: Halts replica modification until clock drift recovers. |
| `SpannerHighCPUUtilization` | `CPU > 65%` | **Horizontal Scale-Out**: Expands non-root servers proportionally: `desired = ceil(current * (CPU / 65))`. |
| `SpannerStorageUtilizationWarning` | `Storage > 80%` | **Capacity Evaluation**: Flags high disk pressure and prepares partition rebalancing candidates. |
| `SpannerStorageUtilizationCritical` | `Storage > 90%` | **Preemptive Scale-Out**: Adds +1 server immediately so Spanner Omni can split and migrate ranges. |

---

## 📈 Integrated Grafana Dashboard

Conforms to **[Use Grafana dashboards to monitor Spanner Omni](https://cloud.google.com/spanner-omni/grafana-dashboards)**.

The bundled dashboard ([`grafana/dashboards/spanner-omni-autoscaler-dashboard.json`](grafana/dashboards/spanner-omni-autoscaler-dashboard.json)) includes:
- **TrueTime Circuit Breakers**: Status indicators for TrueTime availability, clock SLA violations, and microsecond drift.
- **CPU & Storage Thresholds**: Visual lines at 65% (CPU target), 80% (Storage Warning), and 90% (Storage Critical).
- **Replica Topologies**: Real-time current vs. desired StatefulSet replica counts.
- **Tablet Partitions & Range Movement**: Split, move, and merge rates across nodes.

Automatically provisioned as a Kubernetes ConfigMap for instant discovery by Grafana sidecars.

---

## 🏗️ Terraform & Helm Deployment Guide

### Deploying via Terraform

Use the modular Terraform module across GKE, EKS, or AKS:

```hcl
module "spanner_omni_autoscaler" {
  source = "./terraform/modules/autoscaler"

  namespace         = "spanner-autoscaler"
  spanner_namespace = "spanner-ns"

  spanner_deployment_endpoint = "spanner-service.spanner-ns.svc.cluster.local:15000"
  prometheus_address          = "http://prometheus-service.monitoring.svc.cluster.local:9090"

  root_servers_per_zone = 3
  safe_scale_down       = true
  deployment_model      = "unified"

  scaling_targets = [
    {
      name                   = "spanner-a"
      min_replicas           = 3
      max_replicas           = 15
      cpu_target_percent     = 65
      storage_target_percent = 80
    }
  ]

  enable_grafana_dashboard = true
  grafana_namespace        = "monitoring"
}
```

```bash
cd terraform/examples/gke   # or eks, aks
terraform init
terraform plan
terraform apply
```

---

## 🔒 Security & Compliance

- **Restricted Pod Security Standard**: Runs non-root (`UID 65534`), read-only root filesystem, `capabilities: drop: ["ALL"]`, no privilege escalation.
- **Least-Privilege RBAC**: Scoped strictly to `statefulsets` (patch, get, list) and read-only `pods`/`pvcs`.
- **Zero Ingress Attack Surface**: Operates completely in-cluster via internal ServiceAccount tokens.
- **Open-Source Compliance**: Includes official Google [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [LICENSE](LICENSE) (Apache 2.0).

---

## 📄 License & Contributing

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) and [CONTRIBUTING.md](CONTRIBUTING.md) for details.
