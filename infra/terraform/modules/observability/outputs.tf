output "enabled" {
  description = "Whether kube-prometheus-stack is enabled."
  value       = var.enabled
}

output "namespace" {
  description = "Kubernetes namespace where observability components are installed."
  value       = var.namespace
}

output "release_name" {
  description = "Name of the kube-prometheus-stack Helm release."
  value       = var.release_name
}

output "grafana_service_name" {
  description = "Kubernetes Service name for Grafana."
  value       = "${var.release_name}-grafana"
}
