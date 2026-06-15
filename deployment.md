# Deployment Guide

## Local Kubernetes With Kind

### Prerequisites

Install:

- Docker
- kind
- kubectl
- Helm

### Start The Local Cluster

  ```bash
  make kind-up

  This creates a Kind cluster, builds the local Synapse image, loads it into Kind, and creates the synapse-platform namespace.

  ### Install Synapse

  make helm-install

  For repeatable deploys after changes, use:

  make helm-upgrade

  ### Smoke Test

  If values-dev.yaml exposes the app through the Kind NodePort mapping, check readiness with:

  curl http://localhost:8080/readyz

  Expected healthy response:

  {
    "status": "healthy",
    "checks": {
      "sqlite": "ok",
      "kafka": "ok"
    }
  }

  ### Render Without Installing

  make helm-template

  Use this before installing when you want to inspect the generated Kubernetes YAML.

  ### Run Helm Tests

  make helm-test

  ### Uninstall Synapse

  make helm-delete

  ### Destroy The Local Cluster

  make kind-down

  ### Full Local Flow

  make kind-up
  make helm-install
  curl http://localhost:8080/readyz
```

Small practical note: `curl http://localhost:8080/readyz` only works directly if your Kind config maps host `8080` to a worker port and
your dev Service is `NodePort`. If the Service is `ClusterIP`, use `kubectl port-forward` instead.