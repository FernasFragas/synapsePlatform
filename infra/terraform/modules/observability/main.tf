locals {
  dashboard_json_path         = var.dashboard_json_path != "" ? var.dashboard_json_path : abspath("${path.module}/../../../../grafana-dashboard.json")
  dashboard_configmap_enabled = var.enabled && var.dashboard_enabled && fileexists(local.dashboard_json_path)
}

resource "helm_release" "kube_prometheus_stack" {
  count = var.enabled ? 1 : 0

  name             = var.release_name
  namespace        = var.namespace
  create_namespace = true

  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = var.chart_version

  values = [
    yamlencode({
      prometheus = {
        prometheusSpec = {
          serviceMonitorSelectorNilUsesHelmValues = false
          serviceMonitorSelector = {
            matchLabels = {
              (var.service_monitor_label_name) = var.service_monitor_label_value
            }
          }
          serviceMonitorNamespaceSelector = {}
        }
      }

      grafana = {
        sidecar = {
          dashboards = {
            enabled         = var.dashboard_enabled
            label           = "grafana_dashboard"
            labelValue      = "1"
            searchNamespace = "ALL"
          }
        }
      }
    })
  ]
}

resource "kubernetes_config_map" "synapse_grafana_dashboard" {
  count = local.dashboard_configmap_enabled ? 1 : 0

  metadata {
    name      = "synapse-grafana-dashboard"
    namespace = var.namespace

    labels = {
      grafana_dashboard = "1"
    }
  }

  data = {
    "synapse-dashboard.json" = file(local.dashboard_json_path)
  }

  depends_on = [helm_release.kube_prometheus_stack]
}
