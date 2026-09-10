# Spanner Omni Autoscaler: Customer Demo & Field Engineering Guide

*   **Audience:** Google Cloud Customers, Field Engineers, SREs, Database Architects
*   **Target Systems:** Spanner Omni on GKE / EKS / AKS
*   **Estimated Duration:** 15–20 minutes

---

## 🎯 Demo Objectives

In this walkthrough, you will demonstrate to customers:
1. **Multi-Cloud Portability**: Deploying an intelligent autoscaler on Kubernetes with zero cloud lock-in.
2. **Open-Standard Observability**: Querying standard PromQL metrics exported directly by Spanner Omni nodes.
3. **Database-Aware Safety (TrueTime Circuit Breaker)**: Showing how the autoscaler halts actions when TrueTime SLA is violated.
4. **Automated Scale-Out Under Load**: Simulating CPU pressure and watching the autoscaler expand non-root servers.
5. **Safe Scale-Down & Tablet Draining**: Demonstrating ordinal server decommissioning before StatefulSet pod reductions.
6. **Live Grafana Dashboards**: Monitoring TrueTime uncertainty, CPU targets, and tablet moves in real time.

---

## 📋 Pre-Demo Checklist

- [ ] Kubernetes cluster connected (`kubectl get nodes`).
- [ ] Spanner Omni running (`kubectl get statefulsets -n spanner-ns`).
- [ ] Prometheus running (`kubectl get svc -n monitoring`).
- [ ] Port-forwarding available for Grafana and Spanner Omni Console.

---

## 🚀 Step-by-Step Demo Script

### Step 1: Deploy the Autoscaler via Helm or Terraform

Show the customer how easy it is to target their Spanner Omni deployment:

```bash
# Option A: Deploy via Helm
helm upgrade --install spanner-autoscaler ./helm/spanner-omni-autoscaler \
  -f my-spanner-values.yaml \
  --namespace spanner-autoscaler \
  --create-namespace

# Verify autoscaler is running
kubectl get cronjobs,pods -n spanner-autoscaler
```

*Talking Point*: "Unlike generic Kubernetes HPAs that blindly kill pods, this autoscaler understands Spanner Omni's distributed architecture, consensus quorum, and tablet migration."

---

### Step 2: Access the Integrated Grafana Dashboard

Port-forward Grafana to inspect real-time metrics:

```bash
kubectl port-forward svc/grafana 3000:80 -n monitoring
```

Open `http://localhost:3000` and view the **"Spanner Omni - Autoscaler & Cluster Health"** dashboard:
- Point out the **TrueTime Circuit Breaker panel**: Both TrueTime Available and Clock SLA Violations are healthy (Green).
- Point out the **CPU Utilization panel**: Displays the 65% `SpannerHighCPUUtilization` alert threshold line.
- Point out the **StatefulSet Replicas panel**: Currently at baseline (e.g. 3 replicas).

---

### Step 3: Triggering a Scale-Out Event

Simulate query load on Spanner Omni using the SQL benchmark runner or load generator:

```bash
# Check current CPU utilization via PromQL
curl -s "http://localhost:9090/api/v1/query?query=(sum(spanner_cpu_utilization_by_priority_and_category)*100)/sum(spanner_available_milligcu)" | jq .
```

When CPU utilization exceeds 65%:
1. The autoscaler detects `SpannerHighCPUUtilization` (> 65%).
2. Evaluator calculates: `desired = ceil(current * (CPU / 65))`.
3. The autoscaler patches the StatefulSet:
   ```bash
   kubectl get statefulset spanner-a -n spanner-ns -w
   ```
4. Point out in Grafana: The **Tablet Splits & Moves** graph begins ticking up as Spanner Omni dynamically redistributes ranges across the new server!

---

### Step 4: Demonstrating the Safe Scale-Down Protocol

Once load subsides, explain what happens during scale-in:
1. Explain the danger of standard HPA: "Standard Kubernetes kills the highest pod immediately, which can cause transaction timeouts if data splits are active."
2. Show the autoscaler logs:
   ```bash
   kubectl logs -n spanner-autoscaler -l app.kubernetes.io/name=spanner-omni-autoscaler --tail=50
   ```
3. Highlight the sequence:
   - `[Spanner Admin] Draining and decommissioning candidate server spanner-a-4...`
   - Spanner Omni moves tablet splits to remaining nodes.
   - `[Autoscaler] Patching StatefulSet replicas to 3.`
   - Pod `spanner-a-4` terminates safely with zero data impact.

---

### Step 5: TrueTime Circuit Breaker Demonstration

Explain how Spanner Omni preserves ACID external consistency:
- "If node clock drift exceeds the SLA, scaling is immediately frozen (`BLOCKED_BY_TRUETIME`). The autoscaler never compromises database consistency for capacity."

---

## 🏁 Demo Wrap-Up

Summarize key customer value props:
- **Zero Lock-In**: Works uniformly across GKE, AWS EKS, Azure AKS, and Bare Metal.
- **Enterprise Safe**: Root server protection + TrueTime circuit breaker + data split draining.
- **Turnkey**: Deployed in minutes with Helm or Terraform.
