locals {
  provisioned_topics = distinct(concat(var.topics, [var.dlq_topic]))
}

resource "helm_release" "kafka" {
  count = var.enabled ? 1 : 0

  name             = var.release_name
  namespace        = var.namespace
  create_namespace = true

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "kafka"
  version    = var.chart_version

  values = [
    yamlencode({
      kraft = {
        enabled = true
      }

      zookeeper = {
        enabled = false
      }

      controller = {
        replicaCount = var.replica_count
        persistence = {
          enabled = var.persistence_enabled
        }
        logPersistence = {
          enabled = var.persistence_enabled
        }
      }

      broker = {
        replicaCount = 0
        persistence = {
          enabled = var.persistence_enabled
        }
        logPersistence = {
          enabled = var.persistence_enabled
        }
      }

      listeners = {
        client = {
          protocol = "PLAINTEXT"
        }
      }

      sasl = {
        enabled = false
      }

      auth = {
        enabled = false
      }

      persistence = {
        enabled = var.persistence_enabled
      }

      logPersistence = {
        enabled = var.persistence_enabled
      }

      provisioning = {
        enabled = true
        topics = [
          for topic in local.provisioned_topics : {
            name              = topic
            partitions        = 1
            replicationFactor = 1
          }
        ]
      }
    })
  ]

  wait = true
}
