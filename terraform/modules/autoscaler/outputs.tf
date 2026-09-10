output "release_name" {
  value       = helm_release.autoscaler.name
  description = "The name of the deployed Helm release"
}

output "namespace" {
  value       = helm_release.autoscaler.namespace
  description = "The namespace in which the autoscaler was deployed"
}

output "deployment_model" {
  value       = var.deployment_model
  description = "The active autoscaler deployment model (unified or controller)"
}

output "effective_targets" {
  value       = var.scaling_targets
  description = "List of Spanner Omni StatefulSets targeted for autoscaling"
}
