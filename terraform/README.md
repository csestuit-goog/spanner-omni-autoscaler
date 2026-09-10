# Spanner Omni Autoscaler Terraform Infrastructure Suite

This directory provides production-ready, multi-cloud **Terraform** modules to deploy the **Spanner Omni Autoscaler** across **Google Kubernetes Engine (GKE)**, **Amazon Elastic Kubernetes Service (AWS EKS)**, and **Azure Kubernetes Service (Azure AKS)**.

---

## 🏗️ Architecture & Modules

The configuration is structured into a reusable core module and platform-specific deployment examples:

```
terraform/
├── README.md                  # 📖 This documentation file
├── modules/
│   └── autoscaler/            # ⚙️ Core Autoscaler Module (Helm release, namespace, PromQL config)
│       ├── main.tf            # Helm release resource definition
│       ├── variables.tf       # Configurable scaling thresholds, namespaces, Prometheus endpoints
│       └── outputs.tf         # Helm release status, release name, namespace
└── examples/
    ├── gke/                   # 🌐 Google Kubernetes Engine (Regional / Zonal)
    │   └── main.tf            # GKE provider configuration and module instantiation
    ├── eks/                   # ☁️ Amazon Elastic Kubernetes Service
    │   └── main.tf            # AWS EKS provider configuration and module instantiation
    └── aks/                   # 🔷 Azure Kubernetes Service
        └── main.tf            # Azure AKS provider configuration and module instantiation
```

---

## 🚀 Quickstart Deployment

### 1. Prerequisites
- [Terraform >= 1.5.0](https://www.terraform.io/downloads.html) installed.
- Appropriate cloud CLI installed and authenticated:
  - **GKE**: `gcloud auth application-default login`
  - **AWS EKS**: `aws configure`
  - **Azure AKS**: `az login`
- Target Kubernetes cluster with Spanner Omni deployed.

### 2. Deploy to Google Kubernetes Engine (GKE)

```bash
cd terraform/examples/gke
terraform init
terraform fmt -check
terraform validate
terraform apply -var="project_id=my-gcp-project" -var="cluster_name=spanner-gke-cluster" -var="location=europe-west1"
```

### 3. Deploy to Amazon Elastic Kubernetes Service (AWS EKS)

```bash
cd terraform/examples/eks
terraform init
terraform fmt -check
terraform validate
terraform apply -var="cluster_name=my-spanner-eks-cluster" -var="aws_region=eu-central-1"
```

### 4. Deploy to Azure Kubernetes Service (Azure AKS)

```bash
cd terraform/examples/aks
terraform init
terraform fmt -check
terraform validate
terraform apply -var="resource_group_name=my-aks-rg" -var="cluster_name=my-spanner-aks-cluster"
```

---

## ⚙️ Module Inputs (`modules/autoscaler`)

| Name | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `release_name` | `string` | `"spanner-omni-autoscaler"` | Helm release name. |
| `namespace` | `string` | `"spanner-autoscaler"` | Kubernetes namespace for the autoscaler. |
| `create_namespace` | `bool` | `true` | Create namespace if it does not already exist. |
| `spanner_namespace` | `string` | `"spanner-ns"` | Namespace where Spanner Omni StatefulSets reside. |
| `statefulset_names` | `list(string)` | `["spanner-a", "spanner-b", "spanner-c"]` | Names of the Spanner Omni StatefulSets to autoscale. |
| `min_replicas` | `number` | `3` | Minimum replicas per zone (enforces root server quorum). |
| `max_replicas` | `number` | `15` | Maximum replicas per zone. |
| `target_cpu_utilization` | `number` | `65` | Target CPU utilization percentage (0-100). |
| `prometheus_address` | `string` | `"http://prometheus-server.monitoring.svc.cluster.local:9090"` | In-cluster Prometheus HTTP query endpoint. |
| `enable_integrated_monitoring` | `bool` | `false` | Provision bundled Prometheus and Grafana stack. |

---

## 🔒 Security & Compliance

- **Restricted Pod Security Standard**: Runs unprivileged (UID 65534), read-only root filesystem, `capabilities: drop: ["ALL"]`.
- **RBAC Isolation**: ServiceAccount permissions strictly scoped to `statefulsets` (patch, get, list) and read-only `pods`/`pvcs`.
- **Zero Ingress Attack Surface**: Operates completely within the cluster network via ServiceAccount tokens.

---

## 🧹 Destroying Resources

```bash
terraform destroy
```
