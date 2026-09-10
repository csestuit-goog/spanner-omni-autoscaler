# Spanner Omni Autoscaler: Customer Demo & Field Engineering Guide

*   **Audience:** Google Cloud Customers, Field Engineers, SREs, Database Architects
*   **Target Systems:** Spanner Omni on GKE / AWS EKS / Azure AKS
*   **Estimated Duration:** 15–20 minutes

---

## 🎯 Complete Customer Demo, DBA Presentation & Field Engineering Guide

This guide provides field teams, customer engineers, and solutions architects with a step-by-step presentation script and live execution walkthrough for demonstrating horizontal autoscaling of **Google Cloud Spanner Omni** on Kubernetes.

---

## 📑 Table of Contents
- [1. Executive Summary & Value Proposition](#1-executive-summary--value-proposition)
- [2. Feature & Capability Breakdown: Managed Spanner vs. Spanner Omni](#2-feature--capability-breakdown-managed-spanner-vs-spanner-omni)
- [3. Demo Architecture & Cluster Topology](#3-demo-architecture--cluster-topology)
- [4. Step-by-Step Customer Demo Walkthrough (5 Stages)](#4-step-by-step-customer-demo-walkthrough-5-stages)
- [5. How DBAs & SREs Configure and Reconfigure in Production](#5-how-dbas--sres-configure-and-reconfigure-in-production)
- [6. Running Outside Google Cloud (AWS EKS & Azure AKS)](#6-running-outside-google-cloud-aws-eks--azure-aks)
- [7. Tear Down & Resource Cleanup](#7-tear-down--resource-cleanup)
- [📞 Support & Contributing](#-support--contributing)

---

## 1. Executive Summary & Value Proposition

Horizontal autoscaling for distributed databases like Google Cloud Spanner Omni requires intelligence far beyond standard Kubernetes Horizontal Pod Autoscalers (HPA). 

**Key Customer Value Highlights**:
1. **Multi-Cloud Universal Compatibility**: Runs anywhere Kubernetes runs (GKE, EKS, AKS, On-Prem) with zero proprietary cloud APIs.
2. **Open-Standard Observability**: Queries standard Prometheus PromQL HTTP endpoints instead of locking into cloud-provider monitoring tools.
3. **Database-Aware Safety**: Automatically halts scaling operations if TrueTime clock synchronization degrades (`BLOCKED_BY_TRUETIME`).
4. **Guaranteed Paxos Quorum**: Protects root coordination servers from horizontal scale-down.
5. **Zero-Downtime Split Eviction**: Decommissions servers via Spanner Omni's internal management API (`spanner deployment servers delete`) to migrate tablet splits *before* reducing pod counts.

---

## 2. Feature & Capability Breakdown: Managed Spanner vs. Spanner Omni

| Dimension | Managed Cloud Spanner | Spanner Omni on Kubernetes | Autoscaler Capability |
| :--- | :--- | :--- | :--- |
| **Compute Primitive** | Compute Units / Node Count | Kubernetes `StatefulSet` Pods | Dynamic ordinal pod scaling |
| **API Control Plane** | Google Cloud Spanner Admin API | Kubernetes API + In-Pod Management CLI | Native `kubectl` + `spanner deployment` |
| **Metric Source** | Google Cloud Monitoring API | Prometheus / OTel (`:15012/metrics`) | Open-Standard PromQL queries |
| **Consensus Safety** | Managed by Google Infrastructure | Customer-Managed Multi-Zone Quorum | Enforces `rootServersPerZone` minimum |
| **Clock Synchronization** | Google Atomic Clocks & GPS | Customer NTP / PTP or Google Cloud NTP | Real-time TrueTime circuit breaker |
| **Split Migration** | Background Cloud Migration | In-Pod Server Eviction Protocol | Orchestrates split draining prior to scale-in |

---

## 3. Demo Architecture & Cluster Topology

```mermaid
flowchart LR
    Prometheus["Prometheus / OTel (:9090)"] -->|PromQL| Autoscaler["Autoscaler Engine (Go)"]
    Autoscaler -->|1. TrueTime Health Check| Prometheus
    Autoscaler -->|2. Split Drain (:15000)| SpannerPod["Spanner Management Pod"]
    Autoscaler -->|3. Patch Replicas| StatefulSet["StatefulSet: spanner-a/b/c"]
    Prometheus -->|Live Visuals| Grafana["Grafana Dashboard (:3000)"]
```

---

## 4. Step-by-Step Customer Demo Walkthrough (5 Stages)

### Stage 1: Deploying the Autoscaler via Helm or Terraform (2 Minutes)

Demonstrate how simple it is to deploy the autoscaler:

```bash
# Option A: Deploy via Helm
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f my-spanner-values.yaml \
  --namespace spanner-autoscaler \
  --create-namespace

# Option B: Run the interactive quickstart wizard
./quickstart.sh
```

*Customer Talking Point*: "Notice that no cloud-specific credentials or IAM permissions were required. The autoscaler communicates strictly via standard Kubernetes RBAC and in-cluster Prometheus queries."

---

### Stage 2: Inspecting TrueTime & Cluster Health in Grafana (3 Minutes)

Port-forward Grafana to inspect cluster health:

```bash
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

Open `http://localhost:3000` (credentials: `admin` / `prom-operator`) and review the **"Spanner Omni - Autoscaler & Cluster Health"** dashboard:
1. **TrueTime Circuit Breaker**: Highlights TrueTime Availability (1.0) and Clock SLA Violations (0).
2. **CPU Target Gauges**: Shows current cluster CPU utilization against the 65% target.
3. **Active Server Replicas**: Displays current replica counts per zone (`spanner-a`, `spanner-b`, `spanner-c`).

---

### Stage 3: Automated Scale-Out Under Workload (5 Minutes)

Simulate workload pressure using the pre-loaded sample dataset:

```bash
# Open interactive SQL shell to run benchmark queries
./scripts/sql_shell.sh ecommerce_omni
```

Watch the autoscaler logs and Grafana dashboard:
```bash
kubectl logs -n spanner-autoscaler -l app.kubernetes.io/name=spanner-omni-autoscaler -f
```

*Observed Log Output*:
```
[INFO] Cluster CPU utilization: 78.4% (Threshold: 65.0%)
[INFO] TrueTime Health: OK (available=1, violations=0)
[INFO] Calculating desired replicas: 3 -> 7
[INFO] Patching StatefulSet spanner-ns/spanner-a replicas: 3 -> 7
[INFO] Scale-out complete. New servers registered in Spanner Omni topology.
```

---

### Stage 4: TrueTime Safety Circuit Breaker Demonstration (3 Minutes)

Explain how the autoscaler protects data consistency during clock instability:
- If `true_time_is_available < 1` or `sla_tester_violation_count > 0`, the autoscaler **immediately halts all scaling operations**.
- Emphasize to the customer: "Unlike naive autoscalers that kill pods during network or clock instability, our autoscaler acts as an intelligent circuit breaker, preserving Paxos consensus and transaction integrity."

---

### Stage 5: Safe Coordinated Scale-In with Split Eviction (5 Minutes)

When the traffic surge subsides:
1. The autoscaler waits for the scale-in cooldown period (10 minutes) to prevent thrashing.
2. Rather than immediately shrinking the StatefulSet, the engine issues:
   ```bash
   spanner deployment servers delete spanner-a-6 --zone=zone-a
   ```
3. Spanner Omni safely migrates tablet splits away from `spanner-a-6`.
4. The autoscaler then decrements the StatefulSet replica count from 7 to 6.

---

## 5. How DBAs & SREs Configure and Reconfigure in Production

Point the autoscaler at any existing Spanner Omni cluster in seconds:

```bash
./reconfigure.sh \
  --namespace my-spanner-ns \
  --statefulsets spanner-east,spanner-central,spanner-west \
  --prometheus-url http://prometheus.monitoring.svc.cluster.local:9090 \
  --min-replicas 3 \
  --max-replicas 12 \
  --target-cpu 70
```

---

## 6. Running Outside Google Cloud (AWS EKS & Azure AKS)

Pre-configured Helm value overrides are provided for every major cloud:
- **AWS EKS**: `helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler -f ./helm-values-examples/values-aws-eks.yaml`
- **Azure AKS**: `helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler -f ./helm-values-examples/values-azure-aks.yaml`
- **GKE Regional**: `helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler -f ./helm-values-examples/values-gke-regional.yaml`

---

## 7. Tear Down & Resource Cleanup

```bash
# Uninstall autoscaler release
helm uninstall spanner-autoscaler -n spanner-autoscaler

# If using Terraform
cd terraform/examples/gke && terraform destroy
```

---

## 📞 Support & Contributing

For questions, issues, or contributions, please refer to [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).
