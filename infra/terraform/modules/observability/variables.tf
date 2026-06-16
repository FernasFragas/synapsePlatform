variable "enabled" {
  description = "Whether to install kube-prometheus-stack."
  type        = bool
}

variable "namespace" {
  description = "Kubernetes namespace where observability components are installed."
  type        = string
  default     = "monitoring"
}

variable "release_name" {
  description = "Name of the kube-prometheus-stack Helm release."
  type        = string
  default     = "kube-prometheus-stack"
}

variable "chart_version" {
  description = "kube-prometheus-stack chart version to install."
  type        = string
}

variable "service_monitor_label_name" {
  description = "Label name Prometheus uses to select Synapse ServiceMonitor resources."
  type        = string
  default     = "release"

  validation {
    condition     = var.service_monitor_label_name != ""
    error_message = "service_monitor_label_name must not be empty."
  }
}

variable "service_monitor_label_value" {
  description = "Label value Prometheus uses to select Synapse ServiceMonitor resources."
  type        = string
  default     = "kube-prometheus-stack"

  validation {
    condition     = var.service_monitor_label_value != ""
    error_message = "service_monitor_label_value must not be empty."
  }
}

variable "dashboard_enabled" {
  description = "Whether to create a Grafana dashboard ConfigMap when dashboard JSON exists."
  type        = bool
  default     = false
}

variable "dashboard_json_path" {
  description = "Path to a Synapse Grafana dashboard JSON file. Defaults to grafana-dashboard.json at the repository root."
  type        = string
  default     = ""
}
