terraform {
  required_version = ">= 1.3.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
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

variable "cluster_name" {
  type        = string
  description = "EKS Cluster Name"
  default     = "spanner-omni-eks"
}

data "aws_eks_cluster" "cluster" {
  name = var.cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = var.cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  token                  = data.aws_eks_cluster_auth.cluster.token
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.cluster.endpoint
    token                  = data.aws_eks_cluster_auth.cluster.token
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
  }
}

module "spanner_omni_autoscaler" {
  source = "../../modules/autoscaler"

  namespace         = "spanner-autoscaler"
  spanner_namespace = "spanner-ns"

  spanner_deployment_endpoint = "spanner-service.spanner-ns.svc.cluster.local:15000"
  prometheus_address          = "http://prometheus-service.monitoring.svc.cluster.local:9090"

  root_servers_per_zone = 3
  safe_scale_down       = true

  deployment_model = "unified"
  cronjob_schedule = "*/2 * * * *"

  scaling_targets = [
    {
      name                   = "spanner-a"
      min_replicas           = 3
      max_replicas           = 12
      cpu_target_percent     = 65
      storage_target_percent = 80
    }
  ]
}
