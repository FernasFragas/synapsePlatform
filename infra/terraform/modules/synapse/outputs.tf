locals {
  service_name = "${helm_release.synapse.name}-${basename(var.chart_path)}"
}

output "release_name" {
  description = "Name of the Synapse Helm release."
  value       = helm_release.synapse.name
}

output "namespace" {
  description = "Kubernetes namespace where Synapse is installed."
  value       = helm_release.synapse.namespace
}

output "status" {
  description = "Status of the Synapse Helm release."
  value       = helm_release.synapse.status
}

output "service_name" {
  description = "Kubernetes Service name for Synapse using the chart fullname convention."
  value       = local.service_name
}
