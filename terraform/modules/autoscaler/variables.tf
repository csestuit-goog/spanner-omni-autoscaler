variable "release_name" {
  type        = string
  description = "Helm release name for the autoscaler"
  default     = "spanner-omni-autoscaler"
}

variable "namespace" {
  type        = string
  description = "Kubernetes namespace to install the autoscaler"
  default     = "spanner-autoscaler"
}

variable "create_namespace" {
  type        = bool
  description = "Whether to create the namespace if it does not exist"
  default     = true
}

variable "chart_path" {
  type        = string
  description = "Path to the spanner-omni-autoscaler Helm chart"
  default     = ""
}

variable "spanner_namespace" {
  type        = string
  description = "Namespace where Spanner Omni StatefulSets reside"
  default     = "spanner-ns"
}

variable "spanner_deployment_endpoint" {
  type        = string
  description = "Spanner Omni management endpoint (e.g. spanner-service:15000)"
  default     = "spanner-service:15000"
}

variable "prometheus_address" {
  type        = string
  description = "In-cluster Prometheus server or OTel Collector address"
  default     = "http://prometheus-service.monitoring.svc.cluster.local:9090"
}

variable "root_servers_per_zone" {
  type        = number
  description = "Number of root servers per zone (cannot be scaled down)"
  default     = 3
}

variable "safe_scale_down" {
  type        = bool
  description = "Decommission Spanner servers before reducing StatefulSet replica count"
  default     = true
}

variable "scaling_targets" {
  type = list(object({
    name                   = string
    min_replicas           = optional(number, 3)
    max_replicas           = optional(number, 10)
    cpu_target_percent     = optional(number, 65)
    storage_target_percent = optional(number, 80)
  }))
  description = "List of Spanner Omni StatefulSet targets to autoscale"
  default = [
    {
      name                   = "spanner-a"
      min_replicas           = 3
      max_replicas           = 10
      cpu_target_percent     = 65
      storage_target_percent = 80
    }
  ]
}

variable "deployment_model" {
  type        = string
  description = "Deployment topology: 'unified' (CronJob, recommended by GKE guide) or 'controller' (long-running operator)"
  default     = "unified"
}

variable "cronjob_schedule" {
  type        = string
  description = "Cron schedule when deployment_model is unified"
  default     = "*/2 * * * *"
}

variable "enable_grafana_dashboard" {
  type        = bool
  description = "Automatically provision Grafana dashboard ConfigMap for sidecar discovery"
  default     = true
}

variable "grafana_namespace" {
  type        = string
  description = "Namespace where Grafana is installed"
  default     = "monitoring"
}

variable "custom_values" {
  type        = any
  description = "Additional custom values to merge into Helm release"
  default     = {}
}
