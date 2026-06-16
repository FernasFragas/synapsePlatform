variable "name" {
  type    = string
  default = "synapse"
}

variable "namespace" {
  type = string
}

variable "create_namespace" {
  type    = bool
  default = true
}

variable "chart_path" {
  type = string
}

variable "values" {
  type    = list(string)
  default = []
}

variable "values_files" {
  type    = list(string)
  default = []
}

variable "image_tag" {
  type = string
}

variable "replica_count" {
  type = number
}

variable "kafka_brokers" {
  type = list(string)
}

variable "llm_enabled" {
  type = bool
}

variable "auth_jwt_secret" {
  type      = string
  sensitive = true
}

variable "service_monitor_enabled" {
  type = bool
}

variable "timeout" {
  type    = number
  default = 600
}
