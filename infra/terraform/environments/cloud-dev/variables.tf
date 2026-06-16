variable "cluster_endpoint" {
  description = "Kubernetes API server endpoint for the target cloud-dev cluster."
  type        = string
  sensitive   = true
}

variable "cluster_token" {
  description = "Bearer token used to authenticate to the target cloud-dev cluster."
  type        = string
  sensitive   = true
}

variable "cluster_ca_certificate" {
  description = "Base64-encoded Kubernetes cluster CA certificate."
  type        = string
  sensitive   = true
}

variable "namespace" {
  description = "Namespace for Synapse resources."
  type        = string
  default     = "synapse-platform"
}

variable "image_tag" {
  description = "Synapse image tag to deploy."
  type        = string
}

variable "replica_count" {
  description = "Synapse replica count."
  type        = number
  default     = 2
}

variable "jwt_secret" {
  description = "JWT secret for Synapse auth."
  type        = string
  sensitive   = true
}

variable "kafka_brokers" {
  description = "Managed Kafka bootstrap brokers for cloud-dev."
  type        = list(string)
}

variable "topics" {
  description = "Managed Kafka topics consumed by Synapse."
  type        = list(string)
  default     = ["ingestion.raw"]
}

variable "dlq_topic" {
  description = "Managed Kafka dead-letter topic used by Synapse."
  type        = string
  default     = "ingestion.dlq"
}

variable "llm_enabled" {
  description = "Whether LLM features are enabled."
  type        = bool
  default     = false
}

variable "service_monitor_enabled" {
  description = "Whether to create the Synapse ServiceMonitor."
  type        = bool
  default     = true
}

variable "postgres_connection_placeholder" {
  description = "Reserved for future Postgres wiring once the app/chart supports Postgres."
  type        = string
  default     = ""
  sensitive   = true
}
