terraform {
  required_version = ">= 1.3.0"
  required_providers {
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

locals {
  default_chart_path   = "${path.module}/../../../helm/spanner-omni-autoscaler"
  effective_chart_path = var.chart_path != "" ? var.chart_path : local.default_chart_path

  helm_values = merge({
    spannerOmni = {
      namespace          = var.spanner_namespace
      deploymentEndpoint = var.spanner_deployment_endpoint
      prometheusAddress  = var.prometheus_address
      rootServersPerZone = var.root_servers_per_zone
      safeScaleDown      = var.safe_scale_down
    }
    targets = [
      for t in var.scaling_targets : {
        name                 = t.name
        minReplicas          = t.min_replicas
        maxReplicas          = t.max_replicas
        cpuTargetPercent     = t.cpu_target_percent
        storageTargetPercent = t.storage_target_percent
      }
    ]
    unifiedModel = {
      enabled  = var.deployment_model == "unified"
      schedule = var.cronjob_schedule
    }
    customController = {
      enabled = var.deployment_model == "controller"
    }
    grafana = {
      enabled   = var.enable_grafana_dashboard
      namespace = var.grafana_namespace
    }
    monitoring = {
      namespace = var.grafana_namespace
      prometheus = {
        enabled = var.deploy_prometheus
      }
      grafana = {
        enabled = var.deploy_grafana
      }
    }
  }, var.custom_values)
}

resource "helm_release" "autoscaler" {
  name             = var.release_name
  chart            = local.effective_chart_path
  namespace        = var.namespace
  create_namespace = var.create_namespace

  values = [
    yamlencode(local.helm_values)
  ]
}
