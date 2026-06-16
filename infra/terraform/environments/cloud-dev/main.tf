module "synapse" {
  source = "../../modules/synapse"

  name       = "synapse"
  namespace  = var.namespace
  chart_path = abspath("${path.module}/../../../helm/synapse-platform")

  values = [
    yamlencode({
      kafka = {
        enabled  = false
        topics   = var.topics
        dlqTopic = var.dlq_topic
      }
    }),
  ]

  image_tag               = var.image_tag
  replica_count           = var.replica_count
  kafka_brokers           = var.kafka_brokers
  llm_enabled             = var.llm_enabled
  auth_jwt_secret         = var.jwt_secret
  service_monitor_enabled = var.service_monitor_enabled
}
