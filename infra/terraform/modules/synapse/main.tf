resource "helm_release" "synapse" {
  name             = var.name
  namespace        = var.namespace
  create_namespace = var.create_namespace

  chart = var.chart_path

  values = concat(
    [for values_file in var.values_files : file(values_file)],
    var.values,
  )

  set {
    name  = "image.tag"
    value = var.image_tag
  }

  set {
    name  = "replicaCount"
    value = tostring(var.replica_count)
  }

  set_list {
    name  = "kafka.brokers"
    value = var.kafka_brokers
  }

  set {
    name  = "llm.enabled"
    value = tostring(var.llm_enabled)
  }

  set_sensitive {
    name  = "auth.jwt.secret"
    value = var.auth_jwt_secret
  }

  set {
    name  = "observability.serviceMonitor.enabled"
    value = tostring(var.service_monitor_enabled)
  }

  wait    = true
  timeout = var.timeout
}
