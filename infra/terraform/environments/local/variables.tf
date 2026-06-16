variable "namespace" {
  description = "Namespace for local Synapse resources."
  type        = string
  default     = "synapse-platform"
}

variable "kube_context" {
  description = "Kubeconfig context for the local Kind cluster."
  type        = string
  default     = "kind-synapse-platform"
}

variable "image_tag" {
  description = "Synapse image tag loaded into Kind."
  type        = string
  default     = "dev"
}

variable "replica_count" {
  description = "Synapse replica count for local/dev."
  type        = number
  default     = 1
}

variable "jwt_secret" {
  description = "JWT secret for local/dev. Leave empty to generate one with random_password."
  type        = string
  default     = ""
  sensitive   = true
}

variable "enable_observability" {
  description = "Whether to install kube-prometheus-stack and enable Synapse ServiceMonitor."
  type        = bool
  default     = false
}
