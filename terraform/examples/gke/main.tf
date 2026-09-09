terraform {
  required_version = ">= 1.3.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.9.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.20.0"
    }
  }
}

variable "project_id" {
  type        = string
  description = "GCP Project ID"
  default     = "my-gcp-project"
}

variable "cluster_name" {
  type        = string
  description = "GKE Cluster Name"
  default     = "spanner-omni-cluster"
}

variable "cluster_location" {
  type        = string
  description = "GKE Cluster Zone or Region"
  default     = "us-central1-a"
}

data "google_client_config" "default" {}

data "google_container_cluster" "cluster" {
  name     = var.cluster_name
  location = var.cluster_location
  project  = var.project_id
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.cluster.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = "https://${data.google_container_cluster.cluster.endpoint}"
    token                  = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate)
  }
}

module "spanner_omni_autoscaler" {
  source = "../../modules/autoscaler"

  namespace         = "spanner-autoscaler"
  spanner_namespace = "spanner-ns"

  # Point to Spanner Omni deployment
  spanner_deployment_endpoint = "spanner-service.spanner-ns.svc.cluster.local:15000"
  prometheus_address          = "http://prometheus-service.monitoring.svc.cluster.local:9090"

  # Topology settings
  root_servers_per_zone = 3
  safe_scale_down       = true

  # Unified CronJob topology (as recommended by official GKE guide)
  deployment_model = "unified"
  cronjob_schedule = "*/2 * * * *"

  # Target StatefulSets to scale
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
