terraform {
  required_version = ">= 1.3.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.0.0"
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

variable "resource_group_name" {
  type        = string
  description = "Azure Resource Group"
  default     = "spanner-omni-rg"
}

variable "cluster_name" {
  type        = string
  description = "AKS Cluster Name"
  default     = "spanner-omni-aks"
}

provider "azurerm" {
  features {}
}

data "azurerm_kubernetes_cluster" "cluster" {
  name                = var.cluster_name
  resource_group_name = var.resource_group_name
}

provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.cluster.kube_config[0].host
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = data.azurerm_kubernetes_cluster.cluster.kube_config[0].host
    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_certificate)
    client_key             = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].client_key)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.cluster.kube_config[0].cluster_ca_certificate)
  }
}

module "spanner_omni_autoscaler" {
  source = "../../modules/autoscaler"

  namespace         = "spanner-autoscaler"
  spanner_namespace = "spanner-ns"

  spanner_deployment_endpoint = "spanner.spanner-ns.svc.cluster.local:15000"
  prometheus_address          = "http://prometheus-service.monitoring.svc.cluster.local:9090"

  root_servers_per_zone = 3
  safe_scale_down       = true

  deployment_model = "unified"
  cronjob_schedule = "*/2 * * * *"

  # Deploy integrated monitoring if AKS cluster lacks Prometheus/Grafana
  deploy_prometheus = false
  deploy_grafana    = false
  grafana_namespace = "monitoring"

  scaling_targets = [
    {
      name                   = "spanner-a"
      min_replicas           = 3
      max_replicas           = 12
      cpu_target_percent     = 65
      storage_target_percent = 80
    },
    {
      name                   = "spanner-b"
      min_replicas           = 3
      max_replicas           = 12
      cpu_target_percent     = 65
      storage_target_percent = 80
    },
    {
      name                   = "spanner-c"
      min_replicas           = 3
      max_replicas           = 12
      cpu_target_percent     = 65
      storage_target_percent = 80
    }
  ]
}

