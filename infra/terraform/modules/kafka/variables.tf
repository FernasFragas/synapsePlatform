variable "enabled" {
  description = "Whether to install a local/dev Kafka release. Disable when using managed Kafka."
  type        = bool
}

variable "release_name" {
  description = "Name of the Kafka Helm release."
  type        = string
  default     = "synapse-platform-kafka"
}

variable "namespace" {
  description = "Kubernetes namespace where Kafka is installed."
  type        = string
}

variable "chart_version" {
  description = "Bitnami Kafka chart version constraint."
  type        = string
  default     = "~26.0.0"
}

variable "topics" {
  description = "Kafka topics required by Synapse."
  type        = list(string)
  default     = ["ingestion.raw"]
}

variable "dlq_topic" {
  description = "Kafka dead-letter topic required by Synapse."
  type        = string
  default     = "ingestion.dlq"
}

variable "replica_count" {
  description = "Number of KRaft controller replicas for local/dev Kafka."
  type        = number
  default     = 1
}

variable "persistence_enabled" {
  description = "Whether Kafka persistence is enabled. Defaults to false for local/dev."
  type        = bool
  default     = false
}
