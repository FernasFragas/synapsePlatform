locals {
  bootstrap_servers = var.enabled ? ["${var.release_name}:9092"] : []
}

output "enabled" {
  description = "Whether the local/dev Kafka release is enabled."
  value       = var.enabled
}

output "release_name" {
  description = "Name of the Kafka Helm release."
  value       = var.release_name
}

output "bootstrap_servers" {
  description = "Kafka bootstrap servers for local/dev Synapse configuration."
  value       = local.bootstrap_servers
}

output "topics" {
  description = "Kafka topics required by Synapse."
  value       = var.topics
}

output "dlq_topic" {
  description = "Kafka dead-letter topic required by Synapse."
  value       = var.dlq_topic
}
