locals {
  service_monitor_label_name  = "release"
  service_monitor_label_value = "kube-prometheus-stack"
  topics                      = ["ingestion.raw"]
  dlq_topic                   = "ingestion.dlq"
  jwt_secret                  = var.jwt_secret != "" ? var.jwt_secret : random_password.jwt_secret[0].result
}

resource "random_password" "jwt_secret" {
  count = var.jwt_secret == "" ? 1 : 0

  length  = 32
  special = false
}

resource "kubernetes_namespace" "synapse" {
  metadata {
    name = var.namespace

    labels = {
      "app.kubernetes.io/name"    = "synapse-platform"
      "app.kubernetes.io/part-of" = "synapse-platform"
    }
  }
}

module "kafka" {
  source = "../../modules/kafka"

  enabled       = true
  namespace     = kubernetes_namespace.synapse.metadata[0].name
  topics        = local.topics
  dlq_topic     = local.dlq_topic
  replica_count = 1

  depends_on = [kubernetes_namespace.synapse]
}

module "observability" {
  source = "../../modules/observability"

  enabled                     = var.enable_observability
  namespace                   = "monitoring"
  chart_version               = "~75.0.0"
  service_monitor_label_name  = local.service_monitor_label_name
  service_monitor_label_value = local.service_monitor_label_value
  dashboard_enabled           = var.enable_observability
}

module "synapse" {
  source = "../../modules/synapse"

  name             = "synapse"
  namespace        = kubernetes_namespace.synapse.metadata[0].name
  create_namespace = false
  chart_path       = abspath("${path.module}/../../../helm/synapse-platform")

  values_files = [
    abspath("${path.module}/../../../helm/synapse-platform/values-dev.yaml"),
  ]

  values = [
    yamlencode({
      kafka = {
        enabled = false
      }

      observability = {
        serviceMonitor = {
          labels = var.enable_observability ? {
            (local.service_monitor_label_name) = local.service_monitor_label_value
          } : {}
        }
      }
    }),
  ]

  image_tag               = var.image_tag
  replica_count           = var.replica_count
  kafka_brokers           = module.kafka.bootstrap_servers
  llm_enabled             = false
  auth_jwt_secret         = local.jwt_secret
  service_monitor_enabled = var.enable_observability

  depends_on = [
    kubernetes_namespace.synapse,
    module.kafka,
    module.observability,
  ]
}
